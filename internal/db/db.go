package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func New(addr string, maxOpenConns, maxIdleConns int, maxIdleTime string) (*sql.DB, error) {
	return newWithDriver("postgres", addr, maxOpenConns, maxIdleConns, maxIdleTime)
}

func newWithDriver(driverName, addr string, maxOpenConns, maxIdleConns int, maxIdleTime string) (*sql.DB, error) {
	duration, err := time.ParseDuration(maxIdleTime)
	if err != nil {
		return nil, fmt.Errorf("invalid maxIdleTime duration %q: %w", maxIdleTime, err)
	}
	if maxOpenConns <= 0 {
		return nil, fmt.Errorf("maxOpenConns must be > 0")
	}
	if maxIdleConns < 0 {
		return nil, fmt.Errorf("maxIdleConns cannot be negative")
	}

	if maxIdleConns > maxOpenConns {
		maxIdleConns = maxOpenConns
	}

	db, err := sql.Open(driverName, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("storage: postgres ping failed: %w", err)
	}
	return db, nil
}
