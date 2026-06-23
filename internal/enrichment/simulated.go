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

	failureRate := s.FailureRate
	if failureRate == 0 {
		failureRate = 0.10
	}
	if rand.Float64() < failureRate {
		return storage.Profile{}, fmt.Errorf("upstream: simulated failure for profile %q", profileID)
	}

	return storage.Profile{
		ID:         profileID,
		Username:   fmt.Sprintf("User %s", profileID),
		Email:      fmt.Sprintf("%s@mail.com", profileID),
		EnrichedAt: time.Now().UTC(),
	}, nil
}

type Simulated struct {
	FailureRate float64
	MinDelay    time.Duration
	MaxDelay    time.Duration
}

func NewSimulatedClient() *Simulated {
	return &Simulated{
		FailureRate: 0.10,
		MinDelay:    100 * time.Millisecond,
		MaxDelay:    400 * time.Millisecond,
	}
}
