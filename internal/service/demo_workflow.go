package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

var ErrInvalid = errors.New("invalid workflow input")
var ErrDenied = errors.New("action unavailable")

// DemoWorkflow simulates execution only. It intentionally has no Planner,
// Executor, SourceControl, credentials, or production mutation capability.
type DemoWorkflow struct {
	store    ports.WorkflowStore
	planner  ports.DemoPlanner
	policies ports.PolicyEvaluator
	actions  ports.ActionPolicy
	now      func() time.Time
}

func NewDemoWorkflow(store ports.WorkflowStore, planner ports.DemoPlanner, policies ports.PolicyEvaluator, actions ports.ActionPolicy, now func() time.Time) *DemoWorkflow {
	return &DemoWorkflow{store, planner, policies, actions, now}
}
func (s *DemoWorkflow) Create(ctx context.Context, r domain.Review) (domain.Review, error) {
	if !r.Demo {
		return domain.Review{}, ErrInvalid
	}
	r, err := s.store.Create(ctx, r)
	if err != nil {
		return r, err
	}
	return s.present(ctx, r)
}
func (s *DemoWorkflow) Get(ctx context.Context, id string) (domain.Review, error) {
	r, err := s.store.GetReview(ctx, id)
	if err != nil {
		return r, err
	}
	return s.present(ctx, r)
}
func (s *DemoWorkflow) present(ctx context.Context, r domain.Review) (domain.Review, error) {
	now := s.now()
	// Legacy findings remain a projection of the current assessment.
	r.Findings = []domain.Finding{}
	if r.Policy != nil {
		for _, v := range r.Policy.Violations {
			address := ""
			for _, c := range r.Changes {
				if c.ID == v.ResourceID {
					address = c.Address
					break
				}
			}
			r.Findings = append(r.Findings, domain.Finding{ID: v.ID, Severity: v.Severity, Category: "policy", Title: v.Title, ResourceAddress: address, Blocking: v.Blocking})
		}
	}
	r.Actions = []domain.ActionDecision{}
	for i := range r.Acceptances {
		a := &r.Acceptances[i]
		expiry, err := time.Parse(time.RFC3339, a.ExpiresAt)
		if a.Status == "granted" && (err != nil || !now.Before(expiry)) {
			a.Status = "expired"
		}
	}
	for _, a := range []domain.ReviewAction{domain.ActionPlan, domain.ActionApprove, domain.ActionRequestChanges, domain.ActionApply, domain.ActionVerify} {
		d, err := s.actions.EvaluateAction(ctx, domain.ActionPolicyInput{Review: r, Action: a, Now: now})
		if err != nil {
			return domain.Review{}, err
		}
		r.Actions = append(r.Actions, d)
	}
	return r, nil
}

func (s *DemoWorkflow) Act(ctx context.Context, id string, cmd domain.WorkflowCommand) (domain.Review, error) {
	if len(cmd.Reason) > 4000 || len(cmd.Evidence) > 4000 {
		return domain.Review{}, fmt.Errorf("%w: comments must be at most 4,000 characters", ErrInvalid)
	}
	r, err := s.store.Update(ctx, id, cmd.ExpectedVersion, func(r *domain.Review) error {
		now := s.now()
		at := now.UTC().Format(time.RFC3339)
		d, err := s.actions.EvaluateAction(ctx, domain.ActionPolicyInput{Review: *r, Action: cmd.Action, ViolationID: cmd.ViolationID, Now: now})
		if err != nil {
			return err
		}
		if d.Outcome != "permitted" {
			return fmt.Errorf("%w: %s", ErrDenied, d.Reason)
		}
		event := func(title, detail string) {
			r.History = append(r.History, domain.ReviewEvent{Title: title, Detail: detail, CreatedAt: at})
		}
		switch cmd.Action {
		case domain.ActionPlan:
			number := 1
			if r.Plan != nil {
				number = r.Plan.Number + 1
				r.PlanHistory = append(r.PlanHistory, *r.Plan)
			}
			changes, err := s.planner.ChangesForPlan(ctx, *r)
			if err != nil {
				return err
			}
			r.Changes = changes
			r.Plan = &domain.PlanSnapshot{ID: fmt.Sprintf("plan-%d", number), Number: number, CommitSHA: r.HeadSHA, Digest: fmt.Sprintf("synthetic:%s:plan-%d", r.HeadSHA, number), RootIDs: []string{}, ChangeIDs: []string{}, CreatedAt: at}
			for _, c := range r.Changes {
				r.Plan.ChangeIDs = append(r.Plan.ChangeIDs, c.ID)
			}
			for i := range r.Roots {
				root := &r.Roots[i]
				root.Status = "planned"
				root.ApplyStatus = "not_started"
				root.Log = "Synthetic plan completed for " + root.Name + ". No infrastructure was changed."
				r.Plan.RootIDs = append(r.Plan.RootIDs, root.ID)
				r.Attempts = append(r.Attempts, domain.ExecutionRecord{ID: fmt.Sprintf("%s-plan-%s", r.Plan.ID, root.ID), PlanSetID: r.Plan.ID, RootID: root.ID, Operation: "plan", Status: "planned", CreatedAt: at, Log: root.Log})
			}
			e, err := s.policies.EvaluatePlan(ctx, domain.PlanPolicyInput{Review: *r, Now: now})
			if err != nil {
				return err
			}
			r.Policy = &e
			r.State = "ready_for_review"
			r.Verification = "not_started"
			event(fmt.Sprintf("Plan %d collected", number), "All four roots assessed. Earlier approvals and acceptances do not cover this new plan.")
		case domain.ActionRequestChanges, domain.ActionApprove:
			if cmd.Action == domain.ActionRequestChanges && strings.TrimSpace(cmd.Reason) == "" {
				return fmt.Errorf("%w: explain the requested changes", ErrInvalid)
			}
			decision := "approved"
			r.State = "approved"
			if cmd.Action == domain.ActionRequestChanges {
				decision = "changes_requested"
				r.State = "changes_requested"
			}
			r.Decisions = append(r.Decisions, domain.ReviewDecision{Actor: "alex", Decision: decision, PlanSetID: r.Plan.ID, CommitSHA: r.HeadSHA, EvaluationID: r.Policy.ID, CreatedAt: at, Message: strings.TrimSpace(cmd.Reason), Source: "statecraft-demo"})
			event("Alex "+strings.ReplaceAll(decision, "_", " "), r.Plan.ID+" · "+r.HeadSHA+" · "+cmd.Reason)
		case domain.ActionRequestAcceptance:
			if strings.TrimSpace(cmd.Reason) == "" || strings.TrimSpace(cmd.Evidence) == "" {
				return fmt.Errorf("%w: rationale and recovery evidence are required", ErrInvalid)
			}
			var version string
			for _, v := range r.Policy.Violations {
				if v.ID == cmd.ViolationID {
					version = v.PolicyVersion
				}
			}
			r.Acceptances = append(r.Acceptances, domain.ViolationAcceptance{ID: fmt.Sprintf("acceptance-%d", len(r.Acceptances)+1), ViolationID: cmd.ViolationID, PlanSetID: r.Plan.ID, EvaluationID: r.Policy.ID, PolicyVersion: version, Status: "requested", RequestedBy: "alex", Reason: strings.TrimSpace(cmd.Reason), Evidence: strings.TrimSpace(cmd.Evidence), CreatedAt: at})
			event("Acceptance requested", "Awaiting a simulated database-owner decision. Approval is still required.")
		case domain.ActionGrantAcceptance:
			for i := len(r.Acceptances) - 1; i >= 0; i-- {
				a := &r.Acceptances[i]
				if a.ViolationID == cmd.ViolationID && a.Status == "requested" && a.EvaluationID == r.Policy.ID {
					a.Status = "granted"
					a.AuthorizedBy = "morgan · database owner"
					a.ExpiresAt = now.Add(time.Hour).UTC().Format(time.RFC3339)
					break
				}
			}
			event("Morgan accepted the violation", "Simulated database-owner authorization · valid for one hour · scoped to "+r.Plan.ID)
		case domain.ActionApply:
			for i := range r.Roots {
				root := &r.Roots[i]
				changed := false
				for _, c := range r.Changes {
					if c.RootID == root.ID {
						changed = true
					}
				}
				root.ApplyStatus = "no_changes"
				if changed {
					root.ApplyStatus = "applied"
				}
				root.Log = "Synthetic apply complete. This simulation did not contact any infrastructure provider."
				r.Attempts = append(r.Attempts, domain.ExecutionRecord{ID: r.Plan.ID + "-apply-" + root.ID, PlanSetID: r.Plan.ID, RootID: root.ID, Operation: "apply", Status: root.ApplyStatus, CreatedAt: at, Log: root.Log})
			}
			r.State = "applied"
			r.Verification = "pending"
			event("Simulated apply completed", "All changed roots applied. Resulting-state verification is pending.")
		case domain.ActionVerify:
			r.State = "verified"
			r.Verification = "state_matches"
			event("Simulated state matches the plan", "Operational health and data recovery are not verified by this demo.")
		}
		return nil
	})
	if err != nil {
		return domain.Review{}, err
	}
	return s.present(ctx, r)
}
