package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"testing"
)

func TestContainsStringCoverage(t *testing.T) {
	if !containsString([]string{"a", "b"}, "a") {
		t.Fatalf("expected containsString to find existing value")
	}
	if containsString([]string{"a", "b"}, "c") {
		t.Fatalf("expected containsString to report missing value")
	}
}

func TestNormalizeHousingUnitErrors(t *testing.T) {
	housing := HousingUnit{HousingUnit: entitymodel.HousingUnit{State: "invalid"}}
	if err := normalizeHousingUnit(&housing); err == nil {
		t.Fatalf("expected invalid state to error")
	}
	housing = HousingUnit{HousingUnit: entitymodel.HousingUnit{State: domain.HousingStateActive, Environment: "invalid"}}
	if err := normalizeHousingUnit(&housing); err == nil {
		t.Fatalf("expected invalid environment to error")
	}
}
