package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// helper to run a transaction and fail fast
func runTx(t *testing.T, store *memStore, fn func(tx domain.Transaction) error) domain.Result {
	t.Helper()
	res, err := store.RunInTransaction(context.Background(), fn)
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	return res
}

func TestMemStore_FullCRUDAndErrors(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()

	var orgA, orgB domain.Organism
	var cohort domain.Cohort
	var housing domain.HousingUnit
	var breeding domain.BreedingUnit
	var protocol domain.Protocol
	var procedure domain.Procedure
	var project domain.Project

	// Create all entities
	runTx(t, store, func(tx domain.Transaction) error {
		o1, _ := tx.CreateOrganism(domain.Organism{Name: "Alpha", Species: "Frog", Attributes: map[string]any{"a": 1}})
		o2, _ := tx.CreateOrganism(domain.Organism{Name: "Beta", Species: "Frog"})
		orgA, orgB = o1, o2
		c, _ := tx.CreateCohort(domain.Cohort{Name: "C1", Purpose: "testing"})
		cohort = c
		h, _ := tx.CreateHousingUnit(domain.HousingUnit{Name: "H1", Capacity: 2, Environment: "dry", Facility: "F"})
		housing = h
		b, _ := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "B1", FemaleIDs: []string{o1.ID}, MaleIDs: []string{o2.ID}})
		breeding = b
		p, _ := tx.CreateProtocol(domain.Protocol{Code: "P1", Title: "Proto", MaxSubjects: 10})
		protocol = p
		pj, _ := tx.CreateProject(domain.Project{Code: "PRJ1", Title: "Proj"})
		project = pj
		pr, _ := tx.CreateProcedure(domain.Procedure{Name: "Proc", Status: "scheduled", ProtocolID: protocol.ID, OrganismIDs: []string{o1.ID}, ScheduledAt: time.Now().UTC()})
		procedure = pr
		return nil
	})

	// Direct getters to cover GetOrganism/GetHousingUnit and ListProtocols
	if got, ok := store.GetOrganism(orgA.ID); !ok || got.Name != "Alpha" {
		t.Fatalf("GetOrganism mismatch")
	}
	if got, ok := store.GetHousingUnit(housing.ID); !ok || got.Name != "H1" {
		t.Fatalf("GetHousingUnit mismatch")
	}
	if len(store.ListProtocols()) != 1 {
		t.Fatalf("expected 1 protocol")
	}

	// Duplicate create errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateOrganism(orgA); return e }); err == nil {
		t.Fatalf("expected duplicate organism error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateHousingUnit(housing); return e }); err == nil {
		t.Fatalf("expected duplicate housing error")
	}

	// Update & validation errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, e := tx.UpdateOrganism("missing", func(*domain.Organism) error { return nil })
		return e
	}); err == nil {
		t.Fatalf("expected missing organism update error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, e := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error { h.Capacity = 0; return nil })
		return e
	}); err == nil {
		t.Fatalf("expected capacity validation error")
	}

	// Successful updates
	runTx(t, store, func(tx domain.Transaction) error {
		_, _ = tx.UpdateOrganism(orgA.ID, func(o *domain.Organism) error { o.Name = "Alpha2"; o.Attributes["a"] = 2; return nil })
		_, _ = tx.UpdateCohort(cohort.ID, func(c *domain.Cohort) error { c.Purpose = "updated"; return nil })
		_, _ = tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error { h.Environment = "humid"; h.Capacity = 3; return nil })
		_, _ = tx.UpdateBreedingUnit(breeding.ID, func(b *domain.BreedingUnit) error { b.FemaleIDs = append(b.FemaleIDs, orgB.ID); return nil })
		_, _ = tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error { p.Description = "desc"; return nil })
		_, _ = tx.UpdateProcedure(procedure.ID, func(p *domain.Procedure) error { p.Status = "complete"; return nil })
		_, _ = tx.UpdateProject(project.ID, func(p *domain.Project) error { p.Description = "d"; return nil })
		return nil
	})

	// Snapshot export/import consistency
	snap := store.ExportState()
	if len(snap.Organisms) != 2 || len(snap.Housing) != 1 || len(snap.Breeding) != 1 || len(snap.Procedures) != 1 {
		t.Fatalf("unexpected snapshot counts: %+v", snap)
	}
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state after import empty")
	}
	store.ImportState(snap)
	if len(store.ListOrganisms()) != 2 {
		t.Fatalf("expected restore after import")
	}

	// Deletions and missing delete errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tx.DeleteOrganism("missing") }); err == nil {
		t.Fatalf("expected missing delete organism")
	}
	runTx(t, store, func(tx domain.Transaction) error {
		_ = tx.DeleteProcedure(procedure.ID)
		_ = tx.DeleteProtocol(protocol.ID)
		_ = tx.DeleteBreedingUnit(breeding.ID)
		_ = tx.DeleteHousingUnit(housing.ID)
		_ = tx.DeleteCohort(cohort.ID)
		_ = tx.DeleteOrganism(orgB.ID)
		return nil
	})
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected 1 organism left")
	}
	if len(store.ListCohorts()) != 0 {
		t.Fatalf("expected no cohorts left")
	}
	if len(store.ListHousingUnits()) != 0 {
		t.Fatalf("expected no housing units left")
	}
	if len(store.ListBreedingUnits()) != 0 {
		t.Fatalf("expected no breeding units left")
	}
	if len(store.ListProcedures()) != 0 {
		t.Fatalf("expected no procedures left")
	}
}

func TestMemStore_ViewAndFinds(t *testing.T) {
	store := newMemStore(nil)
	runTx(t, store, func(tx domain.Transaction) error { _, _ = tx.CreateOrganism(domain.Organism{Name: "X"}); return nil })
	if err := store.View(context.Background(), func(v domain.TransactionView) error {
		if len(v.ListOrganisms()) != 1 {
			return fmt.Errorf("expected 1 organism in view")
		}
		if _, ok := v.FindOrganism(store.ListOrganisms()[0].ID); !ok {
			return errors.New("organism not found in view")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestSQLiteStore_Persist_Reload_Full(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	engine := domain.NewRulesEngine()
	store, err := NewStore(path, engine)
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	runTx(t, store.memStore, func(tx domain.Transaction) error {
		_, _ = tx.CreateOrganism(domain.Organism{Name: "Persisted"})
		return nil
	})
	// persist via RunInTransaction on outer store for coverage
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateProject(domain.Project{Code: "C", Title: "T"})
		return e
	}); err != nil {
		t.Fatalf("outer tx: %v", err)
	}
	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.ListOrganisms()) == 0 {
		t.Fatalf("expected organisms after reload")
	}
	if reloaded.Path() != path {
		t.Fatalf("expected path match")
	}
	if reloaded.DB() == nil {
		t.Fatalf("expected db handle")
	}
}

func TestSQLiteStore_CorruptBucketHandling(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	// insert corrupt row directly
	if _, err := store.DB().Exec(`INSERT INTO state(bucket, payload) VALUES('organisms', 'not-json')`); err != nil {
		t.Fatalf("insert corrupt: %v", err)
	}
	if _, err := NewStore(path, domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected load error for corrupt json")
	}
}

func TestMemStore_RuleBlockingCoverage(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error { _, e := tx.CreateOrganism(domain.Organism{Name: "Block"}); return e }); err == nil {
		t.Fatalf("expected blocking violation")
	}
}

// NowFunc already exercised indirectly in other tests via transactions; explicit test removed to satisfy lint (unused param warning).

// Covers transaction.Snapshot, transaction.FindHousingUnit/FindProtocol and
// transactionView.ListHousingUnits/ListProtocols/FindHousingUnit which were previously 0%.
func TestMemStore_TransactionViewFinds(t *testing.T) {
	store := newMemStore(nil)
	var housing domain.HousingUnit
	var protocol domain.Protocol
	runTx(t, store, func(tx domain.Transaction) error {
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "T-H", Capacity: 1, Environment: "e", Facility: "F"})
		if err != nil {
			return err
		}
		housing = h
		p, err := tx.CreateProtocol(domain.Protocol{Code: "TP", Title: "T", MaxSubjects: 1})
		if err != nil {
			return err
		}
		protocol = p
		// Direct transaction find methods
		if _, ok := tx.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("tx.FindHousingUnit failed")
		}
		if _, ok := tx.FindProtocol(protocol.ID); !ok {
			t.Fatalf("tx.FindProtocol failed")
		}
		// Snapshot view usage
		v := tx.Snapshot()
		if len(v.ListHousingUnits()) != 1 {
			t.Fatalf("expected 1 housing in snapshot view")
		}
		if _, ok := v.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("view.FindHousingUnit failed")
		}
		if len(v.ListProtocols()) != 1 {
			t.Fatalf("expected 1 protocol in snapshot view")
		}
		// Negative lookups for coverage of false branches
		if _, ok := tx.FindHousingUnit("missing-h"); ok {
			t.Fatalf("expected missing housing unit")
		}
		if _, ok := tx.FindProtocol("missing-p"); ok {
			t.Fatalf("expected missing protocol")
		}
		if _, ok := v.FindHousingUnit("missing-h"); ok {
			t.Fatalf("expected missing housing in view")
		}
		if _, ok := v.FindOrganism("missing-o"); ok {
			t.Fatalf("expected missing organism in view")
		}
		return nil
	})
}

// Covers not found branches for updates/deletes across entities and duplicate creates.
func TestMemStore_ErrorBranchesAdditional(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	// Seed minimal entities for duplicates
	var prot domain.Protocol
	var proj domain.Project
	runTx(t, store, func(tx domain.Transaction) error {
		p, err := tx.CreateProtocol(domain.Protocol{Code: "DUP", Title: "Dup", MaxSubjects: 1})
		if err != nil {
			return err
		}
		prot = p
		pr, err := tx.CreateProject(domain.Project{Code: "DUPP", Title: "DupProj"})
		if err != nil {
			return err
		}
		proj = pr
		return nil
	})
	// Duplicate create errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateProtocol(prot); return e }); err == nil {
		t.Fatalf("expected duplicate protocol error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateProject(proj); return e }); err == nil {
		t.Fatalf("expected duplicate project error")
	}
	// Not found updates
	notFoundUpdates := []struct {
		name string
		fn   func(domain.Transaction) error
	}{
		{"cohort", func(tx domain.Transaction) error {
			_, e := tx.UpdateCohort("missing", func(*domain.Cohort) error { return nil })
			return e
		}},
		{"breeding", func(tx domain.Transaction) error {
			_, e := tx.UpdateBreedingUnit("missing", func(*domain.BreedingUnit) error { return nil })
			return e
		}},
		{"procedure", func(tx domain.Transaction) error {
			_, e := tx.UpdateProcedure("missing", func(*domain.Procedure) error { return nil })
			return e
		}},
		{"protocol", func(tx domain.Transaction) error {
			_, e := tx.UpdateProtocol("missing", func(*domain.Protocol) error { return nil })
			return e
		}},
		{"project", func(tx domain.Transaction) error {
			_, e := tx.UpdateProject("missing", func(*domain.Project) error { return nil })
			return e
		}},
	}
	for _, tc := range notFoundUpdates {
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tc.fn(tx) }); err == nil {
			t.Fatalf("expected not found update for %s", tc.name)
		}
	}
	// Not found deletes
	missingDeletes := []struct {
		name string
		fn   func(domain.Transaction) error
	}{
		{"cohort", func(tx domain.Transaction) error { return tx.DeleteCohort("missing") }},
		{"housing", func(tx domain.Transaction) error { return tx.DeleteHousingUnit("missing") }},
		{"breeding", func(tx domain.Transaction) error { return tx.DeleteBreedingUnit("missing") }},
		{"procedure", func(tx domain.Transaction) error { return tx.DeleteProcedure("missing") }},
		{"protocol", func(tx domain.Transaction) error { return tx.DeleteProtocol("missing") }},
		{"project", func(tx domain.Transaction) error { return tx.DeleteProject("missing") }},
	}
	for _, tc := range missingDeletes {
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tc.fn(tx) }); err == nil {
			t.Fatalf("expected not found delete for %s", tc.name)
		}
	}
}
