// Package sqlite provides an in-memory transactional store plus supporting
// helpers that the SQLite persistent store builds upon. It lives under infra
// to keep domain dependencies one-way (domain -> nothing).
package sqlite

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

// Exported aliases to keep method signatures concise while still exposing
// domain types from this infra package.
type (
	// Organism is a colony organism (alias of domain.Organism).
	Organism = domain.Organism
	// Cohort is an alias of domain.Cohort.
	Cohort = domain.Cohort
	// HousingUnit is an alias of domain.HousingUnit.
	HousingUnit = domain.HousingUnit
	// BreedingUnit is an alias of domain.BreedingUnit.
	BreedingUnit = domain.BreedingUnit
	// Procedure is an alias of domain.Procedure.
	Procedure = domain.Procedure
	// Protocol is an alias of domain.Protocol.
	Protocol = domain.Protocol
	// Project is an alias of domain.Project.
	Project = domain.Project
	// Change is an alias of domain.Change.
	Change = domain.Change
	// Result is an alias of domain.Result.
	Result = domain.Result
	// RulesEngine is an alias of domain.RulesEngine.
	RulesEngine = domain.RulesEngine
	// Transaction is an alias of domain.Transaction.
	Transaction = domain.Transaction
	// TransactionView is an alias of domain.TransactionView.
	TransactionView = domain.TransactionView
	// PersistentStore is an alias of domain.PersistentStore.
	PersistentStore = domain.PersistentStore
)

// Entity constants forwarded from domain for change records.
const (
	EntityOrganism    = domain.EntityOrganism
	EntityCohort      = domain.EntityCohort
	EntityHousingUnit = domain.EntityHousingUnit
	EntityBreeding    = domain.EntityBreeding
	EntityProcedure   = domain.EntityProcedure
	EntityProtocol    = domain.EntityProtocol
	EntityProject     = domain.EntityProject
)

// Action constants forwarded from domain for change records.
const (
	ActionCreate = domain.ActionCreate
	ActionUpdate = domain.ActionUpdate
	ActionDelete = domain.ActionDelete
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

// Snapshot is the serialisable representation of the in-memory state.
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
		organisms:  map[string]Organism{},
		cohorts:    map[string]Cohort{},
		housing:    map[string]HousingUnit{},
		breeding:   map[string]BreedingUnit{},
		procedures: map[string]Procedure{},
		protocols:  map[string]Protocol{},
		projects:   map[string]Project{},
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
	st := newMemoryState()
	for k, v := range s.Organisms {
		st.organisms[k] = cloneOrganism(v)
	}
	for k, v := range s.Cohorts {
		st.cohorts[k] = cloneCohort(v)
	}
	for k, v := range s.Housing {
		st.housing[k] = cloneHousing(v)
	}
	for k, v := range s.Breeding {
		st.breeding[k] = cloneBreeding(v)
	}
	for k, v := range s.Procedures {
		st.procedures[k] = cloneProcedure(v)
	}
	for k, v := range s.Protocols {
		st.protocols[k] = cloneProtocol(v)
	}
	for k, v := range s.Projects {
		st.projects[k] = cloneProject(v)
	}
	return st
}

func (s memoryState) clone() memoryState { return memoryStateFromSnapshot(snapshotFromMemoryState(s)) }

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

type memStore struct {
	mu     sync.RWMutex
	state  memoryState
	engine *RulesEngine
	nowFn  func() time.Time
}

func newMemStore(engine *RulesEngine) *memStore {
	if engine == nil {
		engine = domain.NewRulesEngine()
	}
	return &memStore{state: newMemoryState(), engine: engine, nowFn: func() time.Time { return time.Now().UTC() }}
}
func (s *memStore) newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func (s *memStore) ExportState() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshotFromMemoryState(s.state)
}
func (s *memStore) ImportState(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = memoryStateFromSnapshot(snapshot)
}
func (s *memStore) RulesEngine() *RulesEngine { s.mu.RLock(); defer s.mu.RUnlock(); return s.engine }
func (s *memStore) NowFunc() func() time.Time { s.mu.RLock(); defer s.mu.RUnlock(); return s.nowFn }

type transaction struct {
	store   *memStore
	state   memoryState
	changes []Change
	now     time.Time
}
type transactionView struct{ state *memoryState }

func newTransactionView(state *memoryState) TransactionView { return transactionView{state: state} }
func (v transactionView) ListOrganisms() []Organism {
	out := make([]Organism, 0, len(v.state.organisms))
	for _, o := range v.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}
func (v transactionView) ListHousingUnits() []HousingUnit {
	out := make([]HousingUnit, 0, len(v.state.housing))
	for _, h := range v.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}
func (v transactionView) FindOrganism(id string) (Organism, bool) {
	o, ok := v.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}
func (v transactionView) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := v.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}
func (v transactionView) ListProtocols() []Protocol {
	out := make([]Protocol, 0, len(v.state.protocols))
	for _, p := range v.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}

func (s *memStore) RunInTransaction(ctx context.Context, fn func(tx Transaction) error) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx := &transaction{store: s, state: s.state.clone(), now: s.nowFn()}
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

func (s *memStore) View(_ context.Context, fn func(TransactionView) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.state.clone()
	view := newTransactionView(&snapshot)
	return fn(view)
}
func (tx *transaction) recordChange(change Change) { tx.changes = append(tx.changes, change) }
func (tx *transaction) Snapshot() TransactionView  { return newTransactionView(&tx.state) }
func (tx *transaction) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := tx.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}
func (tx *transaction) FindProtocol(id string) (Protocol, bool) {
	p, ok := tx.state.protocols[id]
	if !ok {
		return Protocol{}, false
	}
	return cloneProtocol(p), true
}
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
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionCreate, After: cloneOrganism(o)})
	return cloneOrganism(o), nil
}
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
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionUpdate, Before: before, After: cloneOrganism(current)})
	return cloneOrganism(current), nil
}
func (tx *transaction) DeleteOrganism(id string) error {
	current, ok := tx.state.organisms[id]
	if !ok {
		return fmt.Errorf("organism %q not found", id)
	}
	delete(tx.state.organisms, id)
	tx.recordChange(Change{Entity: EntityOrganism, Action: ActionDelete, Before: cloneOrganism(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionCreate, After: cloneCohort(c)})
	return cloneCohort(c), nil
}
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
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionUpdate, Before: before, After: cloneCohort(current)})
	return cloneCohort(current), nil
}
func (tx *transaction) DeleteCohort(id string) error {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return fmt.Errorf("cohort %q not found", id)
	}
	delete(tx.state.cohorts, id)
	tx.recordChange(Change{Entity: EntityCohort, Action: ActionDelete, Before: cloneCohort(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionCreate, After: cloneHousing(h)})
	return cloneHousing(h), nil
}
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
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionUpdate, Before: before, After: cloneHousing(current)})
	return cloneHousing(current), nil
}
func (tx *transaction) DeleteHousingUnit(id string) error {
	current, ok := tx.state.housing[id]
	if !ok {
		return fmt.Errorf("housing unit %q not found", id)
	}
	delete(tx.state.housing, id)
	tx.recordChange(Change{Entity: EntityHousingUnit, Action: ActionDelete, Before: cloneHousing(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionCreate, After: cloneBreeding(b)})
	return cloneBreeding(b), nil
}
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
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionUpdate, Before: before, After: cloneBreeding(current)})
	return cloneBreeding(current), nil
}
func (tx *transaction) DeleteBreedingUnit(id string) error {
	current, ok := tx.state.breeding[id]
	if !ok {
		return fmt.Errorf("breeding unit %q not found", id)
	}
	delete(tx.state.breeding, id)
	tx.recordChange(Change{Entity: EntityBreeding, Action: ActionDelete, Before: cloneBreeding(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionCreate, After: cloneProcedure(p)})
	return cloneProcedure(p), nil
}
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
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionUpdate, Before: before, After: cloneProcedure(current)})
	return cloneProcedure(current), nil
}
func (tx *transaction) DeleteProcedure(id string) error {
	current, ok := tx.state.procedures[id]
	if !ok {
		return fmt.Errorf("procedure %q not found", id)
	}
	delete(tx.state.procedures, id)
	tx.recordChange(Change{Entity: EntityProcedure, Action: ActionDelete, Before: cloneProcedure(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionCreate, After: cloneProtocol(p)})
	return cloneProtocol(p), nil
}
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
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionUpdate, Before: before, After: cloneProtocol(current)})
	return cloneProtocol(current), nil
}
func (tx *transaction) DeleteProtocol(id string) error {
	current, ok := tx.state.protocols[id]
	if !ok {
		return fmt.Errorf("protocol %q not found", id)
	}
	delete(tx.state.protocols, id)
	tx.recordChange(Change{Entity: EntityProtocol, Action: ActionDelete, Before: cloneProtocol(current)})
	return nil
}
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
	tx.recordChange(Change{Entity: EntityProject, Action: ActionCreate, After: cloneProject(p)})
	return cloneProject(p), nil
}
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
	tx.recordChange(Change{Entity: EntityProject, Action: ActionUpdate, Before: before, After: cloneProject(current)})
	return cloneProject(current), nil
}
func (tx *transaction) DeleteProject(id string) error {
	current, ok := tx.state.projects[id]
	if !ok {
		return fmt.Errorf("project %q not found", id)
	}
	delete(tx.state.projects, id)
	tx.recordChange(Change{Entity: EntityProject, Action: ActionDelete, Before: cloneProject(current)})
	return nil
}
func (s *memStore) GetOrganism(id string) (Organism, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.state.organisms[id]
	if !ok {
		return Organism{}, false
	}
	return cloneOrganism(o), true
}
func (s *memStore) ListOrganisms() []Organism {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Organism, 0, len(s.state.organisms))
	for _, o := range s.state.organisms {
		out = append(out, cloneOrganism(o))
	}
	return out
}
func (s *memStore) GetHousingUnit(id string) (HousingUnit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.state.housing[id]
	if !ok {
		return HousingUnit{}, false
	}
	return cloneHousing(h), true
}
func (s *memStore) ListHousingUnits() []HousingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HousingUnit, 0, len(s.state.housing))
	for _, h := range s.state.housing {
		out = append(out, cloneHousing(h))
	}
	return out
}
func (s *memStore) ListCohorts() []Cohort {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Cohort, 0, len(s.state.cohorts))
	for _, c := range s.state.cohorts {
		out = append(out, cloneCohort(c))
	}
	return out
}
func (s *memStore) ListProtocols() []Protocol {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Protocol, 0, len(s.state.protocols))
	for _, p := range s.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}
func (s *memStore) ListProjects() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Project, 0, len(s.state.projects))
	for _, p := range s.state.projects {
		out = append(out, cloneProject(p))
	}
	return out
}
func (s *memStore) ListBreedingUnits() []BreedingUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BreedingUnit, 0, len(s.state.breeding))
	for _, b := range s.state.breeding {
		out = append(out, cloneBreeding(b))
	}
	return out
}
func (s *memStore) ListProcedures() []Procedure {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Procedure, 0, len(s.state.procedures))
	for _, p := range s.state.procedures {
		out = append(out, cloneProcedure(p))
	}
	return out
}
