// Package memory provides an in-memory implementation of the core persistence
// store used for tests and ephemeral environments.
package memory

import (
	"colonycore/pkg/domain"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Compile-time contract assertions ensuring memory.Store adheres to the domain persistence interfaces.
var _ domain.PersistentStore = (*Store)(nil)

type (
	// Organism aliases domain.Organism for in-memory persistence operations.
	Organism = domain.Organism
	// Cohort aliases domain.Cohort.
	Cohort = domain.Cohort
	// HousingUnit aliases domain.HousingUnit.
	HousingUnit = domain.HousingUnit
	// BreedingUnit aliases domain.BreedingUnit.
	BreedingUnit = domain.BreedingUnit
	// Procedure aliases domain.Procedure.
	Procedure = domain.Procedure
	// Protocol aliases domain.Protocol.
	Protocol = domain.Protocol
	// Project aliases domain.Project.
	Project = domain.Project
	// Change aliases domain.Change captured in transactions.
	Change = domain.Change
	// Result aliases domain.Result summarizing rule evaluation.
	Result = domain.Result
	// RulesEngine aliases domain.RulesEngine used to evaluate rules.
	RulesEngine = domain.RulesEngine
	// Transaction aliases domain.Transaction representing a mutable unit of work.
	Transaction = domain.Transaction
	// TransactionView aliases domain.TransactionView providing read-only state.
	TransactionView = domain.TransactionView
	// PersistentStore aliases domain.PersistentStore abstraction.
	PersistentStore = domain.PersistentStore
)

// Infra implementations use domain types directly via their interfaces
// No constant aliases needed - use domain.EntityType, domain.Action values directly

type memoryState struct {
	organisms  map[string]Organism
	cohorts    map[string]Cohort
	housing    map[string]HousingUnit
	breeding   map[string]BreedingUnit
	procedures map[string]Procedure
	protocols  map[string]Protocol
	projects   map[string]Project
}

// Snapshot captures a point-in-time clone of the store state.
type Snapshot struct {
	Organisms  map[string]Organism     `json:"organisms"`
	Cohorts    map[string]Cohort       `json:"cohorts"`
	Housing    map[string]HousingUnit  `json:"housing"`
	Breeding   map[string]BreedingUnit `json:"breeding"`
	Procedures map[string]Procedure    `json:"procedures"`
	Protocols  map[string]Protocol     `json:"protocols"`
	Projects   map[string]Project      `json:"projects"`
}

func newMemoryState() memoryState {
	return memoryState{
		organisms:  make(map[string]Organism),
		cohorts:    make(map[string]Cohort),
		housing:    make(map[string]HousingUnit),
		breeding:   make(map[string]BreedingUnit),
		procedures: make(map[string]Procedure),
		protocols:  make(map[string]Protocol),
		projects:   make(map[string]Project),
	}
}

func snapshotFromMemoryState(state memoryState) Snapshot {
	s := Snapshot{
		Organisms:  make(map[string]Organism, len(state.organisms)),
		Cohorts:    make(map[string]Cohort, len(state.cohorts)),
		Housing:    make(map[string]HousingUnit, len(state.housing)),
		Breeding:   make(map[string]BreedingUnit, len(state.breeding)),
		Procedures: make(map[string]Procedure, len(state.procedures)),
		Protocols:  make(map[string]Protocol, len(state.protocols)),
		Projects:   make(map[string]Project, len(state.projects)),
	}
	for k, v := range state.organisms {
		s.Organisms[k] = cloneOrganism(v)
	}
	for k, v := range state.cohorts {
		s.Cohorts[k] = cloneCohort(v)
	}
	for k, v := range state.housing {
		s.Housing[k] = cloneHousing(v)
	}
	for k, v := range state.breeding {
		s.Breeding[k] = cloneBreeding(v)
	}
	for k, v := range state.procedures {
		s.Procedures[k] = cloneProcedure(v)
	}
	for k, v := range state.protocols {
		s.Protocols[k] = cloneProtocol(v)
	}
	for k, v := range state.projects {
		s.Projects[k] = cloneProject(v)
	}
	return s
}

func memoryStateFromSnapshot(s Snapshot) memoryState {
	state := newMemoryState()
	for k, v := range s.Organisms {
		state.organisms[k] = cloneOrganism(v)
	}
	for k, v := range s.Cohorts {
		state.cohorts[k] = cloneCohort(v)
	}
	for k, v := range s.Housing {
		state.housing[k] = cloneHousing(v)
	}
	for k, v := range s.Breeding {
		state.breeding[k] = cloneBreeding(v)
	}
	for k, v := range s.Procedures {
		state.procedures[k] = cloneProcedure(v)
	}
	for k, v := range s.Protocols {
		state.protocols[k] = cloneProtocol(v)
	}
	for k, v := range s.Projects {
		state.projects[k] = cloneProject(v)
	}
	return state
}

func (s memoryState) clone() memoryState {
	cloned := newMemoryState()
	for k, v := range s.organisms {
		cloned.organisms[k] = cloneOrganism(v)
	}
	for k, v := range s.cohorts {
		cloned.cohorts[k] = cloneCohort(v)
	}
	for k, v := range s.housing {
		cloned.housing[k] = cloneHousing(v)
	}
	for k, v := range s.breeding {
		cloned.breeding[k] = cloneBreeding(v)
	}
	for k, v := range s.procedures {
		cloned.procedures[k] = cloneProcedure(v)
	}
	for k, v := range s.protocols {
		cloned.protocols[k] = cloneProtocol(v)
	}
	for k, v := range s.projects {
		cloned.projects[k] = cloneProject(v)
	}
	return cloned
}

func cloneOrganism(o Organism) Organism {
	cp := o
	if o.Attributes != nil {
		cp.Attributes = make(map[string]any, len(o.Attributes))
		for k, v := range o.Attributes {
			cp.Attributes[k] = v
		}
	}
	return cp
}

func cloneCohort(c Cohort) Cohort            { return c }
func cloneHousing(h HousingUnit) HousingUnit { return h }
func cloneBreeding(b BreedingUnit) BreedingUnit {
	cp := b
	cp.FemaleIDs = append([]string(nil), b.FemaleIDs...)
	cp.MaleIDs = append([]string(nil), b.MaleIDs...)
	return cp
}

func cloneProcedure(p Procedure) Procedure {
	cp := p
	cp.OrganismIDs = append([]string(nil), p.OrganismIDs...)
	return cp
}
func cloneProtocol(p Protocol) Protocol { return p }
func cloneProject(p Project) Project    { return p }

// Store provides an in-memory transactional store for the core domain.
type Store struct {
	mu     sync.RWMutex
	state  memoryState
	engine *RulesEngine
	nowFn  func() time.Time
}

// NewStore constructs an in-memory store backed by the provided rules engine.
func NewStore(engine *RulesEngine) *Store {
	if engine == nil {
		engine = domain.NewRulesEngine()
	}
	return &Store{
		state:  newMemoryState(),
		engine: engine,
		nowFn:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Store) newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// ExportState clones the current store state for external persistence.
func (s *Store) ExportState() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshotFromMemoryState(s.state)
}

// ImportState replaces the store state with the provided snapshot.
func (s *Store) ImportState(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = memoryStateFromSnapshot(snapshot)
}

// RulesEngine exposes the currently configured engine for integration points like plugins.
func (s *Store) RulesEngine() *RulesEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.engine
}

// NowFunc returns the time provider used by the in-memory store.
func (s *Store) NowFunc() func() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nowFn
}

// Transaction represents a mutation set applied to the store state.
type transaction struct {
	store   *Store
	state   memoryState
	changes []Change
	now     time.Time
}

// TransactionView exposes a read-only snapshot of the transactional state to rules.
type transactionView struct {
	state *memoryState
}

func newTransactionView(state *memoryState) TransactionView {
	return transactionView{state: state}
}

// ListOrganisms returns all organisms within the transaction snapshot.
func (v transactionView) ListOrganisms() []Organism {
	out := make([]Organism, 0, len(v.state.organisms))
	for _, o := range v.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}

// ListHousingUnits returns all housing units.
func (v transactionView) ListHousingUnits() []HousingUnit {
	out := make([]HousingUnit, 0, len(v.state.housing))
	for _, h := range v.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}

// FindOrganism retrieves an organism by ID from the snapshot.
func (v transactionView) FindOrganism(id string) (Organism, bool) {
	o, ok := v.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}

// FindHousingUnit retrieves a housing unit by ID from the snapshot.
func (v transactionView) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := v.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}

// ListProtocols returns all protocols present in the snapshot.
func (v transactionView) ListProtocols() []Protocol {
	out := make([]Protocol, 0, len(v.state.protocols))
	for _, p := range v.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}

// RunInTransaction executes fn within a transactional copy of the store state.
func (s *Store) RunInTransaction(ctx context.Context, fn func(tx Transaction) error) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &transaction{
		store: s,
		state: s.state.clone(),
		now:   s.nowFn(),
	}

	if err := fn(tx); err != nil {
		return Result{}, err
	}

	var result Result
	if s.engine != nil {
		view := newTransactionView(&tx.state)
		res, err := s.engine.Evaluate(ctx, view, tx.changes)
		if err != nil {
			return Result{}, err
		}
		result = res
		if res.HasBlocking() {
			return res, domain.RuleViolationError{Result: res}
		}
	}

	s.state = tx.state
	return result, nil
}

// View executes fn against a read-only snapshot of the store state.
func (s *Store) View(_ context.Context, fn func(TransactionView) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.state.clone()
	view := newTransactionView(&snapshot)
	return fn(view)
}

// helper to record and append change entries.
func (tx *transaction) recordChange(change Change) {
	tx.changes = append(tx.changes, change)
}

// Snapshot returns a read-only view over the transactional state.
func (tx *transaction) Snapshot() TransactionView {
	return newTransactionView(&tx.state)
}

// FindHousingUnit exposes housing lookup within the transaction scope.
func (tx *transaction) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := tx.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}

// FindProtocol exposes protocol lookup within the transaction scope.
func (tx *transaction) FindProtocol(id string) (Protocol, bool) {
	p, ok := tx.state.protocols[id]
	if !ok {
		return Protocol{}, false
	}
	return cloneProtocol(p), true
}

// CreateOrganism stores a new organism within the transaction.
func (tx *transaction) CreateOrganism(o Organism) (Organism, error) {
	if o.ID == "" {
		o.ID = tx.store.newID()
	}
	if _, exists := tx.state.organisms[o.ID]; exists {
		return Organism{}, fmt.Errorf("organism %q already exists", o.ID)
	}
	o.CreatedAt = tx.now
	o.UpdatedAt = tx.now
	if o.Attributes == nil {
		o.Attributes = map[string]any{}
	}
	tx.state.organisms[o.ID] = cloneOrganism(o)
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionCreate, After: cloneOrganism(o)})
	return cloneOrganism(o), nil
}

// UpdateOrganism mutates an organism using the provided mutator function.
func (tx *transaction) UpdateOrganism(id string, mutator func(*Organism) error) (Organism, error) {
	current, ok := tx.state.organisms[id]
	if !ok {
		return Organism{}, fmt.Errorf("organism %q not found", id)
	}
	before := cloneOrganism(current)
	if err := mutator(&current); err != nil {
		return Organism{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.organisms[id] = cloneOrganism(current)
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionUpdate, Before: before, After: cloneOrganism(current)})
	return cloneOrganism(current), nil
}

// DeleteOrganism removes an organism from the transaction state.
func (tx *transaction) DeleteOrganism(id string) error {
	current, ok := tx.state.organisms[id]
	if !ok {
		return fmt.Errorf("organism %q not found", id)
	}
	delete(tx.state.organisms, id)
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionDelete, Before: cloneOrganism(current)})
	return nil
}

// CreateCohort stores a new cohort.
func (tx *transaction) CreateCohort(c Cohort) (Cohort, error) {
	if c.ID == "" {
		c.ID = tx.store.newID()
	}
	if _, exists := tx.state.cohorts[c.ID]; exists {
		return Cohort{}, fmt.Errorf("cohort %q already exists", c.ID)
	}
	c.CreatedAt = tx.now
	c.UpdatedAt = tx.now
	tx.state.cohorts[c.ID] = cloneCohort(c)
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionCreate, After: cloneCohort(c)})
	return cloneCohort(c), nil
}

// UpdateCohort mutates an existing cohort.
func (tx *transaction) UpdateCohort(id string, mutator func(*Cohort) error) (Cohort, error) {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return Cohort{}, fmt.Errorf("cohort %q not found", id)
	}
	before := cloneCohort(current)
	if err := mutator(&current); err != nil {
		return Cohort{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.cohorts[id] = cloneCohort(current)
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionUpdate, Before: before, After: cloneCohort(current)})
	return cloneCohort(current), nil
}

// DeleteCohort removes a cohort from state.
func (tx *transaction) DeleteCohort(id string) error {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return fmt.Errorf("cohort %q not found", id)
	}
	delete(tx.state.cohorts, id)
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionDelete, Before: cloneCohort(current)})
	return nil
}

// CreateHousingUnit stores new housing metadata.
func (tx *transaction) CreateHousingUnit(h HousingUnit) (HousingUnit, error) {
	if h.ID == "" {
		h.ID = tx.store.newID()
	}
	if _, exists := tx.state.housing[h.ID]; exists {
		return HousingUnit{}, fmt.Errorf("housing unit %q already exists", h.ID)
	}
	if h.Capacity <= 0 {
		return HousingUnit{}, errors.New("housing capacity must be positive")
	}
	h.CreatedAt = tx.now
	h.UpdatedAt = tx.now
	tx.state.housing[h.ID] = cloneHousing(h)
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionCreate, After: cloneHousing(h)})
	return cloneHousing(h), nil
}

// UpdateHousingUnit mutates an existing housing unit.
func (tx *transaction) UpdateHousingUnit(id string, mutator func(*HousingUnit) error) (HousingUnit, error) {
	current, ok := tx.state.housing[id]
	if !ok {
		return HousingUnit{}, fmt.Errorf("housing unit %q not found", id)
	}
	before := cloneHousing(current)
	if err := mutator(&current); err != nil {
		return HousingUnit{}, err
	}
	if current.Capacity <= 0 {
		return HousingUnit{}, errors.New("housing capacity must be positive")
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.housing[id] = cloneHousing(current)
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionUpdate, Before: before, After: cloneHousing(current)})
	return cloneHousing(current), nil
}

// DeleteHousingUnit removes housing metadata.
func (tx *transaction) DeleteHousingUnit(id string) error {
	current, ok := tx.state.housing[id]
	if !ok {
		return fmt.Errorf("housing unit %q not found", id)
	}
	delete(tx.state.housing, id)
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionDelete, Before: cloneHousing(current)})
	return nil
}

// CreateBreedingUnit stores a new breeding unit definition.
func (tx *transaction) CreateBreedingUnit(b BreedingUnit) (BreedingUnit, error) {
	if b.ID == "" {
		b.ID = tx.store.newID()
	}
	if _, exists := tx.state.breeding[b.ID]; exists {
		return BreedingUnit{}, fmt.Errorf("breeding unit %q already exists", b.ID)
	}
	b.CreatedAt = tx.now
	b.UpdatedAt = tx.now
	tx.state.breeding[b.ID] = cloneBreeding(b)
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionCreate, After: cloneBreeding(b)})
	return cloneBreeding(b), nil
}

// UpdateBreedingUnit mutates an existing breeding unit.
func (tx *transaction) UpdateBreedingUnit(id string, mutator func(*BreedingUnit) error) (BreedingUnit, error) {
	current, ok := tx.state.breeding[id]
	if !ok {
		return BreedingUnit{}, fmt.Errorf("breeding unit %q not found", id)
	}
	before := cloneBreeding(current)
	if err := mutator(&current); err != nil {
		return BreedingUnit{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.breeding[id] = cloneBreeding(current)
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionUpdate, Before: before, After: cloneBreeding(current)})
	return cloneBreeding(current), nil
}

// DeleteBreedingUnit removes a breeding unit.
func (tx *transaction) DeleteBreedingUnit(id string) error {
	current, ok := tx.state.breeding[id]
	if !ok {
		return fmt.Errorf("breeding unit %q not found", id)
	}
	delete(tx.state.breeding, id)
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionDelete, Before: cloneBreeding(current)})
	return nil
}

// CreateProcedure stores a procedure record.
func (tx *transaction) CreateProcedure(p Procedure) (Procedure, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.procedures[p.ID]; exists {
		return Procedure{}, fmt.Errorf("procedure %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.procedures[p.ID] = cloneProcedure(p)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionCreate, After: cloneProcedure(p)})
	return cloneProcedure(p), nil
}

// UpdateProcedure mutates a procedure.
func (tx *transaction) UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error) {
	current, ok := tx.state.procedures[id]
	if !ok {
		return Procedure{}, fmt.Errorf("procedure %q not found", id)
	}
	before := cloneProcedure(current)
	if err := mutator(&current); err != nil {
		return Procedure{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.procedures[id] = cloneProcedure(current)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionUpdate, Before: before, After: cloneProcedure(current)})
	return cloneProcedure(current), nil
}

// DeleteProcedure removes a procedure.
func (tx *transaction) DeleteProcedure(id string) error {
	current, ok := tx.state.procedures[id]
	if !ok {
		return fmt.Errorf("procedure %q not found", id)
	}
	delete(tx.state.procedures, id)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionDelete, Before: cloneProcedure(current)})
	return nil
}

// CreateProtocol stores a new protocol record.
func (tx *transaction) CreateProtocol(p Protocol) (Protocol, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.protocols[p.ID]; exists {
		return Protocol{}, fmt.Errorf("protocol %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.protocols[p.ID] = cloneProtocol(p)
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionCreate, After: cloneProtocol(p)})
	return cloneProtocol(p), nil
}

// UpdateProtocol mutates an existing protocol.
func (tx *transaction) UpdateProtocol(id string, mutator func(*Protocol) error) (Protocol, error) {
	current, ok := tx.state.protocols[id]
	if !ok {
		return Protocol{}, fmt.Errorf("protocol %q not found", id)
	}
	before := cloneProtocol(current)
	if err := mutator(&current); err != nil {
		return Protocol{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.protocols[id] = cloneProtocol(current)
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionUpdate, Before: before, After: cloneProtocol(current)})
	return cloneProtocol(current), nil
}

// DeleteProtocol removes a protocol from state.
func (tx *transaction) DeleteProtocol(id string) error {
	current, ok := tx.state.protocols[id]
	if !ok {
		return fmt.Errorf("protocol %q not found", id)
	}
	delete(tx.state.protocols, id)
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionDelete, Before: cloneProtocol(current)})
	return nil
}

// CreateProject stores a project record.
func (tx *transaction) CreateProject(p Project) (Project, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.projects[p.ID]; exists {
		return Project{}, fmt.Errorf("project %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.projects[p.ID] = cloneProject(p)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionCreate, After: cloneProject(p)})
	return cloneProject(p), nil
}

// UpdateProject mutates an existing project record.
func (tx *transaction) UpdateProject(id string, mutator func(*Project) error) (Project, error) {
	current, ok := tx.state.projects[id]
	if !ok {
		return Project{}, fmt.Errorf("project %q not found", id)
	}
	before := cloneProject(current)
	if err := mutator(&current); err != nil {
		return Project{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.projects[id] = cloneProject(current)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionUpdate, Before: before, After: cloneProject(current)})
	return cloneProject(current), nil
}

// DeleteProject removes a project from state.
func (tx *transaction) DeleteProject(id string) error {
	current, ok := tx.state.projects[id]
	if !ok {
		return fmt.Errorf("project %q not found", id)
	}
	delete(tx.state.projects, id)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionDelete, Before: cloneProject(current)})
	return nil
}

// Read helpers ---------------------------------------------------------------

// GetOrganism retrieves an organism by ID from committed state.
func (s *Store) GetOrganism(id string) (Organism, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}

// ListOrganisms returns all organisms from committed state.
func (s *Store) ListOrganisms() []Organism {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Organism, 0, len(s.state.organisms))
	for _, o := range s.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}

// GetHousingUnit retrieves a housing unit by ID.
func (s *Store) GetHousingUnit(id string) (HousingUnit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}

// ListHousingUnits returns all housing units.
func (s *Store) ListHousingUnits() []HousingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HousingUnit, 0, len(s.state.housing))
	for _, h := range s.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}

// ListCohorts returns all cohorts.
func (s *Store) ListCohorts() []Cohort {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Cohort, 0, len(s.state.cohorts))
	for _, c := range s.state.cohorts {
		out = append(out, cloneCohort(c))
	}
	return out
}

// ListProtocols returns all protocol records.
func (s *Store) ListProtocols() []Protocol {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Protocol, 0, len(s.state.protocols))
	for _, p := range s.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}

// ListProjects returns all projects.
func (s *Store) ListProjects() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Project, 0, len(s.state.projects))
	for _, p := range s.state.projects {
		out = append(out, cloneProject(p))
	}
	return out
}

// ListBreedingUnits returns all breeding units.
func (s *Store) ListBreedingUnits() []BreedingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BreedingUnit, 0, len(s.state.breeding))
	for _, b := range s.state.breeding {
		out = append(out, cloneBreeding(b))
	}
	return out
}

// ListProcedures returns all procedures.
func (s *Store) ListProcedures() []Procedure {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Procedure, 0, len(s.state.procedures))
	for _, p := range s.state.procedures {
		out = append(out, cloneProcedure(p))
	}
	return out
}
