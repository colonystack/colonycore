package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffFingerprintsDetectsRemovals(t *testing.T) {
	facilityBase := entityFingerprint{
		Properties: []string{"id", "name"},
		Required:   []string{"id"},
		Invariants: []string{"has_name"},
		Relationships: map[string]relationshipFingerprint{
			"project_ids": {Target: "Project", Cardinality: "0..n", Storage: "join"},
		},
		States: &stateSpec{Enum: "state", Initial: "draft", Terminal: []string{"archived"}},
	}
	baseline := fingerprintDoc{
		Version: "0.1.0",
		Enums: map[string][]string{
			"status": {"approved", "draft"},
		},
		Entities: map[string]entityFingerprint{
			"Facility": facilityBase,
		},
	}
	current := fingerprintDoc{
		Version: "0.1.0",
		Enums: map[string][]string{
			"status": {"approved"},
		},
		Entities: map[string]entityFingerprint{
			"Facility": {
				Properties:    append([]string(nil), facilityBase.Properties...),
				Required:      append([]string(nil), facilityBase.Required...),
				Invariants:    append([]string(nil), facilityBase.Invariants...),
				Relationships: map[string]relationshipFingerprint{},
				States:        facilityBase.States,
			},
		},
	}

	issues := diffFingerprints(baseline, current)
	if len(issues) == 0 {
		t.Fatalf("expected removals detected, got %v", issues)
	}
}

func TestDiffStatesDetectsChanges(t *testing.T) {
	base := &stateSpec{Enum: "state", Initial: "draft", Terminal: []string{"done"}}
	changed := &stateSpec{Enum: "state", Initial: "new", Terminal: []string{"done"}}
	if msg := diffStates("Thing", base, changed); msg == "" {
		t.Fatal("expected initial change detected")
	}
}

func TestComputeFingerprintSortsDeterministically(t *testing.T) {
	doc := schemaDoc{
		Version: "0.2.0",
		Enums: map[string]enumSpec{
			"status": {Values: []string{"beta", "alpha"}},
		},
		Entities: map[string]entitySpec{
			"Thing": {
				Required: []string{"name", "id"},
				Properties: map[string]json.RawMessage{
					"name": json.RawMessage(`{"type":"string"}`),
					"id":   json.RawMessage(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{
					"links": {Target: "Link", Cardinality: "0..n", Storage: "join"},
				},
				Invariants: []string{"zeta", "alpha"},
			},
		},
	}
	fp := computeFingerprint(doc)
	vals := fp.Enums["status"]
	if vals[0] != "alpha" || vals[1] != "beta" {
		t.Fatalf("expected enum values sorted, got %v", vals)
	}
	thing := fp.Entities["Thing"]
	if thing.Properties[0] != "id" || thing.Properties[1] != "name" {
		t.Fatalf("expected property keys sorted, got %v", thing.Properties)
	}
	if thing.Required[0] != "id" {
		t.Fatalf("expected required fields sorted, got %v", thing.Required)
	}
	if thing.Invariants[0] != "alpha" {
		t.Fatalf("expected invariants sorted, got %v", thing.Invariants)
	}
}

func TestDiffFingerprintsVersionAndRelationshipChange(t *testing.T) {
	baseRel := relationshipFingerprint{Target: "Link", Cardinality: "0..n", Storage: "join"}
	baseline := fingerprintDoc{
		Version: "1.0.0",
		Enums:   map[string][]string{"status": {"draft"}},
		Entities: map[string]entityFingerprint{
			"Thing": {
				Properties: []string{"id"},
				Required:   []string{"id"},
				Relationships: map[string]relationshipFingerprint{
					"links": baseRel,
				},
			},
		},
	}
	current := fingerprintDoc{
		Version: "2.0.0",
		Enums:   map[string][]string{"status": {"draft"}},
		Entities: map[string]entityFingerprint{
			"Thing": {
				Properties: []string{"id"},
				Required:   []string{"id"},
				Relationships: map[string]relationshipFingerprint{
					"links": {Target: "Link", Cardinality: "1..1", Storage: "fk"},
				},
			},
		},
	}
	issues := diffFingerprints(baseline, current)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "schema version changed") {
		t.Fatalf("expected schema version change reported, got %v", issues)
	}
	if !strings.Contains(joined, "relationship changed") {
		t.Fatalf("expected relationship change reported, got %v", issues)
	}
}

func TestLoadAndWriteFingerprintRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fingerprint.json")
	input := fingerprintDoc{Version: "0.0.1", Enums: map[string][]string{}, Entities: map[string]entityFingerprint{}}
	if err := writeFingerprint(path, input); err != nil {
		t.Fatalf("write fingerprint: %v", err)
	}
	loaded, err := loadFingerprint(path)
	if err != nil {
		t.Fatalf("load fingerprint: %v", err)
	}
	if loaded.Version != input.Version {
		t.Fatalf("expected version %s, got %s", input.Version, loaded.Version)
	}
}

func TestLoadSchemaReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.json")
	schema := `{"version":"0.0.1","enums":{},"entities":{}}`
	if err := os.WriteFile(path, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if _, err := loadSchema(path); err != nil {
		t.Fatalf("load schema: %v", err)
	}
}

func TestLoadFingerprintParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fingerprint.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid fingerprint: %v", err)
	}
	if _, err := loadFingerprint(path); err == nil {
		t.Fatalf("expected parse error for fingerprint")
	}
}

func TestDiffListDetectsRemovedEntries(t *testing.T) {
	issues := diffList("entity Facility", "property", []string{"name", "id"}, []string{"id"})
	if len(issues) != 1 || !strings.Contains(issues[0], "name") {
		t.Fatalf("expected removal reported for name, got %v", issues)
	}
}

func TestExitErrWritesAndExits(t *testing.T) {
	var capturedCode int
	exitFunc = func(code int) { capturedCode = code }
	defer func() { exitFunc = os.Exit }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	originalStderr := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = originalStderr }()

	exitErr(errors.New("fingerprint mismatch"))

	_ = writer.Close()
	out, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("read stderr: %v", readErr)
	}
	if capturedCode != 1 {
		t.Fatalf("expected exit code 1, got %d", capturedCode)
	}
	if !strings.Contains(string(out), "fingerprint mismatch") {
		t.Fatalf("expected error output, got %q", string(out))
	}
}
