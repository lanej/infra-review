package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lanej/statecraft/internal/adapters/httpapi"
	"github.com/lanej/statecraft/internal/adapters/mock"
	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/service"
)

func TestDemoHTTPBoundary(t *testing.T) {
	workflow := service.NewDemoWorkflow(mock.NewReviewStore(), mock.Planner{}, mock.Policies{}, mock.Policies{}, time.Now)
	handler := httpapi.DemoHandler(workflow, mock.SeedReview)
	call := func(method, path, body, site string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Sec-Fetch-Site", site)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	created := call("POST", "/api/demos", `{"scenario":"ready"}`, "same-origin")
	if created.Code != 200 {
		t.Fatal(created.Code, created.Body.String())
	}
	var r domain.Review
	if err := json.Unmarshal(created.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Demo || r.Version != 1 || r.ID == "pr-1842" {
		t.Fatal("session was not isolated")
	}
	for _, body := range []string{`{"scenario":"unknown"}`, `{"scenario":"ready","actor":"admin"}`, `{"scenario":"ready"} {}`, `{`} {
		if got := call("POST", "/api/demos", body, "same-origin"); got.Code != 400 {
			t.Fatal(got.Code, body)
		}
	}
	if got := call("POST", "/api/demos", `{"scenario":"ready"}`, "cross-site"); got.Code != 403 {
		t.Fatal(got.Code)
	}
	if got := call("DELETE", "/api/demos/"+r.ID, "", "same-origin"); got.Code != http.StatusMethodNotAllowed {
		t.Fatal(got.Code)
	}
	if got := call("GET", "/api/demos/missing", "", "same-origin"); got.Code != 404 {
		t.Fatal(got.Code)
	}
	path := "/api/demos/" + r.ID + "/actions"
	if got := call("POST", path, `{"action":"approve","expectedVersion":1}`, "same-origin"); got.Code != 200 {
		t.Fatal(got.Code, got.Body.String())
	}
	if got := call("POST", path, `{"action":"approve","expectedVersion":1}`, "same-origin"); got.Code != 409 {
		t.Fatal(got.Code)
	}
	fresh, err := workflow.Get(context.Background(), r.ID)
	if err != nil || len(fresh.Decisions) != 1 {
		t.Fatal("duplicate action changed state", err)
	}
}
