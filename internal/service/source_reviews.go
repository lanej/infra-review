package service

import (
	"context"
	"fmt"
	"slices"

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
		Files:     slices.Clone(files),
		Decisions: slices.Clone(decisions),
	}, nil
}

// MergeSourceSnapshot hydrates the source-control-owned portion of a Review.
// Infrastructure evidence and Statecraft's authoritative decisions are preserved.
// External review history is recorded separately: a source review or a plan-set
// marker in its editable body cannot authorize an infrastructure plan.
func MergeSourceSnapshot(review domain.Review, snapshot domain.SourceReviewSnapshot) domain.Review {
	change := snapshot.Change
	review.Repository = change.Repository.FullName()
	review.PullRequest = change.Number
	review.Title = change.Title
	review.HeadSHA = change.HeadSHA

	review.SourceDecisions = slices.Clone(snapshot.Decisions)
	return review
}
