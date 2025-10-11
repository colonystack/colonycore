package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite" // pure go sqlite driver
)

// NOTE: domain import kept indirect through memstore.go aliases to avoid cycles; compile-time assertion lives there.

// Store persists the in-memory state to a single SQLite table as JSON blobs.
// It snapshots the full state after every successful transaction.
type Store struct {
	*memStore
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// NewStore constructs a snapshotting SQLite-backed persistent store.
func NewStore(path string, engine *RulesEngine) (*Store, error) {
	if path == "" {
		path = "colonycore.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("create dirs: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS state (
		bucket TEXT PRIMARY KEY,
		payload BLOB NOT NULL
	)`); err != nil {
		return nil, fmt.Errorf("create state table: %w", err)
	}
	ms := newMemStore(engine)
	s := &Store{memStore: ms, db: db, path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

var sqliteBuckets = []string{"organisms", "cohorts", "housing", "breeding", "procedures", "protocols", "projects"}

func (s *Store) load() error {
	rows, err := s.db.Query(`SELECT bucket, payload FROM state`)
	if err != nil {
		return fmt.Errorf("select state: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type raw struct {
		bucket  string
		payload []byte
	}
	var raws []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.bucket, &r.payload); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		raws = append(raws, r)
	}
	if len(raws) == 0 {
		return nil
	}
	snapshot := Snapshot{}
	for _, r := range raws {
		switch r.bucket {
		case "organisms":
			if err := json.Unmarshal(r.payload, &snapshot.Organisms); err != nil {
				return fmt.Errorf("decode organisms: %w", err)
			}
		case "cohorts":
			if err := json.Unmarshal(r.payload, &snapshot.Cohorts); err != nil {
				return fmt.Errorf("decode cohorts: %w", err)
			}
		case "housing":
			if err := json.Unmarshal(r.payload, &snapshot.Housing); err != nil {
				return fmt.Errorf("decode housing: %w", err)
			}
		case "breeding":
			if err := json.Unmarshal(r.payload, &snapshot.Breeding); err != nil {
				return fmt.Errorf("decode breeding: %w", err)
			}
		case "procedures":
			if err := json.Unmarshal(r.payload, &snapshot.Procedures); err != nil {
				return fmt.Errorf("decode procedures: %w", err)
			}
		case "protocols":
			if err := json.Unmarshal(r.payload, &snapshot.Protocols); err != nil {
				return fmt.Errorf("decode protocols: %w", err)
			}
		case "projects":
			if err := json.Unmarshal(r.payload, &snapshot.Projects); err != nil {
				return fmt.Errorf("decode projects: %w", err)
			}
		}
	}
	s.ImportState(snapshot)
	return nil
}

func (s *Store) persist() (retErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.ExportState()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			_ = tx.Rollback()
		}
	}()
	for _, bucket := range sqliteBuckets {
		var data []byte
		switch bucket {
		case "organisms":
			data, err = json.Marshal(snapshot.Organisms)
		case "cohorts":
			data, err = json.Marshal(snapshot.Cohorts)
		case "housing":
			data, err = json.Marshal(snapshot.Housing)
		case "breeding":
			data, err = json.Marshal(snapshot.Breeding)
		case "procedures":
			data, err = json.Marshal(snapshot.Procedures)
		case "protocols":
			data, err = json.Marshal(snapshot.Protocols)
		case "projects":
			data, err = json.Marshal(snapshot.Projects)
		}
		if err != nil {
			retErr = err
			return retErr
		}
		if _, err = tx.Exec(`INSERT INTO state(bucket,payload) VALUES(?,?) ON CONFLICT(bucket) DO UPDATE SET payload=excluded.payload`, bucket, data); err != nil {
			retErr = fmt.Errorf("upsert %s: %w", bucket, err)
			return retErr
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// RunInTransaction applies the provided function within a transaction, then snapshots state to SQLite if successful.
func (s *Store) RunInTransaction(ctx context.Context, fn func(tx Transaction) error) (Result, error) {
	res, err := s.memStore.RunInTransaction(ctx, fn)
	if err != nil {
		return res, err
	}
	if pErr := s.persist(); pErr != nil {
		return res, pErr
	}
	return res, nil
}

// DB exposes the underlying sql.DB for integration testing hooks.
func (s *Store) DB() *sql.DB { return s.db }

// Path returns the configured database path.
func (s *Store) Path() string { return s.path }
