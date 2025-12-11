package postgres

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNewStoreAppliesDDLAndLoadsSnapshot(t *testing.T) {
	now := time.Now().UTC()
	org := domain.Organism{Organism: entitymodel.Organism{
		ID:        "org-1",
		Name:      "Org",
		Species:   "frog",
		Line:      "L1",
		Stage:     domain.StagePlanned,
		CreatedAt: now,
		UpdatedAt: now,
	}}
	payload, err := json.Marshal(map[string]domain.Organism{"org-1": org})
	if err != nil {
		t.Fatalf("marshal organism: %v", err)
	}

	initial := map[string]json.RawMessage{
		"organisms": payload,
	}
	for _, bucket := range postgresBuckets {
		if _, ok := initial[bucket]; !ok {
			initial[bucket] = json.RawMessage("{}")
		}
	}

	db, conn := newStubDB(initial)
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if got, ok := store.GetOrganism("org-1"); !ok {
		t.Fatalf("organism not loaded")
	} else if got.Name != "Org" || got.Species != "frog" {
		t.Fatalf("unexpected organism: %+v", got)
	}
	if len(conn.execs) == 0 || !strings.Contains(conn.execs[0], "CREATE TABLE IF NOT EXISTS facilities") {
		t.Fatalf("expected entity-model DDL to be applied, got execs: %v", conn.execs)
	}
}

func TestRunInTransactionPersistsState(t *testing.T) {
	db, conn := newStubDB(nil)
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_, err = store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, err := tx.CreateFacility(domain.Facility{
			Facility: entitymodel.Facility{
				Code:         "FAC",
				Name:         "Facility",
				Zone:         "A",
				AccessPolicy: "all",
			},
		})
		return err
	})
	if err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
	if len(conn.state) != len(postgresBuckets) {
		t.Fatalf("expected %d buckets, got %d", len(postgresBuckets), len(conn.state))
	}
	var facilities map[string]domain.Facility
	if err := json.Unmarshal(conn.state["facilities"], &facilities); err != nil {
		t.Fatalf("decode facilities: %v", err)
	}
	if len(facilities) != 1 {
		t.Fatalf("expected 1 facility persisted, got %d", len(facilities))
	}
}

func TestRunInTransactionPersistError(t *testing.T) {
	db, conn := newStubDB(nil)
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	conn.failExec = true
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, err := tx.CreateFacility(domain.Facility{
			Facility: entitymodel.Facility{
				Code:         "FAC",
				Name:         "Facility",
				Zone:         "A",
				AccessPolicy: "all",
			},
		})
		return err
	}); err == nil {
		t.Fatalf("expected persist error")
	}
	if len(conn.state) != 0 {
		t.Fatalf("expected no persisted buckets on failure, got %d", len(conn.state))
	}
}

func TestRunInTransactionUserError(t *testing.T) {
	db, conn := newStubDB(nil)
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	userErr := fmt.Errorf("user fail")
	if _, err := store.RunInTransaction(context.Background(), func(domain.Transaction) error { return userErr }); err != userErr {
		t.Fatalf("expected user error, got %v", err)
	}
	if len(conn.state) != 0 {
		t.Fatalf("expected no persistence on user error, got %d", len(conn.state))
	}
}

func TestLoadSnapshotInvalidPayload(t *testing.T) {
	db, _ := newStubDB(map[string]json.RawMessage{
		"organisms": []byte("not-json"),
	})
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected load error for invalid payload")
	}
}

func TestApplyEntityModelDDLError(t *testing.T) {
	db, conn := newStubDB(nil)
	conn.failExec = true
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected ddl error")
	}
}

func TestEnsureStateTableError(t *testing.T) {
	db, conn := newStubDB(nil)
	conn.failStateDDL = true
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected state table error")
	}
}

func TestLoadSnapshotHandlesEmptyPayloadAndRowsError(t *testing.T) {
	db, conn := newStubDB(map[string]json.RawMessage{
		"permits": {},
	})
	conn.rowsErr = fmt.Errorf("row err")
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected rows error")
	}
}

func TestStoreDBExposesHandle(t *testing.T) {
	db, _ := newStubDB(nil)
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if store.DB() == nil {
		t.Fatalf("expected DB handle")
	}
}

// --- stub driver helpers ---

type stubDriver struct {
	conn *stubConn
}

func (d *stubDriver) Open(string) (driver.Conn, error) {
	return d.conn, nil
}

type stubConn struct {
	execs        []string
	state        map[string]json.RawMessage
	failExec     bool
	failStateDDL bool
	failBegin    bool
	rowsErr      error
}

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("not implemented") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *stubConn) Ping(_ context.Context) error {
	if c.failExec {
		return fmt.Errorf("ping fail")
	}
	return nil
}

func (c *stubConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if c.failBegin {
		return nil, fmt.Errorf("begin fail")
	}
	return &stubTx{conn: c}, nil
}

func (c *stubConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.execs = append(c.execs, query)
	if c.failExec {
		return nil, fmt.Errorf("exec fail")
	}
	if strings.Contains(strings.ToLower(query), "create table if not exists state") && c.failStateDDL {
		return nil, fmt.Errorf("state table fail")
	}
	upper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(upper, "INSERT INTO STATE") && len(args) == 2 {
		bucket, _ := args[0].Value.(string)
		raw, _ := args[1].Value.([]byte)
		if c.state == nil {
			c.state = make(map[string]json.RawMessage)
		}
		c.state[bucket] = append([]byte(nil), raw...)
	}
	return driver.RowsAffected(1), nil
}

func (c *stubConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.state == nil {
		c.state = make(map[string]json.RawMessage)
	}
	rows := make([][]driver.Value, 0, len(c.state))
	for bucket, payload := range c.state {
		rows = append(rows, []driver.Value{bucket, []byte(payload)})
	}
	return &stubRows{
		cols: []string{"bucket", "payload"},
		rows: rows,
		err:  c.rowsErr,
	}, nil
}

type stubTx struct {
	conn *stubConn
}

func (t *stubTx) Commit() error   { return nil }
func (t *stubTx) Rollback() error { return nil }

type stubRows struct {
	cols []string
	rows [][]driver.Value
	idx  int
	err  error
}

func (r *stubRows) Columns() []string { return r.cols }
func (r *stubRows) Close() error      { return nil }

func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		if r.err != nil {
			return r.err
		}
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func newStubDB(initial map[string]json.RawMessage) (*sql.DB, *stubConn) {
	conn := &stubConn{state: initial}
	name := fmt.Sprintf("stubpg%d", time.Now().UnixNano())
	sql.Register(name, &stubDriver{conn: conn})
	db, err := sql.Open(name, "stub")
	if err != nil {
		panic(err)
	}
	return db, conn
}
