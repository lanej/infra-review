package service

import (
	"context"
	"github.com/lanej/infra-review/internal/domain"
	"github.com/lanej/infra-review/internal/ports"
)

type Reviews struct{ store ports.ReviewStore }

func NewReviews(store ports.ReviewStore) *Reviews { return &Reviews{store:store} }
func (s *Reviews) Get(ctx context.Context, id string) (domain.Review, error) {
	return s.store.GetReview(ctx,id)
}
