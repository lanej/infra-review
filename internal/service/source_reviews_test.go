package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lanej/statecraft/internal/domain"
)

func TestSourceReviewsLoad(t *testing.T) {
	repo := domain.RepositoryRef{Owner: "acme", Name: "infra"}
	source := &sourceControlStub{snapshot: domain.SourceReviewSnapshot{
		Change:    domain.SourceChange{Repository: repo, Number: 42, Title: "Scale API", HeadSHA: "head"},
		Files:     []domain.ChangedFile{{Path: "prod/api/main.tf", Status: "modified"}},
		Decisions: []domain.ExternalReviewDecision{{ID: "99", Actor: "reviewer", Decision: "approved", CommitSHA: "head", Source: "github"}},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got, err := NewSourceReviews(source).Load(ctx, repo, 42)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, source.snapshot) {
		t.Fatalf("snapshot = %#v, want %#v", got, source.snapshot)
	}
	if !reflect.DeepEqual(source.calls, []string{"change", "files", "decisions"}) {
		t.Fatalf("calls = %v", source.calls)
	}
	for _, request := range source.requests {
		if request.ctx != ctx || request.repo != repo || request.number != 42 {
			t.Fatalf("request = %#v", request)
		}
	}

	// A snapshot owns its history and file slices even when an adapter returns
	// cached data. Consumer mutations must not alter the source's cache.
	got.Files[0].Path = "changed"
	got.Decisions[0].Actor = "changed"
	if source.snapshot.Files[0].Path != "prod/api/main.tf" || source.snapshot.Decisions[0].Actor != "reviewer" {
		t.Fatal("loaded snapshot aliases source-owned slices")
	}
}

func TestSourceReviewsLoadErrorsStopAtFailedBoundary(t *testing.T) {
	for _, tc := range []struct {
		stage string
		label string
		calls []string
	}{
		{"change", "load source change", []string{"change"}},
		{"files", "load changed files", []string{"change", "files"}},
		{"decisions", "load review decisions", []string{"change", "files", "decisions"}},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			cause := errors.New("provider unavailable")
			source := &sourceControlStub{
				errStage: tc.stage,
				err:      cause,
				snapshot: domain.SourceReviewSnapshot{
					Change: domain.SourceChange{Title: "partial data"},
					Files:  []domain.ChangedFile{{Path: "main.tf"}},
				},
			}
			got, err := NewSourceReviews(source).Load(context.Background(), domain.RepositoryRef{Owner: "acme", Name: "infra"}, 42)
			if !errors.Is(err, cause) || !strings.Contains(err.Error(), tc.label) {
				t.Fatalf("error = %v, want wrapped cause and stage %q", err, tc.label)
			}
			if !reflect.DeepEqual(got, domain.SourceReviewSnapshot{}) {
				t.Fatalf("failed load exposed partial snapshot: %#v", got)
			}
			if !reflect.DeepEqual(source.calls, tc.calls) {
				t.Fatalf("calls = %v, want %v", source.calls, tc.calls)
			}
		})
	}
}

func TestMergeSourceSnapshotPreservesStatecraftDecisionsAndEvidence(t *testing.T) {
	review := domain.Review{
		ID: "review-42", Repository: "old/repo", PullRequest: 1, Title: "Old title", HeadSHA: "old-head", State: "ready_for_review",
		Roots:           []domain.Root{{ID: "api", Name: "prod/api", Status: "planned"}},
		Changes:         []domain.Change{{ID: "c1", RootID: "api", Address: "resource.api"}},
		Findings:        []domain.Finding{{ID: "f1", Title: "destructive change", Blocking: true}},
		Decisions:       []domain.ReviewDecision{{Actor: "local-reviewer", Decision: "approved", PlanSetID: "planset-7", CommitSHA: "old-head"}},
		SourceDecisions: []domain.ExternalReviewDecision{{ID: "old-review", Actor: "old-reviewer", Decision: "approved", Source: "github"}},
	}
	snapshot := domain.SourceReviewSnapshot{
		Change: domain.SourceChange{
			Repository: domain.RepositoryRef{Owner: "acme", Name: "infra"},
			Number:     42, Title: "Scale API", HeadSHA: "new-head",
		},
		Decisions: []domain.ExternalReviewDecision{{
			ID: "99", Actor: "external-reviewer", Decision: "approved", CommitSHA: "new-head", Source: "github",
			Body: "<!-- statecraft-plan-set:planset-7 -->", URL: "https://example.test/reviews/99",
			CreatedAt: time.Date(2026, 9, 25, 20, 10, 0, 0, time.UTC),
		}},
	}

	got := MergeSourceSnapshot(review, snapshot)
	if got.Repository != "acme/infra" || got.PullRequest != 42 || got.Title != "Scale API" || got.HeadSHA != "new-head" {
		t.Fatalf("source metadata = %#v", got)
	}
	if got.ID != review.ID || got.State != review.State || !reflect.DeepEqual(got.Roots, review.Roots) || !reflect.DeepEqual(got.Changes, review.Changes) || !reflect.DeepEqual(got.Findings, review.Findings) {
		t.Fatal("Statecraft review identity, state, or evidence was not preserved")
	}
	if !reflect.DeepEqual(got.Decisions, review.Decisions) {
		t.Fatalf("authoritative decisions changed: %#v", got.Decisions)
	}
	if !reflect.DeepEqual(got.SourceDecisions, snapshot.Decisions) {
		t.Fatalf("source review provenance = %#v, want %#v", got.SourceDecisions, snapshot.Decisions)
	}
	if review.Repository != "old/repo" || review.Title != "Old title" || review.HeadSHA != "old-head" || review.SourceDecisions[0].ID != "old-review" {
		t.Fatal("merge mutated its input review")
	}

	// The editable plan marker remains source evidence only. Copying the
	// external history also prevents later mutations from rewriting the input.
	got.SourceDecisions[0].Body = "changed"
	got.SourceDecisions[0].Actor = "changed"
	if snapshot.Decisions[0].Body != "<!-- statecraft-plan-set:planset-7 -->" || snapshot.Decisions[0].Actor != "external-reviewer" {
		t.Fatal("merged history aliases snapshot-owned decisions")
	}
}

func TestMergeSourceSnapshotRefreshesExternalHistory(t *testing.T) {
	local := domain.ReviewDecision{Actor: "local-reviewer", Decision: "approved", PlanSetID: "planset-7"}
	review := domain.Review{
		Decisions:       []domain.ReviewDecision{local},
		SourceDecisions: []domain.ExternalReviewDecision{{ID: "99", Decision: "approved", Source: "github"}},
	}
	snapshot := domain.SourceReviewSnapshot{Decisions: []domain.ExternalReviewDecision{{ID: "99", Decision: "dismissed", Source: "github"}}}
	got := MergeSourceSnapshot(MergeSourceSnapshot(review, snapshot), snapshot)
	if len(got.SourceDecisions) != 1 || got.SourceDecisions[0].Decision != "dismissed" {
		t.Fatalf("refresh duplicated or retained stale external history: %#v", got.SourceDecisions)
	}
	got = MergeSourceSnapshot(got, domain.SourceReviewSnapshot{})
	if len(got.SourceDecisions) != 0 || !reflect.DeepEqual(got.Decisions, []domain.ReviewDecision{local}) {
		t.Fatalf("empty source history must clear only source decisions: %#v", got)
	}
}

type sourceRequest struct {
	ctx    context.Context
	repo   domain.RepositoryRef
	number int64
}

type sourceControlStub struct {
	snapshot domain.SourceReviewSnapshot
	errStage string
	err      error
	calls    []string
	requests []sourceRequest
}

func (s *sourceControlStub) record(stage string, ctx context.Context, repo domain.RepositoryRef, number int64) error {
	s.calls = append(s.calls, stage)
	s.requests = append(s.requests, sourceRequest{ctx: ctx, repo: repo, number: number})
	if stage == s.errStage {
		return s.err
	}
	return nil
}

func (s *sourceControlStub) GetChange(ctx context.Context, repo domain.RepositoryRef, number int64) (domain.SourceChange, error) {
	return s.snapshot.Change, s.record("change", ctx, repo, number)
}

func (s *sourceControlStub) ListChangedFiles(ctx context.Context, repo domain.RepositoryRef, number int64) ([]domain.ChangedFile, error) {
	return s.snapshot.Files, s.record("files", ctx, repo, number)
}

func (s *sourceControlStub) ListReviewDecisions(ctx context.Context, repo domain.RepositoryRef, number int64) ([]domain.ExternalReviewDecision, error) {
	return s.snapshot.Decisions, s.record("decisions", ctx, repo, number)
}

func (s *sourceControlStub) PublishDecision(context.Context, domain.PublishDecisionRequest) (domain.ExternalReviewDecision, error) {
	panic("unexpected decision publication")
}

func (s *sourceControlStub) PublishStatus(context.Context, domain.StatusReport) error {
	panic("unexpected status publication")
}
