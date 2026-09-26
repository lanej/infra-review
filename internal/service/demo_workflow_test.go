package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lanej/statecraft/internal/adapters/mock"
	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
	"github.com/lanej/statecraft/internal/service"
)

type demo struct {
	service *service.DemoWorkflow
	store   *mock.ReviewStore
	review  domain.Review
	now     time.Time
}

func newDemo(t *testing.T, scenario string) *demo {
	t.Helper()
	d := &demo{store: mock.NewReviewStore(), now: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}
	d.service = service.NewDemoWorkflow(d.store, mock.Planner{}, mock.Policies{}, mock.Policies{}, func() time.Time { return d.now })
	seed, err := mock.SeedReview(scenario, d.now)
	if err != nil {
		t.Fatal(err)
	}
	d.review, err = d.service.Create(context.Background(), seed)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func (d *demo) act(t *testing.T, a domain.ReviewAction, violation string) domain.Review {
	t.Helper()
	r, err := d.service.Act(context.Background(), d.review.ID, domain.WorkflowCommand{Action: a, ExpectedVersion: d.review.Version, ViolationID: violation, Reason: "Reviewed migration", Evidence: "Demo recovery exercise"})
	if err != nil {
		t.Fatal(err)
	}
	d.review = r
	return r
}
func (d *demo) denied(t *testing.T, a domain.ReviewAction) {
	t.Helper()
	_, err := d.service.Act(context.Background(), d.review.ID, domain.WorkflowCommand{Action: a, ExpectedVersion: d.review.Version})
	if !errors.Is(err, service.ErrDenied) {
		t.Fatalf("%s: expected denied, got %v", a, err)
	}
}
func TestAcceptanceAndApprovalAreSeparateThenApplyAndVerify(t *testing.T) {
	d := newDemo(t, "review")
	d.denied(t, domain.ActionApprove)
	d.denied(t, domain.ActionApply)
	v := violationID(t, d.review, "data-protection")
	d.act(t, domain.ActionRequestAcceptance, v)
	d.denied(t, domain.ActionApprove)
	d.act(t, domain.ActionGrantAcceptance, v)
	if d.review.CurrentApproval() {
		t.Fatal("acceptance became approval")
	}
	if len(d.review.Policy.Violations) != 2 || violationID(t, d.review, "data-protection") == "" {
		t.Fatal("acceptance erased the violation")
	}
	d.denied(t, domain.ActionApply)
	d.act(t, domain.ActionApprove, "")
	d.act(t, domain.ActionApply, "")
	if d.review.State != "applied" || d.review.Verification != "pending" {
		t.Fatal("execution was mistaken for verification")
	}
	d.denied(t, domain.ActionApply)
	d.act(t, domain.ActionVerify, "")
	if d.review.State != "verified" || d.review.Verification != "state_matches" {
		t.Fatal("verification not recorded")
	}
}
func TestReplanInvalidatesApprovalAndAcceptanceOnSameCommit(t *testing.T) {
	d := newDemo(t, "review")
	v := violationID(t, d.review, "data-protection")
	d.act(t, domain.ActionRequestAcceptance, v)
	d.act(t, domain.ActionGrantAcceptance, v)
	d.act(t, domain.ActionApprove, "")
	oldPlan := d.review.Plan.ID
	oldEvaluation := d.review.Policy.ID
	d.act(t, domain.ActionPlan, "")
	if d.review.Plan.ID == oldPlan || d.review.Policy.ID == oldEvaluation || d.review.CurrentApproval() {
		t.Fatal("replan preserved authorization")
	}
	if len(d.review.PlanHistory) != 1 || len(d.review.Decisions) != 1 || len(d.review.Acceptances) != 1 {
		t.Fatal("replan lost history")
	}
	d.denied(t, domain.ActionApply)
	d.denied(t, domain.ActionApprove)
}
func TestStaleAndIncompleteEvidenceCannotAuthorize(t *testing.T) {
	for _, scenario := range []string{"stale", "incomplete", "unplanned"} {
		t.Run(scenario, func(t *testing.T) {
			d := newDemo(t, scenario)
			d.denied(t, domain.ActionApprove)
			d.denied(t, domain.ActionApply)
			d.act(t, domain.ActionPlan, "")
			if d.review.Plan.CommitSHA != d.review.HeadSHA || d.review.Policy.Coverage != "complete" {
				t.Fatal("new plan did not recover current evidence")
			}
		})
	}
}
func TestAcceptanceExpiryIsCheckedAtDispatch(t *testing.T) {
	d := newDemo(t, "review")
	v := violationID(t, d.review, "data-protection")
	d.act(t, domain.ActionRequestAcceptance, v)
	d.act(t, domain.ActionGrantAcceptance, v)
	d.act(t, domain.ActionApprove, "")
	d.now = d.now.Add(2 * time.Hour)
	d.denied(t, domain.ActionApply)
	r, err := d.service.Get(context.Background(), d.review.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Acceptances[0].Status != "expired" {
		t.Fatal("expired acceptance still shown as active")
	}
	if !r.CurrentApproval() {
		t.Fatal("expiry erased the historical plan approval")
	}
}
func TestRequestChangesSupersedesApproval(t *testing.T) {
	d := newDemo(t, "ready")
	d.act(t, domain.ActionApprove, "")
	d.act(t, domain.ActionRequestChanges, "")
	d.denied(t, domain.ActionApply)
	if d.review.CurrentApproval() {
		t.Fatal("older approval remained current")
	}
}
func TestPartialApplyRequiresPlanForRemainingWork(t *testing.T) {
	d := newDemo(t, "partial")
	d.denied(t, domain.ActionApply)
	d.denied(t, domain.ActionVerify)
	d.act(t, domain.ActionPlan, "")
	for _, change := range d.review.Changes {
		if change.RootID == "api" {
			t.Fatal("recovery replanned an already applied change")
		}
	}
	if len(d.review.Changes) != 2 || d.review.CurrentApproval() {
		t.Fatal("invalid recovery proposal")
	}
	d.act(t, domain.ActionApprove, "")
	d.act(t, domain.ActionApply, "")
	if d.review.Roots[0].ApplyStatus != "no_changes" {
		t.Fatal("recovery reapplied API changes")
	}
}
func TestAcceptanceRequiresEvidenceAndPermittedViolation(t *testing.T) {
	d := newDemo(t, "review")
	for _, cmd := range []domain.WorkflowCommand{
		{Action: domain.ActionGrantAcceptance, ViolationID: violationID(t, d.review, "data-protection")},
		{Action: domain.ActionRequestAcceptance, ViolationID: violationID(t, d.review, "data-protection")},
		{Action: domain.ActionRequestAcceptance, ViolationID: violationID(t, d.review, "network-access"), Reason: "reason", Evidence: "evidence"},
		{Action: domain.ActionRequestAcceptance, ViolationID: "another-plan", Reason: "reason", Evidence: "evidence"},
	} {
		cmd.ExpectedVersion = d.review.Version
		if _, err := d.service.Act(context.Background(), d.review.ID, cmd); err == nil {
			t.Fatal("invalid acceptance succeeded")
		}
	}
	r, _ := d.service.Get(context.Background(), d.review.ID)
	if r.Version != d.review.Version || len(r.Acceptances) != 0 {
		t.Fatal("failed command partially committed")
	}
}
func TestStoreVersionPreventsDuplicateCommandsAndIsolatesSessions(t *testing.T) {
	d := newDemo(t, "ready")
	seed, _ := mock.SeedReview("ready", d.now)
	other, _ := d.service.Create(context.Background(), seed)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := d.service.Act(context.Background(), d.review.ID, domain.WorkflowCommand{Action: domain.ActionApprove, ExpectedVersion: d.review.Version})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ports.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	r, _ := d.service.Get(context.Background(), other.ID)
	if r.Version != 1 || len(r.Decisions) != 0 {
		t.Fatal("one session changed another")
	}
	r.Changes[0].Properties[0].After = "mutated"
	fresh, _ := d.service.Get(context.Background(), other.ID)
	if fresh.Changes[0].Properties[0].After == "mutated" {
		t.Fatal("store returned aliased data")
	}
}
func TestUnknownPolicyCoverageCannotPermitApproval(t *testing.T) {
	for _, mutate := range []func(*domain.Review){func(r *domain.Review) { r.Policy.Results = nil }, func(r *domain.Review) { r.Policy.Results[0].Outcome = "indeterminate" }, func(r *domain.Review) { r.Policy.PlanSetID = "another-plan" }, func(r *domain.Review) { r.Policy.PolicySetID = "unexpected-version" }} {
		d := newDemo(t, "ready")
		r, err := d.store.Update(context.Background(), d.review.ID, d.review.Version, func(r *domain.Review) error { mutate(r); return nil })
		if err != nil {
			t.Fatal(err)
		}
		d.review = r
		d.denied(t, domain.ActionApprove)
	}
}

func violationID(t *testing.T, r domain.Review, policy string) string {
	t.Helper()
	for _, v := range r.Policy.Violations {
		if v.PolicyID == policy {
			return v.ID
		}
	}
	t.Fatalf("missing violation for %s", policy)
	return ""
}
func TestReplanRestoresFailedRootAndPreservesRecoveryAcrossReplans(t *testing.T) {
	d := newDemo(t, "incomplete")
	if len(d.review.Changes) != 3 || len(d.review.Plan.ChangeIDs) != 3 {
		t.Fatal("failed root advertised evidence")
	}
	d.act(t, domain.ActionPlan, "")
	if len(d.review.Changes) != 4 || len(d.review.Policy.Violations) != 2 {
		t.Fatal("successful replan omitted recovered root")
	}
	d = newDemo(t, "partial")
	d.act(t, domain.ActionPlan, "")
	d.act(t, domain.ActionPlan, "")
	if len(d.review.Changes) != 2 || len(d.review.Findings) != 1 {
		t.Fatal("repeated replan restored applied changes or old findings")
	}
}
