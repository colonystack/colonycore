package pluginapi

import "testing"

func TestChangeAccessors(t *testing.T) {
	before := map[string]any{"a": 1}
	after := map[string]any{"a": 2}
	ch := newChangeForTest(entityOrganism, actionUpdate, before, after)
	if ch.Entity() != entityOrganism {
		t.Fatalf("unexpected entity: %v", ch.Entity())
	}
	if ch.Action() != actionUpdate {
		t.Fatalf("unexpected action: %v", ch.Action())
	}
	// mutate originals after construction; snapshots should be stable
	before["a"] = 99
	after["a"] = 99
	if b := ch.Before().(map[string]any)["a"]; b != 1 {
		t.Fatalf("before snapshot mutated: %v", b)
	}
	if a := ch.After().(map[string]any)["a"]; a != 2 {
		t.Fatalf("after snapshot mutated: %v", a)
	}
	// mutate returned maps; internal snapshots must remain unchanged
	rb := ch.Before().(map[string]any)
	ra := ch.After().(map[string]any)
	rb["a"] = -1
	ra["a"] = -1
	if b := ch.Before().(map[string]any)["a"]; b != 1 {
		t.Fatalf("before accessor not defensive: %v", b)
	}
	if a := ch.After().(map[string]any)["a"]; a != 2 {
		t.Fatalf("after accessor not defensive: %v", a)
	}
}

func TestResultMergeAndBlocking(t *testing.T) {
	r1 := Result{}
	sev := NewSeverityContext().Warn()
	ent := NewEntityContext().Organism()
	r2 := NewResult(NewViolation("r", sev, "", ent, ""))
	r1 = r1.Merge(r2)
	if len(r1.Violations()) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(r1.Violations()))
	}
	if r1.HasBlocking() {
		t.Fatalf("did not expect blocking violation")
	}
	sevBlock := NewSeverityContext().Block()
	entOrg := NewEntityContext().Organism()
	r3 := NewResult(NewViolation("b", sevBlock, "", entOrg, ""))
	if !r3.HasBlocking() {
		t.Fatalf("expected blocking violation detection")
	}
}

func TestRuleViolationError(t *testing.T) {
	sevWarn := NewSeverityContext().Warn()
	entOrg := NewEntityContext().Organism()
	err := RuleViolationError{Result: NewResult(NewViolation("x", sevWarn, "", entOrg, ""))}
	if err.Error() == "" {
		t.Fatalf("expected non-empty error message")
	}
}

// additional tests to exercise snapshotValue defensive copying branches
func TestChangeSnapshotSlicesAndMaps(t *testing.T) {
	beforeSlice := []string{"x", "y"}
	afterSlice := []map[string]any{{"k": "v"}}
	ch := newChangeForTest(entityProtocol, actionUpdate, beforeSlice, afterSlice)
	// mutate originals
	beforeSlice[0] = "z"
	afterSlice[0]["k"] = "w"
	bs, ok := ch.Before().([]string)
	if !ok || bs[0] != "x" {
		t.Fatalf("expected cloned []string with first element 'x', got %v", ch.Before())
	}
	am, ok := ch.After().([]map[string]any)
	if !ok || am[0]["k"] != "v" {
		t.Fatalf("expected cloned []map with value 'v', got %v", ch.After())
	}
}

func TestChangeSnapshotStructFallback(t *testing.T) {
	type simple struct {
		N int `json:"n"`
	}
	s := simple{N: 5}
	ch := newChangeForTest(entityProject, actionCreate, s, nil)
	// JSON round-trip returns map[string]any representation
	b := ch.Before()
	m, ok := b.(map[string]any)
	if !ok {
		t.Fatalf("expected map clone of struct, got %T", b)
	}
	if m["n"].(float64) != 5 { // JSON numbers are float64
		t.Fatalf("expected 5, got %v", m["n"])
	}
}
