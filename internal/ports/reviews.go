package ports

import (
	"context"

	"github.com/lanej/statecraft/internal/domain"
)

type ReviewStore interface {
	GetReview(context.Context, string) (domain.Review, error)
}
