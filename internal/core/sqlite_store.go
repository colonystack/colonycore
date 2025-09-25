package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "modernc.org/sqlite" // pure go sqlite driver
	"os"
	"path/filepath"
	"sync"
)

// SQLiteStore persists the in-memory state to a single SQLite table as JSON blobs.
// It snapshots the full state after every successful transaction.
type SQLiteStore struct {
	*MemoryStore
	db   *sql.DB
	mu   sync.Mutex
	path string
}

func NewSQLiteStore(path string, engine *RulesEngine) (*SQLiteStore, error) {
	if path == "" {
		path = "colonycore.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
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
	ms := NewMemoryStore(engine)
	s := &SQLiteStore{MemoryStore: ms, db: db, path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

var sqliteBuckets = []string{ // one row per bucket
	"organisms", "cohorts", "housing", "breeding", "procedures", "protocols", "projects",
}

func (s *SQLiteStore) load() error {
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
	st := newMemoryState()
	for _, r := range raws {
		switch r.bucket {
		case "organisms":
			if err := json.Unmarshal(r.payload, &st.organisms); err != nil {
				return fmt.Errorf("decode organisms: %w", err)
			}
		case "cohorts":
			if err := json.Unmarshal(r.payload, &st.cohorts); err != nil {
				return fmt.Errorf("decode cohorts: %w", err)
			}
		case "housing":
			if err := json.Unmarshal(r.payload, &st.housing); err != nil {
				return fmt.Errorf("decode housing: %w", err)
			}
		case "breeding":
			if err := json.Unmarshal(r.payload, &st.breeding); err != nil {
				return fmt.Errorf("decode breeding: %w", err)
			}
		case "procedures":
			if err := json.Unmarshal(r.payload, &st.procedures); err != nil {
				return fmt.Errorf("decode procedures: %w", err)
			}
		case "protocols":
			if err := json.Unmarshal(r.payload, &st.protocols); err != nil {
				return fmt.Errorf("decode protocols: %w", err)
			}
		case "projects":
			if err := json.Unmarshal(r.payload, &st.projects); err != nil {
				return fmt.Errorf("decode projects: %w", err)
			}
		}
	}
	s.state = st
	return nil
}

func (s *SQLiteStore) persist() (retErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
			data, err = json.Marshal(s.state.organisms)
		case "cohorts":
			data, err = json.Marshal(s.state.cohorts)
		case "housing":
			data, err = json.Marshal(s.state.housing)
		case "breeding":
			data, err = json.Marshal(s.state.breeding)
		case "procedures":
			data, err = json.Marshal(s.state.procedures)
		case "protocols":
			data, err = json.Marshal(s.state.protocols)
		case "projects":
			data, err = json.Marshal(s.state.projects)
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

func (s *SQLiteStore) RunInTransaction(ctx context.Context, fn func(tx *Transaction) error) (Result, error) {
	res, err := s.MemoryStore.RunInTransaction(ctx, fn)
	if err != nil {
		return res, err
	}
	if pErr := s.persist(); pErr != nil {
		return res, pErr
	}
	return res, nil
}
