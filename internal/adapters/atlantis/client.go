package atlantis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

var _ ports.Planner = (*Client)(nil)
var _ ports.Executor = (*Client)(nil)

type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Atlantis URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("Atlantis URL must include scheme and host")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: parsed, token: token, http: httpClient}, nil
}

type commandRequest struct {
	Repository string        `json:"Repository"`
	Ref        string        `json:"Ref"`
	BaseBranch string        `json:"base_branch,omitempty"`
	Type       string        `json:"Type"`
	Projects   []string      `json:"Projects,omitempty"`
	Paths      []commandPath `json:"Paths,omitempty"`
	PR         int64         `json:"PR,omitempty"`
}

type commandPath struct {
	Directory string `json:"directory,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type commandResult struct {
	Error          json.RawMessage `json:"Error"`
	Failure        string          `json:"Failure"`
	ProjectResults []projectResult `json:"ProjectResults"`
	PlansDeleted   bool            `json:"PlansDeleted"`
}

type projectResult struct {
	Error        json.RawMessage `json:"Error"`
	Failure      string          `json:"Failure"`
	PlanSuccess  *planSuccess    `json:"PlanSuccess,omitempty"`
	ApplySuccess string          `json:"ApplySuccess,omitempty"`
	RepoRelDir   string          `json:"RepoRelDir"`
	Workspace    string          `json:"Workspace"`
	ProjectName  string          `json:"ProjectName"`
}

type planSuccess struct {
	TerraformOutput string `json:"TerraformOutput"`
}

type apiError struct {
	Error string `json:"error"`
}

func (c *Client) Plan(ctx context.Context, req domain.PlanRequest) (domain.PlanRun, error) {
	result, err := c.command(ctx, "/api/plan", commandRequestForPlan(req))
	if err != nil {
		return domain.PlanRun{}, err
	}
	run := domain.PlanRun{Attempts: make([]domain.PlanAttempt, 0, len(result.ProjectResults))}
	for _, project := range result.ProjectResults {
		status := projectStatus(project.Error, project.Failure, project.PlanSuccess != nil)
		attempt := domain.PlanAttempt{
			RootID:       rootID(req.Roots, project.ProjectName, project.RepoRelDir, project.Workspace),
			PlannerRef:    project.ProjectName,
			Directory:    project.RepoRelDir,
			Workspace:    project.Workspace,
			Status:       status,
			Failure:      project.Failure,
			ErrorPresent: rawErrorPresent(project.Error),
		}
		if project.PlanSuccess != nil {
			attempt.Output = project.PlanSuccess.TerraformOutput
		}
		run.Attempts = append(run.Attempts, attempt)
	}
	return run, nil
}

func (c *Client) Apply(ctx context.Context, req domain.ApplyRequest) (domain.ApplyRun, error) {
	result, err := c.command(ctx, "/api/apply", commandRequestForApply(req))
	if err != nil {
		return domain.ApplyRun{}, err
	}
	run := domain.ApplyRun{Attempts: make([]domain.ApplyAttempt, 0, len(result.ProjectResults))}
	for _, project := range result.ProjectResults {
		status := projectStatus(project.Error, project.Failure, project.ApplySuccess != "")
		run.Attempts = append(run.Attempts, domain.ApplyAttempt{
			RootID:       rootID(req.Roots, project.ProjectName, project.RepoRelDir, project.Workspace),
			PlannerRef:    project.ProjectName,
			Directory:    project.RepoRelDir,
			Workspace:    project.Workspace,
			Status:       status,
			Output:       project.ApplySuccess,
			Failure:      project.Failure,
			ErrorPresent: rawErrorPresent(project.Error),
		})
	}
	return run, nil
}

func (c *Client) command(ctx context.Context, path string, payload commandRequest) (commandResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return commandResult{}, fmt.Errorf("encode Atlantis request: %w", err)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return commandResult{}, fmt.Errorf("create Atlantis request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("X-Atlantis-Token", c.token)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return commandResult{}, fmt.Errorf("call Atlantis: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return commandResult{}, fmt.Errorf("read Atlantis response: %w", err)
	}
	var remote apiError
	_ = json.Unmarshal(data, &remote)
	if remote.Error != "" {
		return commandResult{}, fmt.Errorf("Atlantis %s: %s", response.Status, remote.Error)
	}

	var result commandResult
	if err := json.Unmarshal(data, &result); err != nil {
		return commandResult{}, fmt.Errorf("decode Atlantis response: %w", err)
	}

	// Atlantis currently returns HTTP 500 when command.Result.HasErrors() is true.
	// ProjectResults still contain the per-root failure evidence Statecraft needs,
	// so preserve a structured command result even on that status.
	if len(result.ProjectResults) > 0 {
		return result, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return commandResult{}, fmt.Errorf("Atlantis %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	if result.Failure != "" || rawErrorPresent(result.Error) {
		return commandResult{}, fmt.Errorf("Atlantis command failed: %s", result.Failure)
	}
	return result, nil
}

func commandRequestForPlan(req domain.PlanRequest) commandRequest {
	return buildCommandRequest(req.Repository.FullName(), req.Ref, req.BaseBranch, req.PullRequest, req.Roots)
}

func commandRequestForApply(req domain.ApplyRequest) commandRequest {
	return buildCommandRequest(req.Repository.FullName(), req.Ref, req.BaseBranch, req.PullRequest, req.Roots)
}

func buildCommandRequest(repository, ref, baseBranch string, pullRequest int64, roots []domain.RootSelector) commandRequest {
	request := commandRequest{
		Repository: repository,
		Ref:        ref,
		BaseBranch: baseBranch,
		Type:       "Github",
		PR:         pullRequest,
	}
	for _, root := range roots {
		if root.PlannerRef != "" {
			request.Projects = append(request.Projects, root.PlannerRef)
			continue
		}
		request.Paths = append(request.Paths, commandPath{
			Directory: root.Directory,
			Workspace: root.Workspace,
		})
	}
	return request
}

func projectStatus(rawError json.RawMessage, failure string, success bool) string {
	if rawErrorPresent(rawError) || failure != "" {
		return "failed"
	}
	if success {
		return "succeeded"
	}
	return "unknown"
}

func rawErrorPresent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func rootID(roots []domain.RootSelector, project, directory, workspace string) string {
	for _, root := range roots {
		if project != "" && root.PlannerRef == project {
			return root.StableID()
		}
		if project == "" && root.Directory == directory && root.Workspace == workspace {
			return root.StableID()
		}
	}
	if project != "" {
		return project
	}
	return domain.RootSelector{Directory: directory, Workspace: workspace}.StableID()
}
