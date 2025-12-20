package postgres

import (
	"colonycore/internal/entitymodel/sqlbundle"
	"colonycore/internal/infra/persistence/memory"
	pgtu "colonycore/internal/infra/persistence/postgres/testutil"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func firstKey[T any](m map[string]T) (string, bool) {
	for id := range m {
		return id, true
	}
	return "", false
}

type recordingExec struct {
	Execs []string
}

func (r *recordingExec) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	r.Execs = append(r.Execs, query)
	return driver.RowsAffected(1), nil
}

func (r *recordingExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("QueryContext not implemented")
}

type failingExec struct{}

func (f failingExec) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("exec fail")
}

func (f failingExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("query fail")
}

type failAfterExec struct {
	failAt int
	calls  int
	execs  []string
}

func (f *failAfterExec) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	f.calls++
	f.execs = append(f.execs, query)
	if f.failAt > 0 && f.calls == f.failAt {
		return nil, fmt.Errorf("exec fail after %d", f.failAt)
	}
	return driver.RowsAffected(1), nil
}

func (f *failAfterExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("QueryContext not implemented")
}

func TestNewStoreAppliesDDLAndLoadsSnapshot(t *testing.T) {
	ctx := context.Background()
	db, conn := pgtu.NewStubDB()
	fixture := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, fixture); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if len(store.ListOrganisms()) == 0 {
		t.Fatalf("expected organisms loaded from normalized tables")
	}
	var sawDDL bool
	for _, stmt := range conn.Execs {
		if strings.Contains(strings.ToUpper(stmt), "CREATE TABLE") {
			sawDDL = true
			break
		}
	}
	if !sawDDL {
		t.Fatalf("expected entity-model DDL to be applied, got execs: %v", conn.Execs)
	}
}

func TestApplyEntityModelDDLUsesGeneratedPostgresBundle(t *testing.T) {
	ctx := context.Background()
	rec := &recordingExec{}

	ddl := sqlbundle.Postgres()
	if err := applyDDLStatements(ctx, rec, ddl); err != nil {
		t.Fatalf("applyDDLStatements: %v", err)
	}

	expected := sqlbundle.SplitStatements(ddl)
	if len(rec.Execs) != len(expected) {
		t.Fatalf("expected %d DDL statements, got %d", len(expected), len(rec.Execs))
	}
	for i, stmt := range expected {
		if strings.TrimSpace(rec.Execs[i]) != strings.TrimSpace(stmt) {
			t.Fatalf("statement %d mismatch:\nwant: %s\ngot:  %s", i, strings.TrimSpace(stmt), strings.TrimSpace(rec.Execs[i]))
		}
	}
}

func TestRunInTransactionPersistsState(t *testing.T) {
	var conn *pgtu.StubConn
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) {
		db, c := pgtu.NewStubDB()
		conn = c
		return db, nil
	})
	defer restore()

	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_, err = store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, err := tx.CreateFacility(domain.Facility{
			Facility: entitymodel.Facility{
				Code:         "FAC",
				Name:         "Facility",
				Zone:         "A",
				AccessPolicy: "all",
			},
		})
		return err
	})
	if err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}

	var found bool
	for table, rows := range conn.Tables {
		if table == "facilities" && len(rows) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected facility to persist in normalized tables")
	}
}

func TestRunInTransactionUpsertUpdatesExistingRow(t *testing.T) {
	ctx := context.Background()
	db, conn := pgtu.NewStubDB()
	fixture := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, fixture); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	var facilityID string
	for id := range fixture.Facilities {
		facilityID = id
		break
	}
	if facilityID == "" {
		t.Fatalf("fixture missing facilities")
	}

	_, err = store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateFacility(facilityID, func(f *domain.Facility) error {
			f.Name = "Updated Facility"
			f.Zone = "Z2"
			return nil
		})
		return err
	})
	if err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}

	rows := conn.Tables["facilities"]
	if got, want := len(rows), len(fixture.Facilities); got != want {
		t.Fatalf("expected %d facilities after upsert, got %d", want, got)
	}
	var matched bool
	for _, row := range rows {
		if row["id"] == facilityID {
			matched = true
			if row["name"] != "Updated Facility" || row["zone"] != "Z2" {
				t.Fatalf("expected updated facility values, got %+v", row)
			}
		}
	}
	if !matched {
		t.Fatalf("updated facility row not found")
	}
}

func TestRunInTransactionDeletesSupplyAndJoins(t *testing.T) {
	ctx := context.Background()
	db, conn := pgtu.NewStubDB()
	fixture := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, fixture); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	var supplyID string
	for id := range fixture.Supplies {
		supplyID = id
		break
	}
	if supplyID == "" {
		t.Fatalf("fixture missing supplies")
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteSupplyItem(supplyID)
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}

	for table, rows := range map[string][]map[string]any{
		"supply_items":               conn.Tables["supply_items"],
		"supply_items__facility_ids": conn.Tables["supply_items__facility_ids"],
		"projects__supply_item_ids":  conn.Tables["projects__supply_item_ids"],
	} {
		for _, row := range rows {
			if row["id"] == supplyID || row["supply_item_id"] == supplyID {
				t.Fatalf("found deleted supply id %s in table %s", supplyID, table)
			}
		}
	}
}

func TestSnapshotOrCacheFallbackOnLoadError(t *testing.T) {
	ctx := context.Background()
	db, conn := pgtu.NewStubDB()
	fixture := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, fixture); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	conn.FailTables = map[string]bool{"facilities": true}
	facilities := store.ListFacilities()
	if len(facilities) != len(fixture.Facilities) {
		t.Fatalf("expected facilities from cache on load failure, got %d (want %d)", len(facilities), len(fixture.Facilities))
	}
}

func TestStoreReadHelpers(t *testing.T) {
	ctx := context.Background()
	db, _ := pgtu.NewStubDB()
	fixture := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, fixture); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()

	store, err := NewStore("", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.ImportState(fixture)
	export := store.ExportState()
	if len(export.Facilities) == 0 {
		t.Fatalf("expected exported snapshot data")
	}
	if store.RulesEngine() == nil {
		t.Fatalf("expected rules engine")
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		view.ListFacilities()
		view.ListOrganisms()
		view.ListProtocols()
		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if id, ok := firstKey(fixture.Organisms); ok {
		if got, found := store.GetOrganism(id); !found || got.ID != id {
			t.Fatalf("GetOrganism failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Housing); ok {
		if got, found := store.GetHousingUnit(id); !found || got.ID != id {
			t.Fatalf("GetHousingUnit failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Facilities); ok {
		if got, found := store.GetFacility(id); !found || got.ID != id {
			t.Fatalf("GetFacility failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Lines); ok {
		if got, found := store.GetLine(id); !found || got.ID != id {
			t.Fatalf("GetLine failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Strains); ok {
		if got, found := store.GetStrain(id); !found || got.ID != id {
			t.Fatalf("GetStrain failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Markers); ok {
		if got, found := store.GetGenotypeMarker(id); !found || got.ID != id {
			t.Fatalf("GetGenotypeMarker failed for %s", id)
		}
	}
	if id, ok := firstKey(fixture.Permits); ok {
		if got, found := store.GetPermit(id); !found || got.ID != id {
			t.Fatalf("GetPermit failed for %s", id)
		}
	}

	store.ListHousingUnits()
	store.ListLines()
	store.ListStrains()
	store.ListGenotypeMarkers()
	store.ListCohorts()
	store.ListTreatments()
	store.ListObservations()
	store.ListSamples()
	store.ListProtocols()
	store.ListPermits()
	store.ListProjects()
	store.ListBreedingUnits()
	store.ListProcedures()
	store.ListSupplyItems()
}

func TestApplySnapshotDeltaDeletesEntities(t *testing.T) {
	ctx := context.Background()
	before := memory.Snapshot{
		Facilities:   map[string]domain.Facility{"fac": {Facility: entitymodel.Facility{ID: "fac"}}},
		Markers:      map[string]domain.GenotypeMarker{"gm": {GenotypeMarker: entitymodel.GenotypeMarker{ID: "gm"}}},
		Lines:        map[string]domain.Line{"line": {Line: entitymodel.Line{ID: "line"}}},
		Strains:      map[string]domain.Strain{"str": {Strain: entitymodel.Strain{ID: "str"}}},
		Housing:      map[string]domain.HousingUnit{"house": {HousingUnit: entitymodel.HousingUnit{ID: "house"}}},
		Protocols:    map[string]domain.Protocol{"proto": {Protocol: entitymodel.Protocol{ID: "proto"}}},
		Projects:     map[string]domain.Project{"proj": {Project: entitymodel.Project{ID: "proj"}}},
		Permits:      map[string]domain.Permit{"permit": {Permit: entitymodel.Permit{ID: "permit"}}},
		Cohorts:      map[string]domain.Cohort{"cohort": {Cohort: entitymodel.Cohort{ID: "cohort"}}},
		Breeding:     map[string]domain.BreedingUnit{"breed": {BreedingUnit: entitymodel.BreedingUnit{ID: "breed"}}},
		Organisms:    map[string]domain.Organism{"org": {Organism: entitymodel.Organism{ID: "org"}}},
		Procedures:   map[string]domain.Procedure{"proc": {Procedure: entitymodel.Procedure{ID: "proc"}}},
		Observations: map[string]domain.Observation{"obs": {Observation: entitymodel.Observation{ID: "obs"}}},
		Samples:      map[string]domain.Sample{"sample": {Sample: entitymodel.Sample{ID: "sample"}}},
		Supplies:     map[string]domain.SupplyItem{"sup": {SupplyItem: entitymodel.SupplyItem{ID: "sup"}}},
		Treatments:   map[string]domain.Treatment{"treat": {Treatment: entitymodel.Treatment{ID: "treat", ProcedureID: "proc"}}},
	}

	rec := &recordingExec{}
	if err := applySnapshotDelta(ctx, rec, before, memory.Snapshot{}); err != nil {
		t.Fatalf("applySnapshotDelta deletes: %v", err)
	}
	if len(rec.Execs) == 0 {
		t.Fatalf("expected delete statements to be issued")
	}
}

func TestDeleteHelpersExecute(t *testing.T) {
	ctx := context.Background()
	rec := &recordingExec{}
	check := func(label string, fn func() error) {
		t.Helper()
		before := len(rec.Execs)
		if err := fn(); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if len(rec.Execs) == before {
			t.Fatalf("expected exec for %s", label)
		}
	}

	check("facilities", func() error { return deleteFacilities(ctx, rec, []string{"fac"}) })
	check("markers", func() error { return deleteGenotypeMarkers(ctx, rec, []string{"gm"}) })
	check("lines", func() error { return deleteLines(ctx, rec, []string{"line"}) })
	check("strains", func() error { return deleteStrains(ctx, rec, []string{"str"}) })
	check("housing", func() error { return deleteHousingUnits(ctx, rec, []string{"house"}) })
	check("protocols", func() error { return deleteProtocols(ctx, rec, []string{"proto"}) })
	check("projects", func() error { return deleteProjects(ctx, rec, []string{"proj"}) })
	check("permits", func() error { return deletePermits(ctx, rec, []string{"permit"}) })
	check("cohorts", func() error { return deleteCohorts(ctx, rec, []string{"cohort"}) })
	check("breeding", func() error { return deleteBreedingUnits(ctx, rec, []string{"breed"}) })
	check("organisms", func() error { return deleteOrganisms(ctx, rec, []string{"org"}) })
	check("procedures", func() error { return deleteProcedures(ctx, rec, []string{"proc"}) })
	check("observations", func() error { return deleteObservations(ctx, rec, []string{"obs"}) })
	check("samples", func() error { return deleteSamples(ctx, rec, []string{"sample"}) })
	check("supplies", func() error { return deleteSupplyItems(ctx, rec, []string{"sup"}) })
	check("treatments", func() error { return deleteTreatments(ctx, rec, []string{"treat"}) })
}

func TestDeleteHelpersErrorPath(t *testing.T) {
	ctx := context.Background()
	errExec := failingExec{}
	cases := []struct {
		name string
		fn   func() error
	}{
		{"facilities", func() error { return deleteFacilities(ctx, errExec, []string{"fac"}) }},
		{"markers", func() error { return deleteGenotypeMarkers(ctx, errExec, []string{"gm"}) }},
		{"lines", func() error { return deleteLines(ctx, errExec, []string{"line"}) }},
		{"strains", func() error { return deleteStrains(ctx, errExec, []string{"str"}) }},
		{"housing", func() error { return deleteHousingUnits(ctx, errExec, []string{"house"}) }},
		{"protocols", func() error { return deleteProtocols(ctx, errExec, []string{"proto"}) }},
		{"projects", func() error { return deleteProjects(ctx, errExec, []string{"proj"}) }},
		{"permits", func() error { return deletePermits(ctx, errExec, []string{"permit"}) }},
		{"cohorts", func() error { return deleteCohorts(ctx, errExec, []string{"cohort"}) }},
		{"breeding", func() error { return deleteBreedingUnits(ctx, errExec, []string{"breed"}) }},
		{"organisms", func() error { return deleteOrganisms(ctx, errExec, []string{"org"}) }},
		{"procedures", func() error { return deleteProcedures(ctx, errExec, []string{"proc"}) }},
		{"observations", func() error { return deleteObservations(ctx, errExec, []string{"obs"}) }},
		{"samples", func() error { return deleteSamples(ctx, errExec, []string{"sample"}) }},
		{"supplies", func() error { return deleteSupplyItems(ctx, errExec, []string{"sup"}) }},
		{"treatments", func() error { return deleteTreatments(ctx, errExec, []string{"treat"}) }},
	}

	for _, tc := range cases {
		if err := tc.fn(); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestDeleteHelpersIntermediateErrors(t *testing.T) {
	ctx := context.Background()
	type deleter func(context.Context, execQuerier, []string) error
	cases := []struct {
		name   string
		fn     deleter
		failAt []int
	}{
		{"facilities", deleteFacilities, []int{2}},
		{"lines", deleteLines, []int{2}},
		{"strains", deleteStrains, []int{2}},
		{"projects", deleteProjects, []int{2, 3, 4}},
		{"permits", deletePermits, []int{2, 3}},
		{"breeding", deleteBreedingUnits, []int{2, 3}},
		{"organisms", deleteOrganisms, []int{2}},
		{"procedures", deleteProcedures, []int{2}},
		{"supplies", deleteSupplyItems, []int{2, 3}},
		{"treatments", deleteTreatments, []int{2, 3}},
	}
	for _, tc := range cases {
		for _, failAt := range tc.failAt {
			exec := &failAfterExec{failAt: failAt}
			if err := tc.fn(ctx, exec, []string{"id"}); err == nil {
				t.Fatalf("%s expected failure at %d", tc.name, failAt)
			}
			if exec.calls != failAt {
				t.Fatalf("%s failed after %d calls (expected %d)", tc.name, exec.calls, failAt)
			}
		}
	}
}

func TestApplySnapshotDeltaUpsertsEntities(t *testing.T) {
	ctx := context.Background()
	fixture := loadFixtureSnapshot(t)
	rec := &recordingExec{}
	if err := applySnapshotDelta(ctx, rec, memory.Snapshot{}, fixture); err != nil {
		t.Fatalf("applySnapshotDelta upserts: %v", err)
	}
	if len(rec.Execs) == 0 {
		t.Fatalf("expected upsert statements to be issued")
	}
}

func TestApplySnapshotDeltaError(t *testing.T) {
	ctx := context.Background()
	before := memory.Snapshot{
		Treatments: map[string]domain.Treatment{"t": {Treatment: entitymodel.Treatment{ID: "t"}}},
	}
	if err := applySnapshotDelta(ctx, failingExec{}, before, memory.Snapshot{}); err == nil {
		t.Fatalf("expected error when exec fails")
	}
}

func TestApplySnapshotDeltaInsertError(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	after := memory.Snapshot{
		Facilities: map[string]domain.Facility{
			"fac": {Facility: entitymodel.Facility{
				ID:           "fac",
				Code:         "FAC",
				Name:         "Facility",
				Zone:         "Z",
				AccessPolicy: "all",
				CreatedAt:    now,
				UpdatedAt:    now,
			}},
		},
	}
	if err := applySnapshotDelta(ctx, failingExec{}, memory.Snapshot{}, after); err == nil {
		t.Fatalf("expected error when insert exec fails")
	}
}

func TestApplySnapshotDeltaErrorBranches(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	cases := []struct {
		name   string
		before memory.Snapshot
		after  memory.Snapshot
	}{
		{
			name:   "delete projects",
			before: memory.Snapshot{Projects: map[string]domain.Project{"proj": {Project: entitymodel.Project{ID: "proj"}}}},
		},
		{
			name:   "delete facilities",
			before: memory.Snapshot{Facilities: map[string]domain.Facility{"fac": {Facility: entitymodel.Facility{ID: "fac"}}}},
		},
		{
			name:  "insert protocols",
			after: memory.Snapshot{Protocols: map[string]domain.Protocol{"proto": {Protocol: entitymodel.Protocol{ID: "proto", Code: "P", Title: "Protocol", Status: entitymodel.ProtocolStatusDraft, CreatedAt: now, UpdatedAt: now}}}},
		},
		{
			name: "insert treatments",
			after: memory.Snapshot{Treatments: map[string]domain.Treatment{"treat": {Treatment: entitymodel.Treatment{
				ID:          "treat",
				Name:        "Treat",
				Status:      entitymodel.TreatmentStatusPlanned,
				ProcedureID: "proc",
				CreatedAt:   now,
				UpdatedAt:   now,
			}}}},
		},
	}

	for _, tc := range cases {
		if err := applySnapshotDelta(ctx, failingExec{}, tc.before, tc.after); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestApplySnapshotDeltaNoChanges(t *testing.T) {
	ctx := context.Background()
	fixture := loadFixtureSnapshot(t)
	before := cloneSnapshot(fixture)
	after := cloneSnapshot(fixture)
	rec := &recordingExec{}
	if err := applySnapshotDelta(ctx, rec, before, after); err != nil {
		t.Fatalf("expected noop applySnapshotDelta, got %v", err)
	}
	if len(rec.Execs) != 0 {
		t.Fatalf("expected no execs for identical snapshots, got %d", len(rec.Execs))
	}
}

func TestApplySnapshotDeltaUpdates(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	before := memory.Snapshot{
		Facilities: map[string]domain.Facility{
			"fac": {Facility: entitymodel.Facility{
				ID:           "fac",
				Code:         "FAC",
				Name:         "Old",
				Zone:         "Z",
				AccessPolicy: "all",
				CreatedAt:    now,
				UpdatedAt:    now,
			}},
		},
	}
	after := memory.Snapshot{
		Facilities: map[string]domain.Facility{
			"fac": {Facility: entitymodel.Facility{
				ID:           "fac",
				Code:         "FAC",
				Name:         "New",
				Zone:         "Z",
				AccessPolicy: "all",
				CreatedAt:    now,
				UpdatedAt:    now.Add(time.Second),
			}},
		},
	}
	rec := &recordingExec{}
	if err := applySnapshotDelta(ctx, rec, before, after); err != nil {
		t.Fatalf("applySnapshotDelta updates: %v", err)
	}
	if len(rec.Execs) != 1 {
		t.Fatalf("expected one upsert for updated facility, got %d", len(rec.Execs))
	}
	if !strings.Contains(strings.ToLower(rec.Execs[0]), "insert into facilities") {
		t.Fatalf("unexpected statement: %s", rec.Execs[0])
	}
}

func TestImportExportStateErrors(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	store := &Store{db: db, engine: domain.NewRulesEngine()}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on import failure")
		}
	}()
	conn.FailExec = true
	store.ImportState(memory.Snapshot{})

	conn.FailExec = false
	conn.FailTables = map[string]bool{"facilities": true}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on export failure")
		}
	}()
	store.ExportState()
}

func TestInsertHelperValidationErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	exec := &recordingExec{}
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "line missing markers",
			fn: func() error {
				return insertLines(ctx, exec, map[string]domain.Line{
					"line": {Line: entitymodel.Line{ID: "line"}},
				})
			},
		},
		{
			name: "strain missing line",
			fn: func() error {
				return insertStrains(ctx, exec, map[string]domain.Strain{
					"str": {Strain: entitymodel.Strain{ID: "str"}},
				})
			},
		},
		{
			name: "housing missing facility",
			fn: func() error {
				return insertHousingUnits(ctx, exec, map[string]domain.HousingUnit{
					"house": {HousingUnit: entitymodel.HousingUnit{ID: "house"}},
				})
			},
		},
		{
			name: "project missing facilities",
			fn: func() error {
				return insertProjects(ctx, exec, map[string]domain.Project{
					"proj": {Project: entitymodel.Project{ID: "proj"}},
				})
			},
		},
		{
			name: "permit missing facilities",
			fn: func() error {
				return insertPermits(ctx, exec, map[string]domain.Permit{
					"perm": {Permit: entitymodel.Permit{ID: "perm"}},
				})
			},
		},
		{
			name: "permit missing protocols",
			fn: func() error {
				return insertPermits(ctx, exec, map[string]domain.Permit{
					"perm": {Permit: entitymodel.Permit{ID: "perm", FacilityIDs: []string{"fac"}}},
				})
			},
		},
		{
			name: "procedure missing protocol",
			fn: func() error {
				return insertProcedures(ctx, exec, map[string]domain.Procedure{
					"proc": {Procedure: entitymodel.Procedure{ID: "proc"}},
				})
			},
		},
		{
			name: "sample missing chain",
			fn: func() error {
				return insertSamples(ctx, exec, map[string]domain.Sample{
					"sample": {Sample: entitymodel.Sample{ID: "sample"}},
				})
			},
		},
		{
			name: "sample missing facility",
			fn: func() error {
				return insertSamples(ctx, exec, map[string]domain.Sample{
					"sample": {Sample: entitymodel.Sample{
						ID:             "sample",
						ChainOfCustody: []entitymodel.SampleCustodyEvent{{Location: "loc", Timestamp: now}},
					}},
				})
			},
		},
		{
			name: "supply missing facilities",
			fn: func() error {
				return insertSupplyItems(ctx, exec, map[string]domain.SupplyItem{
					"sup": {SupplyItem: entitymodel.SupplyItem{ID: "sup"}},
				})
			},
		},
		{
			name: "supply missing projects",
			fn: func() error {
				return insertSupplyItems(ctx, exec, map[string]domain.SupplyItem{
					"sup": {SupplyItem: entitymodel.SupplyItem{
						ID:          "sup",
						FacilityIDs: []string{"fac"},
					}},
				})
			},
		},
		{
			name: "treatment missing procedure",
			fn: func() error {
				return insertTreatments(ctx, exec, map[string]domain.Treatment{
					"t": {Treatment: entitymodel.Treatment{ID: "t"}},
				})
			},
		},
	}
	for _, tc := range cases {
		if err := tc.fn(); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestLifecycleStatusesRoundTripNormalizedSnapshot(t *testing.T) {
	ctx := context.Background()
	db, _ := pgtu.NewStubDB()

	orig := loadFixtureSnapshot(t)
	if err := persistNormalized(ctx, db, orig); err != nil {
		t.Fatalf("persistNormalized: %v", err)
	}
	loaded, err := loadNormalizedSnapshot(ctx, db)
	if err != nil {
		t.Fatalf("loadNormalizedSnapshot: %v", err)
	}

	validProcedures := map[domain.ProcedureStatus]struct{}{
		domain.ProcedureStatusScheduled:  {},
		domain.ProcedureStatusInProgress: {},
		domain.ProcedureStatusCompleted:  {},
		domain.ProcedureStatusCancelled:  {},
		domain.ProcedureStatusFailed:     {},
	}
	validTreatments := map[domain.TreatmentStatus]struct{}{
		domain.TreatmentStatusPlanned:    {},
		domain.TreatmentStatusInProgress: {},
		domain.TreatmentStatusCompleted:  {},
		domain.TreatmentStatusFlagged:    {},
	}
	validSamples := map[domain.SampleStatus]struct{}{
		domain.SampleStatusStored:    {},
		domain.SampleStatusInTransit: {},
		domain.SampleStatusConsumed:  {},
		domain.SampleStatusDisposed:  {},
	}
	validHousingStates := map[domain.HousingState]struct{}{
		domain.HousingStateQuarantine:     {},
		domain.HousingStateActive:         {},
		domain.HousingStateCleaning:       {},
		domain.HousingStateDecommissioned: {},
	}
	validProtocols := map[domain.ProtocolStatus]struct{}{
		domain.ProtocolStatusDraft:     {},
		domain.ProtocolStatusSubmitted: {},
		domain.ProtocolStatusApproved:  {},
		domain.ProtocolStatusOnHold:    {},
		domain.ProtocolStatusExpired:   {},
		domain.ProtocolStatusArchived:  {},
	}
	validPermits := map[domain.PermitStatus]struct{}{
		domain.PermitStatusDraft:     {},
		domain.PermitStatusSubmitted: {},
		domain.PermitStatusApproved:  {},
		domain.PermitStatusOnHold:    {},
		domain.PermitStatusExpired:   {},
		domain.PermitStatusArchived:  {},
	}

	checkProcedure := false
	for id, proc := range orig.Procedures {
		loadedProc, ok := loaded.Procedures[id]
		if !ok {
			t.Fatalf("missing procedure %s after round-trip", id)
		}
		if _, ok := validProcedures[loadedProc.Status]; !ok {
			t.Fatalf("procedure %s has non-canonical status %s", id, loadedProc.Status)
		}
		if loadedProc.Status != proc.Status {
			t.Fatalf("procedure %s status mismatch: want %s got %s", id, proc.Status, loadedProc.Status)
		}
		checkProcedure = true
		break
	}
	if !checkProcedure {
		t.Fatalf("fixture missing procedures to validate")
	}

	checkTreatment := false
	for id, treatment := range orig.Treatments {
		loadedTreatment, ok := loaded.Treatments[id]
		if !ok {
			t.Fatalf("missing treatment %s after round-trip", id)
		}
		if _, ok := validTreatments[loadedTreatment.Status]; !ok {
			t.Fatalf("treatment %s has non-canonical status %s", id, loadedTreatment.Status)
		}
		if loadedTreatment.Status != treatment.Status {
			t.Fatalf("treatment %s status mismatch: want %s got %s", id, treatment.Status, loadedTreatment.Status)
		}
		checkTreatment = true
		break
	}
	if !checkTreatment {
		t.Fatalf("fixture missing treatments to validate")
	}

	checkSample := false
	for id, sample := range orig.Samples {
		loadedSample, ok := loaded.Samples[id]
		if !ok {
			t.Fatalf("missing sample %s after round-trip", id)
		}
		if _, ok := validSamples[loadedSample.Status]; !ok {
			t.Fatalf("sample %s has non-canonical status %s", id, loadedSample.Status)
		}
		if loadedSample.Status != sample.Status {
			t.Fatalf("sample %s status mismatch: want %s got %s", id, sample.Status, loadedSample.Status)
		}
		checkSample = true
		break
	}
	if !checkSample {
		t.Fatalf("fixture missing samples to validate")
	}

	checkHousing := false
	for id, housing := range orig.Housing {
		loadedHousing, ok := loaded.Housing[id]
		if !ok {
			t.Fatalf("missing housing %s after round-trip", id)
		}
		if _, ok := validHousingStates[loadedHousing.State]; !ok {
			t.Fatalf("housing %s has non-canonical state %s", id, loadedHousing.State)
		}
		if loadedHousing.State != housing.State {
			t.Fatalf("housing %s state mismatch: want %s got %s", id, housing.State, loadedHousing.State)
		}
		checkHousing = true
		break
	}
	if !checkHousing {
		t.Fatalf("fixture missing housing to validate")
	}

	checkProtocol := false
	for id, protocol := range orig.Protocols {
		loadedProtocol, ok := loaded.Protocols[id]
		if !ok {
			t.Fatalf("missing protocol %s after round-trip", id)
		}
		if _, ok := validProtocols[loadedProtocol.Status]; !ok {
			t.Fatalf("protocol %s has non-canonical status %s", id, loadedProtocol.Status)
		}
		if loadedProtocol.Status != protocol.Status {
			t.Fatalf("protocol %s status mismatch: want %s got %s", id, protocol.Status, loadedProtocol.Status)
		}
		checkProtocol = true
		break
	}
	if !checkProtocol {
		t.Fatalf("fixture missing protocols to validate")
	}

	checkPermit := false
	for id, permit := range orig.Permits {
		loadedPermit, ok := loaded.Permits[id]
		if !ok {
			t.Fatalf("missing permit %s after round-trip", id)
		}
		if _, ok := validPermits[loadedPermit.Status]; !ok {
			t.Fatalf("permit %s has non-canonical status %s", id, loadedPermit.Status)
		}
		if loadedPermit.Status != permit.Status {
			t.Fatalf("permit %s status mismatch: want %s got %s", id, permit.Status, loadedPermit.Status)
		}
		checkPermit = true
		break
	}
	if !checkPermit {
		t.Fatalf("fixture missing permits to validate")
	}
}

func TestPersistMissingRequiredRelationshipsError(t *testing.T) {
	db, _ := pgtu.NewStubDB()
	now := time.Now().UTC()
	snapshot := memory.Snapshot{
		Supplies: map[string]domain.SupplyItem{
			"sup-1": {
				SupplyItem: entitymodel.SupplyItem{
					ID:             "sup-1",
					SKU:            "sku-1",
					Name:           "supply",
					QuantityOnHand: 1,
					Unit:           "unit",
					ReorderLevel:   1,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
	}
	err := persistNormalized(context.Background(), db, snapshot)
	if err == nil || !strings.Contains(err.Error(), "facility_ids") {
		t.Fatalf("expected facility_ids requirement error, got %v", err)
	}
}

func TestApplyEntityModelDDLError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.FailExec = true
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected ddl error")
	}
}

func TestApplyDDLStatementsError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.FailExec = true
	if err := applyDDLStatements(context.Background(), db, "CREATE TABLE test(id text);"); err == nil {
		t.Fatalf("expected ddl exec error")
	}
}

func TestNewStoreOpenError(t *testing.T) {
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) {
		return nil, fmt.Errorf("open fail")
	})
	defer restore()
	if _, err := NewStore("", domain.NewRulesEngine()); err == nil || !strings.Contains(err.Error(), "open fail") {
		t.Fatalf("expected open error, got %v", err)
	}
}

func TestInsertPermitsRequireFacilities(t *testing.T) {
	exec := &recordingExec{}
	p := domain.Permit{Permit: entitymodel.Permit{
		ID:                "permit-2",
		PermitNumber:      "P-2",
		Authority:         "auth",
		Status:            domain.PermitStatusDraft,
		ValidFrom:         time.Now(),
		ValidUntil:        time.Now(),
		AllowedActivities: []string{"act"},
		ProtocolIDs:       []string{"proto-1"},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}}
	if err := insertPermits(context.Background(), exec, map[string]domain.Permit{"permit-2": p}); err == nil || !strings.Contains(err.Error(), "facility_ids") {
		t.Fatalf("expected facility_ids error, got %v", err)
	}
}

func TestInsertProjectsRequireFacilities(t *testing.T) {
	exec := &recordingExec{}
	prj := domain.Project{Project: entitymodel.Project{
		ID:        "proj-2",
		Code:      "CODE",
		Title:     "Title",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}
	if err := insertProjects(context.Background(), exec, map[string]domain.Project{"proj-2": prj}); err == nil || !strings.Contains(err.Error(), "facility_ids") {
		t.Fatalf("expected facility_ids error, got %v", err)
	}
}

func TestInsertSupplyItemsRequireProjects(t *testing.T) {
	exec := &recordingExec{}
	supply := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
		ID:             "sup-2",
		SKU:            "SKU",
		Name:           "Name",
		QuantityOnHand: 1,
		Unit:           "unit",
		ReorderLevel:   1,
		FacilityIDs:    []string{"fac-1"},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}}
	if err := insertSupplyItems(context.Background(), exec, map[string]domain.SupplyItem{"sup-2": supply}); err == nil || !strings.Contains(err.Error(), "project_ids") {
		t.Fatalf("expected project_ids error, got %v", err)
	}
}

func TestInsertSamplesRequireFacility(t *testing.T) {
	exec := &recordingExec{}
	s := domain.Sample{Sample: entitymodel.Sample{
		ID:              "s2",
		Identifier:      "ID2",
		SourceType:      "type",
		Status:          domain.SampleStatusStored,
		StorageLocation: "loc",
		AssayType:       "assay",
		ChainOfCustody:  []domain.SampleCustodyEvent{{Actor: "a", Location: "b", Timestamp: time.Now()}},
		CollectedAt:     time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}}
	if err := insertSamples(context.Background(), exec, map[string]domain.Sample{"s2": s}); err == nil || !strings.Contains(err.Error(), "facility_id") {
		t.Fatalf("expected facility_id error, got %v", err)
	}
}

func TestInsertProceduresRequireProtocol(t *testing.T) {
	exec := &recordingExec{}
	p := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-2",
		Name:        "Name",
		Status:      domain.ProcedureStatusScheduled,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}}
	if err := insertProcedures(context.Background(), exec, map[string]domain.Procedure{"proc-2": p}); err == nil || !strings.Contains(err.Error(), "protocol_id") {
		t.Fatalf("expected protocol_id error, got %v", err)
	}
}

func TestInsertHousingUnitsRequireFacility(t *testing.T) {
	exec := &recordingExec{}
	h := domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
		ID:          "h1",
		Name:        "Housing",
		Capacity:    1,
		State:       entitymodel.HousingStateActive,
		Environment: entitymodel.HousingEnvironmentTerrestrial,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}}
	if err := insertHousingUnits(context.Background(), exec, map[string]domain.HousingUnit{"h1": h}); err == nil || !strings.Contains(err.Error(), "facility_id") {
		t.Fatalf("expected facility_id error, got %v", err)
	}
}

func TestInsertTreatmentsRequireProcedure(t *testing.T) {
	exec := &recordingExec{}
	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:         "t2",
		Name:       "Name",
		Status:     domain.TreatmentStatusPlanned,
		DosagePlan: "plan",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}}
	if err := insertTreatments(context.Background(), exec, map[string]domain.Treatment{"t2": treatment}); err == nil || !strings.Contains(err.Error(), "procedure_id") {
		t.Fatalf("expected procedure_id error, got %v", err)
	}
}

func TestPersistNormalizedErrorPaths(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	cases := []struct {
		name     string
		snapshot memory.Snapshot
		wantErr  string
	}{
		{
			name: "line missing markers",
			snapshot: memory.Snapshot{
				Lines: map[string]domain.Line{
					"line-err": {Line: entitymodel.Line{
						ID:        "line-err",
						Code:      "LERR",
						Name:      "Line",
						Origin:    "o",
						CreatedAt: now,
						UpdatedAt: now,
					}},
				},
			},
			wantErr: "genotype_marker_ids",
		},
		{
			name: "project missing facility_ids",
			snapshot: memory.Snapshot{
				Projects: map[string]domain.Project{
					"proj-err": {Project: entitymodel.Project{
						ID:        "proj-err",
						Code:      "PRJ",
						Title:     "Proj",
						CreatedAt: now,
						UpdatedAt: now,
					}},
				},
			},
			wantErr: "facility_ids",
		},
		{
			name: "permit missing protocol_ids",
			snapshot: memory.Snapshot{
				Permits: map[string]domain.Permit{
					"permit-err": {Permit: entitymodel.Permit{
						ID:                "permit-err",
						PermitNumber:      "P-ERR",
						Authority:         "auth",
						Status:            domain.PermitStatusDraft,
						ValidFrom:         now,
						ValidUntil:        now,
						AllowedActivities: []string{"act"},
						FacilityIDs:       []string{"fac-1"},
						CreatedAt:         now,
						UpdatedAt:         now,
					}},
				},
			},
			wantErr: "protocol_ids",
		},
		{
			name: "sample missing chain",
			snapshot: memory.Snapshot{
				Samples: map[string]domain.Sample{
					"sample-err": {Sample: entitymodel.Sample{
						ID:              "sample-err",
						Identifier:      "ID",
						SourceType:      "blood",
						Status:          domain.SampleStatusStored,
						StorageLocation: "loc",
						AssayType:       "assay",
						FacilityID:      "fac-1",
						CollectedAt:     now,
						CreatedAt:       now,
						UpdatedAt:       now,
					}},
				},
			},
			wantErr: "chain_of_custody",
		},
		{
			name: "procedure missing protocol",
			snapshot: memory.Snapshot{
				Procedures: map[string]domain.Procedure{
					"proc-err": {Procedure: entitymodel.Procedure{
						ID:          "proc-err",
						Name:        "Proc",
						Status:      domain.ProcedureStatusScheduled,
						ScheduledAt: now,
						CreatedAt:   now,
						UpdatedAt:   now,
					}},
				},
			},
			wantErr: "protocol_id",
		},
		{
			name: "supply missing projects",
			snapshot: memory.Snapshot{
				Supplies: map[string]domain.SupplyItem{
					"supply-err": {SupplyItem: entitymodel.SupplyItem{
						ID:             "supply-err",
						SKU:            "SKU",
						Name:           "Supply",
						QuantityOnHand: 1,
						Unit:           "unit",
						ReorderLevel:   1,
						FacilityIDs:    []string{"fac-1"},
						CreatedAt:      now,
						UpdatedAt:      now,
					}},
				},
			},
			wantErr: "project_ids",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := pgtu.NewStubDB()
			err := persistNormalized(ctx, db, tc.snapshot)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPersistNormalizedCommitError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.FailCommit = true
	if err := persistNormalized(context.Background(), db, memory.Snapshot{}); err == nil || !strings.Contains(err.Error(), "commit") {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestLoadSnapshotValidatesRequiredJoins(t *testing.T) {
	now := time.Now().UTC()
	db, conn := pgtu.NewStubDB()
	conn.Tables = map[string][]map[string]any{
		"facilities": {{
			"id":                    "fac-1",
			"code":                  "FAC",
			"name":                  "Facility",
			"zone":                  "A",
			"access_policy":         "all",
			"created_at":            now,
			"updated_at":            now,
			"environment_baselines": nil,
		}},
		"projects": {{
			"id":          "proj-1",
			"code":        "PROJ",
			"title":       "Project",
			"description": nil,
			"created_at":  now,
			"updated_at":  now,
		}},
	}
	if _, err := loadNormalizedSnapshot(context.Background(), db); err == nil || !strings.Contains(err.Error(), "facility_ids") {
		t.Fatalf("expected required facility_ids error, got %v", err)
	}
}

func TestLoadSnapshotRowsError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.RowsErr = fmt.Errorf("row err")
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected rows error")
	}
}

func TestLoadFacilitiesDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables = map[string][]map[string]any{
		"facilities": {{
			"id":                    "fac-err",
			"code":                  "FAC",
			"name":                  "Facility",
			"zone":                  "A",
			"access_policy":         "all",
			"created_at":            time.Now(),
			"updated_at":            time.Now(),
			"environment_baselines": []byte("not-json"),
		}},
	}
	if _, err := loadNormalizedSnapshot(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for facilities")
	}
}

func TestLoadGenotypeMarkersDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables = map[string][]map[string]any{
		"genotype_markers": {{
			"id":             "gm-1",
			"name":           "GM",
			"locus":          "loc",
			"alleles":        []byte("bad"),
			"assay_method":   "PCR",
			"interpretation": "interp",
			"version":        "v1",
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		}},
	}
	if _, err := loadNormalizedSnapshot(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for genotype markers")
	}
}

func TestLoadBreedingUnitsDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["breeding_units"] = []map[string]any{{
		"id":                 "breed-err",
		"name":               "Breed",
		"strategy":           "s",
		"housing_id":         nil,
		"line_id":            nil,
		"strain_id":          nil,
		"target_line_id":     nil,
		"target_strain_id":   nil,
		"protocol_id":        nil,
		"pairing_attributes": []byte("bad"),
		"pairing_intent":     nil,
		"pairing_notes":      nil,
		"created_at":         time.Now(),
		"updated_at":         time.Now(),
	}}
	if _, err := loadBreedingUnits(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for breeding units")
	}
}

func TestLoadObservationsDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["observations"] = []map[string]any{{
		"id":           "obs-err",
		"observer":     "o",
		"recorded_at":  time.Now(),
		"procedure_id": nil,
		"organism_id":  nil,
		"cohort_id":    nil,
		"data":         []byte("bad"),
		"notes":        nil,
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}}
	if _, err := loadObservations(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for observations")
	}
}

func TestLoadOrganismsDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["organisms"] = []map[string]any{{
		"id":          "org-err",
		"name":        "Org",
		"species":     "sp",
		"line":        "line",
		"stage":       domain.StageAdult,
		"line_id":     nil,
		"strain_id":   nil,
		"cohort_id":   nil,
		"housing_id":  nil,
		"protocol_id": nil,
		"project_id":  nil,
		"attributes":  []byte("bad"),
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}}
	if _, err := loadOrganisms(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for organisms")
	}
}

func TestLoadSamplesDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["samples"] = []map[string]any{{
		"id":               "s-err",
		"identifier":       "ID",
		"source_type":      "type",
		"status":           domain.SampleStatusStored,
		"storage_location": "loc",
		"assay_type":       "assay",
		"facility_id":      "fac-1",
		"organism_id":      nil,
		"cohort_id":        nil,
		"chain_of_custody": []byte("bad json"),
		"attributes":       []byte("{}"),
		"collected_at":     time.Now(),
		"created_at":       time.Now(),
		"updated_at":       time.Now(),
	}}
	if _, err := loadSamples(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for samples")
	}
}

func TestLoadSupplyItemsDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["supply_items"] = []map[string]any{{
		"id":               "sup-err",
		"sku":              "SKU",
		"name":             "Name",
		"quantity_on_hand": 1,
		"unit":             "u",
		"reorder_level":    1,
		"description":      nil,
		"lot_number":       nil,
		"expires_at":       nil,
		"attributes":       []byte("bad"),
		"created_at":       time.Now(),
		"updated_at":       time.Now(),
	}}
	if _, err := loadSupplyItems(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for supply items")
	}
}

func TestLoadTreatmentsDecodeError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.Tables["treatments"] = []map[string]any{{
		"id":                 "treat-err",
		"name":               "Treat",
		"status":             domain.TreatmentStatusPlanned,
		"procedure_id":       "proc",
		"dosage_plan":        "plan",
		"administration_log": []byte("bad"),
		"adverse_events":     []byte("bad"),
		"created_at":         time.Now(),
		"updated_at":         time.Now(),
	}}
	if _, err := loadTreatments(context.Background(), db); err == nil {
		t.Fatalf("expected decode error for treatments")
	}
}

func TestLoadHelpersReferenceValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		setup func(*pgtu.StubConn)
		fn    func(context.Context, *sql.DB) error
		want  string
	}{
		{
			name: "line marker references missing line",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["lines__genotype_marker_ids"] = []map[string]any{{
					"line_id":            "line-missing",
					"genotype_marker_id": "gm-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadLineMarkers(ctx, db, map[string]domain.Line{})
			},
			want: "missing line",
		},
		{
			name: "line missing markers",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["lines__genotype_marker_ids"] = nil
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadLineMarkers(ctx, db, map[string]domain.Line{"line-1": {Line: entitymodel.Line{ID: "line-1"}}})
			},
			want: "genotype_marker_ids",
		},
		{
			name: "strain marker references missing strain",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["strains__genotype_marker_ids"] = []map[string]any{{
					"strain_id":          "strain-missing",
					"genotype_marker_id": "gm-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadStrainMarkers(ctx, db, map[string]domain.Strain{})
			},
			want: "missing strain",
		},
		{
			name: "project facility references missing project",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["facilities__project_ids"] = []map[string]any{{
					"facility_id": "fac-1",
					"project_id":  "proj-missing",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadProjectFacilities(ctx, db, map[string]domain.Project{}, map[string]domain.Facility{})
			},
			want: "missing project",
		},
		{
			name: "permit facility references missing permit",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["permits__facility_ids"] = []map[string]any{{
					"permit_id":   "permit-missing",
					"facility_id": "fac-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadPermitFacilities(ctx, db, map[string]domain.Permit{})
			},
			want: "missing permit",
		},
		{
			name: "permit protocol references missing permit",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["permits__protocol_ids"] = []map[string]any{{
					"permit_id":   "permit-missing",
					"protocol_id": "proto-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadPermitProtocols(ctx, db, map[string]domain.Permit{})
			},
			want: "missing permit",
		},
		{
			name: "breeding female references missing breeding unit",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["breeding_units__female_ids"] = []map[string]any{{
					"breeding_unit_id": "breed-missing",
					"organism_id":      "org-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadBreedingUnitMembers(ctx, db, map[string]domain.BreedingUnit{})
			},
			want: "missing breeding_unit",
		},
		{
			name: "organism parent references missing organism",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["organisms__parent_ids"] = []map[string]any{{
					"organism_id":   "org-missing",
					"parent_ids_id": "p1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadOrganismParents(ctx, db, map[string]domain.Organism{})
			},
			want: "missing organism",
		},
		{
			name: "procedure organism references missing procedure",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["procedures__organism_ids"] = []map[string]any{{
					"procedure_id": "proc-missing",
					"organism_id":  "org-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadProcedureOrganisms(ctx, db, map[string]domain.Procedure{})
			},
			want: "missing procedure",
		},
		{
			name: "treatment cohort references missing treatment",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["treatments__cohort_ids"] = []map[string]any{{
					"treatment_id": "treat-missing",
					"cohort_id":    "c1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadTreatmentCohorts(ctx, db, map[string]domain.Treatment{})
			},
			want: "missing treatment",
		},
		{
			name: "treatment organism references missing treatment",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["treatments__organism_ids"] = []map[string]any{{
					"treatment_id": "treat-missing",
					"organism_id":  "org-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadTreatmentOrganisms(ctx, db, map[string]domain.Treatment{})
			},
			want: "missing treatment",
		},
		{
			name: "supply facility references missing supply",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["supply_items__facility_ids"] = []map[string]any{{
					"supply_item_id": "sup-missing",
					"facility_id":    "fac-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadSupplyItemFacilities(ctx, db, map[string]domain.SupplyItem{})
			},
			want: "missing supply_item",
		},
		{
			name: "project supply references missing project",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["projects__supply_item_ids"] = []map[string]any{{
					"project_id":     "proj-missing",
					"supply_item_id": "sup-1",
				}}
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadProjectSupplyItems(ctx, db, map[string]domain.Project{}, map[string]domain.SupplyItem{})
			},
			want: "missing project",
		},
		{
			name: "project supply missing project_ids",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["projects__supply_item_ids"] = nil
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				supplies := map[string]domain.SupplyItem{
					"sup-2": {SupplyItem: entitymodel.SupplyItem{ID: "sup-2"}},
				}
				return loadProjectSupplyItems(ctx, db, map[string]domain.Project{}, supplies)
			},
			want: "project_ids",
		},
		{
			name: "supply facility missing facility_ids",
			setup: func(conn *pgtu.StubConn) {
				conn.Tables["supply_items__facility_ids"] = nil
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				supplies := map[string]domain.SupplyItem{
					"sup-3": {SupplyItem: entitymodel.SupplyItem{ID: "sup-3"}},
				}
				return loadSupplyItemFacilities(ctx, db, supplies)
			},
			want: "facility_ids",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, conn := pgtu.NewStubDB()
			if tc.setup != nil {
				tc.setup(conn)
			}
			err := tc.fn(ctx, db)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestLoadNormalizedSnapshotQueryFailures(t *testing.T) {
	ctx := context.Background()
	snapshot := loadFixtureSnapshot(t)
	cases := []struct {
		name  string
		table string
	}{
		{"facilities", "facilities"},
		{"genotype markers", "genotype_markers"},
		{"lines", "lines"},
		{"line markers", "lines__genotype_marker_ids"},
		{"strains", "strains"},
		{"strain markers", "strains__genotype_marker_ids"},
		{"housing", "housing_units"},
		{"protocols", "protocols"},
		{"projects", "projects"},
		{"project facilities", "facilities__project_ids"},
		{"project protocols", "projects__protocol_ids"},
		{"permits", "permits"},
		{"permit facilities", "permits__facility_ids"},
		{"permit protocols", "permits__protocol_ids"},
		{"cohorts", "cohorts"},
		{"breeding units", "breeding_units"},
		{"breeding females", "breeding_units__female_ids"},
		{"organisms", "organisms"},
		{"organism parents", "organisms__parent_ids"},
		{"procedures", "procedures"},
		{"procedure organisms", "procedures__organism_ids"},
		{"observations", "observations"},
		{"samples", "samples"},
		{"supply items", "supply_items"},
		{"supply facilities", "supply_items__facility_ids"},
		{"project supplies", "projects__supply_item_ids"},
		{"treatments", "treatments"},
		{"treatment cohorts", "treatments__cohort_ids"},
		{"treatment organisms", "treatments__organism_ids"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, conn := pgtu.NewStubDB()
			if err := persistNormalized(ctx, db, snapshot); err != nil {
				t.Fatalf("seed snapshot: %v", err)
			}
			conn.FailTables = map[string]bool{tc.table: true}
			if _, err := loadNormalizedSnapshot(ctx, db); err == nil || !strings.Contains(err.Error(), tc.table) {
				t.Fatalf("expected query failure mentioning %s, got %v", tc.table, err)
			}
		})
	}
}

func TestMarshalJSONNullableBranches(t *testing.T) {
	if data, err := marshalJSONNullable(nil); err != nil || data != nil {
		t.Fatalf("expected nil marshal without error, got %v %v", data, err)
	}
	if _, err := marshalJSONNullable(map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatalf("expected marshal error for unsupported type")
	}
}

func TestDecodeStringSliceEmpty(t *testing.T) {
	out, err := decodeStringSlice(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil slice for empty input, got %v", out)
	}
}

func TestSliceEmptyDefaultBranch(t *testing.T) {
	if sliceEmpty([]int{1}) {
		t.Fatalf("expected sliceEmpty to return false for unsupported type")
	}
}

func TestDecodeCustodyInvalidJSON(t *testing.T) {
	if _, err := decodeCustody([]byte("bad")); err == nil {
		t.Fatalf("expected decodeCustody to error on invalid json")
	}
}

func TestStoreDBExposesHandle(t *testing.T) {
	db, _ := pgtu.NewStubDB()
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if store.DB() == nil {
		t.Fatalf("expected DB handle")
	}
}

func TestRunInTransactionPersistsErrorWhenExecFails(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	conn.FailExec = true
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			ID:           "fac-fail",
			Code:         "FF",
			Name:         "Fail",
			Zone:         "Z",
			AccessPolicy: "none",
		}})
		return err
	}); err == nil {
		t.Fatalf("expected persistence error when exec fails")
	}
}

func TestRunInTransactionStopsOnUserError(t *testing.T) {
	var conn *pgtu.StubConn
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) {
		db, c := pgtu.NewStubDB()
		conn = c
		return db, nil
	})
	defer restore()
	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	userErr := fmt.Errorf("user fail")
	if _, err := store.RunInTransaction(context.Background(), func(domain.Transaction) error { return userErr }); !errors.Is(err, userErr) {
		t.Fatalf("expected user error to propagate, got %v", err)
	}
	if conn != nil && len(conn.Tables) != 0 {
		t.Fatalf("expected no persistence when user fn errors")
	}
}

func TestPersistNormalizedBeginTxError(t *testing.T) {
	db, conn := pgtu.NewStubDB()
	conn.FailBegin = true
	err := persistNormalized(context.Background(), db, memory.Snapshot{})
	if err == nil || !strings.Contains(err.Error(), "begin") {
		t.Fatalf("expected begin tx error, got %v", err)
	}
}

func TestInsertFacilitiesMarshalError(t *testing.T) {
	ctx := context.Background()
	conn := &recordingExec{}
	fac := domain.Facility{Facility: entitymodel.Facility{
		ID:           "fac-mar",
		Code:         "FAC",
		Name:         "Facility",
		Zone:         "Z",
		AccessPolicy: "all",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}}
	if err := fac.ApplyEnvironmentBaselines(map[string]any{"bad": make(chan int)}); err != nil {
		t.Fatalf("ApplyEnvironmentBaselines: %v", err)
	}
	if err := insertFacilities(ctx, conn, map[string]domain.Facility{fac.ID: fac}); err == nil {
		t.Fatalf("expected marshal error for facility environment")
	}
}

func TestInsertGenotypeMarkersRequireAlleles(t *testing.T) {
	exec := &recordingExec{}
	marker := domain.GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{
		ID:          "gm-empty",
		Name:        "Marker",
		Locus:       "l",
		AssayMethod: "PCR",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}}
	if err := insertGenotypeMarkers(context.Background(), exec, map[string]domain.GenotypeMarker{marker.ID: marker}); err == nil {
		t.Fatalf("expected alleles required error")
	}
}

func TestInsertStrainsRequireLineID(t *testing.T) {
	exec := &recordingExec{}
	strain := domain.Strain{Strain: entitymodel.Strain{
		ID:        "strain-missing",
		Code:      "S",
		Name:      "Strain",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}
	if err := insertStrains(context.Background(), exec, map[string]domain.Strain{strain.ID: strain}); err == nil || !strings.Contains(err.Error(), "line_id") {
		t.Fatalf("expected line_id required error, got %v", err)
	}
}

func TestInsertLinesRequiresGenotypeMarkers(t *testing.T) {
	exec := &recordingExec{}
	line := domain.Line{Line: entitymodel.Line{
		ID:        "line-1",
		Code:      "L1",
		Name:      "Line",
		Origin:    "lab",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}
	if err := insertLines(context.Background(), exec, map[string]domain.Line{"line-1": line}); err == nil {
		t.Fatalf("expected error for missing genotype_marker_ids")
	}
}

func TestInsertPermitsRequireProtocols(t *testing.T) {
	exec := &recordingExec{}
	p := domain.Permit{Permit: entitymodel.Permit{
		ID:                "permit-1",
		PermitNumber:      "P-1",
		Authority:         "auth",
		Status:            domain.PermitStatusDraft,
		ValidFrom:         time.Now(),
		ValidUntil:        time.Now(),
		AllowedActivities: []string{"act"},
		FacilityIDs:       []string{"fac-1"},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}}
	if err := insertPermits(context.Background(), exec, map[string]domain.Permit{"permit-1": p}); err == nil || !strings.Contains(err.Error(), "protocol_ids") {
		t.Fatalf("expected protocol_ids error, got %v", err)
	}
}

func TestInsertSamplesRequireChainOfCustody(t *testing.T) {
	exec := &recordingExec{}
	s := domain.Sample{Sample: entitymodel.Sample{
		ID:              "s1",
		Identifier:      "ID1",
		SourceType:      "type",
		Status:          domain.SampleStatusStored,
		StorageLocation: "loc",
		AssayType:       "assay",
		FacilityID:      "fac-1",
		CollectedAt:     time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}}
	if err := insertSamples(context.Background(), exec, map[string]domain.Sample{"s1": s}); err == nil {
		t.Fatalf("expected chain_of_custody validation error")
	}
}

func TestInsertObservationsMarshalError(t *testing.T) {
	exec := &recordingExec{}
	obs := domain.Observation{Observation: entitymodel.Observation{
		ID:         "obs-marshal",
		Observer:   "tech",
		RecordedAt: time.Now(),
		Data:       map[string]any{"bad": make(chan int)},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}}
	if err := insertObservations(context.Background(), exec, map[string]domain.Observation{obs.ID: obs}); err == nil {
		t.Fatalf("expected marshal error for observation data")
	}
}

func TestInsertHelpersExecFailures(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	cases := []struct {
		name  string
		table string
		fn    func(context.Context, *sql.DB) error
		want  string
	}{
		{
			name:  "line marker insert fails",
			table: "lines__genotype_marker_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				line := domain.Line{Line: entitymodel.Line{
					ID:                "line-exec",
					Code:              "L",
					Name:              "Line",
					Origin:            "lab",
					GenotypeMarkerIDs: []string{"gm-1"},
					CreatedAt:         now,
					UpdatedAt:         now,
				}}
				return insertLines(ctx, db, map[string]domain.Line{line.ID: line})
			},
			want: "genotype_marker_id",
		},
		{
			name:  "protocol insert fails",
			table: "protocols",
			fn: func(ctx context.Context, db *sql.DB) error {
				p := domain.Protocol{Protocol: entitymodel.Protocol{
					ID:          "proto-exec",
					Code:        "P",
					Title:       "Protocol",
					MaxSubjects: 1,
					Status:      domain.ProtocolStatusApproved,
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertProtocols(ctx, db, map[string]domain.Protocol{p.ID: p})
			},
			want: "insert protocol",
		},
		{
			name:  "cohort insert fails",
			table: "cohorts",
			fn: func(ctx context.Context, db *sql.DB) error {
				c := domain.Cohort{Cohort: entitymodel.Cohort{
					ID:        "cohort-exec",
					Name:      "Cohort",
					Purpose:   "p",
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertCohorts(ctx, db, map[string]domain.Cohort{c.ID: c})
			},
			want: "insert cohort",
		},
		{
			name:  "housing insert fails",
			table: "housing_units",
			fn: func(ctx context.Context, db *sql.DB) error {
				h := domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
					ID:          "house-exec",
					FacilityID:  "fac-1",
					Name:        "House",
					Capacity:    1,
					Environment: entitymodel.HousingEnvironmentTerrestrial,
					State:       entitymodel.HousingStateActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertHousingUnits(ctx, db, map[string]domain.HousingUnit{h.ID: h})
			},
			want: "insert housing",
		},
		{
			name:  "breeding insert fails",
			table: "breeding_units",
			fn: func(ctx context.Context, db *sql.DB) error {
				b := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
					ID:        "breed-main",
					Name:      "Breed",
					Strategy:  "s",
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertBreedingUnits(ctx, db, map[string]domain.BreedingUnit{b.ID: b})
			},
			want: "insert breeding",
		},
		{
			name:  "strain marker insert fails",
			table: "strains__genotype_marker_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				strain := domain.Strain{Strain: entitymodel.Strain{
					ID:                "strain-exec",
					Code:              "S",
					Name:              "Strain",
					LineID:            "line-1",
					GenotypeMarkerIDs: []string{"gm-1"},
					CreatedAt:         now,
					UpdatedAt:         now,
				}}
				return insertStrains(ctx, db, map[string]domain.Strain{strain.ID: strain})
			},
			want: "genotype_marker_id",
		},
		{
			name:  "project facility insert fails",
			table: "facilities__project_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				proj := domain.Project{Project: entitymodel.Project{
					ID:          "proj-exec",
					Code:        "PRJ",
					Title:       "Project",
					FacilityIDs: []string{"fac-1"},
					ProtocolIDs: []string{"proto-1"},
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertProjects(ctx, db, map[string]domain.Project{proj.ID: proj})
			},
			want: "facility",
		},
		{
			name:  "project protocol insert fails",
			table: "projects__protocol_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				proj := domain.Project{Project: entitymodel.Project{
					ID:          "proj-proto",
					Code:        "PRJ2",
					Title:       "Project2",
					FacilityIDs: []string{"fac-1"},
					ProtocolIDs: []string{"proto-1"},
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertProjects(ctx, db, map[string]domain.Project{proj.ID: proj})
			},
			want: "protocol",
		},
		{
			name:  "project supply insert fails",
			table: "projects__supply_item_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				proj := domain.Project{Project: entitymodel.Project{
					ID:            "proj-supply",
					Code:          "PRJ3",
					Title:         "Project3",
					FacilityIDs:   []string{"fac-1"},
					SupplyItemIDs: []string{"sup-1"},
					CreatedAt:     now,
					UpdatedAt:     now,
				}}
				return insertProjects(ctx, db, map[string]domain.Project{proj.ID: proj})
			},
			want: "supply",
		},
		{
			name:  "permit facility insert fails",
			table: "permits__facility_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				permit := domain.Permit{Permit: entitymodel.Permit{
					ID:                "permit-fac",
					PermitNumber:      "PN",
					Authority:         "auth",
					Status:            domain.PermitStatusDraft,
					ValidFrom:         now,
					ValidUntil:        now,
					AllowedActivities: []string{"act"},
					FacilityIDs:       []string{"fac-1"},
					ProtocolIDs:       []string{"proto-1"},
					CreatedAt:         now,
					UpdatedAt:         now,
				}}
				return insertPermits(ctx, db, map[string]domain.Permit{permit.ID: permit})
			},
			want: "facility",
		},
		{
			name:  "permit protocol insert fails",
			table: "permits__protocol_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				permit := domain.Permit{Permit: entitymodel.Permit{
					ID:                "permit-proto",
					PermitNumber:      "PN2",
					Authority:         "auth",
					Status:            domain.PermitStatusDraft,
					ValidFrom:         now,
					ValidUntil:        now,
					AllowedActivities: []string{"act"},
					FacilityIDs:       []string{"fac-1"},
					ProtocolIDs:       []string{"proto-1"},
					CreatedAt:         now,
					UpdatedAt:         now,
				}}
				return insertPermits(ctx, db, map[string]domain.Permit{permit.ID: permit})
			},
			want: "protocol",
		},
		{
			name:  "breeding female insert fails",
			table: "breeding_units__female_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				b := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
					ID:        "breed-fem",
					Name:      "Breed",
					Strategy:  "strategy",
					FemaleIDs: []string{"org-1"},
					MaleIDs:   []string{"org-2"},
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertBreedingUnits(ctx, db, map[string]domain.BreedingUnit{b.ID: b})
			},
			want: "female",
		},
		{
			name:  "breeding male insert fails",
			table: "breeding_units__male_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				b := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
					ID:        "breed-male",
					Name:      "Breed",
					Strategy:  "strategy",
					FemaleIDs: []string{"org-1"},
					MaleIDs:   []string{"org-2"},
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertBreedingUnits(ctx, db, map[string]domain.BreedingUnit{b.ID: b})
			},
			want: "male",
		},
		{
			name:  "organism parent insert fails",
			table: "organisms__parent_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				org := domain.Organism{Organism: entitymodel.Organism{
					ID:        "org-parent",
					Name:      "Org",
					Species:   "sp",
					Line:      "line",
					Stage:     domain.StageAdult,
					ParentIDs: []string{"p1"},
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertOrganisms(ctx, db, map[string]domain.Organism{org.ID: org})
			},
			want: "parent",
		},
		{
			name:  "organism insert fails",
			table: "organisms",
			fn: func(ctx context.Context, db *sql.DB) error {
				org := domain.Organism{Organism: entitymodel.Organism{
					ID:        "org-exec",
					Name:      "Org",
					Species:   "sp",
					Line:      "line",
					Stage:     domain.StageAdult,
					CreatedAt: now,
					UpdatedAt: now,
				}}
				return insertOrganisms(ctx, db, map[string]domain.Organism{org.ID: org})
			},
			want: "insert organism",
		},
		{
			name:  "procedure organism insert fails",
			table: "procedures__organism_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				proc := domain.Procedure{Procedure: entitymodel.Procedure{
					ID:          "proc-org",
					Name:        "Proc",
					Status:      domain.ProcedureStatusScheduled,
					ScheduledAt: now,
					ProtocolID:  "proto-1",
					OrganismIDs: []string{"org-1"},
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertProcedures(ctx, db, map[string]domain.Procedure{proc.ID: proc})
			},
			want: "organism",
		},
		{
			name:  "supply facility insert fails",
			table: "supply_items__facility_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				supply := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
					ID:             "sup-fac",
					SKU:            "SKU",
					Name:           "Supply",
					QuantityOnHand: 1,
					Unit:           "u",
					ReorderLevel:   1,
					FacilityIDs:    []string{"fac-1"},
					ProjectIDs:     []string{"proj-1"},
					CreatedAt:      now,
					UpdatedAt:      now,
				}}
				return insertSupplyItems(ctx, db, map[string]domain.SupplyItem{supply.ID: supply})
			},
			want: "facility",
		},
		{
			name:  "supply project insert fails",
			table: "projects__supply_item_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				supply := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
					ID:             "sup-proj",
					SKU:            "SKU2",
					Name:           "Supply2",
					QuantityOnHand: 1,
					Unit:           "u",
					ReorderLevel:   1,
					FacilityIDs:    []string{"fac-1"},
					ProjectIDs:     []string{"proj-1"},
					CreatedAt:      now,
					UpdatedAt:      now,
				}}
				return insertSupplyItems(ctx, db, map[string]domain.SupplyItem{supply.ID: supply})
			},
			want: "project",
		},
		{
			name:  "sample insert fails",
			table: "samples",
			fn: func(ctx context.Context, db *sql.DB) error {
				s := domain.Sample{Sample: entitymodel.Sample{
					ID:              "sample-exec",
					Identifier:      "S",
					SourceType:      "blood",
					Status:          domain.SampleStatusStored,
					StorageLocation: "loc",
					AssayType:       "assay",
					FacilityID:      "fac-1",
					ChainOfCustody:  []domain.SampleCustodyEvent{{Actor: "a", Location: "b", Timestamp: now}},
					CollectedAt:     now,
					CreatedAt:       now,
					UpdatedAt:       now,
				}}
				return insertSamples(ctx, db, map[string]domain.Sample{s.ID: s})
			},
			want: "insert sample",
		},
		{
			name:  "treatment cohort insert fails",
			table: "treatments__cohort_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				treat := domain.Treatment{Treatment: entitymodel.Treatment{
					ID:          "treat-cohort",
					Name:        "Treat",
					Status:      domain.TreatmentStatusPlanned,
					ProcedureID: "proc-1",
					CohortIDs:   []string{"c1"},
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertTreatments(ctx, db, map[string]domain.Treatment{treat.ID: treat})
			},
			want: "cohort",
		},
		{
			name:  "treatment organism insert fails",
			table: "treatments__organism_ids",
			fn: func(ctx context.Context, db *sql.DB) error {
				treat := domain.Treatment{Treatment: entitymodel.Treatment{
					ID:                "treat-org",
					Name:              "Treat",
					Status:            domain.TreatmentStatusPlanned,
					ProcedureID:       "proc-1",
					OrganismIDs:       []string{"org-1"},
					AdministrationLog: []string{},
					CreatedAt:         now,
					UpdatedAt:         now,
				}}
				return insertTreatments(ctx, db, map[string]domain.Treatment{treat.ID: treat})
			},
			want: "organism",
		},
		{
			name:  "treatment insert fails",
			table: "treatments",
			fn: func(ctx context.Context, db *sql.DB) error {
				treat := domain.Treatment{Treatment: entitymodel.Treatment{
					ID:          "treat-main",
					Name:        "Treat",
					Status:      domain.TreatmentStatusPlanned,
					ProcedureID: "proc-1",
					CreatedAt:   now,
					UpdatedAt:   now,
				}}
				return insertTreatments(ctx, db, map[string]domain.Treatment{treat.ID: treat})
			},
			want: "insert treatment",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, conn := pgtu.NewStubDB()
			conn.FailTables = map[string]bool{tc.table: true}
			err := tc.fn(ctx, db)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q for table %s, got %v", tc.want, tc.table, err)
			}
		})
	}
}

func TestDecodeCustodyErrorsOnEmpty(t *testing.T) {
	if _, err := decodeCustody(nil); err == nil {
		t.Fatalf("expected decodeCustody to error on empty input")
	}
}

func TestPersistAndLoadWithOptionalFields(t *testing.T) {
	ctx := context.Background()
	db, _ := pgtu.NewStubDB()
	now := time.Now().UTC()

	// Seeds with optional fields populated to exercise nullable branches.
	facility := domain.Facility{Facility: entitymodel.Facility{
		ID:           "fac-1",
		Code:         "FAC",
		Name:         "Facility",
		Zone:         "Z1",
		AccessPolicy: "all",
		CreatedAt:    now,
		UpdatedAt:    now,
	}}
	if err := facility.ApplyEnvironmentBaselines(map[string]any{"temp": 22}); err != nil {
		t.Fatalf("ApplyEnvironmentBaselines: %v", err)
	}

	marker := domain.GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{
		ID:             "marker-1",
		Name:           "Marker",
		Locus:          "loc",
		Alleles:        []string{"A", "B"},
		AssayMethod:    "PCR",
		Interpretation: "interp",
		Version:        "v1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}}

	deprecatedAt := now.Add(-time.Hour)
	deprecationReason := "outdated"
	line := domain.Line{Line: entitymodel.Line{
		ID:                "line-1",
		Code:              "L1",
		Name:              "Line",
		Origin:            "origin",
		DeprecatedAt:      &deprecatedAt,
		DeprecationReason: &deprecationReason,
		GenotypeMarkerIDs: []string{marker.ID},
		CreatedAt:         now,
		UpdatedAt:         now,
	}}

	retiredAt := now.Add(time.Hour)
	retirementReason := "complete"
	description := "strain-desc"
	generation := "F1"
	strain := domain.Strain{Strain: entitymodel.Strain{
		ID:                "strain-1",
		Code:              "S1",
		Name:              "Strain",
		LineID:            line.ID,
		Description:       &description,
		Generation:        &generation,
		RetiredAt:         &retiredAt,
		RetirementReason:  &retirementReason,
		GenotypeMarkerIDs: []string{marker.ID},
		CreatedAt:         now,
		UpdatedAt:         now,
	}}

	housing := domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
		ID:          "house-1",
		FacilityID:  facility.ID,
		Name:        "Housing",
		Capacity:    2,
		Environment: entitymodel.HousingEnvironmentAquatic,
		State:       entitymodel.HousingStateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	protoDesc := "protocol"
	protocol := domain.Protocol{Protocol: entitymodel.Protocol{
		ID:          "proto-1",
		Code:        "P1",
		Title:       "Protocol",
		Description: &protoDesc,
		MaxSubjects: 10,
		Status:      domain.ProtocolStatusApproved,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	projectDesc := "project"
	project := domain.Project{Project: entitymodel.Project{
		ID:          "proj-1",
		Code:        "PR1",
		Title:       "Project",
		Description: &projectDesc,
		FacilityIDs: []string{facility.ID},
		ProtocolIDs: []string{protocol.ID},
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	permitNote := "note"
	permit := domain.Permit{Permit: entitymodel.Permit{
		ID:                "permit-1",
		PermitNumber:      "PM1",
		Authority:         "auth",
		Status:            domain.PermitStatusApproved,
		ValidFrom:         now,
		ValidUntil:        now.Add(2 * time.Hour),
		AllowedActivities: []string{"act"},
		FacilityIDs:       []string{facility.ID},
		ProtocolIDs:       []string{protocol.ID},
		Notes:             &permitNote,
		CreatedAt:         now,
		UpdatedAt:         now,
	}}

	cohort := domain.Cohort{Cohort: entitymodel.Cohort{
		ID:         "cohort-1",
		Name:       "Cohort",
		Purpose:    "purpose",
		ProjectID:  &project.ID,
		HousingID:  &housing.ID,
		ProtocolID: &protocol.ID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}}

	pairingIntent := "intent"
	pairingNotes := "notes"
	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
		ID:                "breed-1",
		Name:              "Breeding",
		Strategy:          "strategy",
		HousingID:         &housing.ID,
		LineID:            &line.ID,
		StrainID:          &strain.ID,
		TargetLineID:      &line.ID,
		TargetStrainID:    &strain.ID,
		ProtocolID:        &protocol.ID,
		PairingIntent:     &pairingIntent,
		PairingNotes:      &pairingNotes,
		FemaleIDs:         []string{"org-1"},
		MaleIDs:           []string{"org-2"},
		CreatedAt:         now,
		UpdatedAt:         now,
		PairingAttributes: map[string]any{},
	}}
	if err := breeding.ApplyPairingAttributes(map[string]any{"attr": true}); err != nil {
		t.Fatalf("ApplyPairingAttributes: %v", err)
	}

	projectID := project.ID
	lineID := line.ID
	strainID := strain.ID
	cohortID := cohort.ID
	housingID := housing.ID
	protocolID := protocol.ID
	org1 := domain.Organism{Organism: entitymodel.Organism{
		ID:         "org-1",
		Name:       "Org1",
		Species:    "frog",
		Line:       "line",
		LineID:     &lineID,
		StrainID:   &strainID,
		CohortID:   &cohortID,
		HousingID:  &housingID,
		ProtocolID: &protocolID,
		ProjectID:  &projectID,
		Stage:      domain.StageAdult,
		CreatedAt:  now,
		UpdatedAt:  now,
		Attributes: map[string]any{},
	}}
	if err := org1.SetCoreAttributes(map[string]any{"core": true}); err != nil {
		t.Fatalf("SetCoreAttributes: %v", err)
	}
	org2 := domain.Organism{Organism: entitymodel.Organism{
		ID:         "org-2",
		Name:       "Org2",
		Species:    "frog",
		Line:       "line",
		LineID:     &lineID,
		StrainID:   &strainID,
		HousingID:  &housingID,
		ProtocolID: &protocolID,
		ProjectID:  &projectID,
		Stage:      domain.StageAdult,
		CreatedAt:  now,
		UpdatedAt:  now,
	}}

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-1",
		Name:        "Proc",
		Status:      domain.ProcedureStatusScheduled,
		ScheduledAt: now,
		ProtocolID:  protocol.ID,
		ProjectID:   &projectID,
		CohortID:    &cohortID,
		OrganismIDs: []string{org1.ID},
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	data := map[string]any{"note": "value"}
	obsNote := "obs-note"
	observation := domain.Observation{Observation: entitymodel.Observation{
		ID:          "obs-1",
		Observer:    "observer",
		RecordedAt:  now,
		ProcedureID: &procedure.ID,
		OrganismID:  &org1.ID,
		CohortID:    &cohort.ID,
		Data:        data,
		Notes:       &obsNote,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	chain := []domain.SampleCustodyEvent{{
		Actor:     "tech",
		Location:  "lab",
		Timestamp: now,
	}}
	sample := domain.Sample{Sample: entitymodel.Sample{
		ID:              "sample-1",
		Identifier:      "S1",
		SourceType:      "blood",
		Status:          domain.SampleStatusStored,
		StorageLocation: "fridge",
		AssayType:       "assay",
		FacilityID:      facility.ID,
		OrganismID:      &org1.ID,
		CohortID:        &cohort.ID,
		ChainOfCustody:  chain,
		CollectedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
		Attributes:      map[string]any{},
	}}
	if err := sample.ApplySampleAttributes(map[string]any{"volume": "10ml"}); err != nil {
		t.Fatalf("ApplySampleAttributes: %v", err)
	}

	expiresAt := now.Add(3 * time.Hour)
	lot := "lot-1"
	supplyDesc := "supply-desc"
	supply := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
		ID:             "supply-1",
		SKU:            "SKU1",
		Name:           "Supply",
		QuantityOnHand: 5,
		Unit:           "unit",
		ReorderLevel:   1,
		Description:    &supplyDesc,
		LotNumber:      &lot,
		ExpiresAt:      &expiresAt,
		FacilityIDs:    []string{facility.ID},
		ProjectIDs:     []string{project.ID},
		CreatedAt:      now,
		UpdatedAt:      now,
		Attributes:     map[string]any{},
	}}
	if err := supply.ApplySupplyAttributes(map[string]any{"vendor": "acme"}); err != nil {
		t.Fatalf("ApplySupplyAttributes: %v", err)
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:                "treat-1",
		Name:              "Treatment",
		Status:            domain.TreatmentStatusPlanned,
		ProcedureID:       procedure.ID,
		DosagePlan:        "plan",
		AdministrationLog: []string{"admin"},
		AdverseEvents:     []string{"ae"},
		CohortIDs:         []string{cohort.ID},
		OrganismIDs:       []string{org1.ID},
		CreatedAt:         now,
		UpdatedAt:         now,
	}}

	snapshot := memory.Snapshot{
		Facilities:   map[string]domain.Facility{facility.ID: facility},
		Markers:      map[string]domain.GenotypeMarker{marker.ID: marker},
		Lines:        map[string]domain.Line{line.ID: line},
		Strains:      map[string]domain.Strain{strain.ID: strain},
		Housing:      map[string]domain.HousingUnit{housing.ID: housing},
		Protocols:    map[string]domain.Protocol{protocol.ID: protocol},
		Projects:     map[string]domain.Project{project.ID: project},
		Permits:      map[string]domain.Permit{permit.ID: permit},
		Cohorts:      map[string]domain.Cohort{cohort.ID: cohort},
		Breeding:     map[string]domain.BreedingUnit{breeding.ID: breeding},
		Organisms:    map[string]domain.Organism{org1.ID: org1, org2.ID: org2},
		Procedures:   map[string]domain.Procedure{procedure.ID: procedure},
		Observations: map[string]domain.Observation{observation.ID: observation},
		Samples:      map[string]domain.Sample{sample.ID: sample},
		Supplies:     map[string]domain.SupplyItem{supply.ID: supply},
		Treatments:   map[string]domain.Treatment{treatment.ID: treatment},
	}

	if err := persistNormalized(ctx, db, snapshot); err != nil {
		t.Fatalf("persistNormalized: %v", err)
	}
	loaded, err := loadNormalizedSnapshot(ctx, db)
	if err != nil {
		t.Fatalf("loadNormalizedSnapshot: %v", err)
	}

	gotLine := loaded.Lines[line.ID]
	if gotLine.DeprecatedAt == nil || gotLine.DeprecationReason == nil {
		t.Fatalf("expected line optional fields to persist, got %+v", gotLine)
	}
	gotSupply := loaded.Supplies[supply.ID]
	if gotSupply.Description == nil || gotSupply.LotNumber == nil || gotSupply.ExpiresAt == nil {
		t.Fatalf("expected supply optional fields to persist, got %+v", gotSupply)
	}
	if gotPermit := loaded.Permits[permit.ID]; gotPermit.Notes == nil {
		t.Fatalf("expected permit notes to persist")
	}
}

func loadFixtureSnapshot(t *testing.T) memory.Snapshot {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "..", "..", "testutil", "fixtures", "entity-model", "snapshot.json"))
	// #nosec G304 -- path is a fixed, test-only fixture location
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var snapshot memory.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return snapshot
}
