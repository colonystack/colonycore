package core

import (
	"testing"
	"time"
)

func TestCoerceParameter_StringEnumInvalid(t *testing.T) {
	p := DatasetParameter{Name: "Color", Type: "string", Enum: []string{"red", "green"}}
	if _, err := coerceParameter(p, "blue"); err == nil {
		// expect enumeration error
		t.Fatalf("expected enum error")
	}
}

func TestCoerceParameter_IntegerFromFloatNonInteger(t *testing.T) {
	p := DatasetParameter{Name: "Count", Type: "integer"}
	if _, err := coerceParameter(p, 1.2); err == nil {
		t.Fatalf("expected error for non-integer float")
	}
}

func TestCoerceParameter_BooleanInvalid(t *testing.T) {
	p := DatasetParameter{Name: "Flag", Type: "boolean"}
	if _, err := coerceParameter(p, "notbool"); err == nil {
		t.Fatalf("expected error for invalid bool string")
	}
}

func TestCoerceParameter_TimestampInvalid(t *testing.T) {
	p := DatasetParameter{Name: "When", Type: "timestamp"}
	if _, err := coerceParameter(p, "2023-13-99"); err == nil {
		t.Fatalf("expected timestamp parse error")
	}
}

func TestCoerceParameter_TimestampString(t *testing.T) {
	p := DatasetParameter{Name: "When", Type: "timestamp"}
	val, err := coerceParameter(p, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, ok := val.(time.Time); !ok {
		t.Fatalf("expected time.Time, got %T", val)
	}
}

func TestCoerceParameter_UnsupportedType(t *testing.T) {
	p := DatasetParameter{Name: "X", Type: "weird"}
	if _, err := coerceParameter(p, "anything"); err == nil {
		t.Fatalf("expected unsupported type error")
	}
}

func TestCoerceParameter_NilValue(t *testing.T) {
	p := DatasetParameter{Name: "X", Type: "string"}
	if _, err := coerceParameter(p, nil); err == nil {
		t.Fatalf("expected nil value error")
	}
}

func TestValidateParametersLeftover(t *testing.T) {
	defs := []DatasetParameter{{Name: "A", Type: "string"}, {Name: "B", Type: "integer"}}
	params := map[string]any{"A": "ok", "C": 1, "b": 2}
	cleaned, errs := validateParameters(defs, params)
	if len(errs) == 0 {
		// expect error for leftover param C
		t.Fatalf("expected leftover param error")
	}
	if _, ok := cleaned["A"]; !ok {
		t.Fatalf("expected cleaned param A present")
	}
	if _, ok := cleaned["B"]; !ok {
		// B supplied with lowercase key, should have been matched case-insensitively
		t.Fatalf("expected cleaned param B present from lowercase input")
	}
	// Ensure leftover error names sorted
	last := ""
	for _, e := range errs {
		if last != "" && e.Name < last {
			// not sorted
			t.Fatalf("errors not sorted: %v", errs)
		}
		last = e.Name
	}
}
