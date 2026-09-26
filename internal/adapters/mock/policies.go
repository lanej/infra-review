package mock

import (
	"context"
	"fmt"
	"sort"

	"github.com/lanej/statecraft/internal/domain"
)

// Policies supplies deterministic sample rules, not OPA or production authority.
type Policies struct{}

func (Policies) EvaluatePlan(_ context.Context, input domain.PlanPolicyInput) (domain.PolicyEvaluation, error) {
	r := input.Review
	e := domain.PolicyEvaluation{ID: "evaluation-" + r.Plan.ID, PlanSetID: r.Plan.ID, PolicySetID: "demo-policy-set-v1", Status: "completed", Coverage: "complete", EvaluatedAt: input.Now.UTC().Format("2006-01-02T15:04:05Z"), Results: []domain.PolicyResult{}, Violations: []domain.PolicyViolation{}}
	db := domain.PolicyResult{ID: "data-protection", Version: "1", Title: "Protect production data", Outcome: "satisfied"}
	network := domain.PolicyResult{ID: "network-access", Version: "1", Title: "Review connectivity changes", Outcome: "not_applicable"}
	for _, root := range r.Roots {
		if root.Status != "planned" {
			e.Coverage = "incomplete"
			if root.ID == "network" {
				network.Outcome = "indeterminate"
			}
		}
	}
	for _, c := range r.Changes {
		if c.ID == "c3" && c.Action == "replace" {
			db.Outcome = "violated"
			e.Violations = append(e.Violations, domain.PolicyViolation{ID: e.ID + "-database", PolicyID: db.ID, PolicyVersion: db.Version, ResourceID: c.ID, Title: "Production database will be replaced", Severity: "critical", Blocking: true, AcceptanceAllowed: true, Explanation: "The proposed name change destroys the current database server and creates a new one.", Consequence: "Applications may lose database connectivity; existing data needs a migration or recovery procedure.", Unknown: "Migration readiness and recovery health have not been independently verified.", Evidence: []string{"name: prod-postgres → prod-postgres-v2", c.SourcePath, "Synthetic plan evidence · " + r.Plan.ID}})
		}
		if c.ID == "c2" {
			network.Outcome = "violated"
			e.Violations = append(e.Violations, domain.PolicyViolation{ID: e.ID + "-network", PolicyID: network.ID, PolicyVersion: network.Version, ResourceID: c.ID, Title: "Database access becomes more restrictive", Severity: "high", Explanation: "Only the application subnet will retain inbound access on port 5432.", Consequence: "Backup or administrative clients outside that subnet may lose connectivity.", Unknown: "External clients are not fully represented in the dependency evidence.", Evidence: []string{"10.0.0.0/8 → 10.24.8.0/24", c.SourcePath}})
		}
	}
	sort.SliceStable(e.Violations, func(i, j int) bool { return e.Violations[i].Blocking && !e.Violations[j].Blocking })
	e.Results = []domain.PolicyResult{db, network}
	return e, nil
}

func (Policies) EvaluateAction(_ context.Context, input domain.ActionPolicyInput) (domain.ActionDecision, error) {
	r, a := input.Review, input.Action
	d := domain.ActionDecision{Action: a, Outcome: "denied"}
	deny := func(reason string) (domain.ActionDecision, error) { d.Reason = reason; return d, nil }
	permit := func(reason string) (domain.ActionDecision, error) {
		d.Outcome = "permitted"
		d.Reason = reason
		return d, nil
	}
	if !r.Demo {
		return deny("This workflow only operates on isolated demo reviews.")
	}
	if a == domain.ActionPlan {
		if r.State == "applied" || r.State == "verified" {
			return deny("Verify the applied proposal before starting a new change.")
		}
		return permit("Create a new plan for all four roots. Earlier decisions stay in history.")
	}
	if a == domain.ActionVerify {
		if r.State != "applied" {
			return deny("All changed roots must finish applying before verification.")
		}
		return permit("Compare the simulated resulting state with the applied plan.")
	}
	if r.Plan == nil {
		return deny("Run a plan to collect evidence for this review.")
	}
	if r.HeadSHA == "" || r.Plan.ID == "" || r.Plan.CommitSHA != r.HeadSHA || r.State == "stale" {
		return deny("The commit changed. Run a new plan before making this decision.")
	}
	if r.State == "partial" {
		return deny("Some infrastructure changed. Re-plan the remaining work before another apply.")
	}
	if r.State == "applied" || r.State == "verified" {
		return deny("This plan has already been applied.")
	}
	if r.Policy == nil || r.Policy.Status != "completed" || r.Policy.Coverage != "complete" || r.Policy.PlanSetID != r.Plan.ID {
		return deny("Planning or policy assessment is incomplete. Re-plan the failed roots.")
	}
	if r.Policy.ID == "" || r.Policy.PolicySetID != "demo-policy-set-v1" {
		return deny("The assessment is not bound to the current demo policies.")
	}
	seen := map[string]bool{}
	for _, result := range r.Policy.Results {
		if result.Outcome != "satisfied" && result.Outcome != "violated" && result.Outcome != "not_applicable" {
			return deny("A required policy result is indeterminate.")
		}
		seen[result.ID] = result.Version == "1"
	}
	if !seen["data-protection"] || !seen["network-access"] {
		return deny("Required policy results are missing.")
	}
	for _, root := range r.Roots {
		if root.Status != "planned" {
			return deny("Every expected root needs a current successful plan.")
		}
	}
	if a == domain.ActionRequestChanges {
		return permit("Record concerns against this exact plan.")
	}
	if a == domain.ActionRequestAcceptance || a == domain.ActionGrantAcceptance {
		for _, v := range r.Policy.Violations {
			if v.ID != input.ViolationID {
				continue
			}
			if !v.AcceptanceAllowed {
				return deny("This policy does not permit acceptance.")
			}
			if r.AcceptanceFor(v, input.Now) != nil {
				return deny("This violation already has an active acceptance.")
			}
			pending := false
			for _, ac := range r.Acceptances {
				if ac.ViolationID == v.ID && ac.EvaluationID == r.Policy.ID && ac.Status == "requested" {
					pending = true
				}
			}
			if a == domain.ActionGrantAcceptance && !pending {
				return deny("Request acceptance with rationale and recovery evidence first.")
			}
			if a == domain.ActionRequestAcceptance && pending {
				return deny("A database-owner decision is already pending.")
			}
			return permit("Demo database-owner authorization is scoped to this violation for one hour.")
		}
		return deny("The selected violation is not part of the current assessment.")
	}
	if a != domain.ActionApprove && a != domain.ActionApply {
		return deny("Unknown workflow action.")
	}
	for _, v := range r.Policy.Violations {
		if v.Blocking && r.AcceptanceFor(v, input.Now) == nil {
			return deny(fmt.Sprintf("Resolve or obtain current acceptance for: %s.", v.Title))
		}
	}
	if a == domain.ActionApply && !r.CurrentApproval() {
		return deny("Approve this exact plan and assessment before applying.")
	}
	if a == domain.ActionApprove && r.CurrentApproval() {
		return deny("This exact plan already has a current approval.")
	}
	return permit("All requirements for this simulated action are satisfied.")
}
