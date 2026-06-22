package main

import (
	"encoding/json"
	"enrichment-service/internal/enrichment"
	"enrichment-service/internal/storage"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type application struct {
	config     config
	store      storage.Profiles
	enrichment enrichment.Enrich
}
type config struct {
	db   dbConfig
	addr string
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

func (app *application) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/enrich", app.handleEnrich)
	mux.HandleFunc("GET /v1/profiles/{id}", app.handleGetProfile)

	return mux
}

func (app *application) run(mux *http.ServeMux) error {

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}
	log.Printf("server has started at %s", app.config.addr)

	return srv.ListenAndServe()
}

type enrichRequest struct {
	ProfileIDs []string `json:"profile_ids"`
}

func (s *application) handleEnrich(w http.ResponseWriter, r *http.Request) {
	log.Println("handle enrich")
}

func (s *application) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("profile id is required"))
		return
	}

	profile, err := s.store.Get(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, fmt.Errorf("profile %q has not been enriched", id))
		return
	}
	if err != nil {
		log.Printf("get profile failed: profile_id=%s error=%v", id, err)
		writeError(w, http.StatusInternalServerError, errors.New("internal error"))
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write JSON response failed: %v", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
