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

func TestSQLiteStorePersistsLineStrainMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist-lines.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}

	ctx := context.Background()
	var ids struct {
		line   string
		strain string
		marker string
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{Base: domain.Base{ID: "marker-store"}, Name: "Marker", Locus: "loc", Alleles: []string{"A"}, AssayMethod: "PCR", Interpretation: "ctrl", Version: "v1"})
		if err != nil {
			return err
		}
		ids.marker = marker.ID
		line, err := tx.CreateLine(domain.Line{Base: domain.Base{ID: "line-store"}, Code: "L", Name: "Line", Origin: "field", GenotypeMarkerIDs: []string{marker.ID}})
		if err != nil {
			return err
		}
		ids.line = line.ID
		strain, err := tx.CreateStrain(domain.Strain{Base: domain.Base{ID: "strain-store"}, Code: "S", Name: "Strain", LineID: line.ID, GenotypeMarkerIDs: []string{marker.ID}})
		if err != nil {
			return err
		}
		ids.strain = strain.ID
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	if line, ok := store.GetLine(ids.line); !ok || line.ID != ids.line {
		t.Fatalf("expected line persisted, got ok=%v", ok)
	}
	if strain, ok := store.GetStrain(ids.strain); !ok || strain.ID != ids.strain {
		t.Fatalf("expected strain persisted, got ok=%v", ok)
	}
	if marker, ok := store.GetGenotypeMarker(ids.marker); !ok || marker.ID != ids.marker {
		t.Fatalf("expected marker persisted, got ok=%v", ok)
	}

	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload sqlite: %v", err)
	}
	if len(reloaded.ListLines()) != 1 || len(reloaded.ListStrains()) != 1 || len(reloaded.ListGenotypeMarkers()) != 1 {
		t.Fatalf("expected entities to reload from sqlite")
	}
}
