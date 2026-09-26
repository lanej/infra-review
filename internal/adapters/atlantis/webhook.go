package atlantis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lanej/statecraft/internal/domain"
)

const maxWebhookBytes = 1 << 20

type applyWebhook struct {
	Workspace string `json:"Workspace"`
	Repo      struct {
		FullName string `json:"FullName"`
		Owner    string `json:"Owner"`
		Name     string `json:"Name"`
	} `json:"Repo"`
	Pull struct {
		Num        int64  `json:"Num"`
		HeadCommit string `json:"HeadCommit"`
		HeadBranch string `json:"HeadBranch"`
		BaseBranch string `json:"BaseBranch"`
	} `json:"Pull"`
	User struct {
		Username string `json:"Username"`
	} `json:"User"`
	Success     *bool  `json:"Success"`
	Directory   string `json:"Directory"`
	ProjectName string `json:"ProjectName"`
}

// DecodeApplyWebhook maps a previously authenticated Atlantis apply notification.
// It is a decoder, not an HTTP handler or signature verifier. Before calling it,
// an ingress must authenticate the configured webhook-http-headers secret over
// TLS (or use a trusted transport), authorize the repository, and handle replay.
// Atlantis does not specify a GitHub-style HMAC signature for these webhooks.
func DecodeApplyWebhook(reader io.Reader) (domain.ApplyNotification, error) {
	data, err := readLimited(reader, maxWebhookBytes)
	if err != nil {
		return domain.ApplyNotification{}, fmt.Errorf("read Atlantis apply webhook: %w", err)
	}
	var payload applyWebhook
	if err := json.Unmarshal(data, &payload); err != nil {
		return domain.ApplyNotification{}, fmt.Errorf("decode Atlantis apply webhook: %w", err)
	}
	repo := domain.RepositoryRef{Owner: payload.Repo.Owner, Name: payload.Repo.Name}
	if repo.Owner == "" && repo.Name == "" {
		repo.Owner, repo.Name, _ = strings.Cut(payload.Repo.FullName, "/")
	}
	if repo.Owner == "" || repo.Name == "" || strings.ContainsAny(repo.Owner+repo.Name, "/\\ \t\n") || (payload.Repo.FullName != "" && payload.Repo.FullName != repo.FullName()) {
		return domain.ApplyNotification{}, errors.New("Atlantis apply webhook requires a consistent repository identity")
	}
	if payload.Pull.Num <= 0 || payload.Pull.HeadCommit == "" || payload.User.Username == "" || payload.Success == nil {
		return domain.ApplyNotification{}, errors.New("Atlantis apply webhook requires pull request, commit, actor, and success fields")
	}
	directory, err := normalizeDirectory(payload.Directory)
	if err != nil {
		return domain.ApplyNotification{}, fmt.Errorf("invalid Atlantis apply webhook: %w", err)
	}
	root := RootFromProject(Project{Name: payload.ProjectName, Directory: directory, Workspace: payload.Workspace})
	return domain.ApplyNotification{
		Repository:  repo,
		PullRequest: payload.Pull.Num,
		HeadSHA:     payload.Pull.HeadCommit,
		HeadRef:     payload.Pull.HeadBranch,
		BaseRef:     payload.Pull.BaseBranch,
		Actor:       payload.User.Username,
		Root:        root,
		Success:     *payload.Success,
		Source:      "atlantis",
	}, nil
}
