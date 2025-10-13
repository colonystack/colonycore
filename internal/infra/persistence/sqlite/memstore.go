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
	// Facility is an alias of domain.Facility.
	Facility = domain.Facility
	// Procedure is an alias of domain.Procedure.
	Procedure = domain.Procedure
	// Treatment is an alias of domain.Treatment.
	Treatment = domain.Treatment
	// Observation is an alias of domain.Observation.
	Observation = domain.Observation
	// Sample is an alias of domain.Sample.
	Sample = domain.Sample
	// Protocol is an alias of domain.Protocol.
	Protocol = domain.Protocol
	// Permit is an alias of domain.Permit.
	Permit = domain.Permit
	// Project is an alias of domain.Project.
	Project = domain.Project
	// SupplyItem is an alias of domain.SupplyItem.
	SupplyItem = domain.SupplyItem
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

// Infra implementations use domain types directly via their interfaces
// No constant aliases needed - use domain.EntityType, domain.Action values directly

type memoryState struct {
	organisms    map[string]Organism
	cohorts      map[string]Cohort
	housing      map[string]HousingUnit
	facilities   map[string]Facility
	breeding     map[string]BreedingUnit
	procedures   map[string]Procedure
	treatments   map[string]Treatment
	observations map[string]Observation
	samples      map[string]Sample
	protocols    map[string]Protocol
	permits      map[string]Permit
	projects     map[string]Project
	supplies     map[string]SupplyItem
}

// Snapshot is the serialisable representation of the in-memory state.
type Snapshot struct {
	Organisms    map[string]Organism     `json:"organisms"`
	Cohorts      map[string]Cohort       `json:"cohorts"`
	Housing      map[string]HousingUnit  `json:"housing"`
	Facilities   map[string]Facility     `json:"facilities"`
	Breeding     map[string]BreedingUnit `json:"breeding"`
	Procedures   map[string]Procedure    `json:"procedures"`
	Treatments   map[string]Treatment    `json:"treatments"`
	Observations map[string]Observation  `json:"observations"`
	Samples      map[string]Sample       `json:"samples"`
	Protocols    map[string]Protocol     `json:"protocols"`
	Permits      map[string]Permit       `json:"permits"`
	Projects     map[string]Project      `json:"projects"`
	Supplies     map[string]SupplyItem   `json:"supplies"`
}

func newMemoryState() memoryState {
	return memoryState{
		organisms:    map[string]Organism{},
		cohorts:      map[string]Cohort{},
		housing:      map[string]HousingUnit{},
		facilities:   map[string]Facility{},
		breeding:     map[string]BreedingUnit{},
		procedures:   map[string]Procedure{},
		treatments:   map[string]Treatment{},
		observations: map[string]Observation{},
		samples:      map[string]Sample{},
		protocols:    map[string]Protocol{},
		permits:      map[string]Permit{},
		projects:     map[string]Project{},
		supplies:     map[string]SupplyItem{},
	}
}

func snapshotFromMemoryState(state memoryState) Snapshot {
	s := Snapshot{
		Organisms:    make(map[string]Organism, len(state.organisms)),
		Cohorts:      make(map[string]Cohort, len(state.cohorts)),
		Housing:      make(map[string]HousingUnit, len(state.housing)),
		Facilities:   make(map[string]Facility, len(state.facilities)),
		Breeding:     make(map[string]BreedingUnit, len(state.breeding)),
		Procedures:   make(map[string]Procedure, len(state.procedures)),
		Treatments:   make(map[string]Treatment, len(state.treatments)),
		Observations: make(map[string]Observation, len(state.observations)),
		Samples:      make(map[string]Sample, len(state.samples)),
		Protocols:    make(map[string]Protocol, len(state.protocols)),
		Permits:      make(map[string]Permit, len(state.permits)),
		Projects:     make(map[string]Project, len(state.projects)),
		Supplies:     make(map[string]SupplyItem, len(state.supplies)),
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
	for k, v := range state.facilities {
		s.Facilities[k] = cloneFacility(v)
	}
	for k, v := range state.breeding {
		s.Breeding[k] = cloneBreeding(v)
	}
	for k, v := range state.procedures {
		s.Procedures[k] = cloneProcedure(v)
	}
	for k, v := range state.treatments {
		s.Treatments[k] = cloneTreatment(v)
	}
	for k, v := range state.observations {
		s.Observations[k] = cloneObservation(v)
	}
	for k, v := range state.samples {
		s.Samples[k] = cloneSample(v)
	}
	for k, v := range state.protocols {
		s.Protocols[k] = cloneProtocol(v)
	}
	for k, v := range state.permits {
		s.Permits[k] = clonePermit(v)
	}
	for k, v := range state.projects {
		s.Projects[k] = cloneProject(v)
	}
	for k, v := range state.supplies {
		s.Supplies[k] = cloneSupplyItem(v)
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
	for k, v := range s.Facilities {
		st.facilities[k] = cloneFacility(v)
	}
	for k, v := range s.Breeding {
		st.breeding[k] = cloneBreeding(v)
	}
	for k, v := range s.Procedures {
		st.procedures[k] = cloneProcedure(v)
	}
	for k, v := range s.Treatments {
		st.treatments[k] = cloneTreatment(v)
	}
	for k, v := range s.Observations {
		st.observations[k] = cloneObservation(v)
	}
	for k, v := range s.Samples {
		st.samples[k] = cloneSample(v)
	}
	for k, v := range s.Protocols {
		st.protocols[k] = cloneProtocol(v)
	}
	for k, v := range s.Permits {
		st.permits[k] = clonePermit(v)
	}
	for k, v := range s.Projects {
		st.projects[k] = cloneProject(v)
	}
	for k, v := range s.Supplies {
		st.supplies[k] = cloneSupplyItem(v)
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

func cloneFacility(f Facility) Facility {
	cp := f
	if f.EnvironmentBaselines != nil {
		cp.EnvironmentBaselines = make(map[string]any, len(f.EnvironmentBaselines))
		for k, v := range f.EnvironmentBaselines {
			cp.EnvironmentBaselines[k] = v
		}
	}
	cp.HousingUnitIDs = append([]string(nil), f.HousingUnitIDs...)
	cp.ProjectIDs = append([]string(nil), f.ProjectIDs...)
	return cp
}

func cloneTreatment(t Treatment) Treatment {
	cp := t
	cp.OrganismIDs = append([]string(nil), t.OrganismIDs...)
	cp.CohortIDs = append([]string(nil), t.CohortIDs...)
	cp.AdministrationLog = append([]string(nil), t.AdministrationLog...)
	cp.AdverseEvents = append([]string(nil), t.AdverseEvents...)
	return cp
}

func cloneObservation(o Observation) Observation {
	cp := o
	if o.Data != nil {
		cp.Data = make(map[string]any, len(o.Data))
		for k, v := range o.Data {
			cp.Data[k] = v
		}
	}
	return cp
}

func cloneSample(s Sample) Sample {
	cp := s
	cp.ChainOfCustody = append([]domain.SampleCustodyEvent(nil), s.ChainOfCustody...)
	if s.Attributes != nil {
		cp.Attributes = make(map[string]any, len(s.Attributes))
		for k, v := range s.Attributes {
			cp.Attributes[k] = v
		}
	}
	return cp
}

func clonePermit(p Permit) Permit {
	cp := p
	cp.AllowedActivities = append([]string(nil), p.AllowedActivities...)
	cp.FacilityIDs = append([]string(nil), p.FacilityIDs...)
	cp.ProtocolIDs = append([]string(nil), p.ProtocolIDs...)
	return cp
}

func cloneSupplyItem(s SupplyItem) SupplyItem {
	cp := s
	if s.ExpiresAt != nil {
		t := *s.ExpiresAt
		cp.ExpiresAt = &t
	}
	cp.FacilityIDs = append([]string(nil), s.FacilityIDs...)
	cp.ProjectIDs = append([]string(nil), s.ProjectIDs...)
	if s.Attributes != nil {
		cp.Attributes = make(map[string]any, len(s.Attributes))
		for k, v := range s.Attributes {
			cp.Attributes[k] = v
		}
	}
	return cp
}

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
func (v transactionView) ListFacilities() []Facility {
	out := make([]Facility, 0, len(v.state.facilities))
	for _, f := range v.state.facilities {
		out = append(out, cloneFacility(f))
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
func (v transactionView) FindFacility(id string) (Facility, bool) {
	f, ok := v.state.facilities[id]
	if !ok {
		return Facility{}, false
	}
	return cloneFacility(f), true
}
func (v transactionView) ListProtocols() []Protocol {
	out := make([]Protocol, 0, len(v.state.protocols))
	for _, p := range v.state.protocols {
		out = append(out, cloneProtocol(p))
	}
	return out
}
func (v transactionView) ListTreatments() []Treatment {
	out := make([]Treatment, 0, len(v.state.treatments))
	for _, t := range v.state.treatments {
		out = append(out, cloneTreatment(t))
	}
	return out
}
func (v transactionView) FindTreatment(id string) (Treatment, bool) {
	t, ok := v.state.treatments[id]
	if !ok {
		return Treatment{}, false
	}
	return cloneTreatment(t), true
}
func (v transactionView) ListObservations() []Observation {
	out := make([]Observation, 0, len(v.state.observations))
	for _, o := range v.state.observations {
		out = append(out, cloneObservation(o))
	}
	return out
}
func (v transactionView) FindObservation(id string) (Observation, bool) {
	o, ok := v.state.observations[id]
	if !ok {
		return Observation{}, false
	}
	return cloneObservation(o), true
}
func (v transactionView) ListSamples() []Sample {
	out := make([]Sample, 0, len(v.state.samples))
	for _, s := range v.state.samples {
		out = append(out, cloneSample(s))
	}
	return out
}
func (v transactionView) FindSample(id string) (Sample, bool) {
	s, ok := v.state.samples[id]
	if !ok {
		return Sample{}, false
	}
	return cloneSample(s), true
}
func (v transactionView) ListPermits() []Permit {
	out := make([]Permit, 0, len(v.state.permits))
	for _, p := range v.state.permits {
		out = append(out, clonePermit(p))
	}
	return out
}
func (v transactionView) FindPermit(id string) (Permit, bool) {
	p, ok := v.state.permits[id]
	if !ok {
		return Permit{}, false
	}
	return clonePermit(p), true
}
func (v transactionView) ListProjects() []Project {
	out := make([]Project, 0, len(v.state.projects))
	for _, p := range v.state.projects {
		out = append(out, cloneProject(p))
	}
	return out
}
func (v transactionView) ListSupplyItems() []SupplyItem {
	out := make([]SupplyItem, 0, len(v.state.supplies))
	for _, s := range v.state.supplies {
		out = append(out, cloneSupplyItem(s))
	}
	return out
}
func (v transactionView) FindSupplyItem(id string) (SupplyItem, bool) {
	s, ok := v.state.supplies[id]
	if !ok {
		return SupplyItem{}, false
	}
	return cloneSupplyItem(s), true
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
func (tx *transaction) FindFacility(id string) (Facility, bool) {
	f, ok := tx.state.facilities[id]
	if !ok {
		return Facility{}, false
	}
	return cloneFacility(f), true
}
func (tx *transaction) FindTreatment(id string) (Treatment, bool) {
	t, ok := tx.state.treatments[id]
	if !ok {
		return Treatment{}, false
	}
	return cloneTreatment(t), true
}
func (tx *transaction) FindObservation(id string) (Observation, bool) {
	o, ok := tx.state.observations[id]
	if !ok {
		return Observation{}, false
	}
	return cloneObservation(o), true
}
func (tx *transaction) FindSample(id string) (Sample, bool) {
	s, ok := tx.state.samples[id]
	if !ok {
		return Sample{}, false
	}
	return cloneSample(s), true
}
func (tx *transaction) FindPermit(id string) (Permit, bool) {
	p, ok := tx.state.permits[id]
	if !ok {
		return Permit{}, false
	}
	return clonePermit(p), true
}
func (tx *transaction) FindSupplyItem(id string) (SupplyItem, bool) {
	s, ok := tx.state.supplies[id]
	if !ok {
		return SupplyItem{}, false
	}
	return cloneSupplyItem(s), true
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
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionCreate, After: cloneOrganism(o)})
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
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionUpdate, Before: before, After: cloneOrganism(current)})
	return cloneOrganism(current), nil
}
func (tx *transaction) DeleteOrganism(id string) error {
	current, ok := tx.state.organisms[id]
	if !ok {
		return fmt.Errorf("organism %q not found", id)
	}
	delete(tx.state.organisms, id)
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionDelete, Before: cloneOrganism(current)})
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
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionCreate, After: cloneCohort(c)})
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
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionUpdate, Before: before, After: cloneCohort(current)})
	return cloneCohort(current), nil
}
func (tx *transaction) DeleteCohort(id string) error {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return fmt.Errorf("cohort %q not found", id)
	}
	delete(tx.state.cohorts, id)
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionDelete, Before: cloneCohort(current)})
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
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionCreate, After: cloneHousing(h)})
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
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionUpdate, Before: before, After: cloneHousing(current)})
	return cloneHousing(current), nil
}
func (tx *transaction) DeleteHousingUnit(id string) error {
	current, ok := tx.state.housing[id]
	if !ok {
		return fmt.Errorf("housing unit %q not found", id)
	}
	delete(tx.state.housing, id)
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionDelete, Before: cloneHousing(current)})
	return nil
}
func (tx *transaction) CreateFacility(f Facility) (Facility, error) {
	if f.ID == "" {
		f.ID = tx.store.newID()
	}
	if _, exists := tx.state.facilities[f.ID]; exists {
		return Facility{}, fmt.Errorf("facility %q already exists", f.ID)
	}
	f.CreatedAt = tx.now
	f.UpdatedAt = tx.now
	if f.EnvironmentBaselines == nil {
		f.EnvironmentBaselines = map[string]any{}
	}
	tx.state.facilities[f.ID] = cloneFacility(f)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionCreate, After: cloneFacility(f)})
	return cloneFacility(f), nil
}
func (tx *transaction) UpdateFacility(id string, mutator func(*Facility) error) (Facility, error) {
	current, ok := tx.state.facilities[id]
	if !ok {
		return Facility{}, fmt.Errorf("facility %q not found", id)
	}
	before := cloneFacility(current)
	if err := mutator(&current); err != nil {
		return Facility{}, err
	}
	if current.EnvironmentBaselines == nil {
		current.EnvironmentBaselines = map[string]any{}
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.facilities[id] = cloneFacility(current)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionUpdate, Before: before, After: cloneFacility(current)})
	return cloneFacility(current), nil
}
func (tx *transaction) DeleteFacility(id string) error {
	current, ok := tx.state.facilities[id]
	if !ok {
		return fmt.Errorf("facility %q not found", id)
	}
	delete(tx.state.facilities, id)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionDelete, Before: cloneFacility(current)})
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
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionCreate, After: cloneBreeding(b)})
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
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionUpdate, Before: before, After: cloneBreeding(current)})
	return cloneBreeding(current), nil
}
func (tx *transaction) DeleteBreedingUnit(id string) error {
	current, ok := tx.state.breeding[id]
	if !ok {
		return fmt.Errorf("breeding unit %q not found", id)
	}
	delete(tx.state.breeding, id)
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionDelete, Before: cloneBreeding(current)})
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
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionCreate, After: cloneProcedure(p)})
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
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionUpdate, Before: before, After: cloneProcedure(current)})
	return cloneProcedure(current), nil
}
func (tx *transaction) DeleteProcedure(id string) error {
	current, ok := tx.state.procedures[id]
	if !ok {
		return fmt.Errorf("procedure %q not found", id)
	}
	delete(tx.state.procedures, id)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionDelete, Before: cloneProcedure(current)})
	return nil
}
func (tx *transaction) CreateTreatment(t Treatment) (Treatment, error) {
	if t.ID == "" {
		t.ID = tx.store.newID()
	}
	if _, exists := tx.state.treatments[t.ID]; exists {
		return Treatment{}, fmt.Errorf("treatment %q already exists", t.ID)
	}
	t.CreatedAt = tx.now
	t.UpdatedAt = tx.now
	tx.state.treatments[t.ID] = cloneTreatment(t)
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionCreate, After: cloneTreatment(t)})
	return cloneTreatment(t), nil
}
func (tx *transaction) UpdateTreatment(id string, mutator func(*Treatment) error) (Treatment, error) {
	current, ok := tx.state.treatments[id]
	if !ok {
		return Treatment{}, fmt.Errorf("treatment %q not found", id)
	}
	before := cloneTreatment(current)
	if err := mutator(&current); err != nil {
		return Treatment{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.treatments[id] = cloneTreatment(current)
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionUpdate, Before: before, After: cloneTreatment(current)})
	return cloneTreatment(current), nil
}
func (tx *transaction) DeleteTreatment(id string) error {
	current, ok := tx.state.treatments[id]
	if !ok {
		return fmt.Errorf("treatment %q not found", id)
	}
	delete(tx.state.treatments, id)
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionDelete, Before: cloneTreatment(current)})
	return nil
}
func (tx *transaction) CreateObservation(o Observation) (Observation, error) {
	if o.ID == "" {
		o.ID = tx.store.newID()
	}
	if _, exists := tx.state.observations[o.ID]; exists {
		return Observation{}, fmt.Errorf("observation %q already exists", o.ID)
	}
	o.CreatedAt = tx.now
	o.UpdatedAt = tx.now
	if o.Data == nil {
		o.Data = map[string]any{}
	}
	tx.state.observations[o.ID] = cloneObservation(o)
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionCreate, After: cloneObservation(o)})
	return cloneObservation(o), nil
}
func (tx *transaction) UpdateObservation(id string, mutator func(*Observation) error) (Observation, error) {
	current, ok := tx.state.observations[id]
	if !ok {
		return Observation{}, fmt.Errorf("observation %q not found", id)
	}
	before := cloneObservation(current)
	if err := mutator(&current); err != nil {
		return Observation{}, err
	}
	if current.Data == nil {
		current.Data = map[string]any{}
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.observations[id] = cloneObservation(current)
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionUpdate, Before: before, After: cloneObservation(current)})
	return cloneObservation(current), nil
}
func (tx *transaction) DeleteObservation(id string) error {
	current, ok := tx.state.observations[id]
	if !ok {
		return fmt.Errorf("observation %q not found", id)
	}
	delete(tx.state.observations, id)
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionDelete, Before: cloneObservation(current)})
	return nil
}
func (tx *transaction) CreateSample(s Sample) (Sample, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.samples[s.ID]; exists {
		return Sample{}, fmt.Errorf("sample %q already exists", s.ID)
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if s.Attributes == nil {
		s.Attributes = map[string]any{}
	}
	tx.state.samples[s.ID] = cloneSample(s)
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionCreate, After: cloneSample(s)})
	return cloneSample(s), nil
}
func (tx *transaction) UpdateSample(id string, mutator func(*Sample) error) (Sample, error) {
	current, ok := tx.state.samples[id]
	if !ok {
		return Sample{}, fmt.Errorf("sample %q not found", id)
	}
	before := cloneSample(current)
	if err := mutator(&current); err != nil {
		return Sample{}, err
	}
	if current.Attributes == nil {
		current.Attributes = map[string]any{}
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.samples[id] = cloneSample(current)
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionUpdate, Before: before, After: cloneSample(current)})
	return cloneSample(current), nil
}
func (tx *transaction) DeleteSample(id string) error {
	current, ok := tx.state.samples[id]
	if !ok {
		return fmt.Errorf("sample %q not found", id)
	}
	delete(tx.state.samples, id)
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionDelete, Before: cloneSample(current)})
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
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionCreate, After: cloneProtocol(p)})
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
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionUpdate, Before: before, After: cloneProtocol(current)})
	return cloneProtocol(current), nil
}
func (tx *transaction) DeleteProtocol(id string) error {
	current, ok := tx.state.protocols[id]
	if !ok {
		return fmt.Errorf("protocol %q not found", id)
	}
	delete(tx.state.protocols, id)
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionDelete, Before: cloneProtocol(current)})
	return nil
}
func (tx *transaction) CreatePermit(p Permit) (Permit, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.permits[p.ID]; exists {
		return Permit{}, fmt.Errorf("permit %q already exists", p.ID)
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.permits[p.ID] = clonePermit(p)
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionCreate, After: clonePermit(p)})
	return clonePermit(p), nil
}
func (tx *transaction) UpdatePermit(id string, mutator func(*Permit) error) (Permit, error) {
	current, ok := tx.state.permits[id]
	if !ok {
		return Permit{}, fmt.Errorf("permit %q not found", id)
	}
	before := clonePermit(current)
	if err := mutator(&current); err != nil {
		return Permit{}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.permits[id] = clonePermit(current)
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionUpdate, Before: before, After: clonePermit(current)})
	return clonePermit(current), nil
}
func (tx *transaction) DeletePermit(id string) error {
	current, ok := tx.state.permits[id]
	if !ok {
		return fmt.Errorf("permit %q not found", id)
	}
	delete(tx.state.permits, id)
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionDelete, Before: clonePermit(current)})
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
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionCreate, After: cloneProject(p)})
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
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionUpdate, Before: before, After: cloneProject(current)})
	return cloneProject(current), nil
}
func (tx *transaction) DeleteProject(id string) error {
	current, ok := tx.state.projects[id]
	if !ok {
		return fmt.Errorf("project %q not found", id)
	}
	delete(tx.state.projects, id)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionDelete, Before: cloneProject(current)})
	return nil
}
func (tx *transaction) CreateSupplyItem(s SupplyItem) (SupplyItem, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.supplies[s.ID]; exists {
		return SupplyItem{}, fmt.Errorf("supply item %q already exists", s.ID)
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if s.Attributes == nil {
		s.Attributes = map[string]any{}
	}
	tx.state.supplies[s.ID] = cloneSupplyItem(s)
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionCreate, After: cloneSupplyItem(s)})
	return cloneSupplyItem(s), nil
}
func (tx *transaction) UpdateSupplyItem(id string, mutator func(*SupplyItem) error) (SupplyItem, error) {
	current, ok := tx.state.supplies[id]
	if !ok {
		return SupplyItem{}, fmt.Errorf("supply item %q not found", id)
	}
	before := cloneSupplyItem(current)
	if err := mutator(&current); err != nil {
		return SupplyItem{}, err
	}
	if current.Attributes == nil {
		current.Attributes = map[string]any{}
	}
	if current.ExpiresAt != nil {
		t := *current.ExpiresAt
		current.ExpiresAt = &t
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.supplies[id] = cloneSupplyItem(current)
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionUpdate, Before: before, After: cloneSupplyItem(current)})
	return cloneSupplyItem(current), nil
}
func (tx *transaction) DeleteSupplyItem(id string) error {
	current, ok := tx.state.supplies[id]
	if !ok {
		return fmt.Errorf("supply item %q not found", id)
	}
	delete(tx.state.supplies, id)
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionDelete, Before: cloneSupplyItem(current)})
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
func (s *memStore) GetFacility(id string) (Facility, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.state.facilities[id]
	if !ok {
		return Facility{}, false
	}
	return cloneFacility(f), true
}
func (s *memStore) ListFacilities() []Facility {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Facility, 0, len(s.state.facilities))
	for _, f := range s.state.facilities {
		out = append(out, cloneFacility(f))
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
func (s *memStore) ListTreatments() []Treatment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Treatment, 0, len(s.state.treatments))
	for _, t := range s.state.treatments {
		out = append(out, cloneTreatment(t))
	}
	return out
}
func (s *memStore) ListObservations() []Observation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Observation, 0, len(s.state.observations))
	for _, o := range s.state.observations {
		out = append(out, cloneObservation(o))
	}
	return out
}
func (s *memStore) ListSamples() []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Sample, 0, len(s.state.samples))
	for _, sample := range s.state.samples {
		out = append(out, cloneSample(sample))
	}
	return out
}
func (s *memStore) GetPermit(id string) (Permit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.state.permits[id]
	if !ok {
		return Permit{}, false
	}
	return clonePermit(p), true
}
func (s *memStore) ListPermits() []Permit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Permit, 0, len(s.state.permits))
	for _, p := range s.state.permits {
		out = append(out, clonePermit(p))
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
func (s *memStore) ListSupplyItems() []SupplyItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SupplyItem, 0, len(s.state.supplies))
	for _, sitem := range s.state.supplies {
		out = append(out, cloneSupplyItem(sitem))
	}
	return out
}
