package integration

import (
	"context"
	"testing"
	"time"

	core "colonycore/internal/core"
	domain "colonycore/pkg/domain"
)

func strPtr(v string) *string {
	return &v
}

func TestIntegrationEntityRelationships(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	coreVariants := []struct {
		name string
		open func(t *testing.T) domain.PersistentStore
	}{
		{
			name: "memory-store",
			open: func(_ *testing.T) domain.PersistentStore {
				return core.NewMemoryStore(core.NewDefaultRulesEngine())
			},
		},
		{
			name: "sqlite-store",
			open: func(t *testing.T) domain.PersistentStore {
				path := t.TempDir() + "/relationships.db"
				store, err := core.NewSQLiteStore(path, core.NewDefaultRulesEngine())
				if err != nil {
					t.Fatalf("new sqlite store: %v", err)
				}
				return store
			},
		},
	}

	for _, variant := range coreVariants {
		t.Run(variant.name, func(t *testing.T) {
			store := variant.open(t)
			svc := core.NewService(store)

			facility, res, err := svc.CreateFacility(ctx, domain.Facility{
				Name:         "Facility A",
				Zone:         "zone-a",
				AccessPolicy: "standard",
			})
			if err != nil {
				t.Fatalf("create facility: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected facility violations: %+v", res.Violations)
			}

			if _, _, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{
				Name:       "Invalid Housing",
				FacilityID: "missing-facility",
				Capacity:   1,
			}); err == nil {
				t.Fatalf("expected housing creation to fail for missing facility")
			}

			housing, res, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{
				Name:        "Housing-1",
				FacilityID:  facility.ID,
				Capacity:    2,
				Environment: "dry",
			})
			if err != nil {
				t.Fatalf("create housing: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected housing violations: %+v", res.Violations)
			}

			if _, err := svc.DeleteFacility(ctx, facility.ID); err == nil {
				t.Fatalf("expected facility delete to fail while housing exists")
			}

			project, res, err := svc.CreateProject(ctx, domain.Project{
				Code:        "PROJ-1",
				Title:       "Project 1",
				Description: strPtr("linked to facility"),
				FacilityIDs: []string{facility.ID},
			})
			if err != nil {
				t.Fatalf("create project: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected project violations: %+v", res.Violations)
			}

			protocol, res, err := svc.CreateProtocol(ctx, domain.Protocol{
				Code:        "PR-1",
				Title:       "Protocol 1",
				Description: strPtr("baseline protocol"),
				MaxSubjects: 10,
				Status:      "active",
			})
			if err != nil {
				t.Fatalf("create protocol: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected protocol violations: %+v", res.Violations)
			}

			organism, res, err := svc.CreateOrganism(ctx, domain.Organism{
				Name:    "Specimen-1",
				Species: "Testus example",
			})
			if err != nil {
				t.Fatalf("create organism: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected organism violations: %+v", res.Violations)
			}

			procedure, res, err := svc.CreateProcedure(ctx, domain.Procedure{
				Name:        "Procedure-1",
				Status:      domain.ProcedureStatusScheduled,
				ScheduledAt: now,
				ProtocolID:  protocol.ID,
				OrganismIDs: []string{organism.ID},
			})
			if err != nil {
				t.Fatalf("create procedure: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected procedure violations: %+v", res.Violations)
			}

			if _, _, err := svc.CreateTreatment(ctx, domain.Treatment{
				Name:        "InvalidTreatment",
				ProcedureID: "missing-procedure",
			}); err == nil {
				t.Fatalf("expected treatment creation to fail for missing procedure")
			}

			treatment, res, err := svc.CreateTreatment(ctx, domain.Treatment{
				Name:        "Treatment-1",
				Status:      domain.TreatmentStatusPlanned,
				ProcedureID: procedure.ID,
				OrganismIDs: []string{organism.ID, organism.ID},
				DosagePlan:  "Plan A",
			})
			if err != nil {
				t.Fatalf("create treatment: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected treatment violations: %+v", res.Violations)
			}

			if _, _, err := svc.CreateObservation(ctx, domain.Observation{
				Observer:   "Tech",
				RecordedAt: now,
				Notes:      strPtr("missing context"),
			}); err == nil {
				t.Fatalf("expected observation creation to fail without context")
			}

			procedureID := procedure.ID
			observationInput := domain.Observation{
				ProcedureID: &procedureID,
				Observer:    "Tech",
				RecordedAt:  now,
				Notes:       strPtr("baseline observation"),
			}
			if err := observationInput.ApplyObservationData(map[string]any{"weight": 12.5}); err != nil {
				t.Fatalf("apply observation data: %v", err)
			}
			observation, res, err := svc.CreateObservation(ctx, observationInput)
			if err != nil {
				t.Fatalf("create observation: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected observation violations: %+v", res.Violations)
			}

			if _, err := svc.DeleteProcedure(ctx, procedure.ID); err == nil {
				t.Fatalf("expected procedure delete to fail while attachments exist")
			}

			organismID := organism.ID
			if _, _, err := svc.CreateSample(ctx, domain.Sample{
				Identifier:      "S-Invalid",
				SourceType:      "blood",
				FacilityID:      "missing-facility",
				OrganismID:      &organismID,
				CollectedAt:     now,
				Status:          domain.SampleStatusStored,
				StorageLocation: "freezer-a",
			}); err == nil {
				t.Fatalf("expected sample creation to fail for missing facility")
			}

			if _, _, err := svc.CreateSample(ctx, domain.Sample{
				Identifier:      "S-NoLink",
				SourceType:      "blood",
				FacilityID:      facility.ID,
				CollectedAt:     now,
				Status:          domain.SampleStatusStored,
				StorageLocation: "freezer-a",
			}); err == nil {
				t.Fatalf("expected sample creation to fail without organism or cohort")
			}

			validSample, res, err := svc.CreateSample(ctx, domain.Sample{
				Identifier:      "S-1",
				SourceType:      "blood",
				FacilityID:      facility.ID,
				OrganismID:      &organismID,
				CollectedAt:     now,
				Status:          domain.SampleStatusStored,
				StorageLocation: "freezer-a",
			})
			if err != nil {
				t.Fatalf("create sample: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected sample violations: %+v", res.Violations)
			}

			if _, _, err := svc.CreatePermit(ctx, domain.Permit{
				PermitNumber: "PERM-ERR",
				Authority:    "Gov",
				Status:       domain.PermitStatusPending,
				ValidFrom:    now,
				ValidUntil:   now.AddDate(1, 0, 0),
				FacilityIDs:  []string{facility.ID},
				ProtocolIDs:  []string{"missing-protocol"},
			}); err == nil {
				t.Fatalf("expected permit creation to fail for missing protocol")
			}

			permit, res, err := svc.CreatePermit(ctx, domain.Permit{
				PermitNumber:      "PERM-1",
				Authority:         "Gov",
				Status:            domain.PermitStatusActive,
				ValidFrom:         now,
				ValidUntil:        now.AddDate(1, 0, 0),
				AllowedActivities: []string{"activity"},
				FacilityIDs:       []string{facility.ID, facility.ID},
				ProtocolIDs:       []string{protocol.ID},
			})
			if err != nil {
				t.Fatalf("create permit: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected permit violations: %+v", res.Violations)
			}

			if _, _, err := svc.CreateSupplyItem(ctx, domain.SupplyItem{
				SKU:            "SKU-ERR",
				Name:           "Gloves",
				Description:    strPtr("invalid project reference"),
				QuantityOnHand: 5,
				Unit:           "box",
				FacilityIDs:    []string{facility.ID},
				ProjectIDs:     []string{"missing-project"},
			}); err == nil {
				t.Fatalf("expected supply item creation to fail for missing project")
			}

			supply, res, err := svc.CreateSupplyItem(ctx, domain.SupplyItem{
				SKU:            "SKU-1",
				Name:           "Gloves",
				Description:    strPtr("valid supply"),
				QuantityOnHand: 25,
				Unit:           "box",
				FacilityIDs:    []string{facility.ID},
				ProjectIDs:     []string{project.ID},
				ReorderLevel:   5,
			})
			if err != nil {
				t.Fatalf("create supply: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected supply violations: %+v", res.Violations)
			}

			if _, err := svc.DeleteProject(ctx, project.ID); err == nil {
				t.Fatalf("expected project delete to fail while supply exists")
			}

			if res, err := svc.DeleteSupplyItem(ctx, supply.ID); err != nil {
				t.Fatalf("delete supply: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected supply delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteProject(ctx, project.ID); err != nil {
				t.Fatalf("delete project: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected project delete violations: %+v", res.Violations)
			}

			if _, err := svc.DeleteOrganism(ctx, organism.ID); err == nil {
				t.Fatalf("expected organism delete to fail while sample exists")
			}

			if res, err := svc.DeleteSample(ctx, validSample.ID); err != nil {
				t.Fatalf("delete sample: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected sample delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteOrganism(ctx, organism.ID); err != nil {
				t.Fatalf("delete organism: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected organism delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteObservation(ctx, observation.ID); err != nil {
				t.Fatalf("delete observation: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected observation delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteTreatment(ctx, treatment.ID); err != nil {
				t.Fatalf("delete treatment: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected treatment delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteProcedure(ctx, procedure.ID); err != nil {
				t.Fatalf("delete procedure: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected procedure delete violations: %+v", res.Violations)
			}

			if _, err := svc.DeleteProtocol(ctx, protocol.ID); err == nil {
				t.Fatalf("expected protocol delete to fail while permit exists")
			}

			if res, err := svc.DeletePermit(ctx, permit.ID); err != nil {
				t.Fatalf("delete permit: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected permit delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteProtocol(ctx, protocol.ID); err != nil {
				t.Fatalf("delete protocol: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected protocol delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteHousingUnit(ctx, housing.ID); err != nil {
				t.Fatalf("delete housing: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected housing delete violations: %+v", res.Violations)
			}

			if res, err := svc.DeleteFacility(ctx, facility.ID); err != nil {
				t.Fatalf("delete facility: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected facility delete violations: %+v", res.Violations)
			}
		})
	}
}
