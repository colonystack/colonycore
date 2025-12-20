package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"testing"
)

func TestMigrateSnapshotInitialisesAndFilters(t *testing.T) {
	snapshot := Snapshot{
		Facilities: map[string]Facility{
			"fac-1": {Facility: entitymodel.Facility{ID: "fac-1"}},
		},
		Organisms: map[string]Organism{
			"org-1": {Organism: entitymodel.Organism{
				ID:       "org-1",
				Name:     "Org",
				Species:  "species",
				Stage:    domain.StageAdult,
				LineID:   strPtr("missing-line"),
				StrainID: strPtr("missing-strain"),
			}},
		},
		Breeding: map[string]BreedingUnit{
			"breed-1": {BreedingUnit: entitymodel.BreedingUnit{
				ID:           "breed-1",
				Name:         "B",
				Strategy:     "s",
				LineID:       strPtr("missing-line"),
				StrainID:     strPtr("missing-strain"),
				TargetLineID: strPtr("missing-line"),
			}},
		},
		Markers: map[string]GenotypeMarker{
			"marker-1": {GenotypeMarker: entitymodel.GenotypeMarker{
				ID:             "marker-1",
				Name:           "M",
				Locus:          "locus",
				Alleles:        []string{"A"},
				AssayMethod:    "PCR",
				Interpretation: "interp",
				Version:        "v1",
			}},
		},
		Samples: map[string]Sample{
			"sample-missing": {Sample: entitymodel.Sample{
				ID:         "sample-missing",
				FacilityID: "missing-facility",
			}},
			"sample-orphan": {Sample: entitymodel.Sample{
				ID:         "sample-orphan",
				FacilityID: "fac-1",
			}},
		},
		Treatments: map[string]Treatment{
			"treat-missing": {Treatment: entitymodel.Treatment{
				ID:          "treat-missing",
				ProcedureID: "missing-procedure",
			}},
		},
	}

	migrated := migrateSnapshot(snapshot)

	if migrated.Organisms == nil || migrated.Facilities == nil || migrated.Protocols == nil {
		t.Fatalf("expected migrateSnapshot to initialise nil maps")
	}
	if len(migrated.Samples) != 0 {
		t.Fatalf("expected samples with missing facilities to be dropped, got %d", len(migrated.Samples))
	}
	if len(migrated.Treatments) != 0 {
		t.Fatalf("expected treatments with missing procedures to be dropped, got %d", len(migrated.Treatments))
	}
	if migrated.Organisms["org-1"].LineID != nil || migrated.Organisms["org-1"].StrainID != nil {
		t.Fatalf("expected organism lineage references to be cleared")
	}
	if migrated.Breeding["breed-1"].LineID != nil || migrated.Breeding["breed-1"].StrainID != nil || migrated.Breeding["breed-1"].TargetLineID != nil {
		t.Fatalf("expected breeding lineage references to be cleared")
	}
	if len(migrated.Markers) != 1 {
		t.Fatalf("expected markers to be retained")
	}
}
