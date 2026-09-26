package domain

type Review struct {
	ID          string
	Repository  string
	PullRequest int64
	Title       string
	HeadSHA     string
	State       string
	Roots       []Root
	Changes     []Change
	Findings    []Finding
	Decisions   []ReviewDecision
}

type Root struct {
	ID, Name, Status string
}

type Change struct {
	ID, RootID, Address, ResourceType, Action, Risk, Summary string
}

type Finding struct {
	ID, Severity, Category, Title, ResourceAddress string
	Blocking bool
}

type ReviewDecision struct {
	Actor, Decision, PlanSetID, CommitSHA, CreatedAt string
	ExternalID, Source string
}
