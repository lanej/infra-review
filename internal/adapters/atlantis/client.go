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
	"path"
	"strings"
	"time"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

var _ ports.Planner = (*Client)(nil)
var _ ports.Executor = (*Client)(nil)

const maxResponseBytes = 16 << 20

type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.New("invalid Atlantis URL")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Atlantis URL must be an HTTP(S) origin and optional base path without credentials, query, or fragment")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("Atlantis API token is required for command endpoints")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	client := *httpClient
	// A redirect could leak X-Atlantis-Token or replay an infrastructure command.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{baseURL: parsed, token: token, http: &client}, nil
}

// These wire DTOs mirror the legacy command API, not Atlantis server Go types.
// See runatlantis/atlantis v0.48.0 server/controllers/api_controller.go.
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
	Directory string `json:"directory"`
	Workspace string `json:"workspace"`
}

type commandResult struct {
	Error          json.RawMessage `json:"Error"`
	Failure        string          `json:"Failure"`
	ProjectResults []projectResult `json:"ProjectResults"`
	PlansDeleted   bool            `json:"PlansDeleted"`
}

type projectResult struct {
	Error              json.RawMessage     `json:"Error"`
	Failure            string              `json:"Failure"`
	Command            *int                `json:"Command"`
	PlanSuccess        *planSuccess        `json:"PlanSuccess,omitempty"`
	PolicyCheckResults *policyCheckResults `json:"PolicyCheckResults"`
	ApplySuccess       string              `json:"ApplySuccess,omitempty"`
	RepoRelDir         string              `json:"RepoRelDir"`
	Workspace          string              `json:"Workspace"`
	ProjectName        string              `json:"ProjectName"`
}

// Only presence is used here: command completion is not policy approval.
type policyCheckResults struct {
	PolicySetResults []json.RawMessage `json:"PolicySetResults"`
}

type planSuccess struct {
	TerraformOutput string `json:"TerraformOutput"`
}

func (c *Client) Plan(ctx context.Context, req domain.PlanRequest) (domain.PlanRun, error) {
	payload, err := buildCommandRequest(req.Repository, req.Ref, req.BaseBranch, req.PullRequest, req.Roots)
	if err != nil {
		return domain.PlanRun{}, err
	}
	result, err := c.command(ctx, "/api/plan", payload)
	if err != nil {
		return domain.PlanRun{}, err
	}
	run := domain.PlanRun{
		Attempts:       make([]domain.PlanAttempt, 0, len(result.ProjectResults)),
		Failure:        result.Failure,
		ErrorPresent:   rawErrorPresent(result.Error),
		PlansDiscarded: result.PlansDeleted,
	}
	for _, project := range result.ProjectResults {
		status := projectStatus(project.Error, project.Failure, project.PlanSuccess != nil || project.PolicyCheckResults != nil)
		attempt := domain.PlanAttempt{
			RootID:       rootID(req.Roots, project.ProjectName, project.RepoRelDir, project.Workspace),
			PlannerRef:   project.ProjectName,
			Directory:    project.RepoRelDir,
			Workspace:    project.Workspace,
			Status:       status,
			Phase:        planPhase(project),
			Failure:      project.Failure,
			ErrorPresent: rawErrorPresent(project.Error),
		}
		if project.PlanSuccess != nil {
			attempt.Output = project.PlanSuccess.TerraformOutput
		}
		run.Attempts = append(run.Attempts, attempt)
	}
	// Omitted results cannot be treated as completed roots. This also exposes
	// Atlantis's valid no-projects/skipped response without inventing success.
	for _, root := range req.Roots {
		id := selectedRootID(root)
		found := false
		for _, attempt := range run.Attempts {
			found = found || (attempt.RootID == id && attempt.Phase == "plan")
		}
		if !found {
			run.Attempts = append(run.Attempts, domain.PlanAttempt{RootID: id, PlannerRef: root.PlannerRef, Directory: root.Directory, Workspace: defaultWorkspace(root.Workspace), Phase: "plan", Status: "unknown", Failure: "Atlantis returned no plan result for the selected root"})
		}
	}
	return run, nil
}

func planPhase(project projectResult) string {
	if project.Command != nil {
		switch *project.Command {
		case 1: // Atlantis command.Plan
			return "plan"
		case 3: // Atlantis command.PolicyCheck
			return "policy_check"
		default:
			return "unknown"
		}
	}
	// Public documentation examples omit Command.
	if project.PolicyCheckResults != nil {
		return "policy_check"
	}
	return "plan"
}

func selectedRootID(root domain.RootSelector) string {
	if root.PlannerRef == "" {
		if clean, err := normalizeDirectory(root.Directory); err == nil {
			root.Directory = clean
		}
		root.Workspace = defaultWorkspace(root.Workspace)
	}
	return root.StableID()
}

// Apply invokes Atlantis's plan-then-apply endpoint. It cannot apply a pinned
// Statecraft plan artifact and must not be used as an approval enforcement gate.
func (c *Client) Apply(ctx context.Context, req domain.ApplyRequest) (domain.ApplyRun, error) {
	payload, err := buildCommandRequest(req.Repository, req.Ref, req.BaseBranch, req.PullRequest, req.Roots)
	if err != nil {
		return domain.ApplyRun{}, err
	}
	result, err := c.command(ctx, "/api/apply", payload)
	if err != nil {
		return domain.ApplyRun{}, err
	}
	run := domain.ApplyRun{
		Attempts:     make([]domain.ApplyAttempt, 0, len(result.ProjectResults)),
		Failure:      result.Failure,
		ErrorPresent: rawErrorPresent(result.Error),
	}
	for _, project := range result.ProjectResults {
		run.Attempts = append(run.Attempts, domain.ApplyAttempt{
			RootID:       rootID(req.Roots, project.ProjectName, project.RepoRelDir, project.Workspace),
			PlannerRef:   project.ProjectName,
			Directory:    project.RepoRelDir,
			Workspace:    project.Workspace,
			Status:       projectStatus(project.Error, project.Failure, project.ApplySuccess != ""),
			Output:       project.ApplySuccess,
			Failure:      project.Failure,
			ErrorPresent: rawErrorPresent(project.Error),
		})
	}
	for _, root := range req.Roots {
		id := selectedRootID(root)
		found := false
		for _, attempt := range run.Attempts {
			found = found || attempt.RootID == id
		}
		if !found {
			run.Attempts = append(run.Attempts, domain.ApplyAttempt{RootID: id, PlannerRef: root.PlannerRef, Directory: root.Directory, Workspace: defaultWorkspace(root.Workspace), Status: "unknown", Failure: "Atlantis returned no apply result for the selected root"})
		}
	}
	return run, nil
}

func (c *Client) command(ctx context.Context, route string, payload commandRequest) (commandResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return commandResult{}, fmt.Errorf("encode Atlantis request: %w", err)
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + route
	endpoint.RawPath = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return commandResult{}, fmt.Errorf("create Atlantis request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Atlantis-Token", c.token)

	response, err := c.http.Do(request)
	if err != nil {
		return commandResult{}, fmt.Errorf("call Atlantis: %w", err)
	}
	defer response.Body.Close()
	data, err := readLimited(response.Body, maxResponseBytes)
	if err != nil {
		return commandResult{}, fmt.Errorf("read Atlantis response: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return commandResult{}, fmt.Errorf("Atlantis HTTP %d: expected a JSON command response", response.StatusCode)
	}
	// Request/auth/setup failures use an exact lower-case key. Do not confuse it
	// with command.Result.Error, whose non-nil Go error is encoded as {}.
	if raw, ok := fields["error"]; ok {
		var message string
		if err := json.Unmarshal(raw, &message); err != nil || message == "" {
			return commandResult{}, fmt.Errorf("Atlantis HTTP %d: malformed API error", response.StatusCode)
		}
		return commandResult{}, fmt.Errorf("Atlantis HTTP %d: %s", response.StatusCode, message)
	}
	// HTTP 500 carries command.Result on project failures; other error statuses
	// cannot establish that any returned command evidence is authoritative.
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusInternalServerError {
		return commandResult{}, fmt.Errorf("Atlantis HTTP %d: unexpected command status", response.StatusCode)
	}
	for _, field := range []string{"Error", "Failure", "ProjectResults", "PlansDeleted"} {
		if _, ok := fields[field]; !ok {
			return commandResult{}, fmt.Errorf("Atlantis response is missing %s", field)
		}
	}
	var result commandResult
	if err := json.Unmarshal(data, &result); err != nil {
		return commandResult{}, fmt.Errorf("decode Atlantis command response: %w", err)
	}
	hasFailure := rawErrorPresent(result.Error) || result.Failure != ""
	for _, project := range result.ProjectResults {
		hasFailure = hasFailure || rawErrorPresent(project.Error) || project.Failure != ""
		if project.RepoRelDir == "" || project.Workspace == "" {
			return commandResult{}, errors.New("Atlantis project result is missing directory or workspace")
		}
	}
	if response.StatusCode == http.StatusInternalServerError && !hasFailure {
		return commandResult{}, errors.New("Atlantis HTTP 500 lacks command or project failure evidence")
	}
	// Aggregate failures and PlansDeleted remain observable even when other
	// projects succeeded. An empty, valid result means no selected project ran.
	return result, nil
}

func buildCommandRequest(repository domain.RepositoryRef, ref, baseBranch string, pullRequest int64, roots []domain.RootSelector) (commandRequest, error) {
	if repository.Owner == "" || repository.Name == "" || strings.ContainsAny(repository.Owner+repository.Name, "/\\ \t\n") {
		return commandRequest{}, errors.New("Atlantis requires a repository owner and name")
	}
	if strings.TrimSpace(ref) == "" || pullRequest < 0 {
		return commandRequest{}, errors.New("Atlantis requires a git reference and a non-negative pull request number")
	}
	if len(roots) == 0 {
		return commandRequest{}, errors.New("Atlantis requires at least one explicit root selector")
	}
	request := commandRequest{Repository: repository.FullName(), Ref: ref, BaseBranch: baseBranch, Type: "Github", PR: pullRequest}
	seen := make(map[string]bool)
	seenIDs := make(map[string]bool)
	for _, root := range roots {
		var key string
		if root.PlannerRef != "" {
			name := strings.ReplaceAll(root.PlannerRef, "/", "-")
			if url.QueryEscape(name) != name {
				return commandRequest{}, errors.New("Atlantis project selector must be an exact configured project name")
			}
			key = "project:" + root.PlannerRef
			request.Projects = append(request.Projects, root.PlannerRef)
		} else {
			directory, err := normalizeDirectory(root.Directory)
			if err != nil {
				return commandRequest{}, err
			}
			workspace := defaultWorkspace(root.Workspace)
			key = "path:" + directory + "\x00" + workspace
			request.Paths = append(request.Paths, commandPath{Directory: directory, Workspace: workspace})
		}
		if seen[key] {
			return commandRequest{}, errors.New("Atlantis root selectors must not repeat a project or directory/workspace")
		}
		seen[key] = true
		id := selectedRootID(root)
		if seenIDs[id] {
			return commandRequest{}, errors.New("Atlantis root selectors must have distinct Statecraft IDs")
		}
		seenIDs[id] = true
	}
	// Keep a single selector family across Atlantis versions, and prevent a
	// named project and a path from unintentionally selecting the same root.
	if len(request.Projects) != 0 && len(request.Paths) != 0 {
		return commandRequest{}, errors.New("Atlantis command must use either named projects or directory/workspace selectors, not both")
	}
	return request, nil
}

func normalizeDirectory(directory string) (string, error) {
	if directory == "" || strings.ContainsAny(directory, "\\\x00") || path.IsAbs(directory) {
		return "", errors.New("Atlantis root directory must be an explicit repository-relative path (use . for the root)")
	}
	clean := path.Clean(directory)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("Atlantis root directory must stay within the repository")
	}
	return clean, nil
}

func defaultWorkspace(workspace string) string {
	if workspace == "" {
		return "default"
	}
	return workspace
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
	}
	// Atlantis may resolve a path selector to a named project. Preserve the
	// caller's root ID in that case, but never merge two distinct named roots.
	for _, root := range roots {
		clean, err := normalizeDirectory(root.Directory)
		if err == nil && root.PlannerRef == "" && clean == directory && defaultWorkspace(root.Workspace) == workspace {
			root.Directory, root.Workspace = clean, defaultWorkspace(root.Workspace)
			return root.StableID()
		}
	}
	return domain.RootSelector{PlannerRef: project, Directory: directory, Workspace: workspace}.StableID()
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("payload exceeds size limit")
	}
	return data, nil
}
