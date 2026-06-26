package enrichment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"enrichment-service/internal/storage"
)

func TestEnrichEmptyBatch(t *testing.T) {
	enricher := NewEnricher(&fakeFetcher{}, &fakeProfilesStore{}, 2)

	summary, err := enricher.Enrich(context.Background(), nil)
	if err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	if summary.Requested != 0 || summary.Enriched != 0 || summary.Failed != 0 || len(summary.Failures) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestEnrichCreatesFetchedProfiles(t *testing.T) {
	store := &fakeProfilesStore{}
	fetcher := &fakeFetcher{
		profiles: map[string]storage.Profile{
			"p1": {ID: "p1", Username: "p1", Email: "p1@mail.com"},
			"p2": {ID: "p2", Username: "p2", Email: "p2@mail.com"},
		},
	}
	enricher := NewEnricher(fetcher, store, 2)

	summary, err := enricher.Enrich(context.Background(), []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	if summary.Requested != 2 || summary.Enriched != 2 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	created := store.createdByID()
	for _, id := range []string{"p1", "p2"} {
		if _, ok := created[id]; !ok {
			t.Fatalf("expected profile %q to be created, created=%v", id, created)
		}
	}
}

func TestEnrichReportsFetchAndPersistFailures(t *testing.T) {
	store := &fakeProfilesStore{createErrors: map[string]error{
		"persist-fails": errors.New("database unavailable"),
	}}
	fetcher := &fakeFetcher{
		profiles: map[string]storage.Profile{
			"ok":            {ID: "ok", Username: "OK", Email: "ok@mail.com"},
			"persist-fails": {ID: "persist-fails", Username: "Nope", Email: "nope@mail.com"},
		},
		errors: map[string]error{
			"fetch-fails": errors.New("upstream timeout"),
		},
	}
	enricher := NewEnricher(fetcher, store, 3)

	summary, err := enricher.Enrich(context.Background(), []string{"ok", "fetch-fails", "persist-fails"})
	if err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	if summary.Requested != 3 || summary.Enriched != 1 || summary.Failed != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	failures := failuresByID(summary.Failures)
	if got := failures["fetch-fails"]; !strings.Contains(got, "upstream timeout") {
		t.Fatalf("expected fetch failure reason, got %q", got)
	}
	if got := failures["persist-fails"]; !strings.Contains(got, "persist failed: database unavailable") {
		t.Fatalf("expected persist failure reason, got %q", got)
	}
}

func TestEnrichHonorsConcurrencyLimit(t *testing.T) {
	fetcher := &blockingFetcher{
		profiles: map[string]storage.Profile{
			"p1": {ID: "p1"},
			"p2": {ID: "p2"},
			"p3": {ID: "p3"},
			"p4": {ID: "p4"},
		},
		release: make(chan struct{}),
	}
	enricher := NewEnricher(fetcher, &fakeProfilesStore{}, 2)

	done := make(chan error, 1)
	go func() {
		_, err := enricher.Enrich(context.Background(), []string{"p1", "p2", "p3", "p4"})
		done <- err
	}()

	if err := fetcher.waitForInFlight(2, time.Second); err != nil {
		close(fetcher.release)
		t.Fatal(err)
	}
	if inFlight := fetcher.currentInFlight(); inFlight != 2 {
		close(fetcher.release)
		t.Fatalf("expected exactly 2 in-flight fetches, got %d", inFlight)
	}

	close(fetcher.release)
	if err := <-done; err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}
	if max := fetcher.maxObserved(); max > 2 {
		t.Fatalf("expected max concurrency <= 2, got %d", max)
	}
}

func TestNewEnricherDefaultsInvalidConcurrencyToOne(t *testing.T) {
	fetcher := &blockingFetcher{
		profiles: map[string]storage.Profile{
			"p1": {ID: "p1"},
			"p2": {ID: "p2"},
		},
		release: make(chan struct{}),
	}
	enricher := NewEnricher(fetcher, &fakeProfilesStore{}, 0)

	done := make(chan error, 1)
	go func() {
		_, err := enricher.Enrich(context.Background(), []string{"p1", "p2"})
		done <- err
	}()

	if err := fetcher.waitForInFlight(1, time.Second); err != nil {
		close(fetcher.release)
		t.Fatal(err)
	}
	if inFlight := fetcher.currentInFlight(); inFlight != 1 {
		close(fetcher.release)
		t.Fatalf("expected exactly 1 in-flight fetch, got %d", inFlight)
	}

	close(fetcher.release)
	if err := <-done; err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}
}

func TestSimulatedFetchReturnsProfile(t *testing.T) {
	client := &Simulated{
		FailureRate: -1,
		MinDelay:    time.Nanosecond,
		MaxDelay:    time.Nanosecond,
	}

	profile, err := client.Fetch(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if profile.ID != "p1" || profile.Username != "User p1" || profile.Email != "p1@mail.com" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if profile.EnrichedAt.IsZero() {
		t.Fatal("expected enriched timestamp")
	}
}

func TestSimulatedFetchReturnsFailure(t *testing.T) {
	client := &Simulated{
		FailureRate: 2,
		MinDelay:    time.Nanosecond,
		MaxDelay:    time.Nanosecond,
	}

	_, err := client.Fetch(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `upstream: simulated failure for profile "p1"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSimulatedFetchReturnsContextError(t *testing.T) {
	client := &Simulated{
		FailureRate: -1,
		MinDelay:    time.Hour,
		MaxDelay:    time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Fetch(ctx, "p1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestNewSimulatedClientUsesDefaults(t *testing.T) {
	client := NewSimulatedClient()

	if client.FailureRate != 0.10 {
		t.Fatalf("expected failure rate 0.10, got %f", client.FailureRate)
	}
	if client.MinDelay != 100*time.Millisecond || client.MaxDelay != 400*time.Millisecond {
		t.Fatalf("unexpected delays: min=%s max=%s", client.MinDelay, client.MaxDelay)
	}
}

func failuresByID(failures []Failure) map[string]string {
	byID := make(map[string]string, len(failures))
	for _, failure := range failures {
		byID[failure.ProfileID] = failure.Reason
	}
	return byID
}

type fakeFetcher struct {
	profiles map[string]storage.Profile
	errors   map[string]error
}

func (f *fakeFetcher) Fetch(_ context.Context, profileID string) (storage.Profile, error) {
	if err, ok := f.errors[profileID]; ok {
		return storage.Profile{}, err
	}
	if profile, ok := f.profiles[profileID]; ok {
		return profile, nil
	}
	return storage.Profile{ID: profileID}, nil
}

type fakeProfilesStore struct {
	mu           sync.Mutex
	created      []storage.Profile
	createErrors map[string]error
}

func (s *fakeProfilesStore) Create(_ context.Context, profile storage.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err, ok := s.createErrors[profile.ID]; ok {
		return err
	}
	s.created = append(s.created, profile)
	return nil
}

func (s *fakeProfilesStore) Get(_ context.Context, id string) (storage.Profile, error) {
	return storage.Profile{ID: id}, nil
}

func (s *fakeProfilesStore) createdByID() map[string]storage.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()

	created := make(map[string]storage.Profile, len(s.created))
	for _, profile := range s.created {
		created[profile.ID] = profile
	}
	return created
}

type blockingFetcher struct {
	mu       sync.Mutex
	profiles map[string]storage.Profile
	release  chan struct{}
	inFlight int
	max      int
}

func (f *blockingFetcher) Fetch(ctx context.Context, profileID string) (storage.Profile, error) {
	f.mu.Lock()
	f.inFlight++
	if f.inFlight > f.max {
		f.max = f.inFlight
	}
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.inFlight--
		f.mu.Unlock()
	}()

	select {
	case <-f.release:
	case <-ctx.Done():
		return storage.Profile{}, ctx.Err()
	}

	if profile, ok := f.profiles[profileID]; ok {
		return profile, nil
	}
	return storage.Profile{ID: profileID}, nil
}

func (f *blockingFetcher) waitForInFlight(want int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if f.currentInFlight() == want {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %d in-flight fetches, got %d", want, f.currentInFlight())
}

func (f *blockingFetcher) currentInFlight() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inFlight
}

func (f *blockingFetcher) maxObserved() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.max
}
