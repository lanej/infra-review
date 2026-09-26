package atlantis

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lanej/statecraft/internal/domain"
)

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
	Success     bool   `json:"Success"`
	Directory   string `json:"Directory"`
	ProjectName string `json:"ProjectName"`
}

func DecodeApplyWebhook(reader io.Reader) (domain.ApplyNotification, error) {
	var payload applyWebhook
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return domain.ApplyNotification{}, fmt.Errorf("decode Atlantis apply webhook: %w", err)
	}

	repo := domain.RepositoryRef{Owner: payload.Repo.Owner, Name: payload.Repo.Name}
	if repo.Owner == "" && payload.Repo.FullName != "" {
		for i, ch := range payload.Repo.FullName {
			if ch == '/' {
				repo.Owner = payload.Repo.FullName[:i]
				repo.Name = payload.Repo.FullName[i+1:]
				break
			}
		}
	}

	root := domain.RootSelector{
		ProjectName: payload.ProjectName,
		Directory:   payload.Directory,
		Workspace:   payload.Workspace,
	}

	return domain.ApplyNotification{
		Repository:  repo,
		PullRequest: payload.Pull.Num,
		HeadSHA:     payload.Pull.HeadCommit,
		HeadRef:     payload.Pull.HeadBranch,
		BaseRef:     payload.Pull.BaseBranch,
		Actor:       payload.User.Username,
		Root:        root,
		Success:     payload.Success,
		Source:      "atlantis",
	}, nil
}
