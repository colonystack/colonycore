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
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
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
        "bar_id": {"target": "Bar", "cardinality": "0..1"}
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
		"id_semantics must be declared",
		"enum \"status\" must include at least one value",
		"enum \"status\" is defined but not referenced by any entity states or properties",
		"entity \"Foo\" must require base field \"updated_at\"",
		"entity \"Foo\" natural key #0 must declare at least one field",
		"entity \"Foo\" natural key [<unset>] must declare scope",
		"entity \"Foo\" relationship \"bar_id\" missing property definition",
		"entity \"Foo\" relationship \"bar_id\" missing cardinality",
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
		"id_semantics must be declared",
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
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
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
        "bar_ref": {"target": "", "cardinality": "0..1"}
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
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
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
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
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
		"entity \"Foo\" invariants[0] \"dup\" is not in the allowed invariants list",
		"entity \"Foo\" invariants[1] \"dup\" is not in the allowed invariants list",
		"entity \"Foo\" invariants has duplicate entry \"dup\"",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateRelationshipCardinality(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.0.4",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Bar": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"}
      },
      "relationships": {},
      "invariants": []
    },
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "bar_id": {"type":"string"},
        "invalid_card": {"type":"string"}
      },
      "states": {"enum": "status", "initial": "ok", "terminal": ["ok"]},
      "relationships": {
        "bar_id": {"target": "Bar"},
        "invalid_card": {"target": "Bar", "cardinality": "2..3"}
      },
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
		"entity \"Foo\" relationship \"bar_id\" missing cardinality",
		"entity \"Foo\" relationship \"invalid_card\" has invalid cardinality \"2..3\"",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateIDSemanticsRequired(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.0",
  "id_semantics": { "type": "", "scope": " ", "required": false, "description": "" },
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
        "updated_at": {"type":"string"},
        "status": {"$ref":"#/enums/status"}
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
		"id_semantics.type must be set",
		"id_semantics.scope must be set",
		"id_semantics.required must be true",
		"id_semantics.description must be set",
	}
	for _, want := range expect {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestValidateUnusedEnums(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.1",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "used": { "values": ["ok"] },
    "unused": { "values": ["x"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "status": {"$ref":"#/enums/used"}
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
	if !strings.Contains(msg, "enum \"unused\" is defined but not referenced by any entity states or properties") {
		t.Fatalf("expected unused enum error, got %q", msg)
	}
}

func TestValidatePropertyEnumReferenceUnknown(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.2",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at", "status"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "status": {"$ref":"#/enums/missing"}
      },
      "states": {"enum": "status", "initial": "ok", "terminal": ["ok"]},
      "relationships": {}
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "entity \"Foo\" property \"status\" references unknown enum \"missing\"") {
		t.Fatalf("expected unknown enum reference error, got %q", msg)
	}
}

func TestValidatePropertyRequiresTypeOrRef(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.3",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at", "status"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "status": {}
      },
      "states": {"enum": "status", "initial": "ok", "terminal": ["ok"]},
      "relationships": {}
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "entity \"Foo\" property \"status\" must declare a type or $ref") {
		t.Fatalf("expected property type/ref error, got %q", msg)
	}
}

func TestValidateEnumWhitespaceValue(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.4",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok", " "] }
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

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "enum \"status\" value #1 must not be empty") {
		t.Fatalf("expected enum whitespace error, got %q", msg)
	}
}

func TestValidatePropertyJSONError(t *testing.T) {
	path := writeTemp(t, `{
  "version": "0.1.3",
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
  "metadata": { "status": "seed" },
  "enums": {
    "status": { "values": ["ok"] }
  },
  "entities": {
    "Foo": {
      "natural_keys": [],
      "required": ["id", "created_at", "updated_at", "status"],
      "properties": {
        "id": {"type":"string"},
        "created_at": {"type":"string"},
        "updated_at": {"type":"string"},
        "status": true
      },
      "states": {"enum": "status", "initial": "ok", "terminal": ["ok"]},
      "relationships": {}
    }
  }
}`)

	err := validate(path)
	if err == nil {
		t.Fatalf("validate() expected error")
	}
	if !strings.Contains(err.Error(), "entity \"Foo\" property \"status\" invalid JSON") {
		t.Fatalf("expected property JSON error, got %q", err.Error())
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
  "id_semantics": { "type": "uuidv7", "scope": "global", "required": true, "description": "opaque" },
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
        "updated_at": {"type":"string"},
        "status": {"$ref":"#/enums/status"}
      },
      "relationships": {},
      "states": {"enum": "status", "initial": "ok", "terminal": ["ok"]}
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
