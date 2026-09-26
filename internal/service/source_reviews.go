package service

import (
	"context"
	"fmt"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

type SourceReviews struct {
	source ports.SourceControl
}

func NewSourceReviews(source ports.SourceControl) *SourceReviews {
	return &SourceReviews{source: source}
}

func (s *SourceReviews) Load(ctx context.Context, repo domain.RepositoryRef, number int64) (domain.SourceReviewSnapshot, error) {
	change, err := s.source.GetChange(ctx, repo, number)
	if err != nil {
		return domain.SourceReviewSnapshot{}, fmt.Errorf("load source change: %w", err)
	}
	files, err := s.source.ListChangedFiles(ctx, repo, number)
	if err != nil {
		return domain.SourceReviewSnapshot{}, fmt.Errorf("load changed files: %w", err)
	}
	decisions, err := s.source.ListReviewDecisions(ctx, repo, number)
	if err != nil {
		return domain.SourceReviewSnapshot{}, fmt.Errorf("load review decisions: %w", err)
	}
	return domain.SourceReviewSnapshot{
		Change:    change,
		Files:     files,
		Decisions: decisions,
	}, nil
}

// MergeSourceSnapshot hydrates the source-control-owned portion of a Review.
// Infrastructure roots, plan changes, findings, execution history, and graph data
// remain owned by Statecraft's plan/evidence side and are preserved.
func MergeSourceSnapshot(review domain.Review, snapshot domain.SourceReviewSnapshot) domain.Review {
	change := snapshot.Change
	review.Repository = change.Repository.FullName()
	review.PullRequest = change.Number
	review.Title = change.Title
	review.HeadSHA = change.HeadSHA

	review.Decisions = make([]domain.ReviewDecision, 0, len(snapshot.Decisions))
	for _, decision := range snapshot.Decisions {
		review.Decisions = append(review.Decisions, domain.ReviewDecision{
			Actor:      decision.Actor,
			Decision:   decision.Decision,
			PlanSetID:  decision.PlanSetID,
			CommitSHA:  decision.CommitSHA,
			CreatedAt:  decision.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			ExternalID: decision.ID,
			Source:     decision.Source,
		})
	}
	return review
}
