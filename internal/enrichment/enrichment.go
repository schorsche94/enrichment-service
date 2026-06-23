package enrichment

import (
	"context"
	"enrichment-service/internal/storage"
	"fmt"
	"sync"
)

type Enrich interface {
	Enrich(ctx context.Context, profileIDs []string) (Summary, error)
}

func (e *Enricher) Enrich(ctx context.Context, profileIDs []string) (Summary, error) {
	summary := Summary{Requested: len(profileIDs)}
	if len(profileIDs) == 0 {
		return summary, nil
	}

	results := make(chan result, len(profileIDs))

	var wg sync.WaitGroup
	for _, id := range profileIDs {
		select {
		case <-ctx.Done():
			results <- result{failure: &Failure{ProfileID: id, Reason: "request canceled before enrichment started"}}
			continue
		default:
		}

		wg.Add(1)
		go func(profileID string) {
			defer wg.Done()
			results <- e.enrichOne(ctx, profileID)
		}(id)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.failure != nil {
			summary.Failed++
			summary.Failures = append(summary.Failures, *r.failure)
		} else {
			summary.Enriched++
		}
	}

	if err := ctx.Err(); err != nil {
		return summary, fmt.Errorf("enrichment: batch interrupted: %w", err)
	}
	return summary, nil
}

func (e *Enricher) enrichOne(ctx context.Context, profileID string) result {
	profile, err := e.fetcher.Fetch(ctx, profileID)
	if err != nil {
		return result{failure: &Failure{ProfileID: profileID, Reason: err.Error()}}
	}

	if err := e.storage.Create(ctx, profile); err != nil {
		return result{failure: &Failure{ProfileID: profileID, Reason: fmt.Sprintf("persist failed: %v", err)}}
	}

	return result{}
}

type result struct {
	failure *Failure
}

type Failure struct {
	ProfileID string `json:"profile_id"`
	Reason    string `json:"reason"`
}
type Summary struct {
	Requested int       `json:"requested"`
	Enriched  int       `json:"enriched"`
	Failed    int       `json:"failed"`
	Failures  []Failure `json:"failures,omitempty"`
}

type Enricher struct {
	fetcher Fetcher
	storage storage.Profiles
}

func NewEnricher(fetcher Fetcher, storage storage.Profiles) *Enricher {

	return &Enricher{
		fetcher: fetcher,
		storage: storage,
	}
}
