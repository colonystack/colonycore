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

func TestMigrateSnapshotDropsInvalidEntities(t *testing.T) {
	orgID := "org-1"
	facilityID := "fac-1"
	validProtocol := Protocol{Protocol: entitymodel.Protocol{ID: "prot-ok", Status: domain.ProtocolStatusDraft}}
	validProcedure := Procedure{Procedure: entitymodel.Procedure{ID: "proc-ok", Status: domain.ProcedureStatusScheduled, ProtocolID: validProtocol.ID}}

	snapshot := Snapshot{
		Facilities: map[string]Facility{
			facilityID: {Facility: entitymodel.Facility{ID: facilityID}},
		},
		Organisms: map[string]Organism{
			orgID: {Organism: entitymodel.Organism{ID: orgID, Name: "Org", Species: "species"}},
		},
		Protocols: map[string]Protocol{
			validProtocol.ID: validProtocol,
			"prot-bad":       {Protocol: entitymodel.Protocol{ID: "prot-bad", Status: domain.ProtocolStatus("invalid")}},
		},
		Procedures: map[string]Procedure{
			validProcedure.ID: validProcedure,
			"proc-bad":        {Procedure: entitymodel.Procedure{ID: "proc-bad", Status: domain.ProcedureStatus("invalid")}},
		},
		Housing: map[string]HousingUnit{
			"house-bad": {HousingUnit: entitymodel.HousingUnit{ID: "house-bad", FacilityID: facilityID, Capacity: 1, Environment: "invalid"}},
		},
		Treatments: map[string]Treatment{
			"treat-bad": {Treatment: entitymodel.Treatment{ID: "treat-bad", Name: "Treat", Status: domain.TreatmentStatus("invalid"), ProcedureID: validProcedure.ID}},
		},
		Samples: map[string]Sample{
			"sample-bad": {Sample: entitymodel.Sample{ID: "sample-bad", FacilityID: facilityID, Status: domain.SampleStatus("invalid"), OrganismID: &orgID}},
		},
		Permits: map[string]Permit{
			"permit-bad": {Permit: entitymodel.Permit{ID: "permit-bad", Status: domain.PermitStatus("invalid"), FacilityIDs: []string{facilityID}, ProtocolIDs: []string{validProtocol.ID}}},
		},
	}

	migrated := migrateSnapshot(snapshot)

	if _, ok := migrated.Protocols["prot-bad"]; ok {
		t.Fatalf("expected invalid protocol to be dropped")
	}
	if _, ok := migrated.Procedures["proc-bad"]; ok {
		t.Fatalf("expected invalid procedure to be dropped")
	}
	if _, ok := migrated.Housing["house-bad"]; ok {
		t.Fatalf("expected invalid housing to be dropped")
	}
	if _, ok := migrated.Treatments["treat-bad"]; ok {
		t.Fatalf("expected invalid treatment to be dropped")
	}
	if _, ok := migrated.Samples["sample-bad"]; ok {
		t.Fatalf("expected invalid sample to be dropped")
	}
	if _, ok := migrated.Permits["permit-bad"]; ok {
		t.Fatalf("expected invalid permit to be dropped")
	}
	if _, ok := migrated.Protocols[validProtocol.ID]; !ok {
		t.Fatalf("expected valid protocol to be retained")
	}
	if _, ok := migrated.Procedures[validProcedure.ID]; !ok {
		t.Fatalf("expected valid procedure to be retained")
	}
}
