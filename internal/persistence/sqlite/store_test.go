package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStorePersistAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Name: "Persist"})
		return err
	}); err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload sqlite store: %v", err)
	}
	organisms := reloaded.ListOrganisms()
	if len(organisms) != 1 || organisms[0].Name != "Persist" {
		t.Fatalf("expected persisted organism, got %+v", organisms)
	}
	if err := reloaded.View(context.Background(), func(view domain.TransactionView) error {
		if len(view.ListOrganisms()) != 1 {
			return fmt.Errorf("expected view to list organism")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
	if reloaded.Path() != path {
		t.Fatalf("expected path %s, got %s", path, reloaded.Path())
	}
	if reloaded.DB() == nil {
		t.Fatalf("expected db handle")
	}
}

func TestSQLiteStorePersistError(t *testing.T) {
	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	_ = store.DB().Close()
	if _, err := store.RunInTransaction(context.Background(), func(_ domain.Transaction) error { return nil }); err == nil {
		t.Fatalf("expected persist error after closing db")
	}
}

func TestSQLiteStorePersistAllBuckets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "full.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		project, err := tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project"})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Habitat", Facility: "Main", Capacity: 8, Environment: "Moist"})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PROTO", Title: "Protocol", Description: "desc", MaxSubjects: 5})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Name: "Cohort", ProjectID: &project.ID, HousingID: &housing.ID, ProtocolID: &protocol.ID})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{
			Name:       "Specimen",
			Species:    "Lithobates",
			Stage:      domain.StageJuvenile,
			CohortID:   &cohort.ID,
			HousingID:  &housing.ID,
			ProtocolID: &protocol.ID,
			ProjectID:  &project.ID,
			Attributes: map[string]any{"color": "green"},
		})
		if err != nil {
			return err
		}
		if _, err := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "Pair", HousingID: &housing.ID, ProtocolID: &protocol.ID, FemaleIDs: []string{organism.ID}, MaleIDs: []string{"M"}}); err != nil {
			return err
		}
		if _, err := tx.CreateProcedure(domain.Procedure{Name: "Checkup", Status: "scheduled", ScheduledAt: now, ProtocolID: protocol.ID, CohortID: &cohort.ID, OrganismIDs: []string{organism.ID}}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	if err := store.DB().Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload sqlite store: %v", err)
	}
	defer func() { _ = reloaded.DB().Close() }()

	if got := reloaded.ListOrganisms(); len(got) != 1 {
		t.Fatalf("expected one organism, got %d", len(got))
	}
	if got := reloaded.ListCohorts(); len(got) != 1 {
		t.Fatalf("expected one cohort, got %d", len(got))
	}
	if got := reloaded.ListHousingUnits(); len(got) != 1 {
		t.Fatalf("expected one housing unit, got %d", len(got))
	}
	if got := reloaded.ListBreedingUnits(); len(got) != 1 {
		t.Fatalf("expected one breeding unit, got %d", len(got))
	}
	if got := reloaded.ListProcedures(); len(got) != 1 {
		t.Fatalf("expected one procedure, got %d", len(got))
	}
	if got := reloaded.ListProtocols(); len(got) != 1 {
		t.Fatalf("expected one protocol, got %d", len(got))
	}
	if got := reloaded.ListProjects(); len(got) != 1 {
		t.Fatalf("expected one project, got %d", len(got))
	}
	if err := reloaded.View(ctx, func(view domain.TransactionView) error {
		if len(view.ListOrganisms()) != 1 {
			return fmt.Errorf("expected view to list organism")
		}
		if _, ok := view.FindHousingUnit(reloaded.ListHousingUnits()[0].ID); !ok {
			return fmt.Errorf("expected view to find housing")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestSQLiteStoreLoadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	_, _ = store.DB().Exec(`INSERT INTO state(bucket, payload) VALUES('organisms', 'not-json')`)
	_ = store.DB().Close()
	if _, err := NewStore(path, domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected load error for invalid payload")
	}
}

func TestSQLiteStoreDefaultPath(t *testing.T) {
	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if store.Path() == "" {
		t.Fatalf("expected default path")
	}
}
