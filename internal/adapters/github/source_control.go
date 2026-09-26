package githubadapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v92/github"

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
	if pr == nil {
		return domain.SourceChange{}, fmt.Errorf("get GitHub pull request: empty response")
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
	if pr.GetMerged() {
		change.State = "merged"
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
	opts := &gh.ListOptions{Page: 1, PerPage: 100}
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
		if response.NextPage <= opts.Page {
			return nil, fmt.Errorf("list GitHub pull request files: nonadvancing pagination")
		}
		opts.Page = response.NextPage
	}
	// GitHub caps this endpoint at 3,000 files. Prove completeness before
	// returning a result at the cap, rather than silently accepting truncation.
	if len(result) >= 3000 {
		pr, _, err := s.client.PullRequests.Get(ctx, repo.Owner, repo.Name, int(number))
		if err != nil {
			return nil, fmt.Errorf("verify GitHub pull request file count: %w", err)
		}
		if pr == nil || pr.ChangedFiles == nil || pr.GetChangedFiles() != len(result) {
			return nil, fmt.Errorf("GitHub pull request file list is incomplete: received %d files at the API limit", len(result))
		}
	}
	return result, nil
}

func (s *SourceControl) ListReviewDecisions(ctx context.Context, repo domain.RepositoryRef, number int64) ([]domain.ExternalReviewDecision, error) {
	opts := &gh.ListOptions{Page: 1, PerPage: 100}
	var result []domain.ExternalReviewDecision
	for {
		reviews, response, err := s.client.PullRequests.ListReviews(ctx, repo.Owner, repo.Name, int(number), opts)
		if err != nil {
			return nil, fmt.Errorf("list GitHub pull request reviews: %w", err)
		}
		for _, review := range reviews {
			// Pending reviews are private drafts, not submitted decisions.
			if review == nil || review.SubmittedAt == nil || review.GetState() == "PENDING" {
				continue
			}
			result = append(result, reviewDecision(review))
		}
		if response == nil || response.NextPage == 0 {
			break
		}
		if response.NextPage <= opts.Page {
			return nil, fmt.Errorf("list GitHub pull request reviews: nonadvancing pagination")
		}
		opts.Page = response.NextPage
	}
	return result, nil
}

func (s *SourceControl) PublishDecision(ctx context.Context, req domain.PublishDecisionRequest) (domain.ExternalReviewDecision, error) {
	// Omitting commit_id makes GitHub choose the current head; require the
	// reviewed revision explicitly so a concurrent push cannot retarget a review.
	if strings.TrimSpace(req.CommitSHA) == "" {
		return domain.ExternalReviewDecision{}, fmt.Errorf("publish GitHub pull request review: commit SHA is required")
	}
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
			CommitID: gh.Ptr(req.CommitSHA),
			Body:     gh.Ptr(body),
			Event:    gh.Ptr(event),
		},
	)
	if err != nil {
		return domain.ExternalReviewDecision{}, fmt.Errorf("publish GitHub pull request review: %w", err)
	}
	if review == nil {
		return domain.ExternalReviewDecision{}, fmt.Errorf("publish GitHub pull request review: empty response")
	}
	return reviewDecision(review), nil
}

func (s *SourceControl) PublishStatus(ctx context.Context, report domain.StatusReport) error {
	if strings.TrimSpace(report.HeadSHA) == "" || strings.TrimSpace(report.Name) == "" {
		return fmt.Errorf("publish GitHub check run: head SHA and name are required")
	}
	status := report.Status
	if status == "" {
		status = "queued"
		if report.Conclusion != "" {
			status = "completed"
		}
	}
	switch status {
	case "queued", "in_progress":
		if report.Conclusion != "" {
			return fmt.Errorf("publish GitHub check run: conclusion requires completed status")
		}
	case "completed":
		switch report.Conclusion {
		case "action_required", "cancelled", "failure", "neutral", "success", "skipped", "timed_out":
		default:
			return fmt.Errorf("publish GitHub check run: invalid or missing conclusion %q", report.Conclusion)
		}
	default:
		return fmt.Errorf("publish GitHub check run: unsupported status %q", status)
	}

	opts := gh.CreateCheckRunOptions{
		Name:    report.Name,
		HeadSHA: report.HeadSHA,
		Status:  gh.Ptr(status),
	}
	if report.Conclusion != "" {
		opts.Conclusion = gh.Ptr(report.Conclusion)
		opts.CompletedAt = &gh.Timestamp{Time: time.Now().UTC()}
	}
	if report.DetailsURL != "" {
		opts.DetailsURL = gh.Ptr(report.DetailsURL)
	}
	if report.Summary != "" {
		opts.Output = &gh.CheckRunOutput{
			Title:   gh.Ptr(report.Name),
			Summary: gh.Ptr(report.Summary),
		}
	}

	if _, _, err := s.client.Checks.CreateCheckRun(ctx, report.Repository.Owner, report.Repository.Name, opts); err != nil {
		return fmt.Errorf("publish GitHub check run: %w", err)
	}
	return nil
}

func reviewDecision(review *gh.PullRequestReview) domain.ExternalReviewDecision {
	decision := strings.ToLower(review.GetState())
	// Review bodies are editable user content. A Statecraft trace marker in
	// them cannot establish a trusted relationship to a plan set or approval.
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
