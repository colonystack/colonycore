package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestValidateOK(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.0.1",
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok", "fail"] }
  },
  "entities": {
    "Bar": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "code": {"type":"string"}
      },
      "relationships": {},
      "invariants": []
    },
    "Foo": {
      "natural_keys": [
        {"fields": ["name"], "scope": "global", "description": "name must be unique"}
      ],
      "required": ["id", "created_at", "updated_at", "name"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "name": {"type":"string"},
        "status": {"type":"string"},
        "bar_id": {"type":"string"}
      },
      "states": {"enum": "status", "initial": "ok", "terminal": ["fail"]},
      "relationships": {
        "bar_id": {"target": "Bar"}
      },
      "invariants": ["rule_one"]
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
  "metadata": { "status": "" },
  "enums": {
    "status": { "values": [] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [
        {"fields": [], "scope": ""}
      ],
      "required": ["id", "created_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"}
      },
      "states": {"enum": "missing_enum"},
      "relationships": {
        "bar_id": {"target": "Missing"}
      },
      "invariants": ["", " "]
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
		"metadata.status must be set",
		"enum \"status\" must include at least one value",
		"entity \"Foo\" must require base field \"updated_at\"",
		"entity \"Foo\" natural key #0 must declare at least one field",
		"entity \"Foo\" natural key [<unset>] must declare scope",
		"entity \"Foo\" relationship \"bar_id\" missing property definition",
		"entity \"Foo\" states.enum \"missing_enum\" not found in enums",
		"entity \"Foo\" relationship \"bar_id\" targets unknown entity \"Missing\"",
		"entity \"Foo\" invariants[0] must not be empty",
		"entity \"Foo\" invariants[1] must not be empty",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateTopLevelMissing(t *testing.T) {
	path := writeTemp(t, `{
  "version": "",
  "metadata": { "status": "" },
  "enums": {},
  "entities": {}
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	expect := []string{
		"version must be set",
		"metadata.status must be set",
		"enums must not be empty",
		"entities section must not be empty",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateNaturalKeyAndRelationshipErrors(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.0.1",
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Bar": {
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "bar_ref": {"type":"string"}
      },
      "relationships": {
        "bar_ref": {"target": ""}
      }
    },
    "Foo": {
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "relationships": {},
      "invariants": []
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	expect := []string{
		"entity \"Bar\" must declare natural_keys (empty array allowed)",
		"entity \"Foo\" must declare natural_keys (empty array allowed)",
		"entity \"Bar\" relationship \"bar_ref\" missing target",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "natural key field") {
		t.Fatalf("did not expect natural key field error when no natural keys defined")
	}
}

func TestValidateNaturalKeyFieldMissing(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.0.2",
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [
        {"fields": ["name"], "scope": ""}
      ],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "relationships": {}
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	expect := []string{
		"entity \"Foo\" natural key field \"name\" missing from properties",
		"entity \"Foo\" natural key [name] must declare scope",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateStatesAndDuplicates(t *testing.T) {
	path := writeTemp(t, `{
  "version": "1.0.0",
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["one", "one"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "states": {"enum": "status", "initial": "missing", "terminal": ["one", "two", "one"]},
      "relationships": {},
      "invariants": ["dup", "dup"]
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	expect := []string{
		"enum \"status\" has duplicate value \"one\"",
		"entity \"Foo\" states.initial \"missing\" not found in enum \"status\"",
		"entity \"Foo\" states.terminal value \"two\" not found in enum \"status\"",
		"entity \"Foo\" states.terminal has duplicate value \"one\"",
		"entity \"Foo\" invariants has duplicate entry \"dup\"",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestContains(t *testing.T) {
	t.Helper()
	if !contains([]string{"Id", "Created"}, "id") {
		t.Fatalf("contains should be case insensitive")
	}
	if contains([]string{"foo"}, "bar") {
		t.Fatalf("contains returned true for missing element")
	}
}

func TestMainSuccess(t *testing.T) {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	path := writeTemp(t, `{
  "version": "0.0.3",
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "relationships": {}
    }
  }
}`)
	os.Args = []string{"entitymodelvalidate", path}

	main()
}

func TestExitErr(t *testing.T) {
	defer func() { exitFn = os.Exit }()
	defer func() { errWriter = os.Stderr }()

	var buf bytes.Buffer
	errWriter = &buf

	var code int
	exitFn = func(c int) {
		code = c
	}

	exitErr("boom")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("expected error output to contain message, got %q", buf.String())
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
