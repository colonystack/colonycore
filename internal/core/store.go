package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type memoryState struct {
	organisms  map[string]Organism
	cohorts    map[string]Cohort
	housing    map[string]HousingUnit
	breeding   map[string]BreedingUnit
	procedures map[string]Procedure
	protocols  map[string]Protocol
	projects   map[string]Project
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

// MemoryStore provides an in-memory transactional store for the core domain.
type MemoryStore struct {
	mu     sync.RWMutex
	state  memoryState
	engine *RulesEngine
	nowFn  func() time.Time
}

// NewMemoryStore constructs an in-memory store backed by the provided rules engine.
func NewMemoryStore(engine *RulesEngine) *MemoryStore {
	if engine == nil {
		engine = NewRulesEngine()
	}
	return &MemoryStore{
		state:  newMemoryState(),
		engine: engine,
		nowFn:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *MemoryStore) newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// Transaction represents a mutation set applied to the store state.
type Transaction struct {
	store   *MemoryStore
	state   memoryState
	changes []Change
	now     time.Time
}

// TransactionView exposes a read-only snapshot of the transactional state to rules.
type TransactionView struct {
	state *memoryState
}

func newTransactionView(state *memoryState) TransactionView {
	return TransactionView{state: state}
}

// ListOrganisms returns all organisms within the transaction snapshot.
func (v TransactionView) ListOrganisms() []Organism {
	out := make([]Organism, 0, len(v.state.organisms))
	for _, o := range v.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}

// ListHousingUnits returns all housing units.
func (v TransactionView) ListHousingUnits() []HousingUnit {
	out := make([]HousingUnit, 0, len(v.state.housing))
	for _, h := range v.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}

// FindOrganism retrieves an organism by ID from the snapshot.
func (v TransactionView) FindOrganism(id string) (Organism, bool) {
	o, ok := v.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}

// FindHousingUnit retrieves a housing unit by ID from the snapshot.
func (v TransactionView) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := v.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}

// ListProtocols returns all protocols present in the snapshot.
func (v TransactionView) ListProtocols() []Protocol {
	out := make([]Protocol, 0, len(v.state.protocols))
	for _, p := range v.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}

// RunInTransaction executes fn within a transactional copy of the store state.
func (s *MemoryStore) RunInTransaction(ctx context.Context, fn func(tx *Transaction) error) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &Transaction{
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
			return res, RuleViolationError{Result: res}
		}
	}

	s.state = tx.state
	return result, nil
}

// View executes fn against a read-only snapshot of the store state.
func (s *MemoryStore) View(ctx context.Context, fn func(TransactionView) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.state.clone()
	view := newTransactionView(&snapshot)
	return fn(view)
}

// helper to record and append change entries.
func (tx *Transaction) recordChange(change Change) {
	tx.changes = append(tx.changes, change)
}

// CreateOrganism stores a new organism within the transaction.
func (tx *Transaction) CreateOrganism(o Organism) (Organism, error) {
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
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionCreate, After: cloneOrganism(o)})
	return cloneOrganism(o), nil
}

// UpdateOrganism mutates an organism using the provided mutator function.
func (tx *Transaction) UpdateOrganism(id string, mutator func(*Organism) error) (Organism, error) {
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
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionUpdate, Before: before, After: cloneOrganism(current)})
	return cloneOrganism(current), nil
}

// DeleteOrganism removes an organism from the transaction state.
func (tx *Transaction) DeleteOrganism(id string) error {
	current, ok := tx.state.organisms[id]
	if !ok {
		return fmt.Errorf("organism %q not found", id)
	}
	delete(tx.state.organisms, id)
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionDelete, Before: cloneOrganism(current)})
	return nil
}

// CreateCohort stores a new cohort.
func (tx *Transaction) CreateCohort(c Cohort) (Cohort, error) {
	if c.ID == "" {
		c.ID = tx.store.newID()
	}
	if _, exists := tx.state.cohorts[c.ID]; exists {
		return Cohort{}, fmt.Errorf("cohort %q already exists", c.ID)
	}
	c.CreatedAt = tx.now
	c.UpdatedAt = tx.now
	tx.state.cohorts[c.ID] = cloneCohort(c)
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionCreate, After: cloneCohort(c)})
	return cloneCohort(c), nil
}

// UpdateCohort mutates an existing cohort.
func (tx *Transaction) UpdateCohort(id string, mutator func(*Cohort) error) (Cohort, error) {
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
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionUpdate, Before: before, After: cloneCohort(current)})
	return cloneCohort(current), nil
}

// DeleteCohort removes a cohort from state.
func (tx *Transaction) DeleteCohort(id string) error {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return fmt.Errorf("cohort %q not found", id)
	}
	delete(tx.state.cohorts, id)
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionDelete, Before: cloneCohort(current)})
	return nil
}

// CreateHousingUnit stores new housing metadata.
func (tx *Transaction) CreateHousingUnit(h HousingUnit) (HousingUnit, error) {
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
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionCreate, After: cloneHousing(h)})
	return cloneHousing(h), nil
}

// UpdateHousingUnit mutates an existing housing unit.
func (tx *Transaction) UpdateHousingUnit(id string, mutator func(*HousingUnit) error) (HousingUnit, error) {
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
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionUpdate, Before: before, After: cloneHousing(current)})
	return cloneHousing(current), nil
}

// DeleteHousingUnit removes housing metadata.
func (tx *Transaction) DeleteHousingUnit(id string) error {
	current, ok := tx.state.housing[id]
	if !ok {
		return fmt.Errorf("housing unit %q not found", id)
	}
	delete(tx.state.housing, id)
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionDelete, Before: cloneHousing(current)})
	return nil
}

// CreateBreedingUnit stores a new breeding unit definition.
func (tx *Transaction) CreateBreedingUnit(b BreedingUnit) (BreedingUnit, error) {
	if b.ID == "" {
		b.ID = tx.store.newID()
	}
	if _, exists := tx.state.breeding[b.ID]; exists {
		return BreedingUnit{}, fmt.Errorf("breeding unit %q already exists", b.ID)
	}
	b.CreatedAt = tx.now
	b.UpdatedAt = tx.now
	tx.state.breeding[b.ID] = cloneBreeding(b)
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionCreate, After: cloneBreeding(b)})
	return cloneBreeding(b), nil
}

// UpdateBreedingUnit mutates an existing breeding unit.
func (tx *Transaction) UpdateBreedingUnit(id string, mutator func(*BreedingUnit) error) (BreedingUnit, error) {
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
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionUpdate, Before: before, After: cloneBreeding(current)})
	return cloneBreeding(current), nil
}

// DeleteBreedingUnit removes a breeding unit.
func (tx *Transaction) DeleteBreedingUnit(id string) error {
	current, ok := tx.state.breeding[id]
	if !ok {
		return fmt.Errorf("breeding unit %q not found", id)
	}
	delete(tx.state.breeding, id)
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionDelete, Before: cloneBreeding(current)})
	return nil
}

// CreateProcedure stores a procedure record.
func (tx *Transaction) CreateProcedure(p Procedure) (Procedure, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.procedures[p.ID]; exists {
		return Procedure{}, fmt.Errorf("procedure %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.procedures[p.ID] = cloneProcedure(p)
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionCreate, After: cloneProcedure(p)})
	return cloneProcedure(p), nil
}

// UpdateProcedure mutates a procedure.
func (tx *Transaction) UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error) {
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
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionUpdate, Before: before, After: cloneProcedure(current)})
	return cloneProcedure(current), nil
}

// DeleteProcedure removes a procedure.
func (tx *Transaction) DeleteProcedure(id string) error {
	current, ok := tx.state.procedures[id]
	if !ok {
		return fmt.Errorf("procedure %q not found", id)
	}
	delete(tx.state.procedures, id)
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionDelete, Before: cloneProcedure(current)})
	return nil
}

// CreateProtocol stores a new protocol record.
func (tx *Transaction) CreateProtocol(p Protocol) (Protocol, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.protocols[p.ID]; exists {
		return Protocol{}, fmt.Errorf("protocol %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.protocols[p.ID] = cloneProtocol(p)
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionCreate, After: cloneProtocol(p)})
	return cloneProtocol(p), nil
}

// UpdateProtocol mutates an existing protocol.
func (tx *Transaction) UpdateProtocol(id string, mutator func(*Protocol) error) (Protocol, error) {
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
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionUpdate, Before: before, After: cloneProtocol(current)})
	return cloneProtocol(current), nil
}

// DeleteProtocol removes a protocol from state.
func (tx *Transaction) DeleteProtocol(id string) error {
	current, ok := tx.state.protocols[id]
	if !ok {
		return fmt.Errorf("protocol %q not found", id)
	}
	delete(tx.state.protocols, id)
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionDelete, Before: cloneProtocol(current)})
	return nil
}

// CreateProject stores a project record.
func (tx *Transaction) CreateProject(p Project) (Project, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.projects[p.ID]; exists {
		return Project{}, fmt.Errorf("project %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.projects[p.ID] = cloneProject(p)
	tx.recordChange(Change{Entity: EntityProject, Action: ActionCreate, After: cloneProject(p)})
	return cloneProject(p), nil
}

// UpdateProject mutates an existing project record.
func (tx *Transaction) UpdateProject(id string, mutator func(*Project) error) (Project, error) {
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
	tx.recordChange(Change{Entity: EntityProject, Action: ActionUpdate, Before: before, After: cloneProject(current)})
	return cloneProject(current), nil
}

// DeleteProject removes a project from state.
func (tx *Transaction) DeleteProject(id string) error {
	current, ok := tx.state.projects[id]
	if !ok {
		return fmt.Errorf("project %q not found", id)
	}
	delete(tx.state.projects, id)
	tx.recordChange(Change{Entity: EntityProject, Action: ActionDelete, Before: cloneProject(current)})
	return nil
}

// Read helpers ---------------------------------------------------------------

// GetOrganism retrieves an organism by ID from committed state.
func (s *MemoryStore) GetOrganism(id string) (Organism, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}

// ListOrganisms returns all organisms from committed state.
func (s *MemoryStore) ListOrganisms() []Organism {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Organism, 0, len(s.state.organisms))
	for _, o := range s.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}

// GetHousingUnit retrieves a housing unit by ID.
func (s *MemoryStore) GetHousingUnit(id string) (HousingUnit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}

// ListHousingUnits returns all housing units.
func (s *MemoryStore) ListHousingUnits() []HousingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HousingUnit, 0, len(s.state.housing))
	for _, h := range s.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}

// ListCohorts returns all cohorts.
func (s *MemoryStore) ListCohorts() []Cohort {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Cohort, 0, len(s.state.cohorts))
	for _, c := range s.state.cohorts {
		out = append(out, cloneCohort(c))
	}
	return out
}

// ListProtocols returns all protocol records.
func (s *MemoryStore) ListProtocols() []Protocol {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Protocol, 0, len(s.state.protocols))
	for _, p := range s.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}

// ListProjects returns all projects.
func (s *MemoryStore) ListProjects() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Project, 0, len(s.state.projects))
	for _, p := range s.state.projects {
		out = append(out, cloneProject(p))
	}
	return out
}

// ListBreedingUnits returns all breeding units.
func (s *MemoryStore) ListBreedingUnits() []BreedingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BreedingUnit, 0, len(s.state.breeding))
	for _, b := range s.state.breeding {
		out = append(out, cloneBreeding(b))
	}
	return out
}

// ListProcedures returns all procedures.
func (s *MemoryStore) ListProcedures() []Procedure {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Procedure, 0, len(s.state.procedures))
	for _, p := range s.state.procedures {
		out = append(out, cloneProcedure(p))
	}
	return out
}
