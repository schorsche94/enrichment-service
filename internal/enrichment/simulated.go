package enrichment

import (
	"context"
	"enrichment-service/internal/storage"
	"fmt"
	"time"
)

type Fetcher interface {
	Fetch(ctx context.Context, profileID string) (storage.Profile, error)
}

func (s *Simulated) Fetch(ctx context.Context, profileID string) (storage.Profile, error) {
	return storage.Profile{
		ID:         profileID,
		Username:   fmt.Sprintf("User %s", profileID),
		Email:      fmt.Sprintf("%s@mail.com", profileID),
		EnrichedAt: time.Now().UTC(),
	}, nil
}

type Simulated struct {
}

func NewSimulatedClient() *Simulated {
	return &Simulated{}
}
