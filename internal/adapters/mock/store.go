package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
)

type ReviewStore struct {
	mu      sync.Mutex
	reviews map[string]domain.Review
}

func NewReviewStore() *ReviewStore { return &ReviewStore{reviews: map[string]domain.Review{}} }

func cloneReview(r domain.Review) domain.Review {
	data, _ := json.Marshal(r)
	var copy domain.Review
	_ = json.Unmarshal(data, &copy)
	copy.SourceDecisions = append([]domain.ExternalReviewDecision(nil), r.SourceDecisions...)
	return copy
}

func (s *ReviewStore) GetReview(ctx context.Context, id string) (domain.Review, error) {
	if err := ctx.Err(); err != nil {
		return domain.Review{}, err
	}
	if id == "pr-1842" {
		return baseReview(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reviews[id]
	if !ok {
		return domain.Review{}, ports.ErrNotFound
	}
	return cloneReview(r), nil
}

func (s *ReviewStore) Create(ctx context.Context, r domain.Review) (domain.Review, error) {
	if err := ctx.Err(); err != nil {
		return domain.Review{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.reviews) >= 512 {
		return domain.Review{}, ports.ErrCapacity
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return domain.Review{}, err
	}
	r.ID = "demo-" + hex.EncodeToString(id[:])
	r.Version = 1
	s.reviews[r.ID] = cloneReview(r)
	return cloneReview(r), nil
}

func (s *ReviewStore) Update(ctx context.Context, id string, version uint64, fn func(*domain.Review) error) (domain.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.Review{}, err
	}
	r, ok := s.reviews[id]
	if !ok {
		return domain.Review{}, ports.ErrNotFound
	}
	if version != r.Version {
		return domain.Review{}, ports.ErrConflict
	}
	copy := cloneReview(r)
	if err := fn(&copy); err != nil {
		return domain.Review{}, err
	}
	copy.Version++
	s.reviews[id] = cloneReview(copy)
	return cloneReview(copy), nil
}
