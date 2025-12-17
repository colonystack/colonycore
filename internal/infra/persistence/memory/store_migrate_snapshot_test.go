package memory

import (
	entitymodel "colonycore/pkg/domain/entitymodel"
	"testing"
)

func TestMigrateSnapshotInitialisesAndFilters(t *testing.T) {
	snapshot := Snapshot{
		Samples: map[string]Sample{
			"sample-missing": {Sample: entitymodel.Sample{
				ID:         "sample-missing",
				FacilityID: "missing-facility",
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
}
