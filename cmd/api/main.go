package main

import (
	db2 "enrichment-service/internal/db"
	"enrichment-service/internal/enrichment"
	"enrichment-service/internal/env"
	"enrichment-service/internal/storage"
	"log"
)

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://enricher_user:enricher_pwd@localhost:5432/enrich?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
	}

	db, err := db2.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	log.Println("database connection pool established")

	store := storage.NewProfileStorage(db)
	simulated := enrichment.NewSimulatedClient()
	enrichment := enrichment.NewEnricher(simulated, store)

	app := &application{
		config:     cfg,
		store:      store,
		enrichment: enrichment,
	}

	mux := app.Routes()

	log.Fatal(app.run(mux))
}
