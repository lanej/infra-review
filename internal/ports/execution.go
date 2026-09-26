package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

// Planner produces execution evidence. A nil error only means the response was
// obtained and understood; callers must inspect aggregate and per-root failures,
// unknown attempts, and discarded plans before assessing completeness.
type Planner interface {
	Plan(context.Context, domain.PlanRequest) (domain.PlanRun, error)
}

// Executor starts an infrastructure command. It does not guarantee application
// of a particular approved plan artifact. Authorization and artifact reconciliation
// must be established by a use case before exposing this capability to users.
type Executor interface {
	Apply(context.Context, domain.ApplyRequest) (domain.ApplyRun, error)
}
