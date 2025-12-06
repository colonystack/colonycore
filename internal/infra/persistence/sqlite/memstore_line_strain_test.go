package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
)

func TestMemStoreLineStrainGenotypeLifecycle(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()

	var ids struct {
		line     string
		strain   string
		marker   string
		organism string
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{
			Name:           "Marker-A",
			Locus:          "locus-a",
			Alleles:        []string{"C", "C", "T"},
			AssayMethod:    "PCR",
			Interpretation: "control",
			Version:        "v1",
		})
		if err != nil {
			return err
		}
		ids.marker = marker.ID

		line, err := tx.CreateLine(domain.Line{
			Code:              "L-A",
			Name:              "Line A",
			Origin:            "field",
			GenotypeMarkerIDs: []string{marker.ID, marker.ID},
		})
		if err != nil {
			return err
		}
		ids.line = line.ID

		strain, err := tx.CreateStrain(domain.Strain{
			Code:              "S-A",
			Name:              "Strain A",
			LineID:            line.ID,
			GenotypeMarkerIDs: []string{marker.ID},
		})
		if err != nil {
			return err
		}
		ids.strain = strain.ID

		org, err := tx.CreateOrganism(domain.Organism{
			Name:     "Subject",
			Species:  "Specimen",
			LineID:   &line.ID,
			StrainID: &strain.ID,
		})
		if err != nil {
			return err
		}
		ids.organism = org.ID

		if _, err := tx.UpdateStrain(ids.strain, func(s *domain.Strain) error {
			s.LineID = ""
			return nil
		}); err == nil {
			return fmt.Errorf("expected strain update to fail without line")
		}
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	if len(store.ListLines()) != 1 || len(store.ListStrains()) != 1 || len(store.ListGenotypeMarkers()) != 1 {
		t.Fatalf("expected entities to be persisted")
	}
	if _, ok := store.GetLine(ids.line); !ok {
		t.Fatalf("expected line getter to succeed")
	}
	if _, ok := store.GetStrain(ids.strain); !ok {
		t.Fatalf("expected strain getter to succeed")
	}
	if _, ok := store.GetGenotypeMarker(ids.marker); !ok {
		t.Fatalf("expected marker getter to succeed")
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if err := tx.DeleteLine(ids.line); err == nil {
			return fmt.Errorf("expected line delete to fail while referenced")
		}
		if err := tx.DeleteGenotypeMarker(ids.marker); err == nil {
			return fmt.Errorf("expected marker delete to fail while referenced")
		}
		if _, err := tx.UpdateOrganism(ids.organism, func(o *domain.Organism) error {
			o.LineID = nil
			o.StrainID = nil
			return nil
		}); err != nil {
			return err
		}
		if err := tx.DeleteStrain(ids.strain); err != nil {
			return err
		}
		if err := tx.DeleteLine(ids.line); err != nil {
			return err
		}
		if err := tx.DeleteGenotypeMarker(ids.marker); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("cleanup transaction: %v", err)
	}

	if len(store.ListLines()) != 0 || len(store.ListStrains()) != 0 || len(store.ListGenotypeMarkers()) != 0 {
		t.Fatalf("expected entities removed after cleanup")
	}
}

func TestMemStoreLineStrainGenotypeUpdatesAndFinders(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()

	var ids struct {
		line   string
		strain string
		marker string
	}

	const updatedInterpretation = "reviewed"

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{
			Name:           "Marker-B",
			Locus:          "locus-b",
			Alleles:        []string{"G", "G", "T"},
			AssayMethod:    "PCR",
			Interpretation: "initial",
			Version:        "v1",
		})
		if err != nil {
			return err
		}
		ids.marker = marker.ID

		line, err := tx.CreateLine(domain.Line{
			Code:              "L-B",
			Name:              "Line B",
			Origin:            "field",
			GenotypeMarkerIDs: []string{marker.ID},
		})
		if err != nil {
			return err
		}
		ids.line = line.ID

		strain, err := tx.CreateStrain(domain.Strain{
			Code:              "S-B",
			Name:              "Strain B",
			LineID:            line.ID,
			GenotypeMarkerIDs: []string{marker.ID},
		})
		if err != nil {
			return err
		}
		ids.strain = strain.ID

		if _, ok := tx.FindLine(ids.line); !ok {
			return fmt.Errorf("expected tx.FindLine to succeed")
		}
		if _, ok := tx.FindStrain(ids.strain); !ok {
			return fmt.Errorf("expected tx.FindStrain to succeed")
		}
		if _, ok := tx.FindGenotypeMarker(ids.marker); !ok {
			return fmt.Errorf("expected tx.FindGenotypeMarker to succeed")
		}

		if _, err := tx.UpdateLine(ids.line, func(l *domain.Line) error {
			desc := "updated line"
			l.Description = &desc
			l.GenotypeMarkerIDs = []string{marker.ID, marker.ID, "ghost"}
			return nil
		}); err != nil {
			return err
		}

		if _, err := tx.UpdateStrain(ids.strain, func(s *domain.Strain) error {
			gen := "F2"
			s.Generation = &gen
			s.GenotypeMarkerIDs = []string{marker.ID, "ghost"}
			return nil
		}); err != nil {
			return err
		}

		if _, err := tx.UpdateGenotypeMarker(ids.marker, func(g *domain.GenotypeMarker) error {
			g.Alleles = []string{"C", "C", "T"}
			g.Interpretation = updatedInterpretation
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}

	if err := store.View(ctx, func(v domain.TransactionView) error {
		lines := v.ListLines()
		if len(lines) != 1 {
			return fmt.Errorf("expected 1 line, got %d", len(lines))
		}
		if len(lines[0].GenotypeMarkerIDs) != 1 || lines[0].GenotypeMarkerIDs[0] != ids.marker {
			return fmt.Errorf("expected line markers filtered, got %+v", lines[0].GenotypeMarkerIDs)
		}
		if _, ok := v.FindLine(ids.line); !ok {
			return fmt.Errorf("expected view FindLine to succeed")
		}

		strains := v.ListStrains()
		if len(strains) != 1 {
			return fmt.Errorf("expected 1 strain, got %d", len(strains))
		}
		if len(strains[0].GenotypeMarkerIDs) != 1 || strains[0].GenotypeMarkerIDs[0] != ids.marker {
			return fmt.Errorf("expected strain markers filtered, got %+v", strains[0].GenotypeMarkerIDs)
		}
		if _, ok := v.FindStrain(ids.strain); !ok {
			return fmt.Errorf("expected view FindStrain to succeed")
		}

		markers := v.ListGenotypeMarkers()
		if len(markers) != 1 {
			return fmt.Errorf("expected 1 marker, got %d", len(markers))
		}
		if markers[0].Interpretation != updatedInterpretation || len(markers[0].Alleles) != 2 {
			return fmt.Errorf("expected marker updates applied, got %+v", markers[0])
		}
		if _, ok := v.FindGenotypeMarker(ids.marker); !ok {
			return fmt.Errorf("expected view FindGenotypeMarker to succeed")
		}
		return nil
	}); err != nil {
		t.Fatalf("view assertions: %v", err)
	}
}

func TestMemStoreImportStateLineStrainNormalization(t *testing.T) {
	store := newMemStore(nil)
	lineID := "line-import"
	strainID := "strain-import"
	markerID := "marker-import"
	orgStrain := strainID
	orgLine := lineID
	breedingLine := lineID
	breedingTargetStrain := "missing-strain"

	snapshot := Snapshot{
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
