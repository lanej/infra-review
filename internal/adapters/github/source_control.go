package githubadapter

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v72/github"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

var _ ports.SourceControl = (*SourceControl)(nil)

type SourceControl struct {
	client *gh.Client
}

func NewSourceControl(client *gh.Client) *SourceControl {
	return &SourceControl{client: client}
}

func (s *SourceControl) GetChange(ctx context.Context, repo domain.RepositoryRef, number int64) (domain.SourceChange, error) {
	pr, _, err := s.client.PullRequests.Get(ctx, repo.Owner, repo.Name, int(number))
	if err != nil {
		return domain.SourceChange{}, fmt.Errorf("get GitHub pull request: %w", err)
	}

	change := domain.SourceChange{
		Repository: repo,
		Number:     int64(pr.GetNumber()),
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		Author:     pr.GetUser().GetLogin(),
		State:      pr.GetState(),
		Draft:      pr.GetDraft(),
		URL:        pr.GetHTMLURL(),
	}
	if pr.Head != nil {
		change.HeadSHA = pr.Head.GetSHA()
		change.HeadRef = pr.Head.GetRef()
	}
	if pr.Base != nil {
		change.BaseRef = pr.Base.GetRef()
	}
	return change, nil
}

func (s *SourceControl) ListChangedFiles(ctx context.Context, repo domain.RepositoryRef, number int64) ([]domain.ChangedFile, error) {
	opts := &gh.ListOptions{PerPage: 100}
	var result []domain.ChangedFile
	for {
		files, response, err := s.client.PullRequests.ListFiles(ctx, repo.Owner, repo.Name, int(number), opts)
		if err != nil {
			return nil, fmt.Errorf("list GitHub pull request files: %w", err)
		}
		for _, file := range files {
			result = append(result, domain.ChangedFile{
				Path:         file.GetFilename(),
				PreviousPath: file.GetPreviousFilename(),
				Status:       file.GetStatus(),
				Additions:    file.GetAdditions(),
				Deletions:    file.GetDeletions(),
				Changes:      file.GetChanges(),
			})
		}
		if response == nil || response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}
	return result, nil
}

func (s *SourceControl) ListReviewDecisions(ctx context.Context, repo domain.RepositoryRef, number int64) ([]domain.ExternalReviewDecision, error) {
	opts := &gh.ListOptions{PerPage: 100}
	var result []domain.ExternalReviewDecision
	for {
		reviews, response, err := s.client.PullRequests.ListReviews(ctx, repo.Owner, repo.Name, int(number), opts)
		if err != nil {
			return nil, fmt.Errorf("list GitHub pull request reviews: %w", err)
		}
		for _, review := range reviews {
			result = append(result, reviewDecision(review))
		}
		if response == nil || response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}
	return result, nil
}

func (s *SourceControl) PublishDecision(ctx context.Context, req domain.PublishDecisionRequest) (domain.ExternalReviewDecision, error) {
	event, err := githubReviewEvent(req.Decision)
	if err != nil {
		return domain.ExternalReviewDecision{}, err
	}

	body := req.Body
	if req.PlanSetID != "" {
		if body != "" {
			body += "\n\n"
		}
		body += "<!-- statecraft-plan-set:" + req.PlanSetID + " -->"
	}

	review, _, err := s.client.PullRequests.CreateReview(
		ctx,
		req.Repository.Owner,
		req.Repository.Name,
		int(req.Number),
		&gh.PullRequestReviewRequest{
			CommitID: gh.String(req.CommitSHA),
			Body:     gh.String(body),
			Event:    gh.String(event),
		},
	)
	if err != nil {
		return domain.ExternalReviewDecision{}, fmt.Errorf("publish GitHub pull request review: %w", err)
	}
	return reviewDecision(review), nil
}

func (s *SourceControl) PublishStatus(ctx context.Context, report domain.StatusReport) error {
	status := report.Status
	if status == "" {
		status = "completed"
	}

	opts := gh.CreateCheckRunOptions{
		Name:    report.Name,
		HeadSHA: report.HeadSHA,
		Status:  gh.String(status),
	}
	if report.Conclusion != "" {
		opts.Conclusion = gh.String(report.Conclusion)
	}
	if report.DetailsURL != "" {
		opts.DetailsURL = gh.String(report.DetailsURL)
	}
	if report.Summary != "" {
		opts.Output = &gh.CheckRunOutput{
			Title:   gh.String(report.Name),
			Summary: gh.String(report.Summary),
		}
	}

	if _, _, err := s.client.Checks.CreateCheckRun(ctx, report.Repository.Owner, report.Repository.Name, opts); err != nil {
		return fmt.Errorf("publish GitHub check run: %w", err)
	}
	return nil
}

func reviewDecision(review *gh.PullRequestReview) domain.ExternalReviewDecision {
	decision := strings.ToLower(review.GetState())
	switch decision {
	case "changes_requested":
		decision = "changes_requested"
	case "approved":
		decision = "approved"
	case "commented":
		decision = "commented"
	case "dismissed":
		decision = "dismissed"
	}
	result := domain.ExternalReviewDecision{
		ID:        fmt.Sprintf("%d", review.GetID()),
		Actor:     review.GetUser().GetLogin(),
		Decision:  decision,
		CommitSHA: review.GetCommitID(),
		Body:      review.GetBody(),
		URL:       review.GetHTMLURL(),
		Source:    "github",
	}
	if review.SubmittedAt != nil {
		result.CreatedAt = review.SubmittedAt.Time
	}
	return result
}

func githubReviewEvent(decision string) (string, error) {
	switch strings.ToLower(decision) {
	case "approved", "approve":
		return "APPROVE", nil
	case "changes_requested", "request_changes", "request-changes":
		return "REQUEST_CHANGES", nil
	case "comment", "commented":
		return "COMMENT", nil
	default:
		return "", fmt.Errorf("unsupported review decision %q", decision)
	}
}
