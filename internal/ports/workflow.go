package ports

import (
	"context"
	"errors"

	"github.com/lanej/statecraft/internal/domain"
)

var ErrNotFound = errors.New("review not found")
var ErrConflict = errors.New("review changed; refresh before taking another action")
var ErrCapacity = errors.New("demo session limit reached; restart the demo server")

// WorkflowStore serializes a mutation against an expected version, commits only
// on success, and returns detached values. Production persistence is not supplied.
type WorkflowStore interface {
	ReviewStore
	Create(context.Context, domain.Review) (domain.Review, error)
	Update(context.Context, string, uint64, func(*domain.Review) error) (domain.Review, error)
}

// DemoPlanner supplies synthetic changes, never infrastructure commands. It is
// deliberately distinct from Planner, which can invoke external execution.
type DemoPlanner interface {
	ChangesForPlan(context.Context, domain.Review) ([]domain.Change, error)
}
