package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

type Planner interface {
	Plan(context.Context, domain.PlanRequest) (domain.PlanRun, error)
}

type Executor interface {
	Apply(context.Context, domain.ApplyRequest) (domain.ApplyRun, error)
}
