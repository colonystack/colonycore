package memory

import (
	"colonycore/pkg/domain"
	"context"
	"testing"
)

func TestDeleteLineGuardPaths(t *testing.T) {
	t.Parallel()

	t.Run("fails with strain then breeding then organism before success", func(t *testing.T) {
		store := NewStore(nil)
		ctx := context.Background()
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			line, err := tx.CreateLine(domain.Line{Code: "L", Name: "Line", Origin: "field"})
			if err != nil {
				return err
			}

			strain, err := tx.CreateStrain(domain.Strain{Code: "S", Name: "Strain", LineID: line.ID})
			if err != nil {
				return err
			}

			if err := tx.DeleteLine(line.ID); err == nil {
				t.Fatalf("expected delete line to fail due to strain reference")
			}
			if err := tx.DeleteStrain(strain.ID); err != nil {
				return err
			}

			breeding, err := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "B", Strategy: "pair", LineID: &line.ID})
			if err != nil {
				return err
			}
			if err := tx.DeleteLine(line.ID); err == nil {
				t.Fatalf("expected delete line to fail due to breeding line reference")
			}
			if err := tx.DeleteBreedingUnit(breeding.ID); err != nil {
				return err
			}

			targetLine := line.ID
			breeding, err = tx.CreateBreedingUnit(domain.BreedingUnit{Name: "B2", Strategy: "pair", TargetLineID: &targetLine})
			if err != nil {
				return err
			}
			if err := tx.DeleteLine(line.ID); err == nil {
				t.Fatalf("expected delete line to fail due to breeding target line reference")
			}
			if err := tx.DeleteBreedingUnit(breeding.ID); err != nil {
				return err
			}

			lineRef := line.ID
			org, err := tx.CreateOrganism(domain.Organism{Name: "Org", Species: "Spec", LineID: &lineRef})
			if err != nil {
				return err
			}
			if err := tx.DeleteLine(line.ID); err == nil {
				t.Fatalf("expected delete line to fail due to organism reference")
			}
			if err := tx.DeleteOrganism(org.ID); err != nil {
				return err
			}

			if err := tx.DeleteLine(line.ID); err != nil {
				t.Fatalf("expected final delete line to succeed: %v", err)
			}
			return nil
		}); err != nil {
			t.Fatalf("transaction: %v", err)
		}
	})
}

func TestDeleteStrainAndGenotypeMarkerGuards(t *testing.T) {
	t.Parallel()

	t.Run("strain guard branches", func(t *testing.T) {
		store := NewStore(nil)
		ctx := context.Background()
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			line, err := tx.CreateLine(domain.Line{Base: domain.Base{ID: "line-guard"}, Code: "L", Name: "Line", Origin: "field"})
			if err != nil {
				return err
			}
			strain, err := tx.CreateStrain(domain.Strain{Base: domain.Base{ID: "strain-guard"}, Code: "S", Name: "Strain", LineID: line.ID})
			if err != nil {
				return err
			}

			lineRef := line.ID
			strainRef := strain.ID
			org, err := tx.CreateOrganism(domain.Organism{Base: domain.Base{ID: "org-guard"}, Name: "Org", Species: "Spec", LineID: &lineRef, StrainID: &strainRef})
			if err != nil {
				return err
			}
			if err := tx.DeleteStrain(strain.ID); err == nil {
				t.Fatalf("expected delete strain to fail due to organism reference")
			}

			if _, err := tx.UpdateOrganism(org.ID, func(o *domain.Organism) error {
				o.LineID = nil
				o.StrainID = nil
				return nil
			}); err != nil {
				return err
			}

			targetStrain := strain.ID
			breedingOne, err := tx.CreateBreedingUnit(domain.BreedingUnit{Base: domain.Base{ID: "breed-1"}, Name: "B1", Strategy: "pair", StrainID: &strainRef})
			if err != nil {
				return err
			}
			if err := tx.DeleteStrain(strain.ID); err == nil {
				t.Fatalf("expected delete strain to fail due to breeding strain reference")
			}
			breedingTwo, err := tx.CreateBreedingUnit(domain.BreedingUnit{Base: domain.Base{ID: "breed-2"}, Name: "B2", Strategy: "pair", TargetStrainID: &targetStrain})
			if err != nil {
				return err
			}
			if err := tx.DeleteStrain(strain.ID); err == nil {
				t.Fatalf("expected delete strain to fail due to breeding target strain reference")
			}
			for _, id := range []string{breedingOne.ID, breedingTwo.ID} {
				if err := tx.DeleteBreedingUnit(id); err != nil {
					return err
				}
			}
			if err := tx.DeleteStrain(strain.ID); err != nil {
				t.Fatalf("expected final delete strain to succeed: %v", err)
			}
			return nil
		}); err != nil {
			t.Fatalf("transaction: %v", err)
		}
	})

	t.Run("genotype marker guard branches", func(t *testing.T) {
		store := NewStore(nil)
		ctx := context.Background()
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{Base: domain.Base{ID: "marker-guard"}, Name: "M", Locus: "loc", Alleles: []string{"A"}, AssayMethod: "PCR", Interpretation: "ctrl", Version: "v1"})
			if err != nil {
				return err
			}
			line, err := tx.CreateLine(domain.Line{Base: domain.Base{ID: "line-guard"}, Code: "L", Name: "Line", Origin: "field", GenotypeMarkerIDs: []string{marker.ID}})
			if err != nil {
				return err
			}
			if err := tx.DeleteGenotypeMarker(marker.ID); err == nil {
				t.Fatalf("expected delete marker to fail due to line reference")
			}
			if err := tx.DeleteLine(line.ID); err != nil {
				return err
			}

			freeLine, err := tx.CreateLine(domain.Line{Base: domain.Base{ID: "line-free"}, Code: "L2", Name: "Line2", Origin: "field"})
			if err != nil {
				return err
			}
			strain, err := tx.CreateStrain(domain.Strain{Base: domain.Base{ID: "strain-free"}, Code: "S", Name: "Strain", LineID: freeLine.ID, GenotypeMarkerIDs: []string{marker.ID}})
			if err != nil {
				return err
			}
			if err := tx.DeleteGenotypeMarker(marker.ID); err == nil {
				t.Fatalf("expected delete marker to fail due to strain reference")
			}
			if err := tx.DeleteStrain(strain.ID); err != nil {
				return err
			}
			if err := tx.DeleteGenotypeMarker(marker.ID); err != nil {
				t.Fatalf("expected marker delete to succeed: %v", err)
			}
			return nil
		}); err != nil {
			t.Fatalf("transaction: %v", err)
		}
	})
}
