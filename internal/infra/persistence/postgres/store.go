// Package postgres provides a Postgres-backed persistent store that mirrors the
// in-memory semantics while applying the generated entity-model DDL on startup.
package postgres

import (
	"colonycore/internal/entitymodel/sqlbundle"
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx as a database/sql driver
)

// Compile-time contract assertion ensuring the store satisfies the domain interface.
var _ domain.PersistentStore = (*Store)(nil)

const (
	defaultDriver = "pgx"
	// Default DSN keeps parity with OpenPersistentStore defaults while allowing overrides via env.
	defaultDSN = "postgres://localhost/colonycore?sslmode=disable"
)

var (
	sqlOpen = sql.Open
	openMu  sync.Mutex
)

// Store persists state to Postgres while reusing the in-memory implementation for transactions.
type Store struct {
	*memory.Store
	db *sql.DB
	mu sync.Mutex
}

// NewStore opens a Postgres-backed store using the provided DSN (falls back to defaultDSN).
// It applies the generated entity-model DDL, ensures the snapshot table exists, and
// hydrates the in-memory store from any existing snapshot.
func NewStore(dsn string, engine *domain.RulesEngine) (*Store, error) {
	if dsn == "" {
		dsn = defaultDSN
	}
	openMu.Lock()
	db, err := sqlOpen(defaultDriver, dsn)
	openMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := applyEntityModelDDL(ctx, db); err != nil {
		return nil, err
	}
	if err := ensureStateTable(ctx, db); err != nil {
		return nil, err
	}
	snapshot, err := loadSnapshot(ctx, db)
	if err != nil {
		return nil, err
	}
	mem := memory.NewStore(engine)
	mem.ImportState(snapshot)
	return &Store{Store: mem, db: db}, nil
}

// RunInTransaction applies the provided function within a transaction, then snapshots to Postgres if successful.
func (s *Store) RunInTransaction(ctx context.Context, fn func(domain.Transaction) error) (domain.Result, error) {
	res, err := s.Store.RunInTransaction(ctx, fn)
	if err != nil {
		return res, err
	}
	if err := s.persist(ctx); err != nil {
		return res, err
	}
	return res, nil
}

// DB exposes the underlying sql.DB for integration testing hooks.
func (s *Store) DB() *sql.DB { return s.db }

func applyEntityModelDDL(ctx context.Context, db *sql.DB) error {
	for _, stmt := range sqlbundle.SplitStatements(sqlbundle.Postgres()) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute ddl: %w", err)
		}
	}
	return nil
}

func ensureStateTable(ctx context.Context, db *sql.DB) error {
	ddl := `CREATE TABLE IF NOT EXISTS state (
		bucket TEXT PRIMARY KEY,
		payload JSONB NOT NULL
	)`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("ensure state table: %w", err)
	}
	return nil
}

func loadSnapshot(ctx context.Context, db *sql.DB) (memory.Snapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT bucket, payload FROM state`)
	if err != nil {
		return memory.Snapshot{}, fmt.Errorf("select state: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var snapshot memory.Snapshot
	targets := map[string]any{}
	for _, bucket := range postgresBuckets {
		switch bucket {
		case "organisms":
			targets[bucket] = &snapshot.Organisms
		case "cohorts":
			targets[bucket] = &snapshot.Cohorts
		case "housing":
			targets[bucket] = &snapshot.Housing
		case "facilities":
			targets[bucket] = &snapshot.Facilities
		case "breeding":
			targets[bucket] = &snapshot.Breeding
		case "lines":
			targets[bucket] = &snapshot.Lines
		case "strains":
			targets[bucket] = &snapshot.Strains
		case "markers":
			targets[bucket] = &snapshot.Markers
		case "procedures":
			targets[bucket] = &snapshot.Procedures
		case "treatments":
			targets[bucket] = &snapshot.Treatments
		case "observations":
			targets[bucket] = &snapshot.Observations
		case "samples":
			targets[bucket] = &snapshot.Samples
		case "protocols":
			targets[bucket] = &snapshot.Protocols
		case "permits":
			targets[bucket] = &snapshot.Permits
		case "projects":
			targets[bucket] = &snapshot.Projects
		case "supplies":
			targets[bucket] = &snapshot.Supplies
		}
	}

	for rows.Next() {
		var bucket string
		var payload []byte
		if err := rows.Scan(&bucket, &payload); err != nil {
			return memory.Snapshot{}, fmt.Errorf("scan state: %w", err)
		}
		if len(payload) == 0 {
			continue
		}
		if target, ok := targets[bucket]; ok {
			if err := json.Unmarshal(payload, target); err != nil {
				return memory.Snapshot{}, fmt.Errorf("decode %s: %w", bucket, err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return memory.Snapshot{}, fmt.Errorf("iterate state: %w", err)
	}
	return snapshot, nil
}

var postgresBuckets = []string{
	"organisms",
	"cohorts",
	"housing",
	"facilities",
	"breeding",
	"lines",
	"strains",
	"markers",
	"procedures",
	"treatments",
	"observations",
	"samples",
	"protocols",
	"permits",
	"projects",
	"supplies",
}

func (s *Store) persist(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.ExportState()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	for _, bucket := range postgresBuckets {
		var data []byte
		switch bucket {
		case "organisms":
			data, err = json.Marshal(snapshot.Organisms)
		case "cohorts":
			data, err = json.Marshal(snapshot.Cohorts)
		case "housing":
			data, err = json.Marshal(snapshot.Housing)
		case "facilities":
			data, err = json.Marshal(snapshot.Facilities)
		case "breeding":
			data, err = json.Marshal(snapshot.Breeding)
		case "lines":
			data, err = json.Marshal(snapshot.Lines)
		case "strains":
			data, err = json.Marshal(snapshot.Strains)
		case "markers":
			data, err = json.Marshal(snapshot.Markers)
		case "procedures":
			data, err = json.Marshal(snapshot.Procedures)
		case "treatments":
			data, err = json.Marshal(snapshot.Treatments)
		case "observations":
			data, err = json.Marshal(snapshot.Observations)
		case "samples":
			data, err = json.Marshal(snapshot.Samples)
		case "protocols":
			data, err = json.Marshal(snapshot.Protocols)
		case "permits":
			data, err = json.Marshal(snapshot.Permits)
		case "projects":
			data, err = json.Marshal(snapshot.Projects)
		case "supplies":
			data, err = json.Marshal(snapshot.Supplies)
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO state(bucket,payload) VALUES($1,$2) ON CONFLICT(bucket) DO UPDATE SET payload=EXCLUDED.payload`, bucket, data); err != nil {
			return fmt.Errorf("upsert %s: %w", bucket, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// OverrideSQLOpen swaps the sqlOpen function for tests and returns a restore function.
func OverrideSQLOpen(fn func(driverName, dataSourceName string) (*sql.DB, error)) func() {
	openMu.Lock()
	defer openMu.Unlock()
	prev := sqlOpen
	sqlOpen = fn
	return func() {
		openMu.Lock()
		defer openMu.Unlock()
		sqlOpen = prev
	}
}
