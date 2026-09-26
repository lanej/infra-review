package atlantis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lanej/statecraft/internal/domain"
)

func TestPlanMapsProjectResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plan" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Atlantis-Token"); got != "secret" {
			t.Fatalf("token = %q", got)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["Repository"] != "acme/infra" {
			t.Fatalf("repository = %#v", request["Repository"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
			"Error": null,
			"Failure": "",
			"ProjectResults": [
				{
					"Error": null,
					"Failure": "",
					"PlanSuccess": {"TerraformOutput": "Plan: 1 to add, 1 to change, 0 to destroy."},
					"RepoRelDir": "prod/api",
					"Workspace": "default",
					"ProjectName": "api"
				},
				{
					"Error": {},
					"Failure": "",
					"RepoRelDir": "prod/db",
					"Workspace": "default",
					"ProjectName": "db"
				}
			],
			"PlansDeleted": false
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}

	run, err := client.Plan(context.Background(), domain.PlanRequest{
		Repository:  domain.RepositoryRef{Owner: "acme", Name: "infra"},
		Ref:         "feature/change",
		BaseBranch:  "main",
		PullRequest: 42,
		Roots: []domain.RootSelector{
			{ID: "root-api", PlannerRef: "api", Directory: "prod/api", Workspace: "default"},
			{ID: "root-db", PlannerRef: "db", Directory: "prod/db", Workspace: "default"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 2 {
		t.Fatalf("attempts = %d", len(run.Attempts))
	}
	if run.Attempts[0].RootID != "root-api" || run.Attempts[0].Status != "succeeded" {
		t.Fatalf("api attempt = %#v", run.Attempts[0])
	}
	if run.Attempts[1].RootID != "root-db" || run.Attempts[1].Status != "failed" || !run.Attempts[1].ErrorPresent {
		t.Fatalf("db attempt = %#v", run.Attempts[1])
	}
}

func TestApplyWebhookMapsToDomain(t *testing.T) {
	payload := `{
		"Workspace":"default",
		"Repo":{"FullName":"acme/infra","Owner":"acme","Name":"infra"},
		"Pull":{"Num":42,"HeadCommit":"abc123","HeadBranch":"feature/change","BaseBranch":"main"},
		"User":{"Username":"josh"},
		"Success":true,
		"Directory":"prod/api",
		"ProjectName":"api"
	}`

	got, err := DecodeApplyWebhook(strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if got.Repository.FullName() != "acme/infra" || got.PullRequest != 42 || got.Root.PlannerRef != "api" || !got.Success {
		t.Fatalf("notification = %#v", got)
	}
}

func testPlanRequest() domain.PlanRequest {
	return domain.PlanRequest{Repository: domain.RepositoryRef{Owner: "acme", Name: "infra"}, Ref: "feature/change", PullRequest: 42, Roots: []domain.RootSelector{{ID: "root-api", Directory: "./prod/api/"}}}
}

func TestPlanPathSelectorPreservesRootAndAggregateEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/atlantis/api/plan" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var request commandRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Type != "Github" || request.PR != 42 || len(request.Projects) != 0 || len(request.Paths) != 1 || request.Paths[0].Directory != "prod/api" || request.Paths[0].Workspace != "default" {
			t.Errorf("unexpected request: %#v", request)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"Error":{},"Failure":"another project failed","PlansDeleted":true,"ProjectResults":[{"Error":null,"Failure":"","PlanSuccess":{"TerraformOutput":"Plan: 1 to add."},"ProjectName":"resolved-name","RepoRelDir":"prod/api","Workspace":"default"}]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/atlantis/", "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	run, err := client.Plan(context.Background(), testPlanRequest())
	if err != nil {
		t.Fatal(err)
	}
	if !run.ErrorPresent || !run.PlansDiscarded || run.Failure != "another project failed" || len(run.Attempts) != 1 || run.Attempts[0].RootID != "root-api" {
		t.Fatalf("lost aggregate or root evidence: %#v", run)
	}
}

func TestCommandRejectsUnsafeSelectorsBeforeHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]domain.RootSelector{
		"none":            nil,
		"empty":           {{}},
		"workspace only":  {{Workspace: "prod"}},
		"mixed":           {{PlannerRef: "api"}, {Directory: "prod/db"}},
		"duplicate ID":    {{ID: "same", PlannerRef: "api"}, {ID: "same", PlannerRef: "db"}},
		"regexp selector": {{PlannerRef: ".*"}},
		"duplicate name":  {{PlannerRef: "api"}, {PlannerRef: "api"}},
		"duplicate path":  {{Directory: "./prod"}, {Directory: "prod/", Workspace: "default"}},
		"escape":          {{Directory: "../prod"}},
		"absolute":        {{Directory: "/prod"}},
	}
	for name, roots := range cases {
		t.Run(name, func(t *testing.T) {
			req := testPlanRequest()
			req.Roots = roots
			if _, err := client.Plan(context.Background(), req); err == nil {
				t.Fatal("expected selector error")
			}
			if _, err := client.Apply(context.Background(), domain.ApplyRequest(req)); err == nil {
				t.Fatal("expected apply selector error")
			}
		})
	}
	if calls != 0 {
		t.Fatalf("invalid requests reached Atlantis: %d", calls)
	}
}

func TestCommandRejectsMalformedAndHTTPFailures(t *testing.T) {
	valid := `{"Error":null,"Failure":"","ProjectResults":[],"PlansDeleted":false}`
	cases := []struct {
		name, body string
		status     int
	}{
		{"empty object", `{}`, 200},
		{"null", `null`, 200},
		{"html", `<html>proxy error</html>`, 502},
		{"truncated", `{"Error":null`, 200},
		{"two values", valid + valid, 200},
		{"api failure", `{"error":"missing token"}`, 401},
		{"wrong project shape", `{"Error":null,"Failure":"","ProjectResults":[{}],"PlansDeleted":false}`, 200},
		{"wrong projects type", `{"Error":null,"Failure":"","ProjectResults":{},"PlansDeleted":false}`, 200},
		{"untrusted status", valid, 403},
		{"server failure without command failure", `{"Error":null,"Failure":"","ProjectResults":[{"PlanSuccess":{"TerraformOutput":"No changes."},"RepoRelDir":"prod/api","Workspace":"default"}],"PlansDeleted":false}`, 500},
		{"oversized", valid + strings.Repeat(" ", maxResponseBytes), 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "secret", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Plan(context.Background(), testPlanRequest()); err == nil {
				t.Fatal("expected protocol error")
			}
		})
	}
}

func TestPlanDistinguishesPolicyChecksAndMissingRoots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Error":null,"Failure":"","PlansDeleted":false,"ProjectResults":[
		{"Command":1,"PlanSuccess":{"TerraformOutput":"No changes."},"ProjectName":"api","RepoRelDir":"prod/api","Workspace":"default"},
		{"Command":3,"PolicyCheckResults":{"PolicySetResults":[]},"ProjectName":"api","RepoRelDir":"prod/api","Workspace":"default"}
		]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	req := testPlanRequest()
	req.Roots = []domain.RootSelector{{ID: "root-api", PlannerRef: "api"}, {ID: "root-db", PlannerRef: "db"}}
	run, err := client.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 3 || run.Attempts[0].Phase != "plan" || run.Attempts[1].Phase != "policy_check" || run.Attempts[2].RootID != "root-db" || run.Attempts[2].Status != "unknown" {
		t.Fatalf("attempt phases/completeness = %#v", run.Attempts)
	}
}

func TestApplyMapsFailuresAndMissingRoots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apply" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"Error":{},"Failure":"command failure","PlansDeleted":false,"ProjectResults":[{"Error":{},"Failure":"apply denied","ApplySuccess":"unexpected output","ProjectName":"api","RepoRelDir":"prod/api","Workspace":"default"}]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	req := domain.ApplyRequest(testPlanRequest())
	req.Roots = []domain.RootSelector{{ID: "root-api", PlannerRef: "api"}, {ID: "root-db", PlannerRef: "db"}}
	run, err := client.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !run.ErrorPresent || run.Failure != "command failure" || len(run.Attempts) != 2 || run.Attempts[0].Status != "failed" || !run.Attempts[0].ErrorPresent || run.Attempts[1].Status != "unknown" {
		t.Fatalf("apply evidence = %#v", run)
	}
}

func TestClientDoesNotForwardTokenOrReplayCommandsOnRedirect(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Plan(context.Background(), testPlanRequest()); err == nil {
		t.Fatal("expected redirect rejection")
	}
	if redirected {
		t.Fatal("followed command redirect")
	}
}

func TestApplyWebhookRejectsMissingIdentityAndTrailingPayload(t *testing.T) {
	valid := `{"Repo":{"FullName":"acme/infra"},"Pull":{"Num":42,"HeadCommit":"abc123"},"User":{"Username":"josh"},"Success":false,"Directory":".","Workspace":""}`
	got, err := DecodeApplyWebhook(strings.NewReader(valid))
	if err != nil {
		t.Fatal(err)
	}
	if got.Success || got.Root.Workspace != "default" || got.Root.ID != ".::default" {
		t.Fatalf("notification = %#v", got)
	}
	for _, payload := range []string{
		`{}`, `null`, valid + valid,
		strings.Replace(valid, `"Success":false,`, "", 1),
		strings.Replace(valid, `"Directory":"."`, `"Directory":"../outside"`, 1),
		strings.Replace(valid, `"FullName":"acme/infra"`, `"FullName":"acme/infra","Owner":"other","Name":"infra"`, 1),
		valid + strings.Repeat(" ", maxWebhookBytes),
	} {
		if _, err := DecodeApplyWebhook(strings.NewReader(payload)); err == nil {
			t.Fatal("accepted malformed webhook")
		}
	}
}

func TestPolicyResultUsesAtlantisClearanceIncludingApprovedExceptions(t *testing.T) {
	for _, tc := range []struct {
		name, failure, want string
	}{
		{"uncleared policy", "Some policy sets did not pass.", "failed"},
		{"approved exception", "", "succeeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Upstream PolicyCleared() considers valid approvals as well as Passed.
				// Its command Failure field communicates the final clearance decision.
				result := map[string]any{
					"Error": nil, "Failure": "", "PlansDeleted": false,
					"ProjectResults": []any{map[string]any{
						"Command": 3, "Error": nil, "Failure": tc.failure,
						"RepoRelDir": "prod/api", "Workspace": "default",
						"PolicyCheckResults": map[string]any{"PolicySetResults": []any{map[string]any{"Passed": false, "ReqApprovalCount": 1, "Approvals": []any{map[string]any{"Approver": "operator"}}}}},
					}},
				}
				if tc.failure != "" {
					w.WriteHeader(http.StatusInternalServerError)
				}
				if err := json.NewEncoder(w).Encode(result); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "secret", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			run, err := client.Plan(context.Background(), testPlanRequest())
			if err != nil {
				t.Fatal(err)
			}
			if len(run.Attempts) != 2 || run.Attempts[0].Phase != "policy_check" || run.Attempts[0].Status != tc.want || run.Attempts[1].Status != "unknown" {
				t.Fatalf("policy and missing plan evidence = %#v", run.Attempts)
			}
		})
	}
}
