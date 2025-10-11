package memory

import (
	"colonycore/pkg/domain"
	"context"
	"testing"
	"time"
)

// TestSnapshotAllEntities exercises snapshotFromMemoryState and memoryStateFromSnapshot with
// at least one entry in every entity collection to raise coverage on cloning loops.
func TestSnapshotAllEntities(t *testing.T) {
	store := NewStore(domain.NewRulesEngine())
	ctx := context.Background()
	var housing domain.HousingUnit
	var organism domain.Organism
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		// Create project
		if _, err := tx.CreateProject(domain.Project{Code: "P1", Title: "Project"}); err != nil {
			return err
		}
		// Create housing
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "H1", Facility: "Lab", Capacity: 2})
		if err != nil {
			return err
		}
		housing = h
		// Create protocol
		p, err := tx.CreateProtocol(domain.Protocol{Code: "PR", Title: "Proto", MaxSubjects: 5})
		if err != nil {
			return err
		}
		_ = p // protocol used implicitly via references
		// Create cohort
		if _, err := tx.CreateCohort(domain.Cohort{Name: "C1"}); err != nil {
			return err
		}
		// Create organism with attributes
		o, err := tx.CreateOrganism(domain.Organism{Name: "Spec", Species: "Frog", Attributes: map[string]any{"color": "green"}})
		if err != nil {
			return err
		}
		organism = o
		// Create breeding unit referencing organism
		if _, err := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "Pair", FemaleIDs: []string{o.ID}, MaleIDs: []string{"M"}, HousingID: &h.ID, ProtocolID: &p.ID}); err != nil {
			return err
		}
		// Create procedure referencing organism
		if _, err := tx.CreateProcedure(domain.Procedure{Name: "Check", Status: "scheduled", ScheduledAt: time.Now().UTC(), ProtocolID: p.ID, OrganismIDs: []string{o.ID}}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	snap := store.ExportState()
	if len(snap.Organisms) == 0 || len(snap.Cohorts) == 0 || len(snap.Housing) == 0 || len(snap.Breeding) == 0 || len(snap.Procedures) == 0 || len(snap.Protocols) == 0 || len(snap.Projects) == 0 {
		t.Fatalf("expected populated snapshot: %+v", snap)
	}
	// Clear then re-import to exercise memoryStateFromSnapshot cloning for all maps.
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snap)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored organism")
	}
	// Update housing unit success branch (mutator returns nil and capacity positive)
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 3
			return nil
		})
		return err
	}); err != nil {
		t.Fatalf("update housing success: %v", err)
	}
	// Ensure organism attributes remain isolated (deep copy validated indirectly by modifying snapshot copy)
	snapOrg := snap.Organisms[organism.ID]
	snapOrg.Attributes["color"] = "blue"
	if store.ListOrganisms()[0].Attributes["color"] != "green" {
		t.Fatalf("expected deep copy isolation")
	}
}
