// Package testutil provides a normalized stub database for postgres store tests.
package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"time"
)

// StubConn records normalized statements for the postgres store during tests.
type StubConn struct {
	Execs      []string
	Tables     map[string][]map[string]any
	FailExec   bool
	FailBegin  bool
	RowsErr    error
	FailTables map[string]bool
	FailCommit bool
}

// NewStubDB registers a sql.DB backed by an in-memory stub connection.
func NewStubDB() (*sql.DB, *StubConn) {
	conn := &StubConn{Tables: make(map[string][]map[string]any)}
	name := fmt.Sprintf("stubpg%d", time.Now().UnixNano())
	sql.Register(name, &stubDriver{conn: conn})
	db, err := sql.Open(name, "stub")
	if err != nil {
		panic(err)
	}
	return db, conn
}

type stubDriver struct {
	conn *StubConn
}

func (d *stubDriver) Open(string) (driver.Conn, error) {
	return d.conn, nil
}

// Prepare implements driver.Conn.
func (c *StubConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("not implemented") }

// Close implements driver.Conn.
func (c *StubConn) Close() error { return nil }

// Begin implements driver.Conn.
func (c *StubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// Ping implements driver.Pinger.
func (c *StubConn) Ping(_ context.Context) error {
	if c.FailExec {
		return fmt.Errorf("ping fail")
	}
	return nil
}

// BeginTx implements driver.ConnBeginTx.
func (c *StubConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if c.FailBegin {
		return nil, fmt.Errorf("begin fail")
	}
	return &stubTx{conn: c}, nil
}

// ExecContext implements driver.ExecerContext.
func (c *StubConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.Execs = append(c.Execs, query)
	if c.FailExec {
		return nil, fmt.Errorf("exec fail")
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "TRUNCATE TABLE") {
		c.Tables = make(map[string][]map[string]any)
		return driver.RowsAffected(0), nil
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "INSERT INTO") {
		table, cols, err := parseInsert(query)
		if err != nil {
			return nil, err
		}
		if c.FailTables != nil && c.FailTables[table] {
			return nil, fmt.Errorf("exec fail for %s", table)
		}
		if len(cols) != len(args) {
			return nil, fmt.Errorf("column/arg mismatch for %s", table)
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = args[i].Value
		}
		if strings.Contains(strings.ToUpper(query), "ON CONFLICT") && len(cols) > 0 {
			primary := cols[0]
			var filtered []map[string]any
			for _, existing := range c.Tables[table] {
				if existing[primary] == row[primary] {
					continue
				}
				filtered = append(filtered, existing)
			}
			c.Tables[table] = filtered
		}
		c.Tables[table] = append(c.Tables[table], row)
		return driver.RowsAffected(1), nil
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DELETE FROM") {
		table, col, err := parseDelete(query)
		if err != nil {
			return nil, err
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("missing args for delete %s", table)
		}
		target := args[0].Value
		var filtered []map[string]any
		for _, row := range c.Tables[table] {
			if row[col] == target {
				continue
			}
			filtered = append(filtered, row)
		}
		c.Tables[table] = filtered
		return driver.RowsAffected(1), nil
	}
	return driver.RowsAffected(1), nil
}

// QueryContext implements driver.QueryerContext.
func (c *StubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.Tables == nil {
		c.Tables = make(map[string][]map[string]any)
	}
	table, cols, err := parseSelect(query)
	if err != nil {
		return nil, err
	}
	if c.FailTables != nil && c.FailTables[table] {
		return nil, fmt.Errorf("query fail for %s", table)
	}
	tableRows := c.Tables[table]
	values := make([][]driver.Value, 0, len(tableRows))
	for _, row := range tableRows {
		vals := make([]driver.Value, len(cols))
		for i, col := range cols {
			vals[i] = row[col]
		}
		values = append(values, vals)
	}
	return &stubRows{
		cols: cols,
		rows: values,
		err:  c.RowsErr,
	}, nil
}

type stubTx struct {
	conn *StubConn
}

func (t *stubTx) Commit() error {
	if t.conn.FailCommit {
		return fmt.Errorf("commit fail")
	}
	return nil
}
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

func parseInsert(query string) (string, []string, error) {
	up := strings.ToUpper(query)
	intoIdx := strings.Index(up, "INTO ")
	if intoIdx == -1 {
		return "", nil, fmt.Errorf("cannot parse insert: %s", query)
	}
	rest := strings.TrimSpace(query[intoIdx+len("INTO "):])
	open := strings.Index(rest, "(")
	closeIdx := strings.Index(rest, ")")
	if open == -1 || closeIdx == -1 || closeIdx <= open {
		return "", nil, fmt.Errorf("cannot parse insert: %s", query)
	}
	table := strings.ToLower(strings.TrimSpace(rest[:open]))
	cols := splitColumns(rest[open+1 : closeIdx])
	return table, cols, nil
}

func parseDelete(query string) (string, string, error) {
	lower := strings.ToLower(query)
	prefix := "delete from "
	whereToken := " where "
	if !strings.HasPrefix(lower, prefix) {
		return "", "", fmt.Errorf("cannot parse delete: %s", query)
	}
	rest := strings.TrimSpace(query[len(prefix):])
	whereIdx := strings.Index(strings.ToLower(rest), whereToken)
	if whereIdx == -1 {
		return "", "", fmt.Errorf("cannot parse delete: %s", query)
	}
	table := strings.ToLower(strings.TrimSpace(rest[:whereIdx]))
	where := strings.TrimSpace(rest[whereIdx+len(whereToken):])
	parts := strings.SplitN(where, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("cannot parse delete predicate: %s", query)
	}
	col := strings.ToLower(strings.TrimSpace(parts[0]))
	return table, col, nil
}

func parseSelect(query string) (string, []string, error) {
	lower := strings.ToLower(query)
	selectPrefix := "select "
	fromToken := " from "
	if !strings.HasPrefix(lower, selectPrefix) {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	fromIdx := strings.Index(lower, fromToken)
	if fromIdx == -1 {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	cols := query[len(selectPrefix):fromIdx]
	table := strings.TrimSpace(query[fromIdx+len(fromToken):])
	if table == "" {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	table = strings.Fields(table)[0]
	return strings.ToLower(table), splitColumns(cols), nil
}

func splitColumns(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.ToLower(strings.TrimSpace(part)))
	}
	return out
}
