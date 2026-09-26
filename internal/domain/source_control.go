package domain

import "time"

type RepositoryRef struct {
	Owner string
	Name  string
}

func (r RepositoryRef) FullName() string {
	if r.Owner == "" {
		return r.Name
	}
	return r.Owner + "/" + r.Name
}

type SourceChange struct {
	Repository RepositoryRef
	Number     int64
	Title      string
	Body       string
	Author     string
	State      string
	Draft      bool
	HeadSHA    string
	HeadRef    string
	BaseRef    string
	URL        string
}

type ChangedFile struct {
	Path          string
	PreviousPath  string
	Status        string
	Additions     int
	Deletions     int
	Changes       int
}

type PublishDecisionRequest struct {
	Repository RepositoryRef
	Number     int64
	CommitSHA  string
	Decision   string
	Body       string
	PlanSetID  string
}

type StatusReport struct {
	Repository RepositoryRef
	HeadSHA    string
	Name       string
	Status     string
	Conclusion string
	Summary    string
	DetailsURL string
}

type ExternalReviewDecision struct {
	ID        string
	Actor     string
	Decision  string
	CommitSHA string
	Body      string
	URL       string
	CreatedAt time.Time
	Source    string
}
