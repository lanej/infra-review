package githubadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v92/github"

	"github.com/lanej/statecraft/internal/domain"
)

var testRepo = domain.RepositoryRef{Owner: "acme", Name: "infra"}

func testSource(t *testing.T, handler http.HandlerFunc) *SourceControl {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := gh.NewClient(gh.WithHTTPClient(server.Client()), gh.WithURLs(gh.Ptr(server.URL+"/"), nil), gh.WithAuthToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	return NewSourceControl(client)
}

func TestGetChangeMerged(t *testing.T) {
	source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/repos/acme/infra/pulls/7" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		if got := r.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
			t.Errorf("SDK API version = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization = %q", got)
		}
		fmt.Fprint(w, `{"number":7,"title":"Network","body":"Review me","user":{"login":"josh"},"state":"closed","merged":true,"draft":false,"html_url":"https://github.example/acme/infra/pull/7","head":{"sha":"head-sha","ref":"network"},"base":{"ref":"main"}}`)
	})
	got, err := source.GetChange(context.Background(), testRepo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "merged" || got.HeadSHA != "head-sha" || got.HeadRef != "network" || got.BaseRef != "main" || got.Number != 7 || got.Author != "josh" || got.Repository != testRepo {
		t.Fatalf("unexpected source change: %#v", got)
	}
}

func TestListChangedFilesPagination(t *testing.T) {
	var pages []int
	source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/infra/pulls/7/files" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("unexpected request %s", r.URL)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		if page == 1 {
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/infra/pulls/7/files?page=2>; rel="next"`, r.Host))
			fmt.Fprint(w, `[{"filename":"main.tf","status":"modified","additions":3,"deletions":1,"changes":4}]`)
			return
		}
		fmt.Fprint(w, `[{"filename":"new.tf","previous_filename":"old.tf","status":"renamed","additions":0,"deletions":0,"changes":0}]`)
	})
	got, err := source.ListChangedFiles(context.Background(), testRepo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Changes != 4 || got[1].PreviousPath != "old.tf" || got[1].Status != "renamed" || fmt.Sprint(pages) != "[1 2]" {
		t.Fatalf("files = %#v, pages = %v", got, pages)
	}
}

func TestListChangedFilesAtAPILimit(t *testing.T) {
	for _, count := range []int{3000, 3001} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/files") {
					files := make([]map[string]string, 3000)
					for i := range files {
						files[i] = map[string]string{"filename": fmt.Sprintf("%d.tf", i), "status": "added"}
					}
					json.NewEncoder(w).Encode(files)
					return
				}
				fmt.Fprintf(w, `{"changed_files":%d}`, count)
			})
			got, err := source.ListChangedFiles(context.Background(), testRepo, 7)
			if count == 3000 && (err != nil || len(got) != 3000) {
				t.Fatalf("complete list: count=%d err=%v", len(got), err)
			}
			if count > 3000 && (err == nil || got != nil) {
				t.Fatalf("truncated list accepted: count=%d err=%v", len(got), err)
			}
		})
	}
}

func TestPaginatedReadsRejectPartialResults(t *testing.T) {
	for _, endpoint := range []string{"files", "reviews"} {
		t.Run(endpoint, func(t *testing.T) {
			source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "1" {
					w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/infra/pulls/7/%s?page=2>; rel="next"`, r.Host, endpoint))
					fmt.Fprint(w, `[{"filename":"a.tf","id":42,"state":"APPROVED","submitted_at":"2026-09-25T20:10:00Z"}]`)
					return
				}
				http.Error(w, `{"message":"unavailable"}`, http.StatusServiceUnavailable)
			})
			if endpoint == "files" {
				got, err := source.ListChangedFiles(context.Background(), testRepo, 7)
				if err == nil || got != nil {
					t.Fatalf("partial files accepted: %#v %v", got, err)
				}
			} else {
				got, err := source.ListReviewDecisions(context.Background(), testRepo, 7)
				if err == nil || got != nil {
					t.Fatalf("partial reviews accepted: %#v %v", got, err)
				}
			}
		})
	}
}

func TestListSubmittedReviewsPreservesProvenance(t *testing.T) {
	pages := 0
	source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		if r.URL.Path != "/repos/acme/infra/pulls/7/reviews" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("unexpected request %s", r.URL)
		}
		if r.URL.Query().Get("page") == "1" {
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/infra/pulls/7/reviews?page=2>; rel="next"`, r.Host))
			fmt.Fprint(w, `[{"id":40,"state":"PENDING"},{"id":42,"user":{"login":"josh"},"state":"APPROVED","commit_id":"old-head","body":"<!-- statecraft-plan-set:forged -->","submitted_at":"2026-09-25T20:10:00Z"}]`)
			return
		}
		fmt.Fprint(w, `[{"id":43,"user":{"login":"josh"},"state":"DISMISSED","commit_id":"head","submitted_at":"2026-09-25T20:11:00Z"}]`)
	})
	got, err := source.ListReviewDecisions(context.Background(), testRepo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || pages != 2 || got[0].Decision != "approved" || got[0].CommitSHA != "old-head" || got[0].PlanSetID != "" || !strings.Contains(got[0].Body, "forged") || got[1].Decision != "dismissed" || got[0].Source != "github" {
		t.Fatalf("unexpected external reviews: %#v (pages=%d)", got, pages)
	}
	if want := time.Date(2026, 9, 25, 20, 10, 0, 0, time.UTC); !got[0].CreatedAt.Equal(want) {
		t.Fatalf("submission time: %v", got[0].CreatedAt)
	}
}

func TestPublishDecisionPinsCommit(t *testing.T) {
	source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/repos/acme/infra/pulls/7/reviews" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["commit_id"] != "reviewed-head" || body["event"] != "APPROVE" || body["body"] != "Reviewed\n\n<!-- statecraft-plan-set:plan-7 -->" {
			t.Errorf("unexpected review payload: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":44,"state":"APPROVED","commit_id":"reviewed-head","body":"<!-- statecraft-plan-set:plan-7 -->","submitted_at":"2026-09-25T20:10:00Z"}`)
	})
	got, err := source.PublishDecision(context.Background(), domain.PublishDecisionRequest{Repository: testRepo, Number: 7, CommitSHA: "reviewed-head", Decision: "approved", Body: "Reviewed", PlanSetID: "plan-7"})
	if err != nil || got.CommitSHA != "reviewed-head" || got.PlanSetID != "" {
		t.Fatalf("published decision: %#v %v", got, err)
	}
}

func TestPublishDecisionRequiresCommit(t *testing.T) {
	source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid request reached GitHub")
	})
	if _, err := source.PublishDecision(context.Background(), domain.PublishDecisionRequest{Decision: "approved"}); err == nil {
		t.Fatal("missing commit accepted")
	}
}

func TestPublishStatus(t *testing.T) {
	for _, conclusion := range []string{"", "success"} {
		t.Run("conclusion="+conclusion, func(t *testing.T) {
			source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/repos/acme/infra/check-runs" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				wantStatus := "queued"
				if conclusion != "" {
					wantStatus = "completed"
					if body["conclusion"] != conclusion || body["completed_at"] == nil {
						t.Errorf("missing completion evidence: %#v", body)
					}
				} else if _, ok := body["conclusion"]; ok {
					t.Error("queued check has a conclusion")
				}
				if body["status"] != wantStatus || body["head_sha"] != "head" || body["name"] != "Statecraft" || body["details_url"] != "https://statecraft.example/reviews/7" {
					t.Errorf("unexpected check payload: %#v", body)
				}
				output, _ := body["output"].(map[string]any)
				if output["title"] != "Statecraft" || output["summary"] != "Review summary" {
					t.Errorf("unexpected check output: %#v", output)
				}
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"id":99}`)
			})
			err := source.PublishStatus(context.Background(), domain.StatusReport{Repository: testRepo, HeadSHA: "head", Name: "Statecraft", Conclusion: conclusion, Summary: "Review summary", DetailsURL: "https://statecraft.example/reviews/7"})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishStatusRejectsInvalidStates(t *testing.T) {
	for _, tc := range []struct{ status, conclusion string }{{"completed", ""}, {"queued", "success"}, {"in_progress", "failure"}, {"waiting", ""}, {"completed", "stale"}, {"completed", "unknown"}} {
		t.Run(tc.status+"/"+tc.conclusion, func(t *testing.T) {
			source := testSource(t, func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid request reached GitHub")
			})
			err := source.PublishStatus(context.Background(), domain.StatusReport{Repository: testRepo, HeadSHA: "head", Name: "Statecraft", Status: tc.status, Conclusion: tc.conclusion})
			if err == nil {
				t.Fatal("invalid check state accepted")
			}
		})
	}
}

func TestGitHubReviewEvent(t *testing.T) {
	for input, want := range map[string]string{"approved": "APPROVE", "request_changes": "REQUEST_CHANGES", "changes_requested": "REQUEST_CHANGES", "comment": "COMMENT"} {
		got, err := githubReviewEvent(input)
		if err != nil || got != want {
			t.Fatalf("%s: got %q err=%v want %q", input, got, err, want)
		}
	}
	if _, err := githubReviewEvent("auto_approve"); err == nil {
		t.Fatal("expected unsupported decision error")
	}
}
