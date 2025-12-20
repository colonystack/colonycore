package core

import (
	"context"
	"testing"

	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

// TestNewServicePanicsOnNilStore validates constructor guard.
func TestNewServicePanicsOnNilStore(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on nil store")
		}
	}()
	_ = NewService(nil)
}

// TestAssignOrganismHousingCoversNotFoundAndSuccess exercises both error and success paths.
func TestAssignOrganismHousing(t *testing.T) {
	svc := NewInMemoryService(NewRulesEngine())
	ctx := context.Background()
	org, _, err := svc.CreateOrganism(ctx, domain.Organism{Organism: entitymodel.Organism{Name: "A", Species: "frog"}})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}
	// not found housing
	if _, _, err := svc.AssignOrganismHousing(ctx, org.ID, "missing"); err == nil {
		t.Fatalf("expected not found error")
	}
	// create housing then assign
	facility, _, err := svc.CreateFacility(ctx, domain.Facility{Facility: entitymodel.Facility{Name: "F"}})
	if err != nil {
		t.Fatalf("create facility: %v", err)
	}
	h, _, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "H", FacilityID: facility.ID, Capacity: 10}})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}
	updated, _, err := svc.AssignOrganismHousing(ctx, org.ID, h.ID)
	if err != nil || updated.HousingID == nil || *updated.HousingID != h.ID {
		t.Fatalf("assign housing failed: %+v %v", updated, err)
	}
}

// TestAssignOrganismProtocol covers protocol assignment success and not found.
func TestAssignOrganismProtocol(t *testing.T) {
	svc := NewInMemoryService(NewRulesEngine())
	ctx := context.Background()
	org, _, err := svc.CreateOrganism(ctx, domain.Organism{Organism: entitymodel.Organism{Name: "B", Species: "frog"}})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if _, _, err := svc.AssignOrganismProtocol(ctx, org.ID, "missing"); err == nil {
		t.Fatalf("expected not found error")
	}
	proto, _, err := svc.CreateProtocol(ctx, domain.Protocol{Protocol: entitymodel.Protocol{Code: "P", Title: "Prot", MaxSubjects: 5}})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	updated, _, err := svc.AssignOrganismProtocol(ctx, org.ID, proto.ID)
	if err != nil || updated.ProtocolID == nil || *updated.ProtocolID != proto.ID {
		t.Fatalf("assign protocol failed: %+v %v", updated, err)
	}
}
