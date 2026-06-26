package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Profile struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	EnrichedAt time.Time `json:"enriched_at"`
}
type ProfilesStorage struct {
	db *sql.DB
}

func (s *ProfilesStorage) Create(ctx context.Context, profile Profile) error {
	query := `
		INSERT INTO profiles (id, username, email) 
		VALUES ($1, $2, $3)
		RETURNING id, enriched_at`

	err := s.db.QueryRowContext(ctx, query, profile.ID, profile.Username, profile.Email).Scan(&profile.ID, &profile.EnrichedAt)
	if err != nil {
		return fmt.Errorf("storage: create profile %s: %w", profile.ID, err)
	}

	return nil
}

func (s *ProfilesStorage) Get(ctx context.Context, id string) (Profile, error) {
	const query = `
		SELECT id, username, email, enriched_at
		FROM profiles
		WHERE id = $1`

	var p Profile
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Username, &p.Email, &p.EnrichedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return Profile{}, fmt.Errorf("storage: get profile %s: %w", id, err)
	}
	return p, nil
}

var ErrNotFound = errors.New("profile not found")
