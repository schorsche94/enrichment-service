package storage

import (
	"context"
	"database/sql"
	"time"
)

type Profile struct {
	ID         int       `json:"id"`
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
		INSERT INTO profiles (first_name, last_name, email) 
		VALUES (?, ?, ?)
		RETURNING id, enriched_at`

	err := s.db.QueryRowContext(ctx, query, profile.FirstName, profile.LastName, profile.Email).Scan(&profile.ID, &profile.EnrichedAt)
	if err != nil {
		return err
	}

	return nil
}
