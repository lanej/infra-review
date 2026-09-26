package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lanej/statecraft/internal/domain"
)

// The temporary HTTP bridge encodes Review directly. Its approval list must not
// expose untrusted source history, and its keys must match the TypeScript view.
func TestReviewJSONBoundary(t *testing.T) {
	review := domain.Review{
		ID: "review-1", HeadSHA: "abc123", PullRequest: 42,
		Changes:         []domain.Change{{RootID: "root-1", ResourceType: "test_resource"}},
		Decisions:       []domain.ReviewDecision{{Actor: "reviewer", PlanSetID: "plan-1"}},
		SourceDecisions: []domain.ExternalReviewDecision{{Actor: "external-actor", Body: "untrusted source history"}},
	}
	data, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if string(payload["headSha"]) != `"abc123"` || string(payload["pullRequest"]) != "42" {
		t.Fatalf("frontend identity fields missing: %s", data)
	}
	if !strings.Contains(string(payload["changes"]), `"rootId":"root-1"`) || !strings.Contains(string(payload["changes"]), `"resourceType":"test_resource"`) {
		t.Fatalf("frontend change fields missing: %s", data)
	}
	if !strings.Contains(string(payload["decisions"]), `"planSetId":"plan-1"`) {
		t.Fatalf("Statecraft approval missing: %s", data)
	}
	if strings.Contains(string(data), "external-actor") || strings.Contains(string(data), "untrusted source history") || payload["SourceDecisions"] != nil || payload["sourceDecisions"] != nil {
		t.Fatalf("source history leaked into the approval contract: %s", data)
	}
}
