package ports

import (
	"context"
	"github.com/lanej/infra-review/internal/domain"
)

type ReviewStore interface {
	GetReview(context.Context, string) (domain.Review, error)
}
