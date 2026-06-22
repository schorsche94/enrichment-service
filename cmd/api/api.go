package main

import (
	"encoding/json"
	"enrichment-service/internal/storage"
	"log"
	"net/http"
	"time"
)

type application struct {
	config config
	store  storage.Storage
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

func (app *application) mount() *http.ServeMux {
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
	log.Println("handle get profile")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Fatal("write JSON response failed", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
