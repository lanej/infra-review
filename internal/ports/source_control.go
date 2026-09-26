package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

// SourceControl exchanges code and reviewer evidence with a configured source
// host. Its review history is not Statecraft plan approval; credential selection
// and permission checks belong in the application composition/use case.
type SourceControl interface {
	GetChange(context.Context, domain.RepositoryRef, int64) (domain.SourceChange, error)
	ListChangedFiles(context.Context, domain.RepositoryRef, int64) ([]domain.ChangedFile, error)
	ListReviewDecisions(context.Context, domain.RepositoryRef, int64) ([]domain.ExternalReviewDecision, error)
	PublishDecision(context.Context, domain.PublishDecisionRequest) (domain.ExternalReviewDecision, error)
	PublishStatus(context.Context, domain.StatusReport) error
}
