package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

func TestSQLiteStorePersistMarshalError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()

	_, err = store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Name: "Specimen", Species: "test"})
		if err != nil {
			return err
		}
		sample, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "organism",
			OrganismID:      &organism.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "cold",
			AssayType:       "chromatography",
		})
		if err != nil {
			return err
		}
		_, err = tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			// Introduce a value that cannot be JSON encoded to force marshal failure.
			return s.ApplySampleAttributes(map[string]any{"invalid": func() {}})
		})
		return err
	})
	if err == nil {
		t.Fatalf("expected persist marshal error")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestSQLiteStoreLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "load.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}

	ctx := context.Background()
	_, err = store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Name: "Specimen", Species: "test"})
		if err != nil {
			return err
		}
		_, err = tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "organism",
			OrganismID:      &organism.ID,
			FacilityID:      facility.ID,
			CollectedAt:     time.Now().UTC(),
			Status:          domain.SampleStatusStored,
			StorageLocation: "cold",
			AssayType:       "chromatography",
		})
		return err
	})
	if err != nil {
		t.Fatalf("seed transaction failed: %v", err)
	}

	if _, err := store.DB().Exec(`INSERT OR REPLACE INTO state(bucket,payload) VALUES(?,?)`, "samples", []byte("not-json")); err != nil {
		t.Fatalf("inject invalid state: %v", err)
	}
	if err := store.DB().Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, err = NewStore(path, domain.NewRulesEngine())
	if err == nil {
		t.Fatalf("expected load error due to invalid json")
	}
	if !strings.Contains(err.Error(), "decode samples") {
		t.Fatalf("expected decode samples error, got %v", err)
	}
}
