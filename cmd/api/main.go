package main

import (
	"context"
	db2 "enrichment-service/internal/db"
	"enrichment-service/internal/enrichment"
	"enrichment-service/internal/env"
	"enrichment-service/internal/storage"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	if err := run(); err != nil {
		log.Fatal("server exited with error", "error", err)
		os.Exit(1)
	}

}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := db2.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer db.Close()
	log.Println("database connection pool established")

	store := storage.NewProfileStorage(db)
	simulated := enrichment.NewSimulatedClient()
	enrichment := enrichment.NewEnricher(simulated, store, cfg.concurrency)

	app := &application{
		config:     cfg,
		store:      store,
		enrichment: enrichment,
	}

	mux := app.Routes()

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}
	log.Printf("server has started at %s", app.config.addr)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}

func loadConfig() (config, error) {
	maxOpenConns, err := env.GetInt("DB_MAX_OPEN_CONNS", 30)
	if err != nil {
		return config{}, err
	}

	maxIdleConns, err := env.GetInt("DB_MAX_IDLE_CONNS", 30)
	if err != nil {
		return config{}, err
	}

	concurrency, err := env.GetInt("CONCURRENCY", 5)
	if err != nil {
		return config{}, err
	}

	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://enricher_user:enricher_pwd@db:5432/enrich?sslmode=disable"),
			maxOpenConns: maxOpenConns,
			maxIdleConns: maxIdleConns,
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		concurrency: concurrency,
	}

	return cfg, nil
}
