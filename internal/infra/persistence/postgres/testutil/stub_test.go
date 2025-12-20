package testutil

import (
	"context"
	"database/sql/driver"
	"testing"
)

func TestStubDBStoresAndQueriesRows(t *testing.T) {
	ctx := context.Background()
	_, conn := NewStubDB()

	if err := conn.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	_, err := conn.ExecContext(ctx, "INSERT INTO facilities (id, code) VALUES ($1,$2)", []driver.NamedValue{
		{Value: "fac-1"},
		{Value: "CODE"},
	})
	if err != nil {
		t.Fatalf("ExecContext insert: %v", err)
	}
	if len(conn.Tables["facilities"]) != 1 {
		t.Fatalf("expected facilities row to be stored, got %v", conn.Tables["facilities"])
	}

	_, err = conn.ExecContext(ctx, "DELETE FROM facilities WHERE id=$1", []driver.NamedValue{{Value: "fac-1"}})
	if err != nil {
		t.Fatalf("ExecContext delete: %v", err)
	}

	conn.Tables["facilities"] = []map[string]any{{"id": "fac-2", "code": "CODE2"}}
	rows, err := conn.QueryContext(ctx, "select id, code from facilities", nil)
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	defer func() { _ = rows.Close() }()

	dest := make([]driver.Value, 2)
	if err := rows.Next(dest); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if dest[0] != "fac-2" || dest[1] != "CODE2" {
		t.Fatalf("unexpected row values: %v", dest)
	}
}
