package core

import (
	"colonycore/internal/infra/persistence/memory"
	sqlite "colonycore/internal/infra/persistence/sqlite"
	"colonycore/pkg/domain"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEntityModelFixturesSatisfyRules(t *testing.T) {
	fixtureBytes := readFixture(t)

	var memSnapshot memory.Snapshot
	if err := json.Unmarshal(fixtureBytes, &memSnapshot); err != nil {
		t.Fatalf("unmarshal memory snapshot: %v", err)
	}

	memStore := NewMemoryStore(NewDefaultRulesEngine())
	memStore.ImportState(memSnapshot)
	memSnapshot = memStore.ExportState()

	memChanges := changesFromMemorySnapshot(memSnapshot)
	if len(memChanges) == 0 {
		t.Fatalf("expected fixture changes for memory store")
	}
	memView := captureView(t, memStore)
	memResult := evaluateFixture(t, memStore.RulesEngine(), memView, memChanges)
	if len(memResult.Violations) != 0 {
		t.Fatalf("memory fixture violations: %+v", memResult.Violations)
	}

	sqliteStore, err := NewSQLiteStore(filepath.Join(t.TempDir(), "fixtures.db"), NewDefaultRulesEngine())
	if err != nil {
		t.Skipf("sqlite not available: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.DB().Close() })

	var sqliteSnapshot sqlite.Snapshot
	if err := json.Unmarshal(fixtureBytes, &sqliteSnapshot); err != nil {
		t.Fatalf("unmarshal sqlite snapshot: %v", err)
	}
	sqliteStore.ImportState(sqliteSnapshot)
	sqliteSnapshot = sqliteStore.ExportState()

	sqliteChanges := changesFromSQLiteSnapshot(sqliteSnapshot)
	if len(sqliteChanges) == 0 {
		t.Fatalf("expected fixture changes for sqlite store")
	}
	sqliteView := captureView(t, sqliteStore)
	sqliteResult := evaluateFixture(t, sqliteStore.RulesEngine(), sqliteView, sqliteChanges)
	if len(sqliteResult.Violations) != 0 {
		t.Fatalf("sqlite fixture violations: %+v", sqliteResult.Violations)
	}
}

func readFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testutil", "fixtures", "entity-model", "snapshot.json")
	data, err := os.ReadFile(path) //nolint:gosec // path is repository-local fixture
	if err != nil {
		t.Fatalf("read fixtures from %s: %v", path, err)
	}
	return data
}

func captureView(t *testing.T, store interface {
	View(context.Context, func(domain.TransactionView) error) error
}) domain.TransactionView {
	t.Helper()
	var view domain.TransactionView
	if err := store.View(context.Background(), func(v domain.TransactionView) error {
		view = v
		return nil
	}); err != nil {
		t.Fatalf("capture view: %v", err)
	}
	return view
}

func evaluateFixture(t *testing.T, engine *domain.RulesEngine, view domain.RuleView, changes []domain.Change) domain.Result {
	t.Helper()
	res, err := engine.Evaluate(context.Background(), view, changes)
	if err != nil {
		t.Fatalf("evaluate rules: %v", err)
	}
	return res
}

func changesFromMemorySnapshot(snapshot memory.Snapshot) []domain.Change {
	var changes []domain.Change
	changes = appendChanges(changes, domain.EntityOrganism, snapshot.Organisms)
	changes = appendChanges(changes, domain.EntityCohort, snapshot.Cohorts)
	changes = appendChanges(changes, domain.EntityHousingUnit, snapshot.Housing)
	changes = appendChanges(changes, domain.EntityFacility, snapshot.Facilities)
	changes = appendChanges(changes, domain.EntityBreeding, snapshot.Breeding)
	changes = appendChanges(changes, domain.EntityLine, snapshot.Lines)
	changes = appendChanges(changes, domain.EntityStrain, snapshot.Strains)
	changes = appendChanges(changes, domain.EntityGenotypeMarker, snapshot.Markers)
	changes = appendChanges(changes, domain.EntityProcedure, snapshot.Procedures)
	changes = appendChanges(changes, domain.EntityTreatment, snapshot.Treatments)
	changes = appendChanges(changes, domain.EntityObservation, snapshot.Observations)
	changes = appendChanges(changes, domain.EntitySample, snapshot.Samples)
	changes = appendChanges(changes, domain.EntityProtocol, snapshot.Protocols)
	changes = appendChanges(changes, domain.EntityPermit, snapshot.Permits)
	changes = appendChanges(changes, domain.EntityProject, snapshot.Projects)
	changes = appendChanges(changes, domain.EntitySupplyItem, snapshot.Supplies)
	return changes
}

func changesFromSQLiteSnapshot(snapshot sqlite.Snapshot) []domain.Change {
	var changes []domain.Change
	changes = appendChanges(changes, domain.EntityOrganism, snapshot.Organisms)
	changes = appendChanges(changes, domain.EntityCohort, snapshot.Cohorts)
	changes = appendChanges(changes, domain.EntityHousingUnit, snapshot.Housing)
	changes = appendChanges(changes, domain.EntityFacility, snapshot.Facilities)
	changes = appendChanges(changes, domain.EntityBreeding, snapshot.Breeding)
	changes = appendChanges(changes, domain.EntityLine, snapshot.Lines)
	changes = appendChanges(changes, domain.EntityStrain, snapshot.Strains)
	changes = appendChanges(changes, domain.EntityGenotypeMarker, snapshot.Markers)
	changes = appendChanges(changes, domain.EntityProcedure, snapshot.Procedures)
	changes = appendChanges(changes, domain.EntityTreatment, snapshot.Treatments)
	changes = appendChanges(changes, domain.EntityObservation, snapshot.Observations)
	changes = appendChanges(changes, domain.EntitySample, snapshot.Samples)
	changes = appendChanges(changes, domain.EntityProtocol, snapshot.Protocols)
	changes = appendChanges(changes, domain.EntityPermit, snapshot.Permits)
	changes = appendChanges(changes, domain.EntityProject, snapshot.Projects)
	changes = appendChanges(changes, domain.EntitySupplyItem, snapshot.Supplies)
	return changes
}

func appendChanges[T any](changes []domain.Change, entity domain.EntityType, records map[string]T) []domain.Change {
	for _, record := range records {
		changes = append(changes, domain.Change{
			Entity: entity,
			Action: domain.ActionCreate,
			After:  record,
		})
	}
	return changes
}
