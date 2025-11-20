package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateOK(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.0.1",
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Bar": {
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "relationships": {}
    },
    "Foo": {
      "required": ["id", "created_at", "updated_at", "name"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "name": {"type":"string"},
        "status": {"type":"string"}
      },
      "states": {"enum": "status"},
      "relationships": {
        "bar_id": {"target": "Bar"}
      }
    }
  }
}`)

	if err := validate(path); err != nil {
		t.Fatalf("validate() unexpected error: %v", err)
	}
}

func TestValidateFailures(t *testing.T) {
	path := writeTemp(t, `{
  "version": "",
  "enums": {
    "status": { "values": [] }
  },
  "entities": {
    "Foo": {
      "required": ["id", "created_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"}
      },
      "states": {"enum": "missing_enum"},
      "relationships": {
        "bar_id": {"target": "Missing"}
      }
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	expect := []string{
		"version must be set",
		"enum \"status\" must include at least one value",
		"entity \"Foo\" must require base field \"updated_at\"",
		"entity \"Foo\" states.enum \"missing_enum\" not found in enums",
		"entity \"Foo\" relationship \"bar_id\" targets unknown entity \"Missing\"",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "entity-model-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	return f.Name()
}
