package storage

import (
	"context"
	"database/sql"
)

type Profiles interface {
	Create(ctx context.Context, profile *Profile) error
	Get(ctx context.Context, id string) (Profile, error)
}

func NewProfileStorage(db *sql.DB) Profiles {
	return &ProfilesStorage{db}

}
