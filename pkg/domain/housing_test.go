package domain

import (
	entitymodel "colonycore/pkg/domain/entitymodel"
	"encoding/json"
	"testing"
	"time"
)

func TestHousingUnitJSONRoundTrip(t *testing.T) {
	now := time.Date(2024, time.November, 5, 10, 0, 0, 0, time.UTC)

	original := HousingUnit{
		HousingUnit: entitymodel.HousingUnit{
			ID:          "housing-1",
			CreatedAt:   now,
			UpdatedAt:   now,
			Name:        "Habitat A",
			FacilityID:  "facility-1",
			Capacity:    3,
			State:       HousingStateActive,
			Environment: HousingEnvironmentHumid,
		},
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal housing unit: %v", err)
	}

	var roundTrip HousingUnit
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("unmarshal housing unit: %v", err)
	}

	if roundTrip.State != original.State {
		t.Fatalf("state mismatch: got %q, want %q", roundTrip.State, original.State)
	}
	if roundTrip.Environment != original.Environment {
		t.Fatalf("environment mismatch: got %q, want %q", roundTrip.Environment, original.Environment)
	}
}
