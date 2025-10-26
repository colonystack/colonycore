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
	"sort"
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

//nolint:gocyclo // migrateSnapshot aggregates multiple migration concerns in one pass for parity with existing snapshots.
func migrateSnapshot(snapshot Snapshot) Snapshot {
	if snapshot.Organisms == nil {
		snapshot.Organisms = map[string]Organism{}
	}
	if snapshot.Cohorts == nil {
		snapshot.Cohorts = map[string]Cohort{}
	}
	if snapshot.Housing == nil {
		snapshot.Housing = map[string]HousingUnit{}
	}
	if snapshot.Facilities == nil {
		snapshot.Facilities = map[string]Facility{}
	}
	if snapshot.Breeding == nil {
		snapshot.Breeding = map[string]BreedingUnit{}
	}
	if snapshot.Procedures == nil {
		snapshot.Procedures = map[string]Procedure{}
	}
	if snapshot.Treatments == nil {
		snapshot.Treatments = map[string]Treatment{}
	}
	if snapshot.Observations == nil {
		snapshot.Observations = map[string]Observation{}
	}
	if snapshot.Samples == nil {
		snapshot.Samples = map[string]Sample{}
	}
	if snapshot.Protocols == nil {
		snapshot.Protocols = map[string]Protocol{}
	}
	if snapshot.Permits == nil {
		snapshot.Permits = map[string]Permit{}
	}
	if snapshot.Projects == nil {
		snapshot.Projects = map[string]Project{}
	}
	if snapshot.Supplies == nil {
		snapshot.Supplies = map[string]SupplyItem{}
	}

	facilityExists := func(id string) bool {
		_, ok := snapshot.Facilities[id]
		return ok
	}
	projectExists := func(id string) bool {
		_, ok := snapshot.Projects[id]
		return ok
	}
	organismExists := func(id string) bool {
		_, ok := snapshot.Organisms[id]
		return ok
	}
	cohortExists := func(id string) bool {
		_, ok := snapshot.Cohorts[id]
		return ok
	}
	procedureExists := func(id string) bool {
		_, ok := snapshot.Procedures[id]
		return ok
	}
	protocolExists := func(id string) bool {
		_, ok := snapshot.Protocols[id]
		return ok
	}

	for id, housing := range snapshot.Housing {
		if housing.FacilityID == "" || !facilityExists(housing.FacilityID) {
			delete(snapshot.Housing, id)
			continue
		}
		if housing.Capacity <= 0 {
			housing.Capacity = 1
		}
		snapshot.Housing[id] = housing
	}

	for id, treatment := range snapshot.Treatments {
		if treatment.ProcedureID == "" || !procedureExists(treatment.ProcedureID) {
			delete(snapshot.Treatments, id)
			continue
		}
		if filtered, changed := filterIDs(treatment.OrganismIDs, organismExists); changed {
			treatment.OrganismIDs = filtered
		}
		if filtered, changed := filterIDs(treatment.CohortIDs, cohortExists); changed {
			treatment.CohortIDs = filtered
		}
		snapshot.Treatments[id] = treatment
	}

	for id, observation := range snapshot.Observations {
		if data := observation.DataMap(); data == nil {
			observation.SetData(map[string]any{})
		} else {
			observation.SetData(data)
		}
		if observation.ProcedureID != nil && !procedureExists(*observation.ProcedureID) {
			observation.ProcedureID = nil
		}
		if observation.OrganismID != nil && !organismExists(*observation.OrganismID) {
			observation.OrganismID = nil
		}
		if observation.CohortID != nil && !cohortExists(*observation.CohortID) {
			observation.CohortID = nil
		}
		if observation.ProcedureID == nil && observation.OrganismID == nil && observation.CohortID == nil {
			delete(snapshot.Observations, id)
			continue
		}
		snapshot.Observations[id] = observation
	}

	for id, sample := range snapshot.Samples {
		if attrs := sample.AttributesMap(); attrs == nil {
			sample.SetAttributes(map[string]any{})
		} else {
			sample.SetAttributes(attrs)
		}
		if sample.FacilityID == "" || !facilityExists(sample.FacilityID) {
			delete(snapshot.Samples, id)
			continue
		}
		if sample.OrganismID != nil && !organismExists(*sample.OrganismID) {
			sample.OrganismID = nil
		}
		if sample.CohortID != nil && !cohortExists(*sample.CohortID) {
			sample.CohortID = nil
		}
		if sample.OrganismID == nil && sample.CohortID == nil {
			delete(snapshot.Samples, id)
			continue
		}
		snapshot.Samples[id] = sample
	}

	for id, permit := range snapshot.Permits {
		if filtered, changed := filterIDs(permit.FacilityIDs, facilityExists); changed {
			permit.FacilityIDs = filtered
		}
		if filtered, changed := filterIDs(permit.ProtocolIDs, protocolExists); changed {
			permit.ProtocolIDs = filtered
		}
		snapshot.Permits[id] = permit
	}

	for id, project := range snapshot.Projects {
		if filtered, changed := filterIDs(project.FacilityIDs, facilityExists); changed {
			project.FacilityIDs = filtered
		}
		snapshot.Projects[id] = project
	}

	for id, procedure := range snapshot.Procedures {
		var treatmentIDs []string
		for _, treatment := range snapshot.Treatments {
			if treatment.ProcedureID == id {
				treatmentIDs = append(treatmentIDs, treatment.ID)
			}
		}
		sort.Strings(treatmentIDs)
		procedure.TreatmentIDs = treatmentIDs

		var observationIDs []string
		for _, observation := range snapshot.Observations {
			if observation.ProcedureID != nil && *observation.ProcedureID == id {
				observationIDs = append(observationIDs, observation.ID)
			}
		}
		sort.Strings(observationIDs)
		procedure.ObservationIDs = observationIDs

		snapshot.Procedures[id] = procedure
	}

	for id, item := range snapshot.Supplies {
		if attrs := item.AttributesMap(); attrs == nil {
			item.SetAttributes(map[string]any{})
		} else {
			item.SetAttributes(attrs)
		}
		if filtered, changed := filterIDs(item.FacilityIDs, facilityExists); changed {
			item.FacilityIDs = filtered
		}
		if filtered, changed := filterIDs(item.ProjectIDs, projectExists); changed {
			item.ProjectIDs = filtered
		}
		snapshot.Supplies[id] = item
	}

	for id, facility := range snapshot.Facilities {
		if baselines := facility.EnvironmentBaselinesMap(); baselines == nil {
			facility.SetEnvironmentBaselines(map[string]any{})
		} else {
			facility.SetEnvironmentBaselines(baselines)
		}
		snapshot.Facilities[id] = facility
	}

	for id, facility := range snapshot.Facilities {
		var housingIDs []string
		for _, housing := range snapshot.Housing {
			if housing.FacilityID == id {
				housingIDs = append(housingIDs, housing.ID)
			}
		}
		sort.Strings(housingIDs)
		facility.HousingUnitIDs = housingIDs

		var projectIDs []string
		for _, project := range snapshot.Projects {
			if containsString(project.FacilityIDs, id) {
				projectIDs = append(projectIDs, project.ID)
			}
		}
		sort.Strings(projectIDs)
		facility.ProjectIDs = projectIDs

		snapshot.Facilities[id] = facility
	}

	for id, project := range snapshot.Projects {
		var organismIDs []string
		for _, organism := range snapshot.Organisms {
			if organism.ProjectID != nil && *organism.ProjectID == id {
				organismIDs = append(organismIDs, organism.ID)
			}
		}
		sort.Strings(organismIDs)
		project.OrganismIDs = organismIDs

		var procedureIDs []string
		for _, procedure := range snapshot.Procedures {
			if procedure.ProjectID != nil && *procedure.ProjectID == id {
				procedureIDs = append(procedureIDs, procedure.ID)
			}
		}
		sort.Strings(procedureIDs)
		project.ProcedureIDs = procedureIDs

		var supplyItemIDs []string
		for _, supply := range snapshot.Supplies {
			if containsString(supply.ProjectIDs, id) {
				supplyItemIDs = append(supplyItemIDs, supply.ID)
			}
		}
		sort.Strings(supplyItemIDs)
		project.SupplyItemIDs = supplyItemIDs

		snapshot.Projects[id] = project
	}

	return snapshot
}

func (s memoryState) clone() memoryState { return memoryStateFromSnapshot(snapshotFromMemoryState(s)) }

func cloneOrganism(o Organism) Organism {
	cp := o
	cp.SetAttributes(o.AttributesMap())
	if len(o.ParentIDs) != 0 {
		cp.ParentIDs = append([]string(nil), o.ParentIDs...)
	}
	return cp
}
func cloneCohort(c Cohort) Cohort            { return c }
func cloneHousing(h HousingUnit) HousingUnit { return h }
func cloneBreeding(b BreedingUnit) BreedingUnit {
	cp := b
	cp.FemaleIDs = append([]string(nil), b.FemaleIDs...)
	cp.MaleIDs = append([]string(nil), b.MaleIDs...)
	cp.SetPairingAttributes(b.PairingAttributesMap())
	return cp
}
func cloneProcedure(p Procedure) Procedure {
	cp := p
	cp.OrganismIDs = append([]string(nil), p.OrganismIDs...)
	cp.TreatmentIDs = append([]string(nil), p.TreatmentIDs...)
	cp.ObservationIDs = append([]string(nil), p.ObservationIDs...)
	return cp
}
func cloneProtocol(p Protocol) Protocol { return p }
func cloneProject(p Project) Project {
	cp := p
	cp.FacilityIDs = append([]string(nil), p.FacilityIDs...)
	cp.ProtocolIDs = append([]string(nil), p.ProtocolIDs...)
	cp.OrganismIDs = append([]string(nil), p.OrganismIDs...)
	cp.ProcedureIDs = append([]string(nil), p.ProcedureIDs...)
	cp.SupplyItemIDs = append([]string(nil), p.SupplyItemIDs...)
	return cp
}

func cloneFacility(f Facility) Facility {
	cp := f
	cp.SetEnvironmentBaselines(f.EnvironmentBaselinesMap())
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
	cp.SetData(o.DataMap())
	return cp
}

func cloneSample(s Sample) Sample {
	cp := s
	cp.ChainOfCustody = append([]domain.SampleCustodyEvent(nil), s.ChainOfCustody...)
	cp.SetAttributes(s.AttributesMap())
	return cp
}

func clonePermit(p Permit) Permit {
	cp := p
	cp.AllowedActivities = append([]string(nil), p.AllowedActivities...)
	cp.FacilityIDs = append([]string(nil), p.FacilityIDs...)
	cp.ProtocolIDs = append([]string(nil), p.ProtocolIDs...)
	return cp
}

func containsString(values []string, id string) bool {
	for _, existing := range values {
		if existing == id {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	if len(values) <= 1 {
		return append([]string(nil), values...)
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func filterIDs(values []string, exists func(string) bool) ([]string, bool) {
	if len(values) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	changed := false
	for _, v := range values {
		if _, ok := seen[v]; ok {
			changed = true
			continue
		}
		seen[v] = struct{}{}
		if !exists(v) {
			changed = true
			continue
		}
		out = append(out, v)
	}
	if !changed && len(out) == len(values) {
		return values, false
	}
	return out, true
}

func facilityHousingIDs(state *memoryState, facilityID string) []string {
	var ids []string
	for _, housing := range state.housing {
		if housing.FacilityID == facilityID {
			ids = append(ids, housing.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func facilityProjectIDs(state *memoryState, facilityID string) []string {
	var ids []string
	for _, project := range state.projects {
		if containsString(project.FacilityIDs, facilityID) {
			ids = append(ids, project.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func decorateFacility(state *memoryState, facility Facility) Facility {
	facility.HousingUnitIDs = facilityHousingIDs(state, facility.ID)
	facility.ProjectIDs = facilityProjectIDs(state, facility.ID)
	return facility
}

func procedureTreatmentIDs(state *memoryState, procedureID string) []string {
	var ids []string
	for _, treatment := range state.treatments {
		if treatment.ProcedureID == procedureID {
			ids = append(ids, treatment.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func procedureObservationIDs(state *memoryState, procedureID string) []string {
	var ids []string
	for _, observation := range state.observations {
		if observation.ProcedureID != nil && *observation.ProcedureID == procedureID {
			ids = append(ids, observation.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func decorateProcedure(state *memoryState, procedure Procedure) Procedure {
	procedure.TreatmentIDs = procedureTreatmentIDs(state, procedure.ID)
	procedure.ObservationIDs = procedureObservationIDs(state, procedure.ID)
	return procedure
}

func projectOrganismIDs(state *memoryState, projectID string) []string {
	var ids []string
	for _, organism := range state.organisms {
		if organism.ProjectID != nil && *organism.ProjectID == projectID {
			ids = append(ids, organism.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func projectProcedureIDs(state *memoryState, projectID string) []string {
	var ids []string
	for _, procedure := range state.procedures {
		if procedure.ProjectID != nil && *procedure.ProjectID == projectID {
			ids = append(ids, procedure.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func projectSupplyItemIDs(state *memoryState, projectID string) []string {
	var ids []string
	for _, item := range state.supplies {
		if containsString(item.ProjectIDs, projectID) {
			ids = append(ids, item.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func decorateProject(state *memoryState, project Project) Project {
	project.OrganismIDs = projectOrganismIDs(state, project.ID)
	project.ProcedureIDs = projectProcedureIDs(state, project.ID)
	project.SupplyItemIDs = projectSupplyItemIDs(state, project.ID)
	return project
}

func cloneSupplyItem(s SupplyItem) SupplyItem {
	cp := s
	if s.ExpiresAt != nil {
		t := *s.ExpiresAt
		cp.ExpiresAt = &t
	}
	cp.FacilityIDs = append([]string(nil), s.FacilityIDs...)
	cp.ProjectIDs = append([]string(nil), s.ProjectIDs...)
	cp.SetAttributes(s.AttributesMap())
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
	s.state = memoryStateFromSnapshot(migrateSnapshot(snapshot))
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
		out = append(out, cloneFacility(decorateFacility(v.state, f)))
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
	return cloneFacility(decorateFacility(v.state, f)), true
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
		out = append(out, cloneProject(decorateProject(v.state, p)))
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
	return cloneFacility(decorateFacility(&tx.state, f)), true
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
	if attrs := o.AttributesMap(); attrs == nil {
		o.SetAttributes(map[string]any{})
	} else {
		o.SetAttributes(attrs)
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
	for _, sample := range tx.state.samples {
		if sample.OrganismID != nil && *sample.OrganismID == id {
			return fmt.Errorf("organism %q still referenced by sample %q", id, sample.ID)
		}
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
	for _, sample := range tx.state.samples {
		if sample.CohortID != nil && *sample.CohortID == id {
			return fmt.Errorf("cohort %q still referenced by sample %q", id, sample.ID)
		}
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
	if h.FacilityID == "" {
		return HousingUnit{}, errors.New("housing unit requires facility id")
	}
	if _, ok := tx.state.facilities[h.FacilityID]; !ok {
		return HousingUnit{}, fmt.Errorf("facility %q not found", h.FacilityID)
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
	if current.FacilityID == "" {
		return HousingUnit{}, errors.New("housing unit requires facility id")
	}
	if _, ok := tx.state.facilities[current.FacilityID]; !ok {
		return HousingUnit{}, fmt.Errorf("facility %q not found", current.FacilityID)
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
	f.HousingUnitIDs = nil
	f.ProjectIDs = nil
	if baselines := f.EnvironmentBaselinesMap(); baselines == nil {
		f.SetEnvironmentBaselines(map[string]any{})
	} else {
		f.SetEnvironmentBaselines(baselines)
	}
	tx.state.facilities[f.ID] = cloneFacility(f)
	created := decorateFacility(&tx.state, f)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionCreate, After: cloneFacility(created)})
	return cloneFacility(created), nil
}
func (tx *transaction) UpdateFacility(id string, mutator func(*Facility) error) (Facility, error) {
	current, ok := tx.state.facilities[id]
	if !ok {
		return Facility{}, fmt.Errorf("facility %q not found", id)
	}
	beforeDecorated := decorateFacility(&tx.state, current)
	before := cloneFacility(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Facility{}, err
	}
	if baselines := current.EnvironmentBaselinesMap(); baselines == nil {
		current.SetEnvironmentBaselines(map[string]any{})
	} else {
		current.SetEnvironmentBaselines(baselines)
	}
	current.HousingUnitIDs = nil
	current.ProjectIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.facilities[id] = cloneFacility(current)
	afterDecorated := decorateFacility(&tx.state, current)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionUpdate, Before: before, After: cloneFacility(afterDecorated)})
	return cloneFacility(afterDecorated), nil
}
func (tx *transaction) DeleteFacility(id string) error {
	current, ok := tx.state.facilities[id]
	if !ok {
		return fmt.Errorf("facility %q not found", id)
	}
	decoratedCurrent := decorateFacility(&tx.state, current)
	if count := len(facilityHousingIDs(&tx.state, id)); count > 0 {
		return fmt.Errorf("facility %q has %d housing units; remove them before delete", id, count)
	}
	for _, housing := range tx.state.housing {
		if housing.FacilityID == id {
			return fmt.Errorf("facility %q still referenced by housing unit %q", id, housing.ID)
		}
	}
	for _, sample := range tx.state.samples {
		if sample.FacilityID == id {
			return fmt.Errorf("facility %q still referenced by sample %q", id, sample.ID)
		}
	}
	for _, project := range tx.state.projects {
		if containsString(project.FacilityIDs, id) {
			return fmt.Errorf("facility %q still referenced by project %q", id, project.ID)
		}
	}
	for _, permit := range tx.state.permits {
		if containsString(permit.FacilityIDs, id) {
			return fmt.Errorf("facility %q still referenced by permit %q", id, permit.ID)
		}
	}
	for _, item := range tx.state.supplies {
		if containsString(item.FacilityIDs, id) {
			return fmt.Errorf("facility %q still referenced by supply item %q", id, item.ID)
		}
	}
	delete(tx.state.facilities, id)
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionDelete, Before: cloneFacility(decoratedCurrent)})
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
	p.TreatmentIDs = nil
	p.ObservationIDs = nil
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.procedures[p.ID] = cloneProcedure(p)
	created := decorateProcedure(&tx.state, p)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionCreate, After: cloneProcedure(created)})
	return cloneProcedure(created), nil
}
func (tx *transaction) UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error) {
	current, ok := tx.state.procedures[id]
	if !ok {
		return Procedure{}, fmt.Errorf("procedure %q not found", id)
	}
	beforeDecorated := decorateProcedure(&tx.state, current)
	before := cloneProcedure(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Procedure{}, err
	}
	current.TreatmentIDs = nil
	current.ObservationIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.procedures[id] = cloneProcedure(current)
	afterDecorated := decorateProcedure(&tx.state, current)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionUpdate, Before: before, After: cloneProcedure(afterDecorated)})
	return cloneProcedure(afterDecorated), nil
}
func (tx *transaction) DeleteProcedure(id string) error {
	current, ok := tx.state.procedures[id]
	if !ok {
		return fmt.Errorf("procedure %q not found", id)
	}
	decoratedCurrent := decorateProcedure(&tx.state, current)
	for _, treatment := range tx.state.treatments {
		if treatment.ProcedureID == id {
			return fmt.Errorf("procedure %q still referenced by treatment %q", id, treatment.ID)
		}
	}
	for _, observation := range tx.state.observations {
		if observation.ProcedureID != nil && *observation.ProcedureID == id {
			return fmt.Errorf("procedure %q still referenced by observation %q", id, observation.ID)
		}
	}
	delete(tx.state.procedures, id)
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionDelete, Before: cloneProcedure(decoratedCurrent)})
	return nil
}
func (tx *transaction) CreateTreatment(t Treatment) (Treatment, error) {
	if t.ID == "" {
		t.ID = tx.store.newID()
	}
	if _, exists := tx.state.treatments[t.ID]; exists {
		return Treatment{}, fmt.Errorf("treatment %q already exists", t.ID)
	}
	if t.ProcedureID == "" {
		return Treatment{}, errors.New("treatment requires procedure id")
	}
	if _, ok := tx.state.procedures[t.ProcedureID]; !ok {
		return Treatment{}, fmt.Errorf("procedure %q not found", t.ProcedureID)
	}
	t.OrganismIDs = dedupeStrings(t.OrganismIDs)
	for _, organismID := range t.OrganismIDs {
		if _, ok := tx.state.organisms[organismID]; !ok {
			return Treatment{}, fmt.Errorf("organism %q not found for treatment", organismID)
		}
	}
	t.CohortIDs = dedupeStrings(t.CohortIDs)
	for _, cohortID := range t.CohortIDs {
		if _, ok := tx.state.cohorts[cohortID]; !ok {
			return Treatment{}, fmt.Errorf("cohort %q not found for treatment", cohortID)
		}
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
	if current.ProcedureID == "" {
		return Treatment{}, errors.New("treatment requires procedure id")
	}
	if _, ok := tx.state.procedures[current.ProcedureID]; !ok {
		return Treatment{}, fmt.Errorf("procedure %q not found", current.ProcedureID)
	}
	current.OrganismIDs = dedupeStrings(current.OrganismIDs)
	for _, organismID := range current.OrganismIDs {
		if _, ok := tx.state.organisms[organismID]; !ok {
			return Treatment{}, fmt.Errorf("organism %q not found for treatment", organismID)
		}
	}
	current.CohortIDs = dedupeStrings(current.CohortIDs)
	for _, cohortID := range current.CohortIDs {
		if _, ok := tx.state.cohorts[cohortID]; !ok {
			return Treatment{}, fmt.Errorf("cohort %q not found for treatment", cohortID)
		}
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
	if o.ProcedureID == nil && o.OrganismID == nil && o.CohortID == nil {
		return Observation{}, errors.New("observation requires procedure, organism, or cohort reference")
	}
	if o.ProcedureID != nil {
		if _, ok := tx.state.procedures[*o.ProcedureID]; !ok {
			return Observation{}, fmt.Errorf("procedure %q not found for observation", *o.ProcedureID)
		}
	}
	if o.OrganismID != nil {
		if _, ok := tx.state.organisms[*o.OrganismID]; !ok {
			return Observation{}, fmt.Errorf("organism %q not found for observation", *o.OrganismID)
		}
	}
	if o.CohortID != nil {
		if _, ok := tx.state.cohorts[*o.CohortID]; !ok {
			return Observation{}, fmt.Errorf("cohort %q not found for observation", *o.CohortID)
		}
	}
	o.CreatedAt = tx.now
	o.UpdatedAt = tx.now
	if data := o.DataMap(); data == nil {
		o.SetData(map[string]any{})
	} else {
		o.SetData(data)
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
	if current.ProcedureID == nil && current.OrganismID == nil && current.CohortID == nil {
		return Observation{}, errors.New("observation requires procedure, organism, or cohort reference")
	}
	if current.ProcedureID != nil {
		if _, ok := tx.state.procedures[*current.ProcedureID]; !ok {
			return Observation{}, fmt.Errorf("procedure %q not found for observation", *current.ProcedureID)
		}
	}
	if current.OrganismID != nil {
		if _, ok := tx.state.organisms[*current.OrganismID]; !ok {
			return Observation{}, fmt.Errorf("organism %q not found for observation", *current.OrganismID)
		}
	}
	if current.CohortID != nil {
		if _, ok := tx.state.cohorts[*current.CohortID]; !ok {
			return Observation{}, fmt.Errorf("cohort %q not found for observation", *current.CohortID)
		}
	}
	if data := current.DataMap(); data == nil {
		current.SetData(map[string]any{})
	} else {
		current.SetData(data)
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
	if s.FacilityID == "" {
		return Sample{}, errors.New("sample requires facility id")
	}
	if _, ok := tx.state.facilities[s.FacilityID]; !ok {
		return Sample{}, fmt.Errorf("facility %q not found for sample", s.FacilityID)
	}
	if s.OrganismID == nil && s.CohortID == nil {
		return Sample{}, errors.New("sample requires organism or cohort reference")
	}
	if s.OrganismID != nil {
		if _, ok := tx.state.organisms[*s.OrganismID]; !ok {
			return Sample{}, fmt.Errorf("organism %q not found for sample", *s.OrganismID)
		}
	}
	if s.CohortID != nil {
		if _, ok := tx.state.cohorts[*s.CohortID]; !ok {
			return Sample{}, fmt.Errorf("cohort %q not found for sample", *s.CohortID)
		}
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if attrs := s.AttributesMap(); attrs == nil {
		s.SetAttributes(map[string]any{})
	} else {
		s.SetAttributes(attrs)
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
	if current.FacilityID == "" {
		return Sample{}, errors.New("sample requires facility id")
	}
	if _, ok := tx.state.facilities[current.FacilityID]; !ok {
		return Sample{}, fmt.Errorf("facility %q not found for sample", current.FacilityID)
	}
	if current.OrganismID == nil && current.CohortID == nil {
		return Sample{}, errors.New("sample requires organism or cohort reference")
	}
	if current.OrganismID != nil {
		if _, ok := tx.state.organisms[*current.OrganismID]; !ok {
			return Sample{}, fmt.Errorf("organism %q not found for sample", *current.OrganismID)
		}
	}
	if current.CohortID != nil {
		if _, ok := tx.state.cohorts[*current.CohortID]; !ok {
			return Sample{}, fmt.Errorf("cohort %q not found for sample", *current.CohortID)
		}
	}
	if attrs := current.AttributesMap(); attrs == nil {
		current.SetAttributes(map[string]any{})
	} else {
		current.SetAttributes(attrs)
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
	for _, permit := range tx.state.permits {
		if containsString(permit.ProtocolIDs, id) {
			return fmt.Errorf("protocol %q still referenced by permit %q", id, permit.ID)
		}
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
	p.FacilityIDs = dedupeStrings(p.FacilityIDs)
	for _, facilityID := range p.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Permit{}, fmt.Errorf("facility %q not found for permit", facilityID)
		}
	}
	p.ProtocolIDs = dedupeStrings(p.ProtocolIDs)
	for _, protocolID := range p.ProtocolIDs {
		if _, ok := tx.state.protocols[protocolID]; !ok {
			return Permit{}, fmt.Errorf("protocol %q not found for permit", protocolID)
		}
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
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Permit{}, fmt.Errorf("facility %q not found for permit", facilityID)
		}
	}
	current.ProtocolIDs = dedupeStrings(current.ProtocolIDs)
	for _, protocolID := range current.ProtocolIDs {
		if _, ok := tx.state.protocols[protocolID]; !ok {
			return Permit{}, fmt.Errorf("protocol %q not found for permit", protocolID)
		}
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
	p.FacilityIDs = dedupeStrings(p.FacilityIDs)
	for _, facilityID := range p.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Project{}, fmt.Errorf("facility %q not found for project", facilityID)
		}
	}
	p.OrganismIDs = nil
	p.ProcedureIDs = nil
	p.SupplyItemIDs = nil
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.projects[p.ID] = cloneProject(p)
	created := decorateProject(&tx.state, p)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionCreate, After: cloneProject(created)})
	return cloneProject(created), nil
}
func (tx *transaction) UpdateProject(id string, mutator func(*Project) error) (Project, error) {
	current, ok := tx.state.projects[id]
	if !ok {
		return Project{}, fmt.Errorf("project %q not found", id)
	}
	beforeDecorated := decorateProject(&tx.state, current)
	before := cloneProject(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Project{}, err
	}
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Project{}, fmt.Errorf("facility %q not found for project", facilityID)
		}
	}
	current.OrganismIDs = nil
	current.ProcedureIDs = nil
	current.SupplyItemIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.projects[id] = cloneProject(current)
	afterDecorated := decorateProject(&tx.state, current)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionUpdate, Before: before, After: cloneProject(afterDecorated)})
	return cloneProject(afterDecorated), nil
}
func (tx *transaction) DeleteProject(id string) error {
	current, ok := tx.state.projects[id]
	if !ok {
		return fmt.Errorf("project %q not found", id)
	}
	decoratedCurrent := decorateProject(&tx.state, current)
	for _, supply := range tx.state.supplies {
		if containsString(supply.ProjectIDs, id) {
			return fmt.Errorf("project %q still referenced by supply item %q", id, supply.ID)
		}
	}
	delete(tx.state.projects, id)
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionDelete, Before: cloneProject(decoratedCurrent)})
	return nil
}
func (tx *transaction) CreateSupplyItem(s SupplyItem) (SupplyItem, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.supplies[s.ID]; exists {
		return SupplyItem{}, fmt.Errorf("supply item %q already exists", s.ID)
	}
	s.FacilityIDs = dedupeStrings(s.FacilityIDs)
	for _, facilityID := range s.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return SupplyItem{}, fmt.Errorf("facility %q not found for supply item", facilityID)
		}
	}
	s.ProjectIDs = dedupeStrings(s.ProjectIDs)
	for _, projectID := range s.ProjectIDs {
		if _, ok := tx.state.projects[projectID]; !ok {
			return SupplyItem{}, fmt.Errorf("project %q not found for supply item", projectID)
		}
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if attrs := s.AttributesMap(); attrs == nil {
		s.SetAttributes(map[string]any{})
	} else {
		s.SetAttributes(attrs)
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
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return SupplyItem{}, fmt.Errorf("facility %q not found for supply item", facilityID)
		}
	}
	current.ProjectIDs = dedupeStrings(current.ProjectIDs)
	for _, projectID := range current.ProjectIDs {
		if _, ok := tx.state.projects[projectID]; !ok {
			return SupplyItem{}, fmt.Errorf("project %q not found for supply item", projectID)
		}
	}
	if attrs := current.AttributesMap(); attrs == nil {
		current.SetAttributes(map[string]any{})
	} else {
		current.SetAttributes(attrs)
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
	decorated := decorateFacility(&s.state, f)
	return cloneFacility(decorated), true
}
func (s *memStore) ListFacilities() []Facility {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Facility, 0, len(s.state.facilities))
	for _, f := range s.state.facilities {
		out = append(out, cloneFacility(decorateFacility(&s.state, f)))
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
		out = append(out, cloneProject(decorateProject(&s.state, p)))
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
		out = append(out, cloneProcedure(decorateProcedure(&s.state, p)))
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
