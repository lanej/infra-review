package service

import (
	"testing"
	"time"

	"github.com/lanej/statecraft/internal/domain"
)

func TestMergeSourceSnapshotPreservesInfrastructureReviewData(t *testing.T) {
	review := domain.Review{
		Roots:    []domain.Root{{ID: "api", Name: "prod/api", Status: "planned"}},
		Changes:  []domain.Change{{ID: "c1", RootID: "api", Address: "resource.api"}},
		Findings: []domain.Finding{{ID: "f1", Title: "destructive change"}},
	}
	when := time.Date(2026, 9, 25, 20, 10, 0, 0, time.UTC)
	snapshot := domain.SourceReviewSnapshot{
		Change: domain.SourceChange{
			Repository: domain.RepositoryRef{Owner: "acme", Name: "infra"},
			Number:     42,
			Title:      "Scale API",
			HeadSHA:    "abc123",
		},
		Decisions: []domain.ExternalReviewDecision{{
			ID: "99", Actor: "josh", Decision: "approved", CommitSHA: "abc123",
			PlanSetID: "planset-7", CreatedAt: when, Source: "github",
		}},
	}

	got := MergeSourceSnapshot(review, snapshot)
	if got.Repository != "acme/infra" || got.PullRequest != 42 || got.HeadSHA != "abc123" {
		t.Fatalf("source metadata = %#v", got)
	}
	if len(got.Roots) != 1 || len(got.Changes) != 1 || len(got.Findings) != 1 {
		t.Fatal("infrastructure review data was not preserved")
	}
	if len(got.Decisions) != 1 || got.Decisions[0].PlanSetID != "planset-7" || got.Decisions[0].Source != "github" {
		t.Fatalf("decisions = %#v", got.Decisions)
	}
}
