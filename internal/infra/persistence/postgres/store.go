// Package postgres provides a Postgres-backed persistent store that applies the
// generated entity-model DDL on startup and issues normalized CRUD statements directly.
package postgres

import (
	"colonycore/internal/entitymodel/sqlbundle"
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx as a database/sql driver
)

// Compile-time contract assertion ensuring the store satisfies the domain interface.
var _ domain.PersistentStore = (*Store)(nil)

const (
	defaultDriver = "pgx"
	// Default DSN keeps parity with OpenPersistentStore defaults while allowing overrides via env.
	defaultDSN = "postgres://localhost/colonycore?sslmode=disable"
)

var (
	sqlOpen = sql.Open
	openMu  sync.Mutex
)

// Store persists state to Postgres while executing CRUD directly against the generated DDL.
// It still uses the in-memory transaction engine for rule evaluation but commits deltas to
// the normalized tables instead of snapshot mirroring.
type Store struct {
	db     *sql.DB
	engine *domain.RulesEngine
	mu     sync.Mutex
	cache  memory.Snapshot
}

// NewStore opens a Postgres-backed store using the provided DSN (falls back to defaultDSN).
// It applies the generated entity-model DDL and hydrates an in-memory snapshot cache from Postgres.
func NewStore(dsn string, engine *domain.RulesEngine) (*Store, error) {
	if dsn == "" {
		dsn = defaultDSN
	}
	openMu.Lock()
	db, err := sqlOpen(defaultDriver, dsn)
	openMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := applyEntityModelDDL(ctx, db); err != nil {
		return nil, err
	}
	cache, err := loadNormalizedSnapshot(ctx, db)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:     db,
		engine: engine,
		cache:  cache,
	}, nil
}

// RunInTransaction evaluates the user-supplied function against an in-memory transaction
// and persists the resulting delta directly to the normalized schema inside a single DB transaction.
func (s *Store) RunInTransaction(ctx context.Context, fn func(domain.Transaction) error) (domain.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	before, err := loadNormalizedSnapshot(ctx, tx)
	if err != nil {
		return domain.Result{}, err
	}

	mem := memory.NewStore(s.engine)
	mem.ImportState(before)

	res, err := mem.RunInTransaction(ctx, fn)
	if err != nil {
		return res, err
	}
	after := mem.ExportState()

	if err := applySnapshotDelta(ctx, tx, before, after); err != nil {
		return res, err
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	committed = true
	s.cache = after
	return res, nil
}

// DB exposes the underlying sql.DB for integration testing hooks.
func (s *Store) DB() *sql.DB { return s.db }

func applyEntityModelDDL(ctx context.Context, db *sql.DB) error {
	return applyDDLStatements(ctx, db, sqlbundle.Postgres())
}

// snapshotOrCache returns the latest database snapshot or falls back to the last good cache.
func (s *Store) snapshotOrCache(ctx context.Context) memory.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := loadNormalizedSnapshot(ctx, s.db)
	if err == nil {
		s.cache = snap
		return snap
	}
	return cloneSnapshot(s.cache)
}

// View executes fn against a read-only snapshot of the Postgres-backed state.
func (s *Store) View(ctx context.Context, fn func(domain.TransactionView) error) error {
	snapshot := s.snapshotOrCache(ctx)
	mem := memory.NewStore(s.engine)
	mem.ImportState(snapshot)
	return mem.View(ctx, fn)
}

// GetOrganism returns an organism by ID.
func (s *Store) GetOrganism(id string) (domain.Organism, bool) {
	snap := s.snapshotOrCache(context.Background())
	o, ok := snap.Organisms[id]
	return o, ok
}

// ListOrganisms returns all organisms.
func (s *Store) ListOrganisms() []domain.Organism {
	return mapValues(s.snapshotOrCache(context.Background()).Organisms)
}

// GetHousingUnit returns a housing unit by ID.
func (s *Store) GetHousingUnit(id string) (domain.HousingUnit, bool) {
	snap := s.snapshotOrCache(context.Background())
	h, ok := snap.Housing[id]
	return h, ok
}

// ListHousingUnits returns all housing units.
func (s *Store) ListHousingUnits() []domain.HousingUnit {
	return mapValues(s.snapshotOrCache(context.Background()).Housing)
}

// GetFacility returns a facility by ID.
func (s *Store) GetFacility(id string) (domain.Facility, bool) {
	snap := s.snapshotOrCache(context.Background())
	f, ok := snap.Facilities[id]
	return f, ok
}

// ListFacilities returns all facilities.
func (s *Store) ListFacilities() []domain.Facility {
	return mapValues(s.snapshotOrCache(context.Background()).Facilities)
}

// GetLine returns a line by ID.
func (s *Store) GetLine(id string) (domain.Line, bool) {
	snap := s.snapshotOrCache(context.Background())
	l, ok := snap.Lines[id]
	return l, ok
}

// ListLines returns all lines.
func (s *Store) ListLines() []domain.Line {
	return mapValues(s.snapshotOrCache(context.Background()).Lines)
}

// GetStrain returns a strain by ID.
func (s *Store) GetStrain(id string) (domain.Strain, bool) {
	snap := s.snapshotOrCache(context.Background())
	st, ok := snap.Strains[id]
	return st, ok
}

// ListStrains returns all strains.
func (s *Store) ListStrains() []domain.Strain {
	return mapValues(s.snapshotOrCache(context.Background()).Strains)
}

// GetGenotypeMarker returns a genotype marker by ID.
func (s *Store) GetGenotypeMarker(id string) (domain.GenotypeMarker, bool) {
	snap := s.snapshotOrCache(context.Background())
	gm, ok := snap.Markers[id]
	return gm, ok
}

// ListGenotypeMarkers returns all genotype markers.
func (s *Store) ListGenotypeMarkers() []domain.GenotypeMarker {
	return mapValues(s.snapshotOrCache(context.Background()).Markers)
}

// ListCohorts returns all cohorts.
func (s *Store) ListCohorts() []domain.Cohort {
	return mapValues(s.snapshotOrCache(context.Background()).Cohorts)
}

// ListTreatments returns all treatments.
func (s *Store) ListTreatments() []domain.Treatment {
	return mapValues(s.snapshotOrCache(context.Background()).Treatments)
}

// ListObservations returns all observations.
func (s *Store) ListObservations() []domain.Observation {
	return mapValues(s.snapshotOrCache(context.Background()).Observations)
}

// ListSamples returns all samples.
func (s *Store) ListSamples() []domain.Sample {
	return mapValues(s.snapshotOrCache(context.Background()).Samples)
}

// ListProtocols returns all protocols.
func (s *Store) ListProtocols() []domain.Protocol {
	return mapValues(s.snapshotOrCache(context.Background()).Protocols)
}

// GetPermit returns a permit by ID.
func (s *Store) GetPermit(id string) (domain.Permit, bool) {
	snap := s.snapshotOrCache(context.Background())
	p, ok := snap.Permits[id]
	return p, ok
}

// ListPermits returns all permits.
func (s *Store) ListPermits() []domain.Permit {
	return mapValues(s.snapshotOrCache(context.Background()).Permits)
}

// ListProjects returns all projects.
func (s *Store) ListProjects() []domain.Project {
	return mapValues(s.snapshotOrCache(context.Background()).Projects)
}

// ListBreedingUnits returns all breeding units.
func (s *Store) ListBreedingUnits() []domain.BreedingUnit {
	return mapValues(s.snapshotOrCache(context.Background()).Breeding)
}

// ListProcedures returns all procedures.
func (s *Store) ListProcedures() []domain.Procedure {
	return mapValues(s.snapshotOrCache(context.Background()).Procedures)
}

// ListSupplyItems returns all supply items.
func (s *Store) ListSupplyItems() []domain.SupplyItem {
	return mapValues(s.snapshotOrCache(context.Background()).Supplies)
}

func mapValues[T any](m map[string]T) []T {
	out := make([]T, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func cloneSnapshot(s memory.Snapshot) memory.Snapshot {
	out := memory.Snapshot{
		Organisms:    make(map[string]memory.Organism, len(s.Organisms)),
		Cohorts:      make(map[string]memory.Cohort, len(s.Cohorts)),
		Housing:      make(map[string]memory.HousingUnit, len(s.Housing)),
		Facilities:   make(map[string]memory.Facility, len(s.Facilities)),
		Breeding:     make(map[string]memory.BreedingUnit, len(s.Breeding)),
		Lines:        make(map[string]memory.Line, len(s.Lines)),
		Strains:      make(map[string]memory.Strain, len(s.Strains)),
		Markers:      make(map[string]memory.GenotypeMarker, len(s.Markers)),
		Procedures:   make(map[string]memory.Procedure, len(s.Procedures)),
		Treatments:   make(map[string]memory.Treatment, len(s.Treatments)),
		Observations: make(map[string]memory.Observation, len(s.Observations)),
		Samples:      make(map[string]memory.Sample, len(s.Samples)),
		Protocols:    make(map[string]memory.Protocol, len(s.Protocols)),
		Permits:      make(map[string]memory.Permit, len(s.Permits)),
		Projects:     make(map[string]memory.Project, len(s.Projects)),
		Supplies:     make(map[string]memory.SupplyItem, len(s.Supplies)),
	}
	for k, v := range s.Organisms {
		out.Organisms[k] = v
	}
	for k, v := range s.Cohorts {
		out.Cohorts[k] = v
	}
	for k, v := range s.Housing {
		out.Housing[k] = v
	}
	for k, v := range s.Facilities {
		out.Facilities[k] = v
	}
	for k, v := range s.Breeding {
		out.Breeding[k] = v
	}
	for k, v := range s.Lines {
		out.Lines[k] = v
	}
	for k, v := range s.Strains {
		out.Strains[k] = v
	}
	for k, v := range s.Markers {
		out.Markers[k] = v
	}
	for k, v := range s.Procedures {
		out.Procedures[k] = v
	}
	for k, v := range s.Treatments {
		out.Treatments[k] = v
	}
	for k, v := range s.Observations {
		out.Observations[k] = v
	}
	for k, v := range s.Samples {
		out.Samples[k] = v
	}
	for k, v := range s.Protocols {
		out.Protocols[k] = v
	}
	for k, v := range s.Permits {
		out.Permits[k] = v
	}
	for k, v := range s.Projects {
		out.Projects[k] = v
	}
	for k, v := range s.Supplies {
		out.Supplies[k] = v
	}
	return out
}

// ImportState replaces the normalized data with the provided snapshot (primarily for tests).
func (s *Store) ImportState(snapshot memory.Snapshot) {
	if err := persistNormalized(context.Background(), s.db, snapshot); err != nil {
		panic(fmt.Errorf("postgres import state: %w", err))
	}
	s.cache = cloneSnapshot(snapshot)
}

// ExportState returns the current normalized snapshot (primarily for tests).
func (s *Store) ExportState() memory.Snapshot {
	snap, err := loadNormalizedSnapshot(context.Background(), s.db)
	if err != nil {
		panic(fmt.Errorf("postgres export state: %w", err))
	}
	s.cache = snap
	return snap
}

// RulesEngine exposes the configured rules engine (test helper for parity with other stores).
func (s *Store) RulesEngine() *domain.RulesEngine {
	return s.engine
}

type delta[T any] struct {
	created map[string]T
	updated map[string]T
	deleted []string
}

func diffMaps[T any](before, after map[string]T) delta[T] {
	d := delta[T]{
		created: make(map[string]T),
		updated: make(map[string]T),
	}
	for id, afterVal := range after {
		if prev, ok := before[id]; !ok {
			d.created[id] = afterVal
		} else if !reflect.DeepEqual(prev, afterVal) {
			d.updated[id] = afterVal
		}
	}
	for id := range before {
		if _, ok := after[id]; !ok {
			d.deleted = append(d.deleted, id)
		}
	}
	return d
}

func mergeMaps[T any](first, second map[string]T) map[string]T {
	if len(first) == 0 && len(second) == 0 {
		return nil
	}
	out := make(map[string]T, len(first)+len(second))
	for k, v := range first {
		out[k] = v
	}
	for k, v := range second {
		out[k] = v
	}
	return out
}

// applySnapshotDelta persists the difference between two snapshots inside an active SQL transaction.
func applySnapshotDelta(ctx context.Context, exec execQuerier, before, after memory.Snapshot) error {
	facilities := diffMaps(before.Facilities, after.Facilities)
	markers := diffMaps(before.Markers, after.Markers)
	lines := diffMaps(before.Lines, after.Lines)
	strains := diffMaps(before.Strains, after.Strains)
	housing := diffMaps(before.Housing, after.Housing)
	protocols := diffMaps(before.Protocols, after.Protocols)
	projects := diffMaps(before.Projects, after.Projects)
	permits := diffMaps(before.Permits, after.Permits)
	cohorts := diffMaps(before.Cohorts, after.Cohorts)
	breeding := diffMaps(before.Breeding, after.Breeding)
	organisms := diffMaps(before.Organisms, after.Organisms)
	procedures := diffMaps(before.Procedures, after.Procedures)
	observations := diffMaps(before.Observations, after.Observations)
	samples := diffMaps(before.Samples, after.Samples)
	supplies := diffMaps(before.Supplies, after.Supplies)
	treatments := diffMaps(before.Treatments, after.Treatments)

	// Deletes from leaf to root to satisfy FK constraints.
	if err := deleteTreatments(ctx, exec, treatments.deleted); err != nil {
		return err
	}
	if err := deleteSupplyItems(ctx, exec, supplies.deleted); err != nil {
		return err
	}
	if err := deleteSamples(ctx, exec, samples.deleted); err != nil {
		return err
	}
	if err := deleteObservations(ctx, exec, observations.deleted); err != nil {
		return err
	}
	if err := deleteProcedures(ctx, exec, procedures.deleted); err != nil {
		return err
	}
	if err := deleteBreedingUnits(ctx, exec, breeding.deleted); err != nil {
		return err
	}
	if err := deleteOrganisms(ctx, exec, organisms.deleted); err != nil {
		return err
	}
	if err := deleteCohorts(ctx, exec, cohorts.deleted); err != nil {
		return err
	}
	if err := deletePermits(ctx, exec, permits.deleted); err != nil {
		return err
	}
	if err := deleteProjects(ctx, exec, projects.deleted); err != nil {
		return err
	}
	if err := deleteProtocols(ctx, exec, protocols.deleted); err != nil {
		return err
	}
	if err := deleteHousingUnits(ctx, exec, housing.deleted); err != nil {
		return err
	}
	if err := deleteStrains(ctx, exec, strains.deleted); err != nil {
		return err
	}
	if err := deleteLines(ctx, exec, lines.deleted); err != nil {
		return err
	}
	if err := deleteGenotypeMarkers(ctx, exec, markers.deleted); err != nil {
		return err
	}
	if err := deleteFacilities(ctx, exec, facilities.deleted); err != nil {
		return err
	}

	// Upserts from root to leaf to satisfy FK constraints.
	if err := insertFacilities(ctx, exec, mergeMaps(facilities.created, facilities.updated)); err != nil {
		return err
	}
	if err := insertGenotypeMarkers(ctx, exec, mergeMaps(markers.created, markers.updated)); err != nil {
		return err
	}
	if err := insertLines(ctx, exec, mergeMaps(lines.created, lines.updated)); err != nil {
		return err
	}
	if err := insertStrains(ctx, exec, mergeMaps(strains.created, strains.updated)); err != nil {
		return err
	}
	if err := insertHousingUnits(ctx, exec, mergeMaps(housing.created, housing.updated)); err != nil {
		return err
	}
	if err := insertProtocols(ctx, exec, mergeMaps(protocols.created, protocols.updated)); err != nil {
		return err
	}
	if err := insertProjects(ctx, exec, mergeMaps(projects.created, projects.updated)); err != nil {
		return err
	}
	if err := insertPermits(ctx, exec, mergeMaps(permits.created, permits.updated)); err != nil {
		return err
	}
	if err := insertCohorts(ctx, exec, mergeMaps(cohorts.created, cohorts.updated)); err != nil {
		return err
	}
	if err := insertBreedingUnits(ctx, exec, mergeMaps(breeding.created, breeding.updated)); err != nil {
		return err
	}
	if err := insertOrganisms(ctx, exec, mergeMaps(organisms.created, organisms.updated)); err != nil {
		return err
	}
	if err := insertProcedures(ctx, exec, mergeMaps(procedures.created, procedures.updated)); err != nil {
		return err
	}
	if err := insertObservations(ctx, exec, mergeMaps(observations.created, observations.updated)); err != nil {
		return err
	}
	if err := insertSamples(ctx, exec, mergeMaps(samples.created, samples.updated)); err != nil {
		return err
	}
	if err := insertSupplyItems(ctx, exec, mergeMaps(supplies.created, supplies.updated)); err != nil {
		return err
	}
	if err := insertTreatments(ctx, exec, mergeMaps(treatments.created, treatments.updated)); err != nil {
		return err
	}
	return nil
}

// OverrideSQLOpen swaps the sqlOpen function for tests and returns a restore function.
func OverrideSQLOpen(fn func(driverName, dataSourceName string) (*sql.DB, error)) func() {
	openMu.Lock()
	defer openMu.Unlock()
	prev := sqlOpen
	sqlOpen = fn
	return func() {
		openMu.Lock()
		defer openMu.Unlock()
		sqlOpen = prev
	}
}

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func applyDDLStatements(ctx context.Context, db execQuerier, ddl string) error {
	for _, stmt := range sqlbundle.SplitStatements(ddl) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute ddl: %w", err)
		}
	}
	return nil
}

func persistNormalized(ctx context.Context, db *sql.DB, snapshot memory.Snapshot) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, truncateAllTablesSQL); err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}

	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"insert facilities", func(ctx context.Context) error { return insertFacilities(ctx, tx, snapshot.Facilities) }},
		{"insert genotype markers", func(ctx context.Context) error { return insertGenotypeMarkers(ctx, tx, snapshot.Markers) }},
		{"insert lines", func(ctx context.Context) error { return insertLines(ctx, tx, snapshot.Lines) }},
		{"insert strains", func(ctx context.Context) error { return insertStrains(ctx, tx, snapshot.Strains) }},
		{"insert housing", func(ctx context.Context) error { return insertHousingUnits(ctx, tx, snapshot.Housing) }},
		{"insert protocols", func(ctx context.Context) error { return insertProtocols(ctx, tx, snapshot.Protocols) }},
		{"insert projects", func(ctx context.Context) error { return insertProjects(ctx, tx, snapshot.Projects) }},
		{"insert permits", func(ctx context.Context) error { return insertPermits(ctx, tx, snapshot.Permits) }},
		{"insert cohorts", func(ctx context.Context) error { return insertCohorts(ctx, tx, snapshot.Cohorts) }},
		{"insert breeding units", func(ctx context.Context) error { return insertBreedingUnits(ctx, tx, snapshot.Breeding) }},
		{"insert organisms", func(ctx context.Context) error { return insertOrganisms(ctx, tx, snapshot.Organisms) }},
		{"insert procedures", func(ctx context.Context) error { return insertProcedures(ctx, tx, snapshot.Procedures) }},
		{"insert observations", func(ctx context.Context) error { return insertObservations(ctx, tx, snapshot.Observations) }},
		{"insert samples", func(ctx context.Context) error { return insertSamples(ctx, tx, snapshot.Samples) }},
		{"insert supply items", func(ctx context.Context) error { return insertSupplyItems(ctx, tx, snapshot.Supplies) }},
		{"insert treatments", func(ctx context.Context) error { return insertTreatments(ctx, tx, snapshot.Treatments) }},
	}
	for _, step := range steps {
		if err := step.fn(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// --- delete helpers ---

func deleteFacilities(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteFacilitiesProjectsSQL, id); err != nil {
			return fmt.Errorf("delete facility %s project links: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteFacilitySQL, id); err != nil {
			return fmt.Errorf("delete facility %s: %w", id, err)
		}
	}
	return nil
}

func deleteGenotypeMarkers(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteGenotypeMarkerSQL, id); err != nil {
			return fmt.Errorf("delete genotype marker %s: %w", id, err)
		}
	}
	return nil
}

func deleteLines(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteLineMarkersSQL, id); err != nil {
			return fmt.Errorf("delete line %s markers: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteLineSQL, id); err != nil {
			return fmt.Errorf("delete line %s: %w", id, err)
		}
	}
	return nil
}

func deleteStrains(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteStrainMarkersSQL, id); err != nil {
			return fmt.Errorf("delete strain %s markers: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteStrainSQL, id); err != nil {
			return fmt.Errorf("delete strain %s: %w", id, err)
		}
	}
	return nil
}

func deleteHousingUnits(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteHousingSQL, id); err != nil {
			return fmt.Errorf("delete housing %s: %w", id, err)
		}
	}
	return nil
}

func deleteProtocols(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteProtocolSQL, id); err != nil {
			return fmt.Errorf("delete protocol %s: %w", id, err)
		}
	}
	return nil
}

func deleteProjects(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteProjectFacilitiesSQL, id); err != nil {
			return fmt.Errorf("delete project %s facilities: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectProtocolsSQL, id); err != nil {
			return fmt.Errorf("delete project %s protocols: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectSuppliesSQL, id); err != nil {
			return fmt.Errorf("delete project %s supplies: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectSQL, id); err != nil {
			return fmt.Errorf("delete project %s: %w", id, err)
		}
	}
	return nil
}

func deletePermits(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deletePermitFacilitiesSQL, id); err != nil {
			return fmt.Errorf("delete permit %s facilities: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deletePermitProtocolsSQL, id); err != nil {
			return fmt.Errorf("delete permit %s protocols: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deletePermitSQL, id); err != nil {
			return fmt.Errorf("delete permit %s: %w", id, err)
		}
	}
	return nil
}

func deleteCohorts(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteCohortSQL, id); err != nil {
			return fmt.Errorf("delete cohort %s: %w", id, err)
		}
	}
	return nil
}

func deleteBreedingUnits(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteBreedingFemalesSQL, id); err != nil {
			return fmt.Errorf("delete breeding unit %s females: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteBreedingMalesSQL, id); err != nil {
			return fmt.Errorf("delete breeding unit %s males: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteBreedingSQL, id); err != nil {
			return fmt.Errorf("delete breeding unit %s: %w", id, err)
		}
	}
	return nil
}

func deleteOrganisms(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteOrganismParentsSQL, id); err != nil {
			return fmt.Errorf("delete organism %s parents: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteOrganismSQL, id); err != nil {
			return fmt.Errorf("delete organism %s: %w", id, err)
		}
	}
	return nil
}

func deleteProcedures(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteProcedureOrganismsSQL, id); err != nil {
			return fmt.Errorf("delete procedure %s organisms: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProcedureSQL, id); err != nil {
			return fmt.Errorf("delete procedure %s: %w", id, err)
		}
	}
	return nil
}

func deleteObservations(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteObservationSQL, id); err != nil {
			return fmt.Errorf("delete observation %s: %w", id, err)
		}
	}
	return nil
}

func deleteSamples(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteSampleSQL, id); err != nil {
			return fmt.Errorf("delete sample %s: %w", id, err)
		}
	}
	return nil
}

func deleteSupplyItems(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteSupplyFacilitiesSQL, id); err != nil {
			return fmt.Errorf("delete supply item %s facilities: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectSuppliesBySupplySQL, id); err != nil {
			return fmt.Errorf("delete supply item %s projects: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteSupplySQL, id); err != nil {
			return fmt.Errorf("delete supply item %s: %w", id, err)
		}
	}
	return nil
}

func deleteTreatments(ctx context.Context, exec execQuerier, ids []string) error {
	for _, id := range ids {
		if _, err := exec.ExecContext(ctx, deleteTreatmentCohortsSQL, id); err != nil {
			return fmt.Errorf("delete treatment %s cohorts: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteTreatmentOrganismsSQL, id); err != nil {
			return fmt.Errorf("delete treatment %s organisms: %w", id, err)
		}
		if _, err := exec.ExecContext(ctx, deleteTreatmentSQL, id); err != nil {
			return fmt.Errorf("delete treatment %s: %w", id, err)
		}
	}
	return nil
}

func loadNormalizedSnapshot(ctx context.Context, db execQuerier) (memory.Snapshot, error) {
	facilities, err := loadFacilities(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	markers, err := loadGenotypeMarkers(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	lines, err := loadLines(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadLineMarkers(ctx, db, lines); err != nil {
		return memory.Snapshot{}, err
	}
	strains, err := loadStrains(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadStrainMarkers(ctx, db, strains); err != nil {
		return memory.Snapshot{}, err
	}
	housing, err := loadHousingUnits(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	protocols, err := loadProtocols(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	projects, err := loadProjects(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadProjectFacilities(ctx, db, projects, facilities); err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadProjectProtocols(ctx, db, projects); err != nil {
		return memory.Snapshot{}, err
	}
	permits, err := loadPermits(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadPermitFacilities(ctx, db, permits); err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadPermitProtocols(ctx, db, permits); err != nil {
		return memory.Snapshot{}, err
	}
	cohorts, err := loadCohorts(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	breeding, err := loadBreedingUnits(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadBreedingUnitMembers(ctx, db, breeding); err != nil {
		return memory.Snapshot{}, err
	}
	organisms, err := loadOrganisms(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadOrganismParents(ctx, db, organisms); err != nil {
		return memory.Snapshot{}, err
	}
	procedures, err := loadProcedures(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadProcedureOrganisms(ctx, db, procedures); err != nil {
		return memory.Snapshot{}, err
	}
	observations, err := loadObservations(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	samples, err := loadSamples(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	supplyItems, err := loadSupplyItems(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadSupplyItemFacilities(ctx, db, supplyItems); err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadProjectSupplyItems(ctx, db, projects, supplyItems); err != nil {
		return memory.Snapshot{}, err
	}
	treatments, err := loadTreatments(ctx, db)
	if err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadTreatmentCohorts(ctx, db, treatments); err != nil {
		return memory.Snapshot{}, err
	}
	if err := loadTreatmentOrganisms(ctx, db, treatments); err != nil {
		return memory.Snapshot{}, err
	}

	return memory.Snapshot{
		Facilities:   facilities,
		Markers:      markers,
		Lines:        lines,
		Strains:      strains,
		Housing:      housing,
		Protocols:    protocols,
		Projects:     projects,
		Permits:      permits,
		Cohorts:      cohorts,
		Breeding:     breeding,
		Organisms:    organisms,
		Procedures:   procedures,
		Observations: observations,
		Samples:      samples,
		Supplies:     supplyItems,
		Treatments:   treatments,
	}, nil
}

// --- insert helpers ---

const truncateAllTablesSQL = `
TRUNCATE TABLE
    treatments__organism_ids,
    treatments__cohort_ids,
    treatments,
    supply_items__facility_ids,
    projects__supply_item_ids,
    supply_items,
    samples,
    procedures__organism_ids,
    organisms__parent_ids,
    organisms,
    breeding_units__female_ids,
    breeding_units__male_ids,
    breeding_units,
    observations,
    procedures,
    cohorts,
    permits__protocol_ids,
    permits__facility_ids,
    permits,
    projects__protocol_ids,
    facilities__project_ids,
    projects,
    protocols,
    housing_units,
    strains__genotype_marker_ids,
    strains,
    lines__genotype_marker_ids,
    lines,
    genotype_markers,
    facilities
CASCADE`

func insertFacilities(ctx context.Context, exec execQuerier, facilities map[string]domain.Facility) error {
	keys := sortedKeys(facilities)
	for _, id := range keys {
		f := facilities[id]
		env, err := marshalJSONNullable((&f).EnvironmentBaselines())
		if err != nil {
			return fmt.Errorf("marshal facility environment_baselines: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertFacilitySQL,
			f.ID, f.Code, f.Name, f.Zone, f.AccessPolicy, f.CreatedAt, f.UpdatedAt, env,
		); err != nil {
			return fmt.Errorf("insert facility %s: %w", f.ID, err)
		}
	}
	return nil
}

func insertGenotypeMarkers(ctx context.Context, exec execQuerier, markers map[string]domain.GenotypeMarker) error {
	keys := sortedKeys(markers)
	for _, id := range keys {
		m := markers[id]
		alleles, err := marshalJSONRequired("genotype_marker.alleles", m.Alleles)
		if err != nil {
			return err
		}
		if _, err := exec.ExecContext(ctx, insertGenotypeMarkerSQL,
			m.ID, m.Name, m.Locus, alleles, m.AssayMethod, m.Interpretation, m.Version, m.CreatedAt, m.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert genotype marker %s: %w", m.ID, err)
		}
	}
	return nil
}

func insertLines(ctx context.Context, exec execQuerier, lines map[string]domain.Line) error {
	keys := sortedKeys(lines)
	for _, id := range keys {
		line := lines[id]
		if len(line.GenotypeMarkerIDs) == 0 {
			return fmt.Errorf("line %s missing required genotype_marker_ids", line.ID)
		}
		if _, err := exec.ExecContext(ctx, deleteLineMarkersSQL, line.ID); err != nil {
			return fmt.Errorf("clear line %s markers: %w", line.ID, err)
		}
		defaultAttrs, err := marshalJSONNullable((&line).DefaultAttributes())
		if err != nil {
			return fmt.Errorf("marshal line default_attributes: %w", err)
		}
		overrides, err := marshalJSONNullable((&line).ExtensionOverrides())
		if err != nil {
			return fmt.Errorf("marshal line extension_overrides: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertLineSQL,
			line.ID, line.Code, line.Name, line.Origin, line.Description, defaultAttrs, overrides, line.DeprecatedAt, line.DeprecationReason, line.CreatedAt, line.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert line %s: %w", line.ID, err)
		}
		for _, markerID := range line.GenotypeMarkerIDs {
			if _, err := exec.ExecContext(ctx, insertLineMarkerSQL, line.ID, markerID); err != nil {
				return fmt.Errorf("insert line %s genotype_marker_id %s: %w", line.ID, markerID, err)
			}
		}
	}
	return nil
}

func insertStrains(ctx context.Context, exec execQuerier, strains map[string]domain.Strain) error {
	keys := sortedKeys(strains)
	for _, id := range keys {
		strain := strains[id]
		if strain.LineID == "" {
			return fmt.Errorf("strain %s missing required line_id", strain.ID)
		}
		if _, err := exec.ExecContext(ctx, deleteStrainMarkersSQL, strain.ID); err != nil {
			return fmt.Errorf("clear strain %s markers: %w", strain.ID, err)
		}
		if _, err := exec.ExecContext(ctx, insertStrainSQL,
			strain.ID, strain.Code, strain.Name, strain.LineID, strain.Description, strain.Generation, strain.RetiredAt, strain.RetirementReason, strain.CreatedAt, strain.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert strain %s: %w", strain.ID, err)
		}
		for _, markerID := range strain.GenotypeMarkerIDs {
			if _, err := exec.ExecContext(ctx, insertStrainMarkerSQL, strain.ID, markerID); err != nil {
				return fmt.Errorf("insert strain %s genotype_marker_id %s: %w", strain.ID, markerID, err)
			}
		}
	}
	return nil
}

func insertHousingUnits(ctx context.Context, exec execQuerier, housing map[string]domain.HousingUnit) error {
	keys := sortedKeys(housing)
	for _, id := range keys {
		h := housing[id]
		if h.FacilityID == "" {
			return fmt.Errorf("housing %s missing required facility_id", h.ID)
		}
		if _, err := exec.ExecContext(ctx, insertHousingSQL,
			h.ID, h.FacilityID, h.Name, h.Capacity, h.Environment, h.State, h.CreatedAt, h.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert housing %s: %w", h.ID, err)
		}
	}
	return nil
}

func insertProtocols(ctx context.Context, exec execQuerier, protocols map[string]domain.Protocol) error {
	keys := sortedKeys(protocols)
	for _, id := range keys {
		p := protocols[id]
		if _, err := exec.ExecContext(ctx, insertProtocolSQL,
			p.ID, p.Code, p.Title, p.Description, p.MaxSubjects, p.Status, p.CreatedAt, p.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert protocol %s: %w", p.ID, err)
		}
	}
	return nil
}

func insertProjects(ctx context.Context, exec execQuerier, projects map[string]domain.Project) error {
	keys := sortedKeys(projects)
	for _, id := range keys {
		p := projects[id]
		if len(p.FacilityIDs) == 0 {
			return fmt.Errorf("project %s missing required facility_ids", p.ID)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectFacilitiesSQL, p.ID); err != nil {
			return fmt.Errorf("clear project %s facilities: %w", p.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectProtocolsSQL, p.ID); err != nil {
			return fmt.Errorf("clear project %s protocols: %w", p.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectSuppliesSQL, p.ID); err != nil {
			return fmt.Errorf("clear project %s supplies: %w", p.ID, err)
		}
		if _, err := exec.ExecContext(ctx, insertProjectSQL,
			p.ID, p.Code, p.Title, p.Description, p.CreatedAt, p.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert project %s: %w", p.ID, err)
		}
		for _, facilityID := range p.FacilityIDs {
			if _, err := exec.ExecContext(ctx, insertProjectFacilitySQL, facilityID, p.ID); err != nil {
				return fmt.Errorf("insert project %s facility %s: %w", p.ID, facilityID, err)
			}
		}
		for _, protocolID := range p.ProtocolIDs {
			if _, err := exec.ExecContext(ctx, insertProjectProtocolSQL, p.ID, protocolID); err != nil {
				return fmt.Errorf("insert project %s protocol %s: %w", p.ID, protocolID, err)
			}
		}
		for _, supplyID := range p.SupplyItemIDs {
			if _, err := exec.ExecContext(ctx, insertProjectSupplySQL, p.ID, supplyID); err != nil {
				return fmt.Errorf("insert project %s supply %s: %w", p.ID, supplyID, err)
			}
		}
	}
	return nil
}

func insertPermits(ctx context.Context, exec execQuerier, permits map[string]domain.Permit) error {
	keys := sortedKeys(permits)
	for _, id := range keys {
		p := permits[id]
		if len(p.FacilityIDs) == 0 {
			return fmt.Errorf("permit %s missing required facility_ids", p.ID)
		}
		if len(p.ProtocolIDs) == 0 {
			return fmt.Errorf("permit %s missing required protocol_ids", p.ID)
		}
		if _, err := exec.ExecContext(ctx, deletePermitFacilitiesSQL, p.ID); err != nil {
			return fmt.Errorf("clear permit %s facilities: %w", p.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deletePermitProtocolsSQL, p.ID); err != nil {
			return fmt.Errorf("clear permit %s protocols: %w", p.ID, err)
		}
		activities, err := marshalJSONRequired("permit.allowed_activities", p.AllowedActivities)
		if err != nil {
			return err
		}
		if _, err := exec.ExecContext(ctx, insertPermitSQL,
			p.ID, p.PermitNumber, p.Authority, p.Status, p.ValidFrom, p.ValidUntil, activities, p.Notes, p.CreatedAt, p.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert permit %s: %w", p.ID, err)
		}
		for _, facilityID := range p.FacilityIDs {
			if _, err := exec.ExecContext(ctx, insertPermitFacilitySQL, p.ID, facilityID); err != nil {
				return fmt.Errorf("insert permit %s facility %s: %w", p.ID, facilityID, err)
			}
		}
		for _, protocolID := range p.ProtocolIDs {
			if _, err := exec.ExecContext(ctx, insertPermitProtocolSQL, p.ID, protocolID); err != nil {
				return fmt.Errorf("insert permit %s protocol %s: %w", p.ID, protocolID, err)
			}
		}
	}
	return nil
}

func insertCohorts(ctx context.Context, exec execQuerier, cohorts map[string]domain.Cohort) error {
	keys := sortedKeys(cohorts)
	for _, id := range keys {
		c := cohorts[id]
		if _, err := exec.ExecContext(ctx, insertCohortSQL,
			c.ID, c.Name, c.Purpose, c.ProjectID, c.HousingID, c.ProtocolID, c.CreatedAt, c.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert cohort %s: %w", c.ID, err)
		}
	}
	return nil
}

func insertBreedingUnits(ctx context.Context, exec execQuerier, breeding map[string]domain.BreedingUnit) error {
	keys := sortedKeys(breeding)
	for _, id := range keys {
		b := breeding[id]
		if _, err := exec.ExecContext(ctx, deleteBreedingFemalesSQL, b.ID); err != nil {
			return fmt.Errorf("clear breeding %s females: %w", b.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deleteBreedingMalesSQL, b.ID); err != nil {
			return fmt.Errorf("clear breeding %s males: %w", b.ID, err)
		}
		pairingAttrs, err := marshalJSONNullable((&b).PairingAttributes())
		if err != nil {
			return fmt.Errorf("marshal breeding pairing_attributes: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertBreedingSQL,
			b.ID, b.Name, b.Strategy, b.HousingID, b.LineID, b.StrainID, b.TargetLineID, b.TargetStrainID, b.ProtocolID, pairingAttrs, b.PairingIntent, b.PairingNotes, b.CreatedAt, b.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert breeding %s: %w", b.ID, err)
		}
		for _, femaleID := range b.FemaleIDs {
			if _, err := exec.ExecContext(ctx, insertBreedingFemaleSQL, b.ID, femaleID); err != nil {
				return fmt.Errorf("insert breeding %s female %s: %w", b.ID, femaleID, err)
			}
		}
		for _, maleID := range b.MaleIDs {
			if _, err := exec.ExecContext(ctx, insertBreedingMaleSQL, b.ID, maleID); err != nil {
				return fmt.Errorf("insert breeding %s male %s: %w", b.ID, maleID, err)
			}
		}
	}
	return nil
}

func insertOrganisms(ctx context.Context, exec execQuerier, organisms map[string]domain.Organism) error {
	keys := sortedKeys(organisms)
	for _, id := range keys {
		o := organisms[id]
		if _, err := exec.ExecContext(ctx, deleteOrganismParentsSQL, o.ID); err != nil {
			return fmt.Errorf("clear organism %s parents: %w", o.ID, err)
		}
		attrs, err := marshalJSONNullable((&o).CoreAttributes())
		if err != nil {
			return fmt.Errorf("marshal organism attributes: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertOrganismSQL,
			o.ID, o.Name, o.Species, o.Line, o.Stage, o.LineID, o.StrainID, o.CohortID, o.HousingID, o.ProtocolID, o.ProjectID, attrs, o.CreatedAt, o.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert organism %s: %w", o.ID, err)
		}
		for _, parentID := range o.ParentIDs {
			if _, err := exec.ExecContext(ctx, insertOrganismParentSQL, o.ID, parentID); err != nil {
				return fmt.Errorf("insert organism %s parent %s: %w", o.ID, parentID, err)
			}
		}
	}
	return nil
}

func insertProcedures(ctx context.Context, exec execQuerier, procedures map[string]domain.Procedure) error {
	keys := sortedKeys(procedures)
	for _, id := range keys {
		p := procedures[id]
		if _, err := exec.ExecContext(ctx, deleteProcedureOrganismsSQL, p.ID); err != nil {
			return fmt.Errorf("clear procedure %s organisms: %w", p.ID, err)
		}
		if p.ProtocolID == "" {
			return fmt.Errorf("procedure %s missing required protocol_id", p.ID)
		}
		if _, err := exec.ExecContext(ctx, insertProcedureSQL,
			p.ID, p.Name, p.Status, p.ScheduledAt, p.ProtocolID, p.ProjectID, p.CohortID, p.CreatedAt, p.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert procedure %s: %w", p.ID, err)
		}
		for _, organismID := range p.OrganismIDs {
			if _, err := exec.ExecContext(ctx, insertProcedureOrganismSQL, p.ID, organismID); err != nil {
				return fmt.Errorf("insert procedure %s organism %s: %w", p.ID, organismID, err)
			}
		}
	}
	return nil
}

func insertObservations(ctx context.Context, exec execQuerier, observations map[string]domain.Observation) error {
	keys := sortedKeys(observations)
	for _, id := range keys {
		o := observations[id]
		data, err := marshalJSONNullable(o.Data)
		if err != nil {
			return fmt.Errorf("marshal observation data: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertObservationSQL,
			o.ID, o.Observer, o.RecordedAt, o.ProcedureID, o.OrganismID, o.CohortID, data, o.Notes, o.CreatedAt, o.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert observation %s: %w", o.ID, err)
		}
	}
	return nil
}

func insertSamples(ctx context.Context, exec execQuerier, samples map[string]domain.Sample) error {
	keys := sortedKeys(samples)
	for _, id := range keys {
		s := samples[id]
		if len(s.ChainOfCustody) == 0 {
			return fmt.Errorf("sample %s missing required chain_of_custody", s.ID)
		}
		if s.FacilityID == "" {
			return fmt.Errorf("sample %s missing required facility_id", s.ID)
		}
		chain, err := marshalJSONRequired("sample.chain_of_custody", s.ChainOfCustody)
		if err != nil {
			return err
		}
		attrs, err := marshalJSONNullable((&s).SampleAttributes())
		if err != nil {
			return fmt.Errorf("marshal sample attributes: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertSampleSQL,
			s.ID, s.Identifier, s.SourceType, s.Status, s.StorageLocation, s.AssayType, s.FacilityID, s.OrganismID, s.CohortID, chain, attrs, s.CollectedAt, s.CreatedAt, s.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert sample %s: %w", s.ID, err)
		}
	}
	return nil
}

func insertSupplyItems(ctx context.Context, exec execQuerier, supplies map[string]domain.SupplyItem) error {
	keys := sortedKeys(supplies)
	for _, id := range keys {
		s := supplies[id]
		if len(s.FacilityIDs) == 0 {
			return fmt.Errorf("supply_item %s missing required facility_ids", s.ID)
		}
		if len(s.ProjectIDs) == 0 {
			return fmt.Errorf("supply_item %s missing required project_ids", s.ID)
		}
		if _, err := exec.ExecContext(ctx, deleteSupplyFacilitiesSQL, s.ID); err != nil {
			return fmt.Errorf("clear supply_item %s facilities: %w", s.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deleteProjectSuppliesBySupplySQL, s.ID); err != nil {
			return fmt.Errorf("clear supply_item %s projects: %w", s.ID, err)
		}
		attrs, err := marshalJSONNullable((&s).SupplyAttributes())
		if err != nil {
			return fmt.Errorf("marshal supply_item attributes: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertSupplySQL,
			s.ID, s.SKU, s.Name, s.QuantityOnHand, s.Unit, s.ReorderLevel, s.Description, s.LotNumber, s.ExpiresAt, attrs, s.CreatedAt, s.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert supply_item %s: %w", s.ID, err)
		}
		for _, facilityID := range s.FacilityIDs {
			if _, err := exec.ExecContext(ctx, insertSupplyFacilitySQL, s.ID, facilityID); err != nil {
				return fmt.Errorf("insert supply_item %s facility %s: %w", s.ID, facilityID, err)
			}
		}
		for _, projectID := range s.ProjectIDs {
			if _, err := exec.ExecContext(ctx, insertProjectSupplySQL, projectID, s.ID); err != nil {
				return fmt.Errorf("insert supply_item %s project %s: %w", s.ID, projectID, err)
			}
		}
	}
	return nil
}

func insertTreatments(ctx context.Context, exec execQuerier, treatments map[string]domain.Treatment) error {
	keys := sortedKeys(treatments)
	for _, id := range keys {
		treatment := treatments[id]
		if treatment.ProcedureID == "" {
			return fmt.Errorf("treatment %s missing required procedure_id", treatment.ID)
		}
		if _, err := exec.ExecContext(ctx, deleteTreatmentCohortsSQL, treatment.ID); err != nil {
			return fmt.Errorf("clear treatment %s cohorts: %w", treatment.ID, err)
		}
		if _, err := exec.ExecContext(ctx, deleteTreatmentOrganismsSQL, treatment.ID); err != nil {
			return fmt.Errorf("clear treatment %s organisms: %w", treatment.ID, err)
		}
		adminLog, err := marshalJSONNullable(treatment.AdministrationLog)
		if err != nil {
			return fmt.Errorf("marshal treatment administration_log: %w", err)
		}
		adverse, err := marshalJSONNullable(treatment.AdverseEvents)
		if err != nil {
			return fmt.Errorf("marshal treatment adverse_events: %w", err)
		}
		if _, err := exec.ExecContext(ctx, insertTreatmentSQL,
			treatment.ID, treatment.Name, treatment.Status, treatment.ProcedureID, treatment.DosagePlan, adminLog, adverse, treatment.CreatedAt, treatment.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert treatment %s: %w", treatment.ID, err)
		}
		for _, cohortID := range treatment.CohortIDs {
			if _, err := exec.ExecContext(ctx, insertTreatmentCohortSQL, treatment.ID, cohortID); err != nil {
				return fmt.Errorf("insert treatment %s cohort %s: %w", treatment.ID, cohortID, err)
			}
		}
		for _, organismID := range treatment.OrganismIDs {
			if _, err := exec.ExecContext(ctx, insertTreatmentOrganismSQL, treatment.ID, organismID); err != nil {
				return fmt.Errorf("insert treatment %s organism %s: %w", treatment.ID, organismID, err)
			}
		}
	}
	return nil
}

// --- load helpers ---

func loadFacilities(ctx context.Context, db execQuerier) (map[string]domain.Facility, error) {
	rows, err := db.QueryContext(ctx, selectFacilitiesSQL)
	if err != nil {
		return nil, fmt.Errorf("select facilities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Facility)
	for rows.Next() {
		var (
			id, code, name, zone, policy string
			createdAt, updatedAt         time.Time
			envRaw                       []byte
		)
		if err := rows.Scan(&id, &code, &name, &zone, &policy, &createdAt, &updatedAt, &envRaw); err != nil {
			return nil, fmt.Errorf("scan facilities: %w", err)
		}
		env, err := decodeMap(envRaw)
		if err != nil {
			return nil, fmt.Errorf("decode facility %s environment_baselines: %w", id, err)
		}
		out[id] = domain.Facility{Facility: entitymodel.Facility{
			ID:                   id,
			Code:                 code,
			Name:                 name,
			Zone:                 zone,
			AccessPolicy:         policy,
			CreatedAt:            createdAt,
			UpdatedAt:            updatedAt,
			EnvironmentBaselines: env,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate facilities: %w", err)
	}
	return out, nil
}

func loadGenotypeMarkers(ctx context.Context, db execQuerier) (map[string]domain.GenotypeMarker, error) {
	rows, err := db.QueryContext(ctx, selectGenotypeMarkersSQL)
	if err != nil {
		return nil, fmt.Errorf("select genotype_markers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.GenotypeMarker)
	for rows.Next() {
		var (
			id, name, locus, assayMethod, interpretation, version string
			createdAt, updatedAt                                  time.Time
			allelesRaw                                            []byte
		)
		if err := rows.Scan(&id, &name, &locus, &allelesRaw, &assayMethod, &interpretation, &version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan genotype_markers: %w", err)
		}
		alleles, err := decodeStringSlice(allelesRaw)
		if err != nil {
			return nil, fmt.Errorf("decode genotype_marker %s alleles: %w", id, err)
		}
		out[id] = domain.GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{
			ID:             id,
			Name:           name,
			Locus:          locus,
			Alleles:        alleles,
			AssayMethod:    assayMethod,
			Interpretation: interpretation,
			Version:        version,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate genotype_markers: %w", err)
	}
	return out, nil
}

func loadLines(ctx context.Context, db execQuerier) (map[string]domain.Line, error) {
	rows, err := db.QueryContext(ctx, selectLinesSQL)
	if err != nil {
		return nil, fmt.Errorf("select lines: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Line)
	for rows.Next() {
		var (
			id, code, name, origin        string
			description                   sql.NullString
			defaultAttrsRaw, overridesRaw []byte
			deprecatedAt                  sql.NullTime
			deprecationReason             sql.NullString
			createdAt, updatedAt          time.Time
		)
		if err := rows.Scan(&id, &code, &name, &origin, &description, &defaultAttrsRaw, &overridesRaw, &deprecatedAt, &deprecationReason, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan lines: %w", err)
		}
		defaultAttrs, err := decodeMap(defaultAttrsRaw)
		if err != nil {
			return nil, fmt.Errorf("decode line %s default_attributes: %w", id, err)
		}
		overrides, err := decodeMap(overridesRaw)
		if err != nil {
			return nil, fmt.Errorf("decode line %s extension_overrides: %w", id, err)
		}
		var deprecatedPtr *time.Time
		if deprecatedAt.Valid {
			deprecatedPtr = &deprecatedAt.Time
		}
		var deprecationReasonPtr *string
		if deprecationReason.Valid {
			deprecationReasonPtr = &deprecationReason.String
		}
		var descriptionPtr *string
		if description.Valid {
			descriptionPtr = &description.String
		}
		out[id] = domain.Line{Line: entitymodel.Line{
			ID:                 id,
			Code:               code,
			Name:               name,
			Origin:             origin,
			Description:        descriptionPtr,
			DefaultAttributes:  defaultAttrs,
			ExtensionOverrides: overrides,
			DeprecatedAt:       deprecatedPtr,
			DeprecationReason:  deprecationReasonPtr,
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lines: %w", err)
	}
	return out, nil
}

func loadLineMarkers(ctx context.Context, db execQuerier, lines map[string]domain.Line) error {
	rows, err := db.QueryContext(ctx, selectLineMarkersSQL)
	if err != nil {
		return fmt.Errorf("select line markers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var lineID, markerID string
		if err := rows.Scan(&lineID, &markerID); err != nil {
			return fmt.Errorf("scan line markers: %w", err)
		}
		line, ok := lines[lineID]
		if !ok {
			return fmt.Errorf("line marker row references missing line %s", lineID)
		}
		line.GenotypeMarkerIDs = append(line.GenotypeMarkerIDs, markerID)
		lines[lineID] = line
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate line markers: %w", err)
	}
	for id, line := range lines {
		if len(line.GenotypeMarkerIDs) == 0 {
			return fmt.Errorf("line %s missing genotype_marker_ids", id)
		}
		sort.Strings(line.GenotypeMarkerIDs)
		lines[id] = line
	}
	return nil
}

func loadStrains(ctx context.Context, db execQuerier) (map[string]domain.Strain, error) {
	rows, err := db.QueryContext(ctx, selectStrainsSQL)
	if err != nil {
		return nil, fmt.Errorf("select strains: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Strain)
	for rows.Next() {
		var (
			id, code, name, lineID  string
			description, generation sql.NullString
			retiredAt               sql.NullTime
			retirementReason        sql.NullString
			createdAt, updatedAt    time.Time
		)
		if err := rows.Scan(&id, &code, &name, &lineID, &description, &generation, &retiredAt, &retirementReason, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan strains: %w", err)
		}
		var descriptionPtr *string
		if description.Valid {
			descriptionPtr = &description.String
		}
		var generationPtr *string
		if generation.Valid {
			generationPtr = &generation.String
		}
		var retiredAtPtr *time.Time
		if retiredAt.Valid {
			retiredAtPtr = &retiredAt.Time
		}
		var retirementReasonPtr *string
		if retirementReason.Valid {
			retirementReasonPtr = &retirementReason.String
		}
		out[id] = domain.Strain{Strain: entitymodel.Strain{
			ID:               id,
			Code:             code,
			Name:             name,
			LineID:           lineID,
			Description:      descriptionPtr,
			Generation:       generationPtr,
			RetiredAt:        retiredAtPtr,
			RetirementReason: retirementReasonPtr,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate strains: %w", err)
	}
	return out, nil
}

func loadStrainMarkers(ctx context.Context, db execQuerier, strains map[string]domain.Strain) error {
	rows, err := db.QueryContext(ctx, selectStrainMarkersSQL)
	if err != nil {
		return fmt.Errorf("select strain markers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var strainID, markerID string
		if err := rows.Scan(&strainID, &markerID); err != nil {
			return fmt.Errorf("scan strain markers: %w", err)
		}
		strain, ok := strains[strainID]
		if !ok {
			return fmt.Errorf("strain marker row references missing strain %s", strainID)
		}
		strain.GenotypeMarkerIDs = append(strain.GenotypeMarkerIDs, markerID)
		strains[strainID] = strain
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate strain markers: %w", err)
	}
	for id, strain := range strains {
		sort.Strings(strain.GenotypeMarkerIDs)
		strains[id] = strain
	}
	return nil
}

func loadHousingUnits(ctx context.Context, db execQuerier) (map[string]domain.HousingUnit, error) {
	rows, err := db.QueryContext(ctx, selectHousingSQL)
	if err != nil {
		return nil, fmt.Errorf("select housing_units: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.HousingUnit)
	for rows.Next() {
		var (
			id, facilityID, name string
			capacity             int
			environment          domain.HousingEnvironment
			state                domain.HousingState
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&id, &facilityID, &name, &capacity, &environment, &state, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan housing_units: %w", err)
		}
		out[id] = domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
			ID:          id,
			FacilityID:  facilityID,
			Name:        name,
			Capacity:    capacity,
			Environment: entitymodel.HousingEnvironment(environment),
			State:       entitymodel.HousingState(state),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate housing_units: %w", err)
	}
	return out, nil
}

func loadProtocols(ctx context.Context, db execQuerier) (map[string]domain.Protocol, error) {
	rows, err := db.QueryContext(ctx, selectProtocolSQL)
	if err != nil {
		return nil, fmt.Errorf("select protocols: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Protocol)
	for rows.Next() {
		var (
			id, code, title      string
			description          sql.NullString
			maxSubjects          int
			status               domain.ProtocolStatus
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&id, &code, &title, &description, &maxSubjects, &status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan protocols: %w", err)
		}
		var descriptionPtr *string
		if description.Valid {
			descriptionPtr = &description.String
		}
		out[id] = domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          id,
			Code:        code,
			Title:       title,
			Description: descriptionPtr,
			MaxSubjects: maxSubjects,
			Status:      entitymodel.ProtocolStatus(status),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate protocols: %w", err)
	}
	return out, nil
}

func loadProjects(ctx context.Context, db execQuerier) (map[string]domain.Project, error) {
	rows, err := db.QueryContext(ctx, selectProjectSQL)
	if err != nil {
		return nil, fmt.Errorf("select projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Project)
	for rows.Next() {
		var (
			id, code, title      string
			description          sql.NullString
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&id, &code, &title, &description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan projects: %w", err)
		}
		var descriptionPtr *string
		if description.Valid {
			descriptionPtr = &description.String
		}
		out[id] = domain.Project{Project: entitymodel.Project{
			ID:          id,
			Code:        code,
			Title:       title,
			Description: descriptionPtr,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return out, nil
}

func loadProjectFacilities(ctx context.Context, db execQuerier, projects map[string]domain.Project, facilities map[string]domain.Facility) error {
	rows, err := db.QueryContext(ctx, selectProjectFacilitiesSQL)
	if err != nil {
		return fmt.Errorf("select project facilities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var facilityID, projectID string
		if err := rows.Scan(&facilityID, &projectID); err != nil {
			return fmt.Errorf("scan project facilities: %w", err)
		}
		project, ok := projects[projectID]
		if !ok {
			return fmt.Errorf("project facility row references missing project %s", projectID)
		}
		project.FacilityIDs = append(project.FacilityIDs, facilityID)
		projects[projectID] = project
		if facility, ok := facilities[facilityID]; ok {
			facility.ProjectIDs = append(facility.ProjectIDs, projectID)
			facilities[facilityID] = facility
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate project facilities: %w", err)
	}
	for id, project := range projects {
		if len(project.FacilityIDs) == 0 {
			return fmt.Errorf("project %s missing required facility_ids", id)
		}
		sort.Strings(project.FacilityIDs)
		projects[id] = project
	}
	for id, facility := range facilities {
		sort.Strings(facility.ProjectIDs)
		facilities[id] = facility
	}
	return nil
}

func loadProjectProtocols(ctx context.Context, db execQuerier, projects map[string]domain.Project) error {
	rows, err := db.QueryContext(ctx, selectProjectProtocolsSQL)
	if err != nil {
		return fmt.Errorf("select project protocols: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var projectID, protocolID string
		if err := rows.Scan(&projectID, &protocolID); err != nil {
			return fmt.Errorf("scan project protocols: %w", err)
		}
		project, ok := projects[projectID]
		if !ok {
			return fmt.Errorf("project protocol row references missing project %s", projectID)
		}
		project.ProtocolIDs = append(project.ProtocolIDs, protocolID)
		projects[projectID] = project
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate project protocols: %w", err)
	}
	for id, project := range projects {
		sort.Strings(project.ProtocolIDs)
		projects[id] = project
	}
	return nil
}

func loadPermits(ctx context.Context, db execQuerier) (map[string]domain.Permit, error) {
	rows, err := db.QueryContext(ctx, selectPermitSQL)
	if err != nil {
		return nil, fmt.Errorf("select permits: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Permit)
	for rows.Next() {
		var (
			id, permitNumber, authority string
			status                      domain.PermitStatus
			validFrom, validUntil       time.Time
			activitiesRaw               []byte
			notes                       sql.NullString
			createdAt, updatedAt        time.Time
		)
		if err := rows.Scan(&id, &permitNumber, &authority, &status, &validFrom, &validUntil, &activitiesRaw, &notes, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan permits: %w", err)
		}
		activities, err := decodeStringSlice(activitiesRaw)
		if err != nil {
			return nil, fmt.Errorf("decode permit %s allowed_activities: %w", id, err)
		}
		var notesPtr *string
		if notes.Valid {
			notesPtr = &notes.String
		}
		out[id] = domain.Permit{Permit: entitymodel.Permit{
			ID:                id,
			PermitNumber:      permitNumber,
			Authority:         authority,
			Status:            entitymodel.PermitStatus(status),
			ValidFrom:         validFrom,
			ValidUntil:        validUntil,
			AllowedActivities: activities,
			Notes:             notesPtr,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permits: %w", err)
	}
	return out, nil
}

func loadPermitFacilities(ctx context.Context, db execQuerier, permits map[string]domain.Permit) error {
	rows, err := db.QueryContext(ctx, selectPermitFacilitiesSQL)
	if err != nil {
		return fmt.Errorf("select permit facilities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var permitID, facilityID string
		if err := rows.Scan(&permitID, &facilityID); err != nil {
			return fmt.Errorf("scan permit facilities: %w", err)
		}
		permit, ok := permits[permitID]
		if !ok {
			return fmt.Errorf("permit facility row references missing permit %s", permitID)
		}
		permit.FacilityIDs = append(permit.FacilityIDs, facilityID)
		permits[permitID] = permit
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate permit facilities: %w", err)
	}
	for id, permit := range permits {
		if len(permit.FacilityIDs) == 0 {
			return fmt.Errorf("permit %s missing required facility_ids", id)
		}
		sort.Strings(permit.FacilityIDs)
		permits[id] = permit
	}
	return nil
}

func loadPermitProtocols(ctx context.Context, db execQuerier, permits map[string]domain.Permit) error {
	rows, err := db.QueryContext(ctx, selectPermitProtocolsSQL)
	if err != nil {
		return fmt.Errorf("select permit protocols: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var permitID, protocolID string
		if err := rows.Scan(&permitID, &protocolID); err != nil {
			return fmt.Errorf("scan permit protocols: %w", err)
		}
		permit, ok := permits[permitID]
		if !ok {
			return fmt.Errorf("permit protocol row references missing permit %s", permitID)
		}
		permit.ProtocolIDs = append(permit.ProtocolIDs, protocolID)
		permits[permitID] = permit
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate permit protocols: %w", err)
	}
	for id, permit := range permits {
		if len(permit.ProtocolIDs) == 0 {
			return fmt.Errorf("permit %s missing required protocol_ids", id)
		}
		sort.Strings(permit.ProtocolIDs)
		permits[id] = permit
	}
	return nil
}

func loadCohorts(ctx context.Context, db execQuerier) (map[string]domain.Cohort, error) {
	rows, err := db.QueryContext(ctx, selectCohortSQL)
	if err != nil {
		return nil, fmt.Errorf("select cohorts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Cohort)
	for rows.Next() {
		var (
			id, name, purpose                string
			projectID, housingID, protocolID sql.NullString
			createdAt, updatedAt             time.Time
		)
		if err := rows.Scan(&id, &name, &purpose, &projectID, &housingID, &protocolID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan cohorts: %w", err)
		}
		out[id] = domain.Cohort{Cohort: entitymodel.Cohort{
			ID:         id,
			Name:       name,
			Purpose:    purpose,
			ProjectID:  nullableString(projectID),
			HousingID:  nullableString(housingID),
			ProtocolID: nullableString(protocolID),
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cohorts: %w", err)
	}
	return out, nil
}

func loadBreedingUnits(ctx context.Context, db execQuerier) (map[string]domain.BreedingUnit, error) {
	rows, err := db.QueryContext(ctx, selectBreedingSQL)
	if err != nil {
		return nil, fmt.Errorf("select breeding_units: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.BreedingUnit)
	for rows.Next() {
		var (
			id, name, strategy                        string
			housingID, lineID, strainID, targetLineID sql.NullString
			targetStrainID, protocolID                sql.NullString
			pairingAttrsRaw                           []byte
			pairingIntent, pairingNotes               sql.NullString
			createdAt, updatedAt                      time.Time
		)
		if err := rows.Scan(&id, &name, &strategy, &housingID, &lineID, &strainID, &targetLineID, &targetStrainID, &protocolID, &pairingAttrsRaw, &pairingIntent, &pairingNotes, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan breeding_units: %w", err)
		}
		pairingAttrs, err := decodeMap(pairingAttrsRaw)
		if err != nil {
			return nil, fmt.Errorf("decode breeding_unit %s pairing_attributes: %w", id, err)
		}
		out[id] = domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
			ID:                id,
			Name:              name,
			Strategy:          strategy,
			HousingID:         nullableString(housingID),
			LineID:            nullableString(lineID),
			StrainID:          nullableString(strainID),
			TargetLineID:      nullableString(targetLineID),
			TargetStrainID:    nullableString(targetStrainID),
			ProtocolID:        nullableString(protocolID),
			PairingAttributes: pairingAttrs,
			PairingIntent:     nullableString(pairingIntent),
			PairingNotes:      nullableString(pairingNotes),
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate breeding_units: %w", err)
	}
	return out, nil
}

func loadBreedingUnitMembers(ctx context.Context, db execQuerier, breeding map[string]domain.BreedingUnit) error {
	femaleRows, err := db.QueryContext(ctx, selectBreedingFemalesSQL)
	if err != nil {
		return fmt.Errorf("select breeding female_ids: %w", err)
	}
	defer func() { _ = femaleRows.Close() }()
	for femaleRows.Next() {
		var breedingID, organismID string
		if err := femaleRows.Scan(&breedingID, &organismID); err != nil {
			return fmt.Errorf("scan breeding female_ids: %w", err)
		}
		unit, ok := breeding[breedingID]
		if !ok {
			return fmt.Errorf("breeding female row references missing breeding_unit %s", breedingID)
		}
		unit.FemaleIDs = append(unit.FemaleIDs, organismID)
		breeding[breedingID] = unit
	}
	if err := femaleRows.Err(); err != nil {
		return fmt.Errorf("iterate breeding female_ids: %w", err)
	}

	maleRows, err := db.QueryContext(ctx, selectBreedingMalesSQL)
	if err != nil {
		return fmt.Errorf("select breeding male_ids: %w", err)
	}
	defer func() { _ = maleRows.Close() }()
	for maleRows.Next() {
		var breedingID, organismID string
		if err := maleRows.Scan(&breedingID, &organismID); err != nil {
			return fmt.Errorf("scan breeding male_ids: %w", err)
		}
		unit, ok := breeding[breedingID]
		if !ok {
			return fmt.Errorf("breeding male row references missing breeding_unit %s", breedingID)
		}
		unit.MaleIDs = append(unit.MaleIDs, organismID)
		breeding[breedingID] = unit
	}
	if err := maleRows.Err(); err != nil {
		return fmt.Errorf("iterate breeding male_ids: %w", err)
	}
	for id, unit := range breeding {
		sort.Strings(unit.FemaleIDs)
		sort.Strings(unit.MaleIDs)
		breeding[id] = unit
	}
	return nil
}

func loadOrganisms(ctx context.Context, db execQuerier) (map[string]domain.Organism, error) {
	rows, err := db.QueryContext(ctx, selectOrganismSQL)
	if err != nil {
		return nil, fmt.Errorf("select organisms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Organism)
	for rows.Next() {
		var (
			id, name, species, line string
			stage                   domain.LifecycleStage
			lineID, strainID        sql.NullString
			cohortID, housingID     sql.NullString
			protocolID, projectID   sql.NullString
			attributesRaw           []byte
			createdAt, updatedAt    time.Time
		)
		if err := rows.Scan(&id, &name, &species, &line, &stage, &lineID, &strainID, &cohortID, &housingID, &protocolID, &projectID, &attributesRaw, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan organisms: %w", err)
		}
		attrs, err := decodeMap(attributesRaw)
		if err != nil {
			return nil, fmt.Errorf("decode organism %s attributes: %w", id, err)
		}
		out[id] = domain.Organism{Organism: entitymodel.Organism{
			ID:         id,
			Name:       name,
			Species:    species,
			Line:       line,
			Stage:      entitymodel.LifecycleStage(stage),
			LineID:     nullableString(lineID),
			StrainID:   nullableString(strainID),
			CohortID:   nullableString(cohortID),
			HousingID:  nullableString(housingID),
			ProtocolID: nullableString(protocolID),
			ProjectID:  nullableString(projectID),
			Attributes: attrs,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisms: %w", err)
	}
	return out, nil
}

func loadOrganismParents(ctx context.Context, db execQuerier, organisms map[string]domain.Organism) error {
	rows, err := db.QueryContext(ctx, selectOrganismParentsSQL)
	if err != nil {
		return fmt.Errorf("select organism parents: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var organismID, parentID string
		if err := rows.Scan(&organismID, &parentID); err != nil {
			return fmt.Errorf("scan organism parents: %w", err)
		}
		org, ok := organisms[organismID]
		if !ok {
			return fmt.Errorf("organism parent row references missing organism %s", organismID)
		}
		org.ParentIDs = append(org.ParentIDs, parentID)
		organisms[organismID] = org
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate organism parents: %w", err)
	}
	for id, org := range organisms {
		sort.Strings(org.ParentIDs)
		organisms[id] = org
	}
	return nil
}

func loadProcedures(ctx context.Context, db execQuerier) (map[string]domain.Procedure, error) {
	rows, err := db.QueryContext(ctx, selectProcedureSQL)
	if err != nil {
		return nil, fmt.Errorf("select procedures: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Procedure)
	for rows.Next() {
		var (
			id, name                          string
			status                            domain.ProcedureStatus
			scheduledAt, createdAt, updatedAt time.Time
			protocolID                        string
			projectID, cohortID               sql.NullString
		)
		if err := rows.Scan(&id, &name, &status, &scheduledAt, &protocolID, &projectID, &cohortID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan procedures: %w", err)
		}
		out[id] = domain.Procedure{Procedure: entitymodel.Procedure{
			ID:          id,
			Name:        name,
			Status:      entitymodel.ProcedureStatus(status),
			ScheduledAt: scheduledAt,
			ProtocolID:  protocolID,
			ProjectID:   nullableString(projectID),
			CohortID:    nullableString(cohortID),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate procedures: %w", err)
	}
	return out, nil
}

func loadProcedureOrganisms(ctx context.Context, db execQuerier, procedures map[string]domain.Procedure) error {
	rows, err := db.QueryContext(ctx, selectProcedureOrganismsSQL)
	if err != nil {
		return fmt.Errorf("select procedure organisms: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var procedureID, organismID string
		if err := rows.Scan(&procedureID, &organismID); err != nil {
			return fmt.Errorf("scan procedure organisms: %w", err)
		}
		proc, ok := procedures[procedureID]
		if !ok {
			return fmt.Errorf("procedure organism row references missing procedure %s", procedureID)
		}
		proc.OrganismIDs = append(proc.OrganismIDs, organismID)
		procedures[procedureID] = proc
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate procedure organisms: %w", err)
	}
	for id, proc := range procedures {
		sort.Strings(proc.OrganismIDs)
		procedures[id] = proc
	}
	return nil
}

func loadObservations(ctx context.Context, db execQuerier) (map[string]domain.Observation, error) {
	rows, err := db.QueryContext(ctx, selectObservationSQL)
	if err != nil {
		return nil, fmt.Errorf("select observations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Observation)
	for rows.Next() {
		var (
			id, observer                      string
			recordedAt, createdAt, updatedAt  time.Time
			procedureID, organismID, cohortID sql.NullString
			dataRaw                           []byte
			notes                             sql.NullString
		)
		if err := rows.Scan(&id, &observer, &recordedAt, &procedureID, &organismID, &cohortID, &dataRaw, &notes, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan observations: %w", err)
		}
		data, err := decodeMap(dataRaw)
		if err != nil {
			return nil, fmt.Errorf("decode observation %s data: %w", id, err)
		}
		out[id] = domain.Observation{Observation: entitymodel.Observation{
			ID:          id,
			Observer:    observer,
			RecordedAt:  recordedAt,
			ProcedureID: nullableString(procedureID),
			OrganismID:  nullableString(organismID),
			CohortID:    nullableString(cohortID),
			Data:        data,
			Notes:       nullableString(notes),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observations: %w", err)
	}
	return out, nil
}

func loadSamples(ctx context.Context, db execQuerier) (map[string]domain.Sample, error) {
	rows, err := db.QueryContext(ctx, selectSampleSQL)
	if err != nil {
		return nil, fmt.Errorf("select samples: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Sample)
	for rows.Next() {
		var (
			id, identifier, sourceType, status, storageLocation, assayType string
			facilityID                                                     string
			organismID, cohortID                                           sql.NullString
			chainRaw, attrsRaw                                             []byte
			collectedAt, createdAt, updatedAt                              time.Time
		)
		if err := rows.Scan(&id, &identifier, &sourceType, &status, &storageLocation, &assayType, &facilityID, &organismID, &cohortID, &chainRaw, &attrsRaw, &collectedAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan samples: %w", err)
		}
		chain, err := decodeCustody(chainRaw)
		if err != nil {
			return nil, fmt.Errorf("decode sample %s chain_of_custody: %w", id, err)
		}
		attrs, err := decodeMap(attrsRaw)
		if err != nil {
			return nil, fmt.Errorf("decode sample %s attributes: %w", id, err)
		}
		out[id] = domain.Sample{Sample: entitymodel.Sample{
			ID:              id,
			Identifier:      identifier,
			SourceType:      sourceType,
			Status:          entitymodel.SampleStatus(status),
			StorageLocation: storageLocation,
			AssayType:       assayType,
			FacilityID:      facilityID,
			OrganismID:      nullableString(organismID),
			CohortID:        nullableString(cohortID),
			ChainOfCustody:  chain,
			Attributes:      attrs,
			CollectedAt:     collectedAt,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate samples: %w", err)
	}
	return out, nil
}

func loadSupplyItems(ctx context.Context, db execQuerier) (map[string]domain.SupplyItem, error) {
	rows, err := db.QueryContext(ctx, selectSupplySQL)
	if err != nil {
		return nil, fmt.Errorf("select supply_items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.SupplyItem)
	for rows.Next() {
		var (
			id, sku, name, unit  string
			quantity, reorder    int
			description, lot     sql.NullString
			expiresAt            sql.NullTime
			attrsRaw             []byte
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&id, &sku, &name, &quantity, &unit, &reorder, &description, &lot, &expiresAt, &attrsRaw, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan supply_items: %w", err)
		}
		attrs, err := decodeMap(attrsRaw)
		if err != nil {
			return nil, fmt.Errorf("decode supply_item %s attributes: %w", id, err)
		}
		out[id] = domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
			ID:             id,
			SKU:            sku,
			Name:           name,
			QuantityOnHand: quantity,
			Unit:           unit,
			ReorderLevel:   reorder,
			Description:    nullableString(description),
			LotNumber:      nullableString(lot),
			ExpiresAt:      nullableTime(expiresAt),
			Attributes:     attrs,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supply_items: %w", err)
	}
	return out, nil
}

func loadSupplyItemFacilities(ctx context.Context, db execQuerier, supplies map[string]domain.SupplyItem) error {
	rows, err := db.QueryContext(ctx, selectSupplyFacilitiesSQL)
	if err != nil {
		return fmt.Errorf("select supply facilities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var supplyID, facilityID string
		if err := rows.Scan(&supplyID, &facilityID); err != nil {
			return fmt.Errorf("scan supply facilities: %w", err)
		}
		supply, ok := supplies[supplyID]
		if !ok {
			return fmt.Errorf("supply facility row references missing supply_item %s", supplyID)
		}
		supply.FacilityIDs = append(supply.FacilityIDs, facilityID)
		supplies[supplyID] = supply
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate supply facilities: %w", err)
	}
	for id, supply := range supplies {
		if len(supply.FacilityIDs) == 0 {
			return fmt.Errorf("supply_item %s missing required facility_ids", id)
		}
		sort.Strings(supply.FacilityIDs)
		supplies[id] = supply
	}
	return nil
}

func loadProjectSupplyItems(ctx context.Context, db execQuerier, projects map[string]domain.Project, supplies map[string]domain.SupplyItem) error {
	rows, err := db.QueryContext(ctx, selectProjectSupplySQL)
	if err != nil {
		return fmt.Errorf("select project supply items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var projectID, supplyID string
		if err := rows.Scan(&projectID, &supplyID); err != nil {
			return fmt.Errorf("scan project supply items: %w", err)
		}
		project, ok := projects[projectID]
		if !ok {
			return fmt.Errorf("project supply row references missing project %s", projectID)
		}
		project.SupplyItemIDs = append(project.SupplyItemIDs, supplyID)
		projects[projectID] = project
		if supply, ok := supplies[supplyID]; ok {
			supply.ProjectIDs = append(supply.ProjectIDs, projectID)
			supplies[supplyID] = supply
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate project supply items: %w", err)
	}
	for id, project := range projects {
		sort.Strings(project.SupplyItemIDs)
		projects[id] = project
	}
	for id, supply := range supplies {
		if len(supply.ProjectIDs) == 0 {
			return fmt.Errorf("supply_item %s missing required project_ids", id)
		}
		sort.Strings(supply.ProjectIDs)
		supplies[id] = supply
	}
	return nil
}

func loadTreatments(ctx context.Context, db execQuerier) (map[string]domain.Treatment, error) {
	rows, err := db.QueryContext(ctx, selectTreatmentSQL)
	if err != nil {
		return nil, fmt.Errorf("select treatments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]domain.Treatment)
	for rows.Next() {
		var (
			id, name, procedureID, dosagePlan string
			status                            domain.TreatmentStatus
			adminLogRaw, adverseRaw           []byte
			createdAt, updatedAt              time.Time
		)
		if err := rows.Scan(&id, &name, &status, &procedureID, &dosagePlan, &adminLogRaw, &adverseRaw, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan treatments: %w", err)
		}
		adminLog, err := decodeStringSlice(adminLogRaw)
		if err != nil {
			return nil, fmt.Errorf("decode treatment %s administration_log: %w", id, err)
		}
		adverseEvents, err := decodeStringSlice(adverseRaw)
		if err != nil {
			return nil, fmt.Errorf("decode treatment %s adverse_events: %w", id, err)
		}
		out[id] = domain.Treatment{Treatment: entitymodel.Treatment{
			ID:                id,
			Name:              name,
			Status:            entitymodel.TreatmentStatus(status),
			ProcedureID:       procedureID,
			DosagePlan:        dosagePlan,
			AdministrationLog: adminLog,
			AdverseEvents:     adverseEvents,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate treatments: %w", err)
	}
	return out, nil
}

func loadTreatmentCohorts(ctx context.Context, db execQuerier, treatments map[string]domain.Treatment) error {
	rows, err := db.QueryContext(ctx, selectTreatmentCohortsSQL)
	if err != nil {
		return fmt.Errorf("select treatment cohorts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var treatmentID, cohortID string
		if err := rows.Scan(&treatmentID, &cohortID); err != nil {
			return fmt.Errorf("scan treatment cohorts: %w", err)
		}
		t, ok := treatments[treatmentID]
		if !ok {
			return fmt.Errorf("treatment cohort row references missing treatment %s", treatmentID)
		}
		t.CohortIDs = append(t.CohortIDs, cohortID)
		treatments[treatmentID] = t
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate treatment cohorts: %w", err)
	}
	for id, treatment := range treatments {
		sort.Strings(treatment.CohortIDs)
		treatments[id] = treatment
	}
	return nil
}

func loadTreatmentOrganisms(ctx context.Context, db execQuerier, treatments map[string]domain.Treatment) error {
	rows, err := db.QueryContext(ctx, selectTreatmentOrganismsSQL)
	if err != nil {
		return fmt.Errorf("select treatment organisms: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var treatmentID, organismID string
		if err := rows.Scan(&treatmentID, &organismID); err != nil {
			return fmt.Errorf("scan treatment organisms: %w", err)
		}
		t, ok := treatments[treatmentID]
		if !ok {
			return fmt.Errorf("treatment organism row references missing treatment %s", treatmentID)
		}
		t.OrganismIDs = append(t.OrganismIDs, organismID)
		treatments[treatmentID] = t
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate treatment organisms: %w", err)
	}
	for id, treatment := range treatments {
		sort.Strings(treatment.OrganismIDs)
		treatments[id] = treatment
	}
	return nil
}

// --- SQL constants ---

const (
	insertFacilitySQL           = `INSERT INTO facilities (id, code, name, zone, access_policy, created_at, updated_at, environment_baselines) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET code=EXCLUDED.code, name=EXCLUDED.name, zone=EXCLUDED.zone, access_policy=EXCLUDED.access_policy, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at, environment_baselines=EXCLUDED.environment_baselines`
	deleteFacilitySQL           = `DELETE FROM facilities WHERE id=$1`
	deleteFacilitiesProjectsSQL = `DELETE FROM facilities__project_ids WHERE facility_id=$1`
	selectFacilitiesSQL         = `SELECT id, code, name, zone, access_policy, created_at, updated_at, environment_baselines FROM facilities`

	insertGenotypeMarkerSQL  = `INSERT INTO genotype_markers (id, name, locus, alleles, assay_method, interpretation, version, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, locus=EXCLUDED.locus, alleles=EXCLUDED.alleles, assay_method=EXCLUDED.assay_method, interpretation=EXCLUDED.interpretation, version=EXCLUDED.version, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteGenotypeMarkerSQL  = `DELETE FROM genotype_markers WHERE id=$1`
	selectGenotypeMarkersSQL = `SELECT id, name, locus, alleles, assay_method, interpretation, version, created_at, updated_at FROM genotype_markers`

	insertLineSQL        = `INSERT INTO lines (id, code, name, origin, description, default_attributes, extension_overrides, deprecated_at, deprecation_reason, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (id) DO UPDATE SET code=EXCLUDED.code, name=EXCLUDED.name, origin=EXCLUDED.origin, description=EXCLUDED.description, default_attributes=EXCLUDED.default_attributes, extension_overrides=EXCLUDED.extension_overrides, deprecated_at=EXCLUDED.deprecated_at, deprecation_reason=EXCLUDED.deprecation_reason, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteLineSQL        = `DELETE FROM lines WHERE id=$1`
	insertLineMarkerSQL  = `INSERT INTO lines__genotype_marker_ids (line_id, genotype_marker_id) VALUES ($1,$2)`
	deleteLineMarkersSQL = `DELETE FROM lines__genotype_marker_ids WHERE line_id=$1`
	selectLinesSQL       = `SELECT id, code, name, origin, description, default_attributes, extension_overrides, deprecated_at, deprecation_reason, created_at, updated_at FROM lines`
	selectLineMarkersSQL = `SELECT line_id, genotype_marker_id FROM lines__genotype_marker_ids`

	insertStrainSQL        = `INSERT INTO strains (id, code, name, line_id, description, generation, retired_at, retirement_reason, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (id) DO UPDATE SET code=EXCLUDED.code, name=EXCLUDED.name, line_id=EXCLUDED.line_id, description=EXCLUDED.description, generation=EXCLUDED.generation, retired_at=EXCLUDED.retired_at, retirement_reason=EXCLUDED.retirement_reason, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteStrainSQL        = `DELETE FROM strains WHERE id=$1`
	insertStrainMarkerSQL  = `INSERT INTO strains__genotype_marker_ids (strain_id, genotype_marker_id) VALUES ($1,$2)`
	deleteStrainMarkersSQL = `DELETE FROM strains__genotype_marker_ids WHERE strain_id=$1`
	selectStrainsSQL       = `SELECT id, code, name, line_id, description, generation, retired_at, retirement_reason, created_at, updated_at FROM strains`
	selectStrainMarkersSQL = `SELECT strain_id, genotype_marker_id FROM strains__genotype_marker_ids`

	insertHousingSQL = `INSERT INTO housing_units (id, facility_id, name, capacity, environment, state, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET facility_id=EXCLUDED.facility_id, name=EXCLUDED.name, capacity=EXCLUDED.capacity, environment=EXCLUDED.environment, state=EXCLUDED.state, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteHousingSQL = `DELETE FROM housing_units WHERE id=$1`
	selectHousingSQL = `SELECT id, facility_id, name, capacity, environment, state, created_at, updated_at FROM housing_units`

	insertProtocolSQL = `INSERT INTO protocols (id, code, title, description, max_subjects, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET code=EXCLUDED.code, title=EXCLUDED.title, description=EXCLUDED.description, max_subjects=EXCLUDED.max_subjects, status=EXCLUDED.status, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteProtocolSQL = `DELETE FROM protocols WHERE id=$1`
	selectProtocolSQL = `SELECT id, code, title, description, max_subjects, status, created_at, updated_at FROM protocols`

	insertProjectSQL           = `INSERT INTO projects (id, code, title, description, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO UPDATE SET code=EXCLUDED.code, title=EXCLUDED.title, description=EXCLUDED.description, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteProjectSQL           = `DELETE FROM projects WHERE id=$1`
	insertProjectFacilitySQL   = `INSERT INTO facilities__project_ids (facility_id, project_id) VALUES ($1,$2)`
	deleteProjectFacilitiesSQL = `DELETE FROM facilities__project_ids WHERE project_id=$1`
	insertProjectProtocolSQL   = `INSERT INTO projects__protocol_ids (project_id, protocol_id) VALUES ($1,$2)`
	deleteProjectProtocolsSQL  = `DELETE FROM projects__protocol_ids WHERE project_id=$1`
	insertProjectSupplySQL     = `INSERT INTO projects__supply_item_ids (project_id, supply_item_id) VALUES ($1,$2)`
	deleteProjectSuppliesSQL   = `DELETE FROM projects__supply_item_ids WHERE project_id=$1`
	selectProjectSQL           = `SELECT id, code, title, description, created_at, updated_at FROM projects`
	selectProjectFacilitiesSQL = `SELECT facility_id, project_id FROM facilities__project_ids`
	selectProjectProtocolsSQL  = `SELECT project_id, protocol_id FROM projects__protocol_ids`
	selectProjectSupplySQL     = `SELECT project_id, supply_item_id FROM projects__supply_item_ids`

	insertPermitSQL           = `INSERT INTO permits (id, permit_number, authority, status, valid_from, valid_until, allowed_activities, notes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (id) DO UPDATE SET permit_number=EXCLUDED.permit_number, authority=EXCLUDED.authority, status=EXCLUDED.status, valid_from=EXCLUDED.valid_from, valid_until=EXCLUDED.valid_until, allowed_activities=EXCLUDED.allowed_activities, notes=EXCLUDED.notes, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deletePermitSQL           = `DELETE FROM permits WHERE id=$1`
	insertPermitFacilitySQL   = `INSERT INTO permits__facility_ids (permit_id, facility_id) VALUES ($1,$2)`
	deletePermitFacilitiesSQL = `DELETE FROM permits__facility_ids WHERE permit_id=$1`
	insertPermitProtocolSQL   = `INSERT INTO permits__protocol_ids (permit_id, protocol_id) VALUES ($1,$2)`
	deletePermitProtocolsSQL  = `DELETE FROM permits__protocol_ids WHERE permit_id=$1`
	selectPermitSQL           = `SELECT id, permit_number, authority, status, valid_from, valid_until, allowed_activities, notes, created_at, updated_at FROM permits`
	selectPermitFacilitiesSQL = `SELECT permit_id, facility_id FROM permits__facility_ids`
	selectPermitProtocolsSQL  = `SELECT permit_id, protocol_id FROM permits__protocol_ids`

	insertCohortSQL   = `INSERT INTO cohorts (id, name, purpose, project_id, housing_id, protocol_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, purpose=EXCLUDED.purpose, project_id=EXCLUDED.project_id, housing_id=EXCLUDED.housing_id, protocol_id=EXCLUDED.protocol_id, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteCohortSQL   = `DELETE FROM cohorts WHERE id=$1`
	selectCohortSQL   = `SELECT id, name, purpose, project_id, housing_id, protocol_id, created_at, updated_at FROM cohorts`
	selectBreedingSQL = `SELECT id, name, strategy, housing_id, line_id, strain_id, target_line_id, target_strain_id, protocol_id, pairing_attributes, pairing_intent, pairing_notes, created_at, updated_at FROM breeding_units`

	insertBreedingSQL        = `INSERT INTO breeding_units (id, name, strategy, housing_id, line_id, strain_id, target_line_id, target_strain_id, protocol_id, pairing_attributes, pairing_intent, pairing_notes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, strategy=EXCLUDED.strategy, housing_id=EXCLUDED.housing_id, line_id=EXCLUDED.line_id, strain_id=EXCLUDED.strain_id, target_line_id=EXCLUDED.target_line_id, target_strain_id=EXCLUDED.target_strain_id, protocol_id=EXCLUDED.protocol_id, pairing_attributes=EXCLUDED.pairing_attributes, pairing_intent=EXCLUDED.pairing_intent, pairing_notes=EXCLUDED.pairing_notes, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteBreedingSQL        = `DELETE FROM breeding_units WHERE id=$1`
	insertBreedingFemaleSQL  = `INSERT INTO breeding_units__female_ids (breeding_unit_id, organism_id) VALUES ($1,$2)`
	deleteBreedingFemalesSQL = `DELETE FROM breeding_units__female_ids WHERE breeding_unit_id=$1`
	insertBreedingMaleSQL    = `INSERT INTO breeding_units__male_ids (breeding_unit_id, organism_id) VALUES ($1,$2)`
	deleteBreedingMalesSQL   = `DELETE FROM breeding_units__male_ids WHERE breeding_unit_id=$1`
	selectBreedingFemalesSQL = `SELECT breeding_unit_id, organism_id FROM breeding_units__female_ids`
	selectBreedingMalesSQL   = `SELECT breeding_unit_id, organism_id FROM breeding_units__male_ids`

	insertOrganismSQL        = `INSERT INTO organisms (id, name, species, line, stage, line_id, strain_id, cohort_id, housing_id, protocol_id, project_id, attributes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, species=EXCLUDED.species, line=EXCLUDED.line, stage=EXCLUDED.stage, line_id=EXCLUDED.line_id, strain_id=EXCLUDED.strain_id, cohort_id=EXCLUDED.cohort_id, housing_id=EXCLUDED.housing_id, protocol_id=EXCLUDED.protocol_id, project_id=EXCLUDED.project_id, attributes=EXCLUDED.attributes, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteOrganismSQL        = `DELETE FROM organisms WHERE id=$1`
	insertOrganismParentSQL  = `INSERT INTO organisms__parent_ids (organism_id, parent_ids_id) VALUES ($1,$2)`
	deleteOrganismParentsSQL = `DELETE FROM organisms__parent_ids WHERE organism_id=$1`
	selectOrganismSQL        = `SELECT id, name, species, line, stage, line_id, strain_id, cohort_id, housing_id, protocol_id, project_id, attributes, created_at, updated_at FROM organisms`
	selectOrganismParentsSQL = `SELECT organism_id, parent_ids_id FROM organisms__parent_ids`

	insertProcedureSQL          = `INSERT INTO procedures (id, name, status, scheduled_at, protocol_id, project_id, cohort_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, status=EXCLUDED.status, scheduled_at=EXCLUDED.scheduled_at, protocol_id=EXCLUDED.protocol_id, project_id=EXCLUDED.project_id, cohort_id=EXCLUDED.cohort_id, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteProcedureSQL          = `DELETE FROM procedures WHERE id=$1`
	insertProcedureOrganismSQL  = `INSERT INTO procedures__organism_ids (procedure_id, organism_id) VALUES ($1,$2)`
	deleteProcedureOrganismsSQL = `DELETE FROM procedures__organism_ids WHERE procedure_id=$1`
	selectProcedureSQL          = `SELECT id, name, status, scheduled_at, protocol_id, project_id, cohort_id, created_at, updated_at FROM procedures`
	selectProcedureOrganismsSQL = `SELECT procedure_id, organism_id FROM procedures__organism_ids`

	insertObservationSQL = `INSERT INTO observations (id, observer, recorded_at, procedure_id, organism_id, cohort_id, data, notes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (id) DO UPDATE SET observer=EXCLUDED.observer, recorded_at=EXCLUDED.recorded_at, procedure_id=EXCLUDED.procedure_id, organism_id=EXCLUDED.organism_id, cohort_id=EXCLUDED.cohort_id, data=EXCLUDED.data, notes=EXCLUDED.notes, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteObservationSQL = `DELETE FROM observations WHERE id=$1`
	selectObservationSQL = `SELECT id, observer, recorded_at, procedure_id, organism_id, cohort_id, data, notes, created_at, updated_at FROM observations`

	insertSampleSQL = `INSERT INTO samples (id, identifier, source_type, status, storage_location, assay_type, facility_id, organism_id, cohort_id, chain_of_custody, attributes, collected_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (id) DO UPDATE SET identifier=EXCLUDED.identifier, source_type=EXCLUDED.source_type, status=EXCLUDED.status, storage_location=EXCLUDED.storage_location, assay_type=EXCLUDED.assay_type, facility_id=EXCLUDED.facility_id, organism_id=EXCLUDED.organism_id, cohort_id=EXCLUDED.cohort_id, chain_of_custody=EXCLUDED.chain_of_custody, attributes=EXCLUDED.attributes, collected_at=EXCLUDED.collected_at, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteSampleSQL = `DELETE FROM samples WHERE id=$1`
	selectSampleSQL = `SELECT id, identifier, source_type, status, storage_location, assay_type, facility_id, organism_id, cohort_id, chain_of_custody, attributes, collected_at, created_at, updated_at FROM samples`

	insertSupplySQL                  = `INSERT INTO supply_items (id, sku, name, quantity_on_hand, unit, reorder_level, description, lot_number, expires_at, attributes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (id) DO UPDATE SET sku=EXCLUDED.sku, name=EXCLUDED.name, quantity_on_hand=EXCLUDED.quantity_on_hand, unit=EXCLUDED.unit, reorder_level=EXCLUDED.reorder_level, description=EXCLUDED.description, lot_number=EXCLUDED.lot_number, expires_at=EXCLUDED.expires_at, attributes=EXCLUDED.attributes, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteSupplySQL                  = `DELETE FROM supply_items WHERE id=$1`
	insertSupplyFacilitySQL          = `INSERT INTO supply_items__facility_ids (supply_item_id, facility_id) VALUES ($1,$2)`
	deleteSupplyFacilitiesSQL        = `DELETE FROM supply_items__facility_ids WHERE supply_item_id=$1`
	selectSupplyFacilitiesSQL        = `SELECT supply_item_id, facility_id FROM supply_items__facility_ids`
	deleteProjectSuppliesBySupplySQL = `DELETE FROM projects__supply_item_ids WHERE supply_item_id=$1`
	selectSupplySQL                  = `SELECT id, sku, name, quantity_on_hand, unit, reorder_level, description, lot_number, expires_at, attributes, created_at, updated_at FROM supply_items`

	insertTreatmentSQL          = `INSERT INTO treatments (id, name, status, procedure_id, dosage_plan, administration_log, adverse_events, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, status=EXCLUDED.status, procedure_id=EXCLUDED.procedure_id, dosage_plan=EXCLUDED.dosage_plan, administration_log=EXCLUDED.administration_log, adverse_events=EXCLUDED.adverse_events, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at`
	deleteTreatmentSQL          = `DELETE FROM treatments WHERE id=$1`
	insertTreatmentCohortSQL    = `INSERT INTO treatments__cohort_ids (treatment_id, cohort_id) VALUES ($1,$2)`
	deleteTreatmentCohortsSQL   = `DELETE FROM treatments__cohort_ids WHERE treatment_id=$1`
	insertTreatmentOrganismSQL  = `INSERT INTO treatments__organism_ids (treatment_id, organism_id) VALUES ($1,$2)`
	deleteTreatmentOrganismsSQL = `DELETE FROM treatments__organism_ids WHERE treatment_id=$1`
	selectTreatmentSQL          = `SELECT id, name, status, procedure_id, dosage_plan, administration_log, adverse_events, created_at, updated_at FROM treatments`
	selectTreatmentCohortsSQL   = `SELECT treatment_id, cohort_id FROM treatments__cohort_ids`
	selectTreatmentOrganismsSQL = `SELECT treatment_id, organism_id FROM treatments__organism_ids`
)

// --- helpers ---

func marshalJSONNullable(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func marshalJSONRequired(label string, value any) ([]byte, error) {
	if sliceEmpty(value) {
		return nil, fmt.Errorf("%s is required", label)
	}
	return json.Marshal(value)
}

func sliceEmpty(v any) bool {
	switch t := v.(type) {
	case []string:
		return len(t) == 0
	case []domain.SampleCustodyEvent:
		return len(t) == 0
	default:
		return false
	}
}

func decodeStringSlice(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeCustody(raw []byte) ([]domain.SampleCustodyEvent, error) {
	if len(raw) == 0 {
		return nil, errors.New("chain_of_custody cannot be empty")
	}
	var out []domain.SampleCustodyEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func nullableString(val sql.NullString) *string {
	if val.Valid {
		return &val.String
	}
	return nil
}

func nullableTime(val sql.NullTime) *time.Time {
	if val.Valid {
		return &val.Time
	}
	return nil
}
