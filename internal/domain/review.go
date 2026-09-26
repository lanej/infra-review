package domain

type Review struct {
	ID           string                `json:"id"`
	Repository   string                `json:"repository"`
	PullRequest  int64                 `json:"pullRequest"`
	Title        string                `json:"title"`
	HeadSHA      string                `json:"headSha"`
	State        string                `json:"state"`
	Roots        []Root                `json:"roots"`
	Changes      []Change              `json:"changes"`
	Findings     []Finding             `json:"findings"`
	Decisions    []ReviewDecision      `json:"decisions"`
	Demo         bool                  `json:"demo"`
	Version      uint64                `json:"version"`
	Scenario     string                `json:"scenario"`
	Plan         *PlanSnapshot         `json:"plan"`
	Policy       *PolicyEvaluation     `json:"policy"`
	Acceptances  []ViolationAcceptance `json:"acceptances"`
	Attempts     []ExecutionRecord     `json:"attempts"`
	History      []ReviewEvent         `json:"history"`
	PlanHistory  []PlanSnapshot        `json:"planHistory"`
	Actions      []ActionDecision      `json:"actions"`
	Verification string                `json:"verification"`
	// Source history is evidence, not a Statecraft approval. It stays internal
	// until a separate public source-history contract is introduced.
	SourceDecisions []ExternalReviewDecision `json:"-"`
}

type Root struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	ApplyStatus string `json:"applyStatus"`
	Log         string `json:"log"`
}

type Change struct {
	ID            string                 `json:"id"`
	RootID        string                 `json:"rootId"`
	Address       string                 `json:"address"`
	ResourceType  string                 `json:"resourceType"`
	Action        string                 `json:"action"`
	Risk          string                 `json:"risk"`
	Summary       string                 `json:"summary"`
	Name          string                 `json:"name"`
	Properties    []PropertyChange       `json:"properties"`
	Relationships []ResourceRelationship `json:"relationships"`
	PlanText      string                 `json:"planText"`
	SourcePath    string                 `json:"sourcePath"`
	SourceText    string                 `json:"sourceText"`
}

type Finding struct {
	ID              string `json:"id"`
	Severity        string `json:"severity"`
	Category        string `json:"category"`
	Title           string `json:"title"`
	ResourceAddress string `json:"resourceAddress"`
	Blocking        bool   `json:"blocking"`
}

type ReviewDecision struct {
	Actor        string `json:"actor"`
	Decision     string `json:"decision"`
	PlanSetID    string `json:"planSetId"`
	CommitSHA    string `json:"commitSha"`
	CreatedAt    string `json:"createdAt"`
	ExternalID   string `json:"externalId"`
	Source       string `json:"source"`
	EvaluationID string `json:"evaluationId"`
	Message      string `json:"message"`
}
