package githubadapter

import (
	"testing"
	"time"

	gh "github.com/google/go-github/v72/github"
)

func TestReviewDecision(t *testing.T) {
	when := time.Date(2026, 9, 25, 20, 10, 0, 0, time.UTC)
	review := &gh.PullRequestReview{
		ID:          gh.Int64(42),
		User:        &gh.User{Login: gh.String("josh")},
		State:       gh.String("APPROVED"),
		CommitID:    gh.String("abc123"),
		Body:        gh.String("looks good\n\n<!-- statecraft-plan-set:planset-7 -->"),
		HTMLURL:     gh.String("https://github.example/review/42"),
		SubmittedAt: &gh.Timestamp{Time: when},
	}

	got := reviewDecision(review)
	if got.ID != "42" || got.Actor != "josh" || got.Decision != "approved" || got.CommitSHA != "abc123" || got.PlanSetID != "planset-7" {
		t.Fatalf("unexpected decision: %#v", got)
	}
	if !got.CreatedAt.Equal(when) {
		t.Fatalf("created at = %v, want %v", got.CreatedAt, when)
	}
}

func TestGitHubReviewEvent(t *testing.T) {
	tests := map[string]string{
		"approved":          "APPROVE",
		"request_changes":   "REQUEST_CHANGES",
		"changes_requested": "REQUEST_CHANGES",
		"comment":           "COMMENT",
	}
	for input, want := range tests {
		got, err := githubReviewEvent(input)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got != want {
			t.Fatalf("%s: got %q want %q", input, got, want)
		}
	}
	if _, err := githubReviewEvent("auto_approve"); err == nil {
		t.Fatal("expected unsupported decision error")
	}
}
