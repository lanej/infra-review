package domain

import "time"

type PropertyChange struct {
	Name   string `json:"name"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type ResourceRelationship struct {
	ResourceID string `json:"resourceId"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Evidence   string `json:"evidence"`
}

// PlanSnapshot identifies an immutable proposal. Demo digests are deliberately
// labeled synthetic and cannot be used as production artifact attestations.
type PlanSnapshot struct {
	ID        string   `json:"id"`
	Number    int      `json:"number"`
	CommitSHA string   `json:"commitSha"`
	Digest    string   `json:"digest"`
	RootIDs   []string `json:"rootIds"`
	CreatedAt string   `json:"createdAt"`
	ChangeIDs []string `json:"changeIds"`
}

type PolicyResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Title   string `json:"title"`
	Outcome string `json:"outcome"`
}

type PolicyViolation struct {
	ID                string   `json:"id"`
	PolicyID          string   `json:"policyId"`
	PolicyVersion     string   `json:"policyVersion"`
	ResourceID        string   `json:"resourceId"`
	Title             string   `json:"title"`
	Severity          string   `json:"severity"`
	Blocking          bool     `json:"blocking"`
	AcceptanceAllowed bool     `json:"acceptanceAllowed"`
	Explanation       string   `json:"explanation"`
	Consequence       string   `json:"consequence"`
	Unknown           string   `json:"unknown"`
	Evidence          []string `json:"evidence"`
}

type PolicyEvaluation struct {
	ID          string            `json:"id"`
	PlanSetID   string            `json:"planSetId"`
	PolicySetID string            `json:"policySetId"`
	Status      string            `json:"status"`
	Coverage    string            `json:"coverage"`
	EvaluatedAt string            `json:"evaluatedAt"`
	Results     []PolicyResult    `json:"results"`
	Violations  []PolicyViolation `json:"violations"`
}

type ViolationAcceptance struct {
	ID            string `json:"id"`
	ViolationID   string `json:"violationId"`
	PlanSetID     string `json:"planSetId"`
	EvaluationID  string `json:"evaluationId"`
	PolicyVersion string `json:"policyVersion"`
	Status        string `json:"status"`
	RequestedBy   string `json:"requestedBy"`
	AuthorizedBy  string `json:"authorizedBy"`
	Reason        string `json:"reason"`
	Evidence      string `json:"evidence"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
}

type ReviewAction string

const (
	ActionPlan              ReviewAction = "plan"
	ActionApprove           ReviewAction = "approve"
	ActionRequestChanges    ReviewAction = "request_changes"
	ActionRequestAcceptance ReviewAction = "request_acceptance"
	ActionGrantAcceptance   ReviewAction = "grant_acceptance"
	ActionApply             ReviewAction = "apply"
	ActionVerify            ReviewAction = "verify"
)

type ActionDecision struct {
	Action  ReviewAction `json:"action"`
	Outcome string       `json:"outcome"`
	Reason  string       `json:"reason"`
}

type WorkflowCommand struct {
	Action          ReviewAction `json:"action"`
	ExpectedVersion uint64       `json:"expectedVersion"`
	ViolationID     string       `json:"violationId"`
	Reason          string       `json:"reason"`
	Evidence        string       `json:"evidence"`
}

type ExecutionRecord struct {
	ID        string `json:"id"`
	PlanSetID string `json:"planSetId"`
	RootID    string `json:"rootId"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Log       string `json:"log"`
}

type ReviewEvent struct {
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"createdAt"`
}

type PlanPolicyInput struct {
	Review Review
	Now    time.Time
}
type ActionPolicyInput struct {
	Review      Review
	Action      ReviewAction
	ViolationID string
	Now         time.Time
}

func (r Review) CurrentApproval() bool {
	if r.Plan == nil || r.Policy == nil || r.HeadSHA == "" || r.Plan.ID == "" || r.Policy.ID == "" || r.Policy.PlanSetID != r.Plan.ID || r.Plan.CommitSHA != r.HeadSHA || r.State == "stale" {
		return false
	}
	// Later decisions by the same reviewer supersede earlier decisions on this
	// exact plan, including a request for changes after approval.
	latest := map[string]string{}
	for _, d := range r.Decisions {
		if d.PlanSetID == r.Plan.ID && d.CommitSHA == r.HeadSHA && d.EvaluationID == r.Policy.ID {
			latest[d.Actor] = d.Decision
		}
	}
	approved := false
	for _, decision := range latest {
		if decision == "changes_requested" {
			return false
		}
		if decision == "approved" {
			approved = true
		}
	}
	return approved
}

func (r Review) AcceptanceFor(v PolicyViolation, now time.Time) *ViolationAcceptance {
	if r.Plan == nil || r.Policy == nil {
		return nil
	}
	for i := len(r.Acceptances) - 1; i >= 0; i-- {
		a := r.Acceptances[i]
		expires, err := time.Parse(time.RFC3339, a.ExpiresAt)
		if a.ViolationID == v.ID && a.PlanSetID == r.Plan.ID && a.EvaluationID == r.Policy.ID && a.PolicyVersion == v.PolicyVersion && a.Status == "granted" && a.AuthorizedBy != "" && err == nil && now.Before(expires) {
			return &a
		}
	}
	return nil
}
