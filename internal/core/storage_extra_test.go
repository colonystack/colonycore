package core

import (
	"os"
	"testing"
)

func TestOpenPersistentStoreVariants(t *testing.T) {
	engine := NewDefaultRulesEngine()
	// default (sqlite)
	_ = os.Unsetenv("COLONYCORE_STORAGE_DRIVER")
	st, err := OpenPersistentStore(engine)
	if err != nil || st == nil {
		t.Fatalf("default open: %v %v", st, err)
	}
	// memory
	if err := os.Setenv("COLONYCORE_STORAGE_DRIVER", "memory"); err != nil {
		t.Fatalf("setenv memory: %v", err)
	}
	st, err = OpenPersistentStore(engine)
	if err != nil || st == nil {
		t.Fatalf("memory open: %v %v", st, err)
	}
	// unknown
	if err := os.Setenv("COLONYCORE_STORAGE_DRIVER", "gibberish"); err != nil {
		t.Fatalf("setenv gibberish: %v", err)
	}
	if _, err := OpenPersistentStore(engine); err == nil {
		t.Fatalf("expected unknown driver error")
	}
}

func TestPostgresStorePlaceholder(t *testing.T) {
	engine := NewDefaultRulesEngine()
	if err := os.Setenv("COLONYCORE_STORAGE_DRIVER", "postgres"); err != nil {
		t.Fatalf("setenv postgres: %v", err)
	}
	_, err := OpenPersistentStore(engine)
	if err == nil {
		t.Fatalf("expected postgres placeholder error")
	}
}
