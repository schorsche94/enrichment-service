package enrichment

import (
	"context"
	"enrichment-service/internal/storage"
	"fmt"
)

type Enrich interface {
	Enrich(ctx context.Context, profileIDs []string) (Summary, error)
}

func (e *Enricher) Enrich(ctx context.Context, profileIDs []string) (Summary, error) {
	summary := Summary{Requested: len(profileIDs)}
	if len(profileIDs) == 0 {
		return summary, nil
	}
	e.enrichOne(ctx, profileIDs[0])
	return Summary{}, nil
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
