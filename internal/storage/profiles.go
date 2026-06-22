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
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Email      string    `json:"email"`
	EnrichedAt time.Time `json:"enriched_at"`
}
type ProfilesStorage struct {
	db *sql.DB
}

func (s *ProfilesStorage) Create(ctx context.Context, profile *Profile) error {
	query := `
		INSERT INTO profiles (id, first_name, last_name, email) 
		VALUES (?, ?, ?, ?)
		RETURNING id, enriched_at`

	err := s.db.QueryRowContext(ctx, query, profile.FirstName, profile.LastName, profile.Email).Scan(&profile.ID, &profile.EnrichedAt)
	if err != nil {
		return err
	}

	return nil
}

func (s *ProfilesStorage) Get(ctx context.Context, id string) (Profile, error) {
	const query = `
		SELECT id, first_name, last_name, email, enriched_at
		FROM profiles
		WHERE id = $1`

	var p Profile
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.EnrichedAt,
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
