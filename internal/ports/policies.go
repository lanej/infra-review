package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

type PolicyEvaluator interface {
	EvaluatePlan(context.Context, domain.PlanPolicyInput) (domain.PolicyEvaluation, error)
}

type ActionPolicy interface {
	EvaluateAction(context.Context, domain.ActionPolicyInput) (domain.ActionDecision, error)
}
