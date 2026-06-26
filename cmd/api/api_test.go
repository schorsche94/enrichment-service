package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"enrichment-service/internal/enrichment"
	"enrichment-service/internal/storage"
)

func TestHandleEnrichRejectsInvalidJSON(t *testing.T) {
	app := &application{enrichment: &fakeEnricher{}}
	req := httptest.NewRequest(http.MethodPost, "/v1/enrich", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertJSONErrorContains(t, rec.Body.Bytes(), "decode request body")
}

func TestHandleEnrichRejectsEmptyProfileIDs(t *testing.T) {
	app := &application{enrichment: &fakeEnricher{}}
	req := httptest.NewRequest(http.MethodPost, "/v1/enrich", bytes.NewBufferString(`{"profile_ids":[]}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertJSONErrorContains(t, rec.Body.Bytes(), "profile_ids must contain at least one ID")
}

func TestHandleEnrichReturnsSummary(t *testing.T) {
	enricher := &fakeEnricher{
		summary: enrichment.Summary{
			Requested: 2,
			Enriched:  1,
			Failed:    1,
			Failures:  []enrichment.Failure{{ProfileID: "p2", Reason: "upstream timeout"}},
		},
	}
	app := &application{enrichment: enricher}
	req := httptest.NewRequest(http.MethodPost, "/v1/enrich", bytes.NewBufferString(`{"profile_ids":["p1","p2"]}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	if want := []string{"p1", "p2"}; !sameStrings(enricher.profileIDs, want) {
		t.Fatalf("expected profile IDs %v, got %v", want, enricher.profileIDs)
	}

	var got enrichment.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Requested != 2 || got.Enriched != 1 || got.Failed != 1 || len(got.Failures) != 1 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestHandleGetProfileReturnsProfile(t *testing.T) {
	enrichedAt := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	store := &fakeProfileStore{
		profiles: map[string]storage.Profile{
			"p1": {ID: "p1", Username: "p1", Email: "p1@mail.com", EnrichedAt: enrichedAt},
		},
	}
	app := &application{store: store}
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/p1", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got storage.Profile
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "p1" || got.Username != "p1" || got.Email != "p1@mail.com" || !got.EnrichedAt.Equal(enrichedAt) {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestHandleGetProfileMapsNotFound(t *testing.T) {
	app := &application{store: &fakeProfileStore{getErr: storage.ErrNotFound}}
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/missing", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	assertJSONErrorContains(t, rec.Body.Bytes(), `profile "missing" has not been enriched`)
}

func TestHandleGetProfileMapsUnexpectedStorageError(t *testing.T) {
	app := &application{store: &fakeProfileStore{getErr: errors.New("database down")}}
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/p1", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	assertJSONErrorContains(t, rec.Body.Bytes(), "internal error")
}

func TestWriteJSONLogsEncodeError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusOK, func() {})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
}

func assertJSONErrorContains(t *testing.T, body []byte, want string) {
	t.Helper()

	var got errorResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode error response: %v; body=%s", err, body)
	}
	if !strings.Contains(got.Error, want) {
		t.Fatalf("expected error containing %q, got %q", want, got.Error)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

type fakeEnricher struct {
	profileIDs []string
	summary    enrichment.Summary
	err        error
}

func (e *fakeEnricher) Enrich(_ context.Context, profileIDs []string) (enrichment.Summary, error) {
	e.profileIDs = append([]string(nil), profileIDs...)
	return e.summary, e.err
}

type fakeProfileStore struct {
	profiles map[string]storage.Profile
	getErr   error
}

func (s *fakeProfileStore) Create(_ context.Context, _ storage.Profile) error {
	return nil
}

func (s *fakeProfileStore) Get(_ context.Context, id string) (storage.Profile, error) {
	if s.getErr != nil {
		return storage.Profile{}, s.getErr
	}
	profile, ok := s.profiles[id]
	if !ok {
		return storage.Profile{}, storage.ErrNotFound
	}
	return profile, nil
}
