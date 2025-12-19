package postgres

import (
	"colonycore/internal/entitymodel/sqlbundle"
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewStoreAppliesDDLAndLoadsSnapshot(t *testing.T) {
	ctx := context.Background()
	db, conn := newStubDB()
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
	for _, stmt := range conn.execs {
		if strings.Contains(strings.ToUpper(stmt), "CREATE TABLE") {
			sawDDL = true
			break
		}
	}
	if !sawDDL {
		t.Fatalf("expected entity-model DDL to be applied, got execs: %v", conn.execs)
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
	if len(rec.execs) != len(expected) {
		t.Fatalf("expected %d DDL statements, got %d", len(expected), len(rec.execs))
	}
	for i, stmt := range expected {
		if strings.TrimSpace(rec.execs[i]) != strings.TrimSpace(stmt) {
			t.Fatalf("statement %d mismatch:\nwant: %s\ngot:  %s", i, strings.TrimSpace(stmt), strings.TrimSpace(rec.execs[i]))
		}
	}
}

func TestRunInTransactionPersistsState(t *testing.T) {
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) {
		db, _ := newStubDB()
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

	conn := store.dbConn()
	if conn == nil {
		t.Fatalf("expected stub connection")
	}
	var found bool
	for table, rows := range conn.tables {
		if table == "facilities" && len(rows) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected facility to persist in normalized tables")
	}
}

func TestLifecycleStatusesRoundTripNormalizedSnapshot(t *testing.T) {
	ctx := context.Background()
	db, _ := newStubDB()

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
	db, _ := newStubDB()
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
	db, conn := newStubDB()
	conn.failExec = true
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected ddl error")
	}
}

func TestApplyDDLStatementsError(t *testing.T) {
	db, conn := newStubDB()
	conn.failExec = true
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
			db, _ := newStubDB()
			err := persistNormalized(ctx, db, tc.snapshot)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPersistNormalizedCommitError(t *testing.T) {
	db, conn := newStubDB()
	conn.failCommit = true
	if err := persistNormalized(context.Background(), db, memory.Snapshot{}); err == nil || !strings.Contains(err.Error(), "commit") {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestLoadSnapshotValidatesRequiredJoins(t *testing.T) {
	now := time.Now().UTC()
	db, conn := newStubDB()
	conn.tables = map[string][]map[string]any{
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
	db, conn := newStubDB()
	conn.rowsErr = fmt.Errorf("row err")
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	if _, err := NewStore("ignored", domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected rows error")
	}
}

func TestLoadFacilitiesDecodeError(t *testing.T) {
	db, conn := newStubDB()
	conn.tables = map[string][]map[string]any{
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
	db, conn := newStubDB()
	conn.tables = map[string][]map[string]any{
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
	db, conn := newStubDB()
	conn.tables["breeding_units"] = []map[string]any{{
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
	db, conn := newStubDB()
	conn.tables["observations"] = []map[string]any{{
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
	db, conn := newStubDB()
	conn.tables["organisms"] = []map[string]any{{
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
	db, conn := newStubDB()
	conn.tables["samples"] = []map[string]any{{
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
	db, conn := newStubDB()
	conn.tables["supply_items"] = []map[string]any{{
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
	db, conn := newStubDB()
	conn.tables["treatments"] = []map[string]any{{
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
		setup func(*stubConn)
		fn    func(context.Context, *sql.DB) error
		want  string
	}{
		{
			name: "line marker references missing line",
			setup: func(conn *stubConn) {
				conn.tables["lines__genotype_marker_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["lines__genotype_marker_ids"] = nil
			},
			fn: func(ctx context.Context, db *sql.DB) error {
				return loadLineMarkers(ctx, db, map[string]domain.Line{"line-1": {Line: entitymodel.Line{ID: "line-1"}}})
			},
			want: "genotype_marker_ids",
		},
		{
			name: "strain marker references missing strain",
			setup: func(conn *stubConn) {
				conn.tables["strains__genotype_marker_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["facilities__project_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["permits__facility_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["permits__protocol_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["breeding_units__female_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["organisms__parent_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["procedures__organism_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["treatments__cohort_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["treatments__organism_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["supply_items__facility_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["projects__supply_item_ids"] = []map[string]any{{
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
			setup: func(conn *stubConn) {
				conn.tables["projects__supply_item_ids"] = nil
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
			setup: func(conn *stubConn) {
				conn.tables["supply_items__facility_ids"] = nil
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
			db, conn := newStubDB()
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
			db, conn := newStubDB()
			if err := persistNormalized(ctx, db, snapshot); err != nil {
				t.Fatalf("seed snapshot: %v", err)
			}
			conn.failTables = map[string]bool{tc.table: true}
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
	db, _ := newStubDB()
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
	db, conn := newStubDB()
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) { return db, nil })
	defer restore()
	store, err := NewStore("ignored", domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	conn.failExec = true
	if _, err := store.RunInTransaction(context.Background(), func(domain.Transaction) error { return nil }); err == nil {
		t.Fatalf("expected persistence error when exec fails")
	}
}

func TestRunInTransactionStopsOnUserError(t *testing.T) {
	restore := OverrideSQLOpen(func(_, _ string) (*sql.DB, error) {
		db, _ := newStubDB()
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
	if conn := store.dbConn(); conn != nil && len(conn.tables) != 0 {
		t.Fatalf("expected no persistence when user fn errors")
	}
}

func TestPersistNormalizedBeginTxError(t *testing.T) {
	db, conn := newStubDB()
	conn.failBegin = true
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
			db, conn := newStubDB()
			conn.failTables = map[string]bool{tc.table: true}
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
	db, _ := newStubDB()
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

// --- stub driver helpers ---

type stubDriver struct {
	conn *stubConn
}

func (d *stubDriver) Open(string) (driver.Conn, error) {
	return d.conn, nil
}

type stubConn struct {
	execs      []string
	tables     map[string][]map[string]any
	failExec   bool
	failBegin  bool
	rowsErr    error
	failTables map[string]bool
	failCommit bool
}

func newStubDB() (*sql.DB, *stubConn) {
	conn := &stubConn{tables: make(map[string][]map[string]any)}
	name := fmt.Sprintf("stubpg%d", time.Now().UnixNano())
	sql.Register(name, &stubDriver{conn: conn})
	db, err := sql.Open(name, "stub")
	if err != nil {
		panic(err)
	}
	return db, conn
}

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("not implemented") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *stubConn) Ping(_ context.Context) error {
	if c.failExec {
		return fmt.Errorf("ping fail")
	}
	return nil
}

func (c *stubConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if c.failBegin {
		return nil, fmt.Errorf("begin fail")
	}
	return &stubTx{conn: c}, nil
}

func (c *stubConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.execs = append(c.execs, query)
	if c.failExec {
		return nil, fmt.Errorf("exec fail")
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "TRUNCATE TABLE") {
		c.tables = make(map[string][]map[string]any)
		return driver.RowsAffected(0), nil
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "INSERT INTO") {
		table, cols, err := parseInsert(query)
		if err != nil {
			return nil, err
		}
		if c.failTables != nil && c.failTables[table] {
			return nil, fmt.Errorf("exec fail for %s", table)
		}
		if len(cols) != len(args) {
			return nil, fmt.Errorf("column/arg mismatch for %s", table)
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = args[i].Value
		}
		c.tables[table] = append(c.tables[table], row)
		return driver.RowsAffected(1), nil
	}
	return driver.RowsAffected(1), nil
}

func (c *stubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.tables == nil {
		c.tables = make(map[string][]map[string]any)
	}
	table, cols, err := parseSelect(query)
	if err != nil {
		return nil, err
	}
	if c.failTables != nil && c.failTables[table] {
		return nil, fmt.Errorf("query fail for %s", table)
	}
	tableRows := c.tables[table]
	values := make([][]driver.Value, 0, len(tableRows))
	for _, row := range tableRows {
		vals := make([]driver.Value, len(cols))
		for i, col := range cols {
			vals[i] = row[col]
		}
		values = append(values, vals)
	}
	return &stubRows{
		cols: cols,
		rows: values,
		err:  c.rowsErr,
	}, nil
}

type stubTx struct {
	conn *stubConn
}

func (t *stubTx) Commit() error {
	if t.conn.failCommit {
		return fmt.Errorf("commit fail")
	}
	return nil
}
func (t *stubTx) Rollback() error { return nil }

type stubRows struct {
	cols []string
	rows [][]driver.Value
	idx  int
	err  error
}

func (r *stubRows) Columns() []string { return r.cols }
func (r *stubRows) Close() error      { return nil }

func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		if r.err != nil {
			return r.err
		}
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func parseInsert(query string) (string, []string, error) {
	up := strings.ToUpper(query)
	intoIdx := strings.Index(up, "INTO ")
	if intoIdx == -1 {
		return "", nil, fmt.Errorf("cannot parse insert: %s", query)
	}
	rest := strings.TrimSpace(query[intoIdx+len("INTO "):])
	open := strings.Index(rest, "(")
	closeIdx := strings.Index(rest, ")")
	if open == -1 || closeIdx == -1 || closeIdx <= open {
		return "", nil, fmt.Errorf("cannot parse insert: %s", query)
	}
	table := strings.ToLower(strings.TrimSpace(rest[:open]))
	cols := splitColumns(rest[open+1 : closeIdx])
	return table, cols, nil
}

func parseSelect(query string) (string, []string, error) {
	lower := strings.ToLower(query)
	selectPrefix := "select "
	fromToken := " from "
	if !strings.HasPrefix(lower, selectPrefix) {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	fromIdx := strings.Index(lower, fromToken)
	if fromIdx == -1 {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	cols := query[len(selectPrefix):fromIdx]
	table := strings.TrimSpace(query[fromIdx+len(fromToken):])
	if table == "" {
		return "", nil, fmt.Errorf("cannot parse select: %s", query)
	}
	table = strings.Fields(table)[0]
	return strings.ToLower(table), splitColumns(cols), nil
}

func splitColumns(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.ToLower(strings.TrimSpace(part)))
	}
	return out
}

type recordingExec struct {
	execs []string
}

func (r *recordingExec) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	r.execs = append(r.execs, query)
	return driver.RowsAffected(1), nil
}

func (r *recordingExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("QueryContext not implemented")
}

// dbConn exposes the stub connection to tests without leaking the driver types elsewhere.
func (s *Store) dbConn() *stubConn {
	if s == nil || s.db == nil {
		return nil
	}
	if connector, ok := s.db.Driver().(*stubDriver); ok {
		return connector.conn
	}
	return nil
}
