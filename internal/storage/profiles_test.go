package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProfilesStorageCreateInsertsProfile(t *testing.T) {
	enrichedAt := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	db := openProfilesTestDB(t, func(query string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "INSERT INTO profiles") {
			t.Fatalf("expected insert query, got %q", query)
		}
		assertNamedValue(t, args, 1, "p1")
		assertNamedValue(t, args, 2, "p1")
		assertNamedValue(t, args, 3, "p1@mail.com")

		return &profilesTestRows{
			columns: []string{"id", "enriched_at"},
			values:  [][]driver.Value{{"p1", enrichedAt}},
		}, nil
	})
	defer db.Close()

	store := &ProfilesStorage{db: db}
	err := store.Create(context.Background(), Profile{
		ID:       "p1",
		Username: "p1",
		Email:    "p1@mail.com",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

func TestProfilesStorageCreateWrapsDatabaseError(t *testing.T) {
	db := openProfilesTestDB(t, func(_ string, _ []driver.NamedValue) (driver.Rows, error) {
		return nil, errors.New("insert failed")
	})
	defer db.Close()

	store := &ProfilesStorage{db: db}
	err := store.Create(context.Background(), Profile{ID: "p1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "storage: create profile p1") || !strings.Contains(err.Error(), "insert failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProfilesStorageGetReturnsProfile(t *testing.T) {
	enrichedAt := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	db := openProfilesTestDB(t, func(query string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "SELECT id, username, email, enriched_at") {
			t.Fatalf("expected select query, got %q", query)
		}
		assertNamedValue(t, args, 1, "p1")

		return &profilesTestRows{
			columns: []string{"id", "username", "email", "enriched_at"},
			values:  [][]driver.Value{{"p1", "p1", "p1@mail.com", enrichedAt}},
		}, nil
	})
	defer db.Close()

	store := &ProfilesStorage{db: db}
	profile, err := store.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if profile.ID != "p1" || profile.Username != "p1" || profile.Email != "p1@mail.com" || !profile.EnrichedAt.Equal(enrichedAt) {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestProfilesStorageGetMapsNoRowsToErrNotFound(t *testing.T) {
	db := openProfilesTestDB(t, func(_ string, _ []driver.NamedValue) (driver.Rows, error) {
		return &profilesTestRows{
			columns: []string{"id", "username", "email", "enriched_at"},
		}, nil
	})
	defer db.Close()

	store := &ProfilesStorage{db: db}
	_, err := store.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "storage: get profile missing") {
		t.Fatalf("expected wrapped storage error, got %v", err)
	}
}

func TestProfilesStorageGetWrapsDatabaseError(t *testing.T) {
	db := openProfilesTestDB(t, func(_ string, _ []driver.NamedValue) (driver.Rows, error) {
		return nil, errors.New("select failed")
	})
	defer db.Close()

	store := &ProfilesStorage{db: db}
	_, err := store.Get(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "storage: get profile p1") || !strings.Contains(err.Error(), "select failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewProfileStorage(t *testing.T) {
	db := openProfilesTestDB(t, func(_ string, _ []driver.NamedValue) (driver.Rows, error) {
		return nil, errors.New("unexpected query")
	})
	defer db.Close()

	store, ok := NewProfileStorage(db).(*ProfilesStorage)
	if !ok {
		t.Fatalf("expected *ProfilesStorage, got %T", NewProfileStorage(db))
	}
	if store.db != db {
		t.Fatal("expected storage to keep provided database")
	}
}

func assertNamedValue(t *testing.T, args []driver.NamedValue, ordinal int, want driver.Value) {
	t.Helper()

	for _, arg := range args {
		if arg.Ordinal == ordinal {
			if arg.Value != want {
				t.Fatalf("arg %d: expected %v, got %v", ordinal, want, arg.Value)
			}
			return
		}
	}
	t.Fatalf("arg %d not found in %v", ordinal, args)
}

type profilesQueryFunc func(query string, args []driver.NamedValue) (driver.Rows, error)

var (
	registerProfilesTestDriver sync.Once
	profilesTestDriverMu       sync.Mutex
	profilesTestHandlers       = map[string]profilesQueryFunc{}
)

func openProfilesTestDB(t *testing.T, query profilesQueryFunc) *sql.DB {
	t.Helper()

	registerProfilesTestDriver.Do(func() {
		sql.Register("profiles-test", profilesTestDriver{})
	})

	name := strings.ReplaceAll(t.Name(), "/", "-")
	profilesTestDriverMu.Lock()
	profilesTestHandlers[name] = query
	profilesTestDriverMu.Unlock()

	t.Cleanup(func() {
		profilesTestDriverMu.Lock()
		delete(profilesTestHandlers, name)
		profilesTestDriverMu.Unlock()
	})

	db, err := sql.Open("profiles-test", name)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

type profilesTestDriver struct{}

func (profilesTestDriver) Open(name string) (driver.Conn, error) {
	return profilesTestConn{name: name}, nil
}

type profilesTestConn struct {
	name string
}

func (c profilesTestConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported")
}

func (c profilesTestConn) Close() error {
	return nil
}

func (c profilesTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c profilesTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	profilesTestDriverMu.Lock()
	handler := profilesTestHandlers[c.name]
	profilesTestDriverMu.Unlock()
	if handler == nil {
		return nil, errors.New("test query handler not found")
	}
	return handler(query, args)
}

var _ driver.QueryerContext = profilesTestConn{}

type profilesTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *profilesTestRows) Columns() []string {
	return r.columns
}

func (r *profilesTestRows) Close() error {
	return nil
}

func (r *profilesTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}
