package sqlite

import (
	"fmt"
	"testing"

	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

func TestMemStoreDeleteProtocolBlockedByPermit(t *testing.T) {
	store := newMemStore(nil)

	runTx(t, store, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P-DEL", Title: "Protocol", MaxSubjects: 1}})
		if err != nil {
			return err
		}
		if _, err := tx.CreatePermit(domain.Permit{Permit: entitymodel.Permit{
			PermitNumber:      "PER-DEL",
			Authority:         "Gov",
			AllowedActivities: []string{"use"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		}}); err != nil {
			return err
		}
		if err := tx.DeleteProtocol(protocol.ID); err == nil {
			return fmt.Errorf("expected delete protocol to fail while permit exists")
		}
		return nil
	})
}

func TestMemStoreDeleteProjectBlockedBySupplyItem(t *testing.T) {
	store := newMemStore(nil)

	runTx(t, store, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{Code: "PRJ-DEL", Title: "Project", FacilityIDs: []string{facility.ID}}})
		if err != nil {
			return err
		}
		if _, err := tx.CreateSupplyItem(domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
			SKU:            "SKU-DEL",
			Name:           "Item",
			QuantityOnHand: 1,
			Unit:           "unit",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
		}}); err != nil {
			return err
		}
		if err := tx.DeleteProject(project.ID); err == nil {
			return fmt.Errorf("expected delete project to fail while supply item exists")
		}
		return nil
	})
}
