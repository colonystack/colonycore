package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

func TestMustApplyNoPanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("unexpected panic: %v", err)
		}
	}()
	mustApply("noop", nil)
}

func TestMustApplyPanicsOnError(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatalf("expected panic when err is non-nil")
		}
	}()
	mustApply("fail", fmt.Errorf("boom"))
}

func TestStoreGettersMissing(t *testing.T) {
	store := NewStore(nil)
	if _, ok := store.GetOrganism("missing"); ok {
		t.Fatalf("expected missing organism to return false")
	}
	if _, ok := store.GetHousingUnit("missing"); ok {
		t.Fatalf("expected missing housing unit to return false")
	}
	if _, ok := store.GetFacility("missing"); ok {
		t.Fatalf("expected missing facility to return false")
	}
	if _, ok := store.GetPermit("missing"); ok {
		t.Fatalf("expected missing permit to return false")
	}
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if err := tx.DeleteProtocol("missing"); err == nil {
			t.Fatalf("expected delete protocol to error on missing id")
		}
		if err := tx.DeleteProject("missing"); err == nil {
			t.Fatalf("expected delete project to error on missing id")
		}
		if err := tx.DeleteSupplyItem("missing"); err == nil {
			t.Fatalf("expected delete supply to error on missing id")
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected transaction error: %v", err)
	}
}

func TestStoreDeleteGuards(t *testing.T) {
	store := NewStore(nil)
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Test Facility"})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PR", Title: "Protocol", MaxSubjects: 1})
		if err != nil {
			return err
		}
		validFrom := time.Now().UTC()
		validUntil := validFrom.Add(time.Hour)
		_, err = tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-1",
			Authority:         "Gov",
			Status:            domain.PermitStatusApproved,
			ValidFrom:         validFrom,
			ValidUntil:        validUntil,
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		})
		if err != nil {
			return err
		}
		if err := tx.DeleteProtocol(protocol.ID); err == nil {
			t.Fatalf("expected protocol delete to fail when referenced")
		}

		project, err := tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}
		_, err = tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Supply",
			QuantityOnHand: 1,
			Unit:           "unit",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
			ReorderLevel:   1,
		})
		if err != nil {
			return err
		}
		if err := tx.DeleteProject(project.ID); err == nil {
			t.Fatalf("expected project delete to fail when referenced")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}
