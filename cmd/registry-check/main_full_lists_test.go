package main

import (
	"os"
	"testing"
	"time"
)

// TestRegistryAllListFields exercises appendList/resetList branches for every supported list key.
func TestRegistryAllListFields(t *testing.T) {
	content := "documents:\n" +
		"  - id: RFC-ALL\n" +
		"    type: RFC\n" +
		"    title: Lists\n" +
		"    status: Draft\n" +
		"    path: docs/rfc/rfc-all.md\n" +
		"    authors:\n" +
		"      - A1\n" +
		"    stakeholders:\n" +
		"      - S1\n" +
		"    reviewers:\n" +
		"      - R1\n" +
		"    owners:\n" +
		"      - O1\n" +
		"    deciders:\n" +
		"      - D1\n" +
		"    linked_annexes:\n" +
		"      - Annex-1\n" +
		"    linked_adrs:\n" +
		"      - ADR-1\n" +
		"    linked_rfcs:\n" +
		"      - RFC-1\n"
	// create a relative file name to satisfy validatePath (no absolute paths)
	name := time.Now().UTC().Format("20060102_150405") + "_full_lists_registry.yaml"
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer func() { _ = os.Remove(name) }()
	if err := run(name); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
