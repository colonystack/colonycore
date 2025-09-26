package datasets

import (
	"colonycore/internal/core"
	"testing"
)

func TestWorkerMaterializeJSONMarshalError(t *testing.T) {
	w := &Worker{}
	// Build template with columns so schema available
	template := core.DatasetTemplate{Key: "k", Version: "v1", Title: "t", Dialect: core.DatasetDialectSQL, Query: "select 1", Columns: []core.DatasetColumn{{Name: "a", Type: "string"}}, OutputFormats: []core.DatasetFormat{core.FormatJSON}, Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) { return nil, nil }}
	// Result with unsupported value (functions can't be marshaled)
	result := core.DatasetRunResult{Schema: template.Columns, Rows: []map[string]any{{"a": func() {}}}}
	if _, err := w.materialize(core.FormatJSON, template, result); err == nil {
		// Expect marshal error
		t.Fatalf("expected marshal error")
	}
}
