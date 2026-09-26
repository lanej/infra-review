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
