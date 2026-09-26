package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

type SourceControl interface {
	GetChange(context.Context, domain.RepositoryRef, int64) (domain.SourceChange, error)
	ListChangedFiles(context.Context, domain.RepositoryRef, int64) ([]domain.ChangedFile, error)
	ListReviewDecisions(context.Context, domain.RepositoryRef, int64) ([]domain.ExternalReviewDecision, error)
	PublishDecision(context.Context, domain.PublishDecisionRequest) (domain.ExternalReviewDecision, error)
	PublishStatus(context.Context, domain.StatusReport) error
}
