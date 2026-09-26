package domain

type RootSelector struct {
	ID         string
	PlannerRef string
	Directory  string
	Workspace  string
}

func (r RootSelector) StableID() string {
	if r.ID != "" {
		return r.ID
	}
	if r.PlannerRef != "" {
		return r.PlannerRef
	}
	return r.Directory + "::" + r.Workspace
}

type PlanRequest struct {
	Repository  RepositoryRef
	Ref         string
	BaseBranch  string
	PullRequest int64
	Roots       []RootSelector
}

type PlanRun struct {
	Attempts []PlanAttempt
	// Command-level failure can coexist with successful per-root attempts.
	Failure        string
	ErrorPresent   bool
	PlansDiscarded bool
}

type PlanAttempt struct {
	RootID string
	// A planner can report a policy check separately from plan generation.
	Phase        string
	PlannerRef   string
	Directory    string
	Workspace    string
	Status       string
	Output       string
	Failure      string
	ErrorPresent bool
}

type ApplyRequest struct {
	Repository  RepositoryRef
	Ref         string
	BaseBranch  string
	PullRequest int64
	Roots       []RootSelector
}

type ApplyRun struct {
	Attempts     []ApplyAttempt
	Failure      string
	ErrorPresent bool
}

type ApplyAttempt struct {
	RootID       string
	PlannerRef   string
	Directory    string
	Workspace    string
	Status       string
	Output       string
	Failure      string
	ErrorPresent bool
}

type ApplyNotification struct {
	Repository  RepositoryRef
	PullRequest int64
	HeadSHA     string
	HeadRef     string
	BaseRef     string
	Actor       string
	Root        RootSelector
	Success     bool
	Source      string
}
