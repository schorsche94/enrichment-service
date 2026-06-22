package storage

import (
	"context"
	"database/sql"
)

type Storage struct {
	Profiles interface {
		Create(context.Context, *Profile) error
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Profiles: &ProfilesStorage{db},
	}
}
