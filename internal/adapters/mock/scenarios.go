package mock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lanej/statecraft/internal/domain"
)

func SeedReview(scenario string, now time.Time) (domain.Review, error) {
	switch scenario {
	case "review", "unplanned", "ready", "incomplete", "stale", "expired", "partial":
	default:
		return domain.Review{}, errors.New("unknown demo scenario")
	}
	r, _ := baseReview("pr-1842")
	r.Demo = true
	r.Scenario = scenario
	r.Decisions = []domain.ReviewDecision{}
	r.Acceptances = []domain.ViolationAcceptance{}
	r.Attempts = []domain.ExecutionRecord{}
	r.History = []domain.ReviewEvent{}
	r.PlanHistory = []domain.PlanSnapshot{}
	r.Verification = "not_started"
	names := []string{"API node pool", "Database access rule", "Primary database", "Database diagnostics"}
	props := [][]domain.PropertyChange{
		{{Name: "node_count", Before: "3", After: "5"}, {Name: "vm_size", Before: "Standard_D4s_v5", After: "Standard_D4s_v5"}},
		{{Name: "source_address_prefix", Before: "10.0.0.0/8", After: "10.24.8.0/24"}, {Name: "destination_port_range", Before: "5432", After: "5432"}},
		{{Name: "name", Before: "prod-postgres", After: "prod-postgres-v2"}, {Name: "sku_name", Before: "GP_Standard_D2s_v3", After: "GP_Standard_D2s_v3"}},
		{{Name: "enabled_log", Before: "Not configured", After: "PostgreSQLLogs"}},
	}
	for i := range r.Changes {
		c := &r.Changes[i]
		c.Name = names[i]
		c.Properties = props[i]
		c.SourcePath = "azure/prod/" + c.RootID + "/main.tf"
		c.PlanText = fmt.Sprintf("# %s\n# action: %s\n", c.Address, c.Action)
		c.SourceText = fmt.Sprintf("resource %q {\n", c.Address)
		for _, p := range c.Properties {
			c.PlanText += fmt.Sprintf("  %s: %s → %s\n", p.Name, p.Before, p.After)
			c.SourceText += fmt.Sprintf("  %s = %q\n", p.Name, p.After)
		}
		c.SourceText += "}\n# Illustrative configuration excerpt"
		c.Relationships = []domain.ResourceRelationship{{ResourceID: "c3", Label: "Primary database", Kind: "uses", Evidence: "Illustrative reference in application configuration"}}
		if c.ID == "c3" {
			c.Relationships = []domain.ResourceRelationship{{ResourceID: "c1", Label: "API node pool", Kind: "used by", Evidence: "Illustrative application database connection"}, {Label: "Connection secret", Kind: "referenced by", Evidence: "Unchanged context · not included in this plan"}}
		}
	}
	if scenario == "ready" {
		r.Changes = append(r.Changes[:2], r.Changes[3:]...)
		r.Findings = r.Findings[1:]
	}
	for i := range r.Roots {
		r.Roots[i].ApplyStatus = "not_started"
		r.Roots[i].Log = "Synthetic plan complete. Evidence collected for " + r.Roots[i].Name + "."
	}
	if scenario == "unplanned" {
		r.State = "unplanned"
		r.Changes = []domain.Change{}
		r.Findings = []domain.Finding{}
		for i := range r.Roots {
			r.Roots[i].Status = "pending"
			r.Roots[i].Log = "No plan has run."
		}
		return r, nil
	}
	r.Plan = &domain.PlanSnapshot{ID: "plan-7", Number: 7, CommitSHA: r.HeadSHA, Digest: "synthetic:plan-7", RootIDs: []string{"api", "network", "observability", "identity"}, CreatedAt: now.Add(-10 * time.Minute).UTC().Format(time.RFC3339), ChangeIDs: []string{}}
	if scenario == "incomplete" {
		r.State = "incomplete"
		r.Roots[1].Status = "failed"
		r.Roots[1].Log = "Error: provider authentication failed.\nNo current plan artifact was produced.\nThe other roots remain available for investigation."
		r.Changes = append(r.Changes[:1], r.Changes[2:]...)
	}
	for _, c := range r.Changes {
		r.Plan.ChangeIDs = append(r.Plan.ChangeIDs, c.ID)
	}
	e, _ := (Policies{}).EvaluatePlan(context.Background(), domain.PlanPolicyInput{Review: r, Now: now.Add(-9 * time.Minute)})
	r.Policy = &e
	for _, root := range r.Roots {
		r.Attempts = append(r.Attempts, domain.ExecutionRecord{ID: "seed-plan-" + root.ID, PlanSetID: r.Plan.ID, RootID: root.ID, Operation: "plan", Status: root.Status, CreatedAt: r.Plan.CreatedAt, Log: root.Log})
	}
	r.History = append(r.History, domain.ReviewEvent{Title: "Plan 7 assessed", Detail: "Synthetic evidence · " + e.Coverage + " policy coverage", CreatedAt: r.Plan.CreatedAt})
	if scenario == "expired" || scenario == "stale" || scenario == "partial" {
		var v domain.PolicyViolation
		for _, candidate := range r.Policy.Violations {
			if candidate.PolicyID == "data-protection" {
				v = candidate
				break
			}
		}
		expires := now.Add(time.Hour)
		if scenario == "expired" {
			expires = now.Add(-time.Minute)
		}
		r.Acceptances = append(r.Acceptances, domain.ViolationAcceptance{ID: "acceptance-seed", ViolationID: v.ID, PlanSetID: r.Plan.ID, EvaluationID: r.Policy.ID, PolicyVersion: v.PolicyVersion, Status: "granted", RequestedBy: "alex", AuthorizedBy: "morgan · database owner", Reason: "Planned database migration", Evidence: "Demo recovery exercise and migration checklist", CreatedAt: now.Add(-2 * time.Hour).UTC().Format(time.RFC3339), ExpiresAt: expires.UTC().Format(time.RFC3339)})
		r.Decisions = append(r.Decisions, domain.ReviewDecision{Actor: "alex", Decision: "approved", PlanSetID: r.Plan.ID, CommitSHA: r.HeadSHA, EvaluationID: r.Policy.ID, CreatedAt: now.Add(-5 * time.Minute).UTC().Format(time.RFC3339), Message: "Reviewed recovery evidence.", Source: "statecraft-demo"})
		r.State = "approved"
	}
	if scenario == "stale" {
		r.HeadSHA = "def5678"
		r.State = "stale"
		for i := range r.Roots {
			r.Roots[i].Status = "stale"
		}
		r.History = append(r.History, domain.ReviewEvent{Title: "New commit invalidated readiness", Detail: "abc1234 → def5678. Earlier approval is retained as history.", CreatedAt: now.UTC().Format(time.RFC3339)})
	}
	if scenario == "partial" {
		r.State = "partial"
		r.Roots[0].ApplyStatus = "applied"
		r.Roots[0].Log = "Synthetic apply completed. The API root already changed."
		r.Roots[3].ApplyStatus = "no_changes"
		r.Roots[1].ApplyStatus = "failed"
		r.Roots[1].Log = "Apply stopped: network provider rejected the access rule.\nThe API root already changed. Observability has not started.\nRe-plan from the resulting state before another apply."
		for _, root := range r.Roots {
			if root.ApplyStatus != "applied" && root.ApplyStatus != "failed" {
				continue
			}
			r.Attempts = append(r.Attempts, domain.ExecutionRecord{ID: "seed-apply-" + root.ID, PlanSetID: r.Plan.ID, RootID: root.ID, Operation: "apply", Status: root.ApplyStatus, CreatedAt: now.Add(-time.Minute).UTC().Format(time.RFC3339), Log: root.Log})
		}
		r.History = append(r.History, domain.ReviewEvent{Title: "Apply stopped after partial execution", Detail: "API applied; network failed; observability not started.", CreatedAt: now.Add(-time.Minute).UTC().Format(time.RFC3339)})
	}
	return r, nil
}

// Planner rebuilds synthetic evidence after an incomplete plan and excludes roots
// already changed by this session's earlier apply. Seed data never reaches a live
// execution adapter.
type Planner struct{}

func (Planner) ChangesForPlan(ctx context.Context, current domain.Review) ([]domain.Change, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scenario := "review"
	if current.Scenario == "ready" {
		scenario = "ready"
	}
	seed, err := SeedReview(scenario, time.Now())
	if err != nil {
		return nil, err
	}
	applied := map[string]bool{}
	for _, attempt := range current.Attempts {
		if attempt.Operation == "apply" && attempt.Status == "applied" {
			applied[attempt.RootID] = true
		}
	}
	changes := []domain.Change{}
	for _, change := range seed.Changes {
		if !applied[change.RootID] {
			changes = append(changes, change)
		}
	}
	return changes, nil
}
