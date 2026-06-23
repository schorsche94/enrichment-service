package enrichment

import (
	"context"
	"enrichment-service/internal/storage"
	"fmt"
	"math/rand"
	"time"
)

type Fetcher interface {
	Fetch(ctx context.Context, profileID string) (storage.Profile, error)
}

func (s *Simulated) Fetch(ctx context.Context, profileID string) (storage.Profile, error) {
	minDelay := s.MinDelay
	maxDelay := s.MaxDelay
	if minDelay == 0 && maxDelay == 0 {
		minDelay, maxDelay = 100*time.Millisecond, 400*time.Millisecond
	}

	delay := minDelay
	if maxDelay > minDelay {
		delay += time.Duration(rand.Int63n(int64(maxDelay - minDelay)))
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return storage.Profile{}, ctx.Err()
	case <-timer.C:
	}

	return storage.Profile{
		ID:         profileID,
		Username:   fmt.Sprintf("User %s", profileID),
		Email:      fmt.Sprintf("%s@mail.com", profileID),
		EnrichedAt: time.Now().UTC(),
	}, nil
}

type Simulated struct {
	MinDelay time.Duration
	MaxDelay time.Duration
}

func NewSimulatedClient() *Simulated {
	return &Simulated{
		MinDelay: 100 * time.Millisecond,
		MaxDelay: 400 * time.Millisecond,
	}
}
