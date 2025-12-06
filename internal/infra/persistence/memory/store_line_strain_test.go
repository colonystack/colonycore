package memory_test

import (
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
)

func TestMemoryStoreLineStrainGenotypeLifecycle(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()

	var ids struct {
		line     string
		strain   string
		marker   string
		organism string
		breeding string
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{
			Name:           "Marker-1",
			Locus:          "loc-1",
			Alleles:        []string{"A", "A", "B"},
			AssayMethod:    "PCR",
			Interpretation: "control",
			Version:        "v1",
		})
		if err != nil {
			return err
		}
		ids.marker = marker.ID

		if _, err := tx.CreateStrain(domain.Strain{Code: "bad", Name: "No line"}); err == nil {
			return fmt.Errorf("expected strain creation to require line")
		}

		line, err := tx.CreateLine(domain.Line{
			Code:              "L-1",
			Name:              "Line",
			Origin:            "wild",
			GenotypeMarkerIDs: []string{marker.ID, "missing-marker", marker.ID},
		})
		if err != nil {
			return err
		}
		ids.line = line.ID
		if got := len(line.GenotypeMarkerIDs); got != 1 {
			return fmt.Errorf("expected line marker dedupe, got %d", got)
		}

		strain, err := tx.CreateStrain(domain.Strain{
			Code:              "S-1",
			Name:              "Strain",
			LineID:            line.ID,
			GenotypeMarkerIDs: []string{marker.ID, "missing-marker"},
		})
		if err != nil {
			return err
		}
		ids.strain = strain.ID

		breeding, err := tx.CreateBreedingUnit(domain.BreedingUnit{
			Name:     "Breeding",
			Strategy: "pair",
			LineID:   &line.ID,
			StrainID: &strain.ID,
			FemaleIDs: []string{
				ids.strain,
			},
		})
		if err != nil {
			return err
		}
		ids.breeding = breeding.ID

		org, err := tx.CreateOrganism(domain.Organism{
			Name:     "Org",
			Species:  "Spec",
			LineID:   &line.ID,
			StrainID: &strain.ID,
		})
		if err != nil {
			return err
		}
		ids.organism = org.ID
		if _, ok := tx.FindLine(ids.line); !ok {
			return fmt.Errorf("expected tx to find line")
		}
		if _, ok := tx.FindStrain(ids.strain); !ok {
			return fmt.Errorf("expected tx to find strain")
		}
		if _, ok := tx.FindGenotypeMarker(ids.marker); !ok {
			return fmt.Errorf("expected tx to find marker")
		}
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	if markers := store.ListGenotypeMarkers(); len(markers) != 1 || len(markers[0].Alleles) != 2 {
		t.Fatalf("expected deduped marker alleles, got %+v", markers)
	}
	if lines := store.ListLines(); len(lines) != 1 || lines[0].ID != ids.line {
		t.Fatalf("expected line to be persisted")
	}
	if strains := store.ListStrains(); len(strains) != 1 || strains[0].ID != ids.strain {
		t.Fatalf("expected strain to be persisted")
	}
	if _, ok := store.GetLine(ids.line); !ok {
		t.Fatalf("expected line getter to succeed")
	}
	if _, ok := store.GetStrain(ids.strain); !ok {
		t.Fatalf("expected strain getter to succeed")
	}
	if _, ok := store.GetGenotypeMarker(ids.marker); !ok {
		t.Fatalf("expected genotype marker getter to succeed")
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		if _, ok := view.FindLine(ids.line); !ok {
			return fmt.Errorf("expected line in view")
		}
		if _, ok := view.FindStrain(ids.strain); !ok {
			return fmt.Errorf("expected strain in view")
		}
		if _, ok := view.FindGenotypeMarker(ids.marker); !ok {
			return fmt.Errorf("expected marker in view")
		}
		if len(view.ListLines()) != 1 || len(view.ListStrains()) != 1 {
			return fmt.Errorf("expected line and strain lists")
		}
		if len(view.ListGenotypeMarkers()) != 1 {
			return fmt.Errorf("expected marker list length 1")
		}
		return nil
	}); err != nil {
		t.Fatalf("view check: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateLine(ids.line, func(l *domain.Line) error {
			l.GenotypeMarkerIDs = []string{ids.marker, ids.marker}
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateStrain(ids.strain, func(s *domain.Strain) error {
			gen := "F1"
			s.Generation = &gen
			s.GenotypeMarkerIDs = []string{ids.marker, "missing"}
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateGenotypeMarker(ids.marker, func(m *domain.GenotypeMarker) error {
			m.Interpretation = "updated"
			m.Alleles = append(m.Alleles, "G")
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateStrain(ids.strain, func(s *domain.Strain) error {
			s.LineID = "missing-line"
			return nil
		}); err == nil {
			return fmt.Errorf("expected strain update to fail with missing line")
		}
		if err := tx.DeleteGenotypeMarker(ids.marker); err == nil {
			return fmt.Errorf("expected marker delete to fail while referenced")
		}
		if err := tx.DeleteLine(ids.line); err == nil {
			return fmt.Errorf("expected line delete to fail while strain exists")
		}
		if err := tx.DeleteStrain(ids.strain); err == nil {
			return fmt.Errorf("expected strain delete to fail while organism references it")
		}
		if _, err := tx.UpdateOrganism(ids.organism, func(o *domain.Organism) error {
			o.LineID = nil
			o.StrainID = nil
			return nil
		}); err != nil {
			return err
		}
		if err := tx.DeleteBreedingUnit(ids.breeding); err != nil {
			return fmt.Errorf("delete breeding: %w", err)
		}
		if err := tx.DeleteStrain(ids.strain); err != nil {
			return fmt.Errorf("delete strain: %w", err)
		}
		if err := tx.DeleteLine(ids.line); err != nil {
			return fmt.Errorf("delete line: %w", err)
		}
		if err := tx.DeleteGenotypeMarker(ids.marker); err != nil {
			return fmt.Errorf("delete marker: %w", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("mutation transaction: %v", err)
	}

	if len(store.ListLines()) != 0 || len(store.ListStrains()) != 0 || len(store.ListGenotypeMarkers()) != 0 {
		t.Fatalf("expected cleanup after deletions")
	}
}

func TestMemoryImportStateLineStrainNormalization(t *testing.T) {
	store := memory.NewStore(nil)
	lineID := "line-import"
	strainID := "strain-import"
	markerID := "marker-import"
	orgStrain := strainID
	orgLine := lineID
	breedingLine := lineID
	breedingTargetStrain := "missing-strain"

	snapshot := memory.Snapshot{
		Lines: map[string]domain.Line{
			lineID: {
				Base:              domain.Base{ID: lineID},
				Code:              "L-IMP",
				Name:              "Imported Line",
				Origin:            "field",
				GenotypeMarkerIDs: []string{markerID, markerID, "missing-marker"},
			},
		},
		Strains: map[string]domain.Strain{
			strainID: {
				Base:              domain.Base{ID: strainID},
				Code:              "S-IMP",
				Name:              "Imported Strain",
				LineID:            "missing-line",
				GenotypeMarkerIDs: []string{"missing-marker"},
			},
		},
		Markers: map[string]domain.GenotypeMarker{
			markerID: {
				Base:        domain.Base{ID: markerID},
				Name:        "Marker",
				Locus:       "locus",
				Alleles:     []string{"A", "A"},
				AssayMethod: "PCR",
				Version:     "v1",
			},
		},
		Organisms: map[string]domain.Organism{
			"org-import": {Base: domain.Base{ID: "org-import"}, Name: "Org", Species: "Spec", LineID: &orgLine, StrainID: &orgStrain},
		},
		Breeding: map[string]domain.BreedingUnit{
			"breed-import": {Base: domain.Base{ID: "breed-import"}, Name: "Breed", LineID: &breedingLine, TargetStrainID: &breedingTargetStrain},
		},
	}

	store.ImportState(snapshot)
	exported := store.ExportState()

	line, ok := exported.Lines[lineID]
	if !ok {
		t.Fatalf("expected line to survive import")
	}
	if len(line.GenotypeMarkerIDs) != 1 {
		t.Fatalf("expected line markers deduped and filtered, got %+v", line.GenotypeMarkerIDs)
	}
	if _, ok := exported.Strains[strainID]; ok {
		t.Fatalf("expected strain dropped due to missing line reference")
	}
	org := exported.Organisms["org-import"]
	if org.StrainID != nil {
		t.Fatalf("expected organism strain cleared during import")
	}
	breed := exported.Breeding["breed-import"]
	if breed.TargetStrainID != nil {
		t.Fatalf("expected breeding target strain cleared")
	}
}
