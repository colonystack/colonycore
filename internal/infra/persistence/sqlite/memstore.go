// Package sqlite provides an in-memory transactional store plus supporting
// helpers that the SQLite persistent store builds upon. It lives under infra
// to keep domain dependencies one-way (domain -> nothing).
package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
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
	// Line is an alias of domain.Line.
	Line = domain.Line
	// Strain is an alias of domain.Strain.
	Strain = domain.Strain
	// GenotypeMarker is an alias of domain.GenotypeMarker.
	GenotypeMarker = domain.GenotypeMarker
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

func mustApply(label string, err error) {
	if err != nil {
		panic(fmt.Errorf("sqlite memstore %s: %w", label, err))
	}
}

var (
	defaultHousingState       = domain.HousingStateQuarantine
	defaultHousingEnvironment = domain.HousingEnvironmentTerrestrial
	validHousingStates        = map[domain.HousingState]struct{}{
		domain.HousingStateQuarantine:     {},
		domain.HousingStateActive:         {},
		domain.HousingStateCleaning:       {},
		domain.HousingStateDecommissioned: {},
	}
	validHousingEnvironments = map[domain.HousingEnvironment]struct{}{
		domain.HousingEnvironmentAquatic:     {},
		domain.HousingEnvironmentTerrestrial: {},
		domain.HousingEnvironmentArboreal:    {},
		domain.HousingEnvironmentHumid:       {},
	}
	defaultProtocolStatus = domain.ProtocolStatusDraft
	validProtocolStatuses = map[domain.ProtocolStatus]struct{}{
		domain.ProtocolStatusDraft:     {},
		domain.ProtocolStatusSubmitted: {},
		domain.ProtocolStatusApproved:  {},
		domain.ProtocolStatusOnHold:    {},
		domain.ProtocolStatusExpired:   {},
		domain.ProtocolStatusArchived:  {},
	}
	defaultPermitStatus = domain.PermitStatusDraft
	validPermitStatuses = map[domain.PermitStatus]struct{}{
		domain.PermitStatusDraft:     {},
		domain.PermitStatusSubmitted: {},
		domain.PermitStatusApproved:  {},
		domain.PermitStatusOnHold:    {},
		domain.PermitStatusExpired:   {},
		domain.PermitStatusArchived:  {},
	}
	defaultProcedureStatus = domain.ProcedureStatusScheduled
	validProcedureStatuses = map[domain.ProcedureStatus]struct{}{
		domain.ProcedureStatusScheduled:  {},
		domain.ProcedureStatusInProgress: {},
		domain.ProcedureStatusCompleted:  {},
		domain.ProcedureStatusCancelled:  {},
		domain.ProcedureStatusFailed:     {},
	}
	defaultTreatmentStatus = domain.TreatmentStatusPlanned
	validTreatmentStatuses = map[domain.TreatmentStatus]struct{}{
		domain.TreatmentStatusPlanned:    {},
		domain.TreatmentStatusInProgress: {},
		domain.TreatmentStatusCompleted:  {},
		domain.TreatmentStatusFlagged:    {},
	}
	defaultSampleStatus = domain.SampleStatusStored
	validSampleStatuses = map[domain.SampleStatus]struct{}{
		domain.SampleStatusStored:    {},
		domain.SampleStatusInTransit: {},
		domain.SampleStatusConsumed:  {},
		domain.SampleStatusDisposed:  {},
	}
)

func normalizeHousingUnit(h *HousingUnit) error {
	if h.State == "" {
		h.State = defaultHousingState
	}
	if _, ok := validHousingStates[h.State]; !ok {
		return fmt.Errorf("unsupported housing state %q", h.State)
	}
	if h.Environment == "" {
		h.Environment = defaultHousingEnvironment
	}
	if _, ok := validHousingEnvironments[h.Environment]; !ok {
		return fmt.Errorf("unsupported housing environment %q", h.Environment)
	}
	return nil
}

func normalizeProtocol(p *Protocol) error {
	if p.Status == "" {
		p.Status = defaultProtocolStatus
	}
	if _, ok := validProtocolStatuses[p.Status]; !ok {
		return fmt.Errorf("unsupported protocol status %q", p.Status)
	}
	return nil
}

func normalizePermit(p *Permit) error {
	if p.Status == "" {
		p.Status = defaultPermitStatus
	}
	if _, ok := validPermitStatuses[p.Status]; !ok {
		return fmt.Errorf("unsupported permit status %q", p.Status)
	}
	return nil
}

func normalizeProcedure(p *Procedure) error {
	if p.Status == "" {
		p.Status = defaultProcedureStatus
	}
	if _, ok := validProcedureStatuses[p.Status]; !ok {
		return fmt.Errorf("unsupported procedure status %q", p.Status)
	}
	return nil
}

func normalizeTreatment(t *Treatment) error {
	if t.Status == "" {
		t.Status = defaultTreatmentStatus
	}
	if _, ok := validTreatmentStatuses[t.Status]; !ok {
		return fmt.Errorf("unsupported treatment status %q", t.Status)
	}
	return nil
}

func normalizeSample(s *Sample) error {
	if s.Status == "" {
		s.Status = defaultSampleStatus
	}
	if _, ok := validSampleStatuses[s.Status]; !ok {
		return fmt.Errorf("unsupported sample status %q", s.Status)
	}
	return nil
}

// Infra implementations use domain types directly via their interfaces
// No constant aliases needed - use domain.EntityType, domain.Action values directly

type memoryState struct {
	organisms    map[string]Organism
	cohorts      map[string]Cohort
	housing      map[string]HousingUnit
	facilities   map[string]Facility
	breeding     map[string]BreedingUnit
	lines        map[string]Line
	strains      map[string]Strain
	markers      map[string]GenotypeMarker
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
	Organisms    map[string]Organism       `json:"organisms"`
	Cohorts      map[string]Cohort         `json:"cohorts"`
	Housing      map[string]HousingUnit    `json:"housing"`
	Facilities   map[string]Facility       `json:"facilities"`
	Breeding     map[string]BreedingUnit   `json:"breeding"`
	Lines        map[string]Line           `json:"lines"`
	Strains      map[string]Strain         `json:"strains"`
	Markers      map[string]GenotypeMarker `json:"markers"`
	Procedures   map[string]Procedure      `json:"procedures"`
	Treatments   map[string]Treatment      `json:"treatments"`
	Observations map[string]Observation    `json:"observations"`
	Samples      map[string]Sample         `json:"samples"`
	Protocols    map[string]Protocol       `json:"protocols"`
	Permits      map[string]Permit         `json:"permits"`
	Projects     map[string]Project        `json:"projects"`
	Supplies     map[string]SupplyItem     `json:"supplies"`
}

func newMemoryState() memoryState {
	return memoryState{
		organisms:    map[string]Organism{},
		cohorts:      map[string]Cohort{},
		housing:      map[string]HousingUnit{},
		facilities:   map[string]Facility{},
		breeding:     map[string]BreedingUnit{},
		lines:        map[string]Line{},
		strains:      map[string]Strain{},
		markers:      map[string]GenotypeMarker{},
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
		Lines:        make(map[string]Line, len(state.lines)),
		Strains:      make(map[string]Strain, len(state.strains)),
		Markers:      make(map[string]GenotypeMarker, len(state.markers)),
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
	for k, v := range state.lines {
		s.Lines[k] = cloneLine(v)
	}
	for k, v := range state.strains {
		s.Strains[k] = cloneStrain(v)
	}
	for k, v := range state.markers {
		s.Markers[k] = cloneGenotypeMarker(v)
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
	for k, v := range s.Lines {
		st.lines[k] = cloneLine(v)
	}
	for k, v := range s.Strains {
		st.strains[k] = cloneStrain(v)
	}
	for k, v := range s.Markers {
		st.markers[k] = cloneGenotypeMarker(v)
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
	if snapshot.Lines == nil {
		snapshot.Lines = map[string]Line{}
	}
	if snapshot.Strains == nil {
		snapshot.Strains = map[string]Strain{}
	}
	if snapshot.Markers == nil {
		snapshot.Markers = map[string]GenotypeMarker{}
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
	markerExists := func(id string) bool {
		_, ok := snapshot.Markers[id]
		return ok
	}
	lineExists := func(id string) bool {
		_, ok := snapshot.Lines[id]
		return ok
	}
	strainExists := func(id string) bool {
		_, ok := snapshot.Strains[id]
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

	for id, organism := range snapshot.Organisms {
		if attrs := organism.CoreAttributes(); attrs == nil {
			mustApply("apply organism attributes", organism.SetCoreAttributes(map[string]any{}))
		} else {
			mustApply("apply organism attributes", organism.SetCoreAttributes(attrs))
		}
		if organism.LineID != nil && !lineExists(*organism.LineID) {
			organism.LineID = nil
		}
		if organism.StrainID != nil && !strainExists(*organism.StrainID) {
			organism.StrainID = nil
		}
		snapshot.Organisms[id] = organism
	}

	for id, breeding := range snapshot.Breeding {
		if attrs := breeding.PairingAttributes(); attrs == nil {
			mustApply("apply breeding attributes", breeding.ApplyPairingAttributes(map[string]any{}))
		} else {
			mustApply("apply breeding attributes", breeding.ApplyPairingAttributes(attrs))
		}
		if breeding.LineID != nil && !lineExists(*breeding.LineID) {
			breeding.LineID = nil
		}
		if breeding.StrainID != nil && !strainExists(*breeding.StrainID) {
			breeding.StrainID = nil
		}
		if breeding.TargetLineID != nil && !lineExists(*breeding.TargetLineID) {
			breeding.TargetLineID = nil
		}
		if breeding.TargetStrainID != nil && !strainExists(*breeding.TargetStrainID) {
			breeding.TargetStrainID = nil
		}
		snapshot.Breeding[id] = breeding
	}

	for id, marker := range snapshot.Markers {
		if attrs := marker.GenotypeMarkerAttributesByPlugin(); attrs == nil {
			mustApply("apply genotype marker attributes", marker.ApplyGenotypeMarkerAttributes(map[string]any{}))
		} else {
			mustApply("apply genotype marker attributes", marker.ApplyGenotypeMarkerAttributes(attrs))
		}
		if len(marker.Alleles) > 0 {
			marker.Alleles = dedupeStrings(marker.Alleles)
		}
		snapshot.Markers[id] = marker
	}

	for id, line := range snapshot.Lines {
		if attrs := line.DefaultAttributes(); attrs == nil {
			mustApply("apply line default attributes", line.ApplyDefaultAttributes(map[string]any{}))
		} else {
			mustApply("apply line default attributes", line.ApplyDefaultAttributes(attrs))
		}
		if overrides := line.ExtensionOverrides(); overrides == nil {
			mustApply("apply line extension overrides", line.ApplyExtensionOverrides(map[string]any{}))
		} else {
			mustApply("apply line extension overrides", line.ApplyExtensionOverrides(overrides))
		}
		if filtered, changed := filterIDs(line.GenotypeMarkerIDs, markerExists); changed {
			line.GenotypeMarkerIDs = filtered
		}
		snapshot.Lines[id] = line
	}

	for id, strain := range snapshot.Strains {
		if !lineExists(strain.LineID) {
			delete(snapshot.Strains, id)
			continue
		}
		if attrs := strain.StrainAttributesByPlugin(); attrs == nil {
			mustApply("apply strain attributes", strain.ApplyStrainAttributes(map[string]any{}))
		} else {
			mustApply("apply strain attributes", strain.ApplyStrainAttributes(attrs))
		}
		if filtered, changed := filterIDs(strain.GenotypeMarkerIDs, markerExists); changed {
			strain.GenotypeMarkerIDs = filtered
		}
		snapshot.Strains[id] = strain
	}

	for id, organism := range snapshot.Organisms {
		if organism.LineID != nil && !lineExists(*organism.LineID) {
			organism.LineID = nil
		}
		if organism.StrainID != nil && !strainExists(*organism.StrainID) {
			organism.StrainID = nil
		}
		snapshot.Organisms[id] = organism
	}

	for id, protocol := range snapshot.Protocols {
		if err := normalizeProtocol(&protocol); err != nil {
			delete(snapshot.Protocols, id)
			continue
		}
		snapshot.Protocols[id] = protocol
	}

	for id, housing := range snapshot.Housing {
		if housing.FacilityID == "" || !facilityExists(housing.FacilityID) {
			delete(snapshot.Housing, id)
			continue
		}
		if housing.Capacity <= 0 {
			housing.Capacity = 1
		}
		if err := normalizeHousingUnit(&housing); err != nil {
			delete(snapshot.Housing, id)
			continue
		}
		snapshot.Housing[id] = housing
	}

	for id, treatment := range snapshot.Treatments {
		if treatment.ProcedureID == "" || !procedureExists(treatment.ProcedureID) {
			delete(snapshot.Treatments, id)
			continue
		}
		if err := normalizeTreatment(&treatment); err != nil {
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
		if data := observation.ObservationData(); data == nil {
			mustApply("apply observation data", observation.ApplyObservationData(map[string]any{}))
		} else {
			mustApply("apply observation data", observation.ApplyObservationData(data))
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
		if attrs := sample.SampleAttributes(); attrs == nil {
			mustApply("apply sample attributes", sample.ApplySampleAttributes(map[string]any{}))
		} else {
			mustApply("apply sample attributes", sample.ApplySampleAttributes(attrs))
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
		if err := normalizeSample(&sample); err != nil {
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
		if err := normalizePermit(&permit); err != nil {
			delete(snapshot.Permits, id)
			continue
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
		if err := normalizeProcedure(&procedure); err != nil {
			delete(snapshot.Procedures, id)
			continue
		}
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
		if attrs := item.SupplyAttributes(); attrs == nil {
			mustApply("apply supply attributes", item.ApplySupplyAttributes(map[string]any{}))
		} else {
			mustApply("apply supply attributes", item.ApplySupplyAttributes(attrs))
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
		if baselines := facility.EnvironmentBaselines(); baselines == nil {
			mustApply("apply facility baselines", facility.ApplyEnvironmentBaselines(map[string]any{}))
		} else {
			mustApply("apply facility baselines", facility.ApplyEnvironmentBaselines(baselines))
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
	container, err := o.OrganismExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone organism extensions: %w", err))
	}
	if err := cp.SetOrganismExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set organism extensions: %w", err))
	}
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
	container, err := b.BreedingUnitExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone breeding attributes: %w", err))
	}
	if err := cp.SetBreedingUnitExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set breeding attributes: %w", err))
	}
	return cp
}

func cloneLine(l Line) Line {
	cp := l
	if l.Description != nil {
		desc := *l.Description
		cp.Description = &desc
	}
	if l.DeprecatedAt != nil {
		t := *l.DeprecatedAt
		cp.DeprecatedAt = &t
	}
	if l.DeprecationReason != nil {
		reason := *l.DeprecationReason
		cp.DeprecationReason = &reason
	}
	cp.GenotypeMarkerIDs = append([]string(nil), l.GenotypeMarkerIDs...)
	container, err := l.LineExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone line extensions: %w", err))
	}
	if err := cp.SetLineExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set line extensions: %w", err))
	}
	return cp
}

func cloneStrain(s Strain) Strain {
	cp := s
	if s.Description != nil {
		desc := *s.Description
		cp.Description = &desc
	}
	if s.Generation != nil {
		gen := *s.Generation
		cp.Generation = &gen
	}
	if s.RetiredAt != nil {
		t := *s.RetiredAt
		cp.RetiredAt = &t
	}
	if s.RetirementReason != nil {
		reason := *s.RetirementReason
		cp.RetirementReason = &reason
	}
	cp.GenotypeMarkerIDs = append([]string(nil), s.GenotypeMarkerIDs...)
	container, err := s.StrainExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone strain extensions: %w", err))
	}
	if err := cp.SetStrainExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set strain extensions: %w", err))
	}
	return cp
}

func cloneGenotypeMarker(g GenotypeMarker) GenotypeMarker {
	cp := g
	cp.Alleles = append([]string(nil), g.Alleles...)
	container, err := g.GenotypeMarkerExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone genotype marker extensions: %w", err))
	}
	if err := cp.SetGenotypeMarkerExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set genotype marker extensions: %w", err))
	}
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
	container, err := f.FacilityExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone facility baselines: %w", err))
	}
	if err := cp.SetFacilityExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set facility baselines: %w", err))
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
	container, err := o.ObservationExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone observation data: %w", err))
	}
	if err := cp.SetObservationExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set observation data: %w", err))
	}
	return cp
}

func cloneSample(s Sample) Sample {
	cp := s
	cp.ChainOfCustody = append([]domain.SampleCustodyEvent(nil), s.ChainOfCustody...)
	container, err := s.SampleExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone sample attributes: %w", err))
	}
	if err := cp.SetSampleExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set sample attributes: %w", err))
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

func requireNonEmpty(field string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s requires at least one value", field)
	}
	return nil
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
	container, err := s.SupplyItemExtensions()
	if err != nil {
		panic(fmt.Errorf("sqlite: clone supply attributes: %w", err))
	}
	if err := cp.SetSupplyItemExtensions(container); err != nil {
		panic(fmt.Errorf("sqlite: set supply attributes: %w", err))
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
func (v transactionView) ListLines() []Line {
	out := make([]Line, 0, len(v.state.lines))
	for _, line := range v.state.lines {
		out = append(out, cloneLine(line))
	}
	return out
}
func (v transactionView) ListStrains() []Strain {
	out := make([]Strain, 0, len(v.state.strains))
	for _, strain := range v.state.strains {
		out = append(out, cloneStrain(strain))
	}
	return out
}
func (v transactionView) ListGenotypeMarkers() []GenotypeMarker {
	out := make([]GenotypeMarker, 0, len(v.state.markers))
	for _, marker := range v.state.markers {
		out = append(out, cloneGenotypeMarker(marker))
	}
	return out
}
func (v transactionView) FindOrganism(id string) (Organism, bool) {
	o, ok := v.state.organisms[id]
	if !ok {
		return Organism{Organism: entitymodel.Organism{}}, false
	}
	return cloneOrganism(o), true
}
func (v transactionView) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := v.state.housing[id]
	if !ok {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, false
	}
	return cloneHousing(h), true
}
func (v transactionView) FindFacility(id string) (Facility, bool) {
	f, ok := v.state.facilities[id]
	if !ok {
		return Facility{Facility: entitymodel.Facility{}}, false
	}
	return cloneFacility(decorateFacility(v.state, f)), true
}
func (v transactionView) FindLine(id string) (Line, bool) {
	line, ok := v.state.lines[id]
	if !ok {
		return Line{Line: entitymodel.Line{}}, false
	}
	return cloneLine(line), true
}
func (v transactionView) FindStrain(id string) (Strain, bool) {
	strain, ok := v.state.strains[id]
	if !ok {
		return Strain{Strain: entitymodel.Strain{}}, false
	}
	return cloneStrain(strain), true
}
func (v transactionView) FindGenotypeMarker(id string) (GenotypeMarker, bool) {
	marker, ok := v.state.markers[id]
	if !ok {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, false
	}
	return cloneGenotypeMarker(marker), true
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
		return Treatment{Treatment: entitymodel.Treatment{}}, false
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
		return Observation{Observation: entitymodel.Observation{}}, false
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
		return Sample{Sample: entitymodel.Sample{}}, false
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
		return Permit{Permit: entitymodel.Permit{}}, false
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
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, false
	}
	return cloneSupplyItem(s), true
}

func (v transactionView) FindProcedure(id string) (Procedure, bool) {
	p, ok := v.state.procedures[id]
	if !ok {
		return Procedure{Procedure: entitymodel.Procedure{}}, false
	}
	return cloneProcedure(p), true
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

// changePayloadFromValue encodes value into a domain.ChangePayload.
// On success it returns the encoded payload. If encoding fails it returns
// domain.UndefinedChangePayload() and an error that wraps the underlying encoding error.
func changePayloadFromValue[T any](value T) (domain.ChangePayload, error) {
	payload, err := domain.NewChangePayloadFromValue(value)
	if err != nil {
		return domain.UndefinedChangePayload(), fmt.Errorf("encode change payload: %w", err)
	}
	return payload, nil
}
func (tx *transaction) Snapshot() TransactionView { return newTransactionView(&tx.state) }
func (tx *transaction) FindHousingUnit(id string) (HousingUnit, bool) {
	h, ok := tx.state.housing[id]
	if !ok {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, false
	}
	return cloneHousing(h), true
}
func (tx *transaction) FindProtocol(id string) (Protocol, bool) {
	p, ok := tx.state.protocols[id]
	if !ok {
		return Protocol{Protocol: entitymodel.Protocol{}}, false
	}
	return cloneProtocol(p), true
}
func (tx *transaction) FindFacility(id string) (Facility, bool) {
	f, ok := tx.state.facilities[id]
	if !ok {
		return Facility{Facility: entitymodel.Facility{}}, false
	}
	return cloneFacility(decorateFacility(&tx.state, f)), true
}

func (tx *transaction) FindLine(id string) (Line, bool) {
	line, ok := tx.state.lines[id]
	if !ok {
		return Line{Line: entitymodel.Line{}}, false
	}
	return cloneLine(line), true
}

func (tx *transaction) FindStrain(id string) (Strain, bool) {
	strain, ok := tx.state.strains[id]
	if !ok {
		return Strain{Strain: entitymodel.Strain{}}, false
	}
	return cloneStrain(strain), true
}

func (tx *transaction) FindGenotypeMarker(id string) (GenotypeMarker, bool) {
	marker, ok := tx.state.markers[id]
	if !ok {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, false
	}
	return cloneGenotypeMarker(marker), true
}

func (tx *transaction) FindTreatment(id string) (Treatment, bool) {
	t, ok := tx.state.treatments[id]
	if !ok {
		return Treatment{Treatment: entitymodel.Treatment{}}, false
	}
	return cloneTreatment(t), true
}
func (tx *transaction) FindObservation(id string) (Observation, bool) {
	o, ok := tx.state.observations[id]
	if !ok {
		return Observation{Observation: entitymodel.Observation{}}, false
	}
	return cloneObservation(o), true
}
func (tx *transaction) FindSample(id string) (Sample, bool) {
	s, ok := tx.state.samples[id]
	if !ok {
		return Sample{Sample: entitymodel.Sample{}}, false
	}
	return cloneSample(s), true
}
func (tx *transaction) FindPermit(id string) (Permit, bool) {
	p, ok := tx.state.permits[id]
	if !ok {
		return Permit{Permit: entitymodel.Permit{}}, false
	}
	return clonePermit(p), true
}
func (tx *transaction) FindSupplyItem(id string) (SupplyItem, bool) {
	s, ok := tx.state.supplies[id]
	if !ok {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, false
	}
	return cloneSupplyItem(s), true
}

func (tx *transaction) FindProcedure(id string) (Procedure, bool) {
	p, ok := tx.state.procedures[id]
	if !ok {
		return Procedure{Procedure: entitymodel.Procedure{}}, false
	}
	return cloneProcedure(p), true
}
func (tx *transaction) CreateOrganism(o Organism) (Organism, error) {
	if o.ID == "" {
		o.ID = tx.store.newID()
	}
	if o.Stage == "" {
		o.Stage = domain.StagePlanned
	}
	if _, exists := tx.state.organisms[o.ID]; exists {
		return Organism{Organism: entitymodel.Organism{}}, fmt.Errorf("organism %q already exists", o.ID)
	}
	o.CreatedAt = tx.now
	o.UpdatedAt = tx.now
	if attrs := o.CoreAttributes(); attrs == nil {
		mustApply("apply organism attributes", o.SetCoreAttributes(map[string]any{}))
	} else {
		mustApply("apply organism attributes", o.SetCoreAttributes(attrs))
	}
	tx.state.organisms[o.ID] = cloneOrganism(o)
	after, err := changePayloadFromValue(cloneOrganism(o))
	if err != nil {
		return Organism{Organism: entitymodel.Organism{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionCreate, After: after})
	return cloneOrganism(o), nil
}
func (tx *transaction) UpdateOrganism(id string, mutator func(*Organism) error) (Organism, error) {
	current, ok := tx.state.organisms[id]
	if !ok {
		return Organism{Organism: entitymodel.Organism{}}, fmt.Errorf("organism %q not found", id)
	}
	before := cloneOrganism(current)
	if err := mutator(&current); err != nil {
		return Organism{Organism: entitymodel.Organism{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.organisms[id] = cloneOrganism(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Organism{Organism: entitymodel.Organism{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneOrganism(current))
	if err != nil {
		return Organism{Organism: entitymodel.Organism{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneOrganism(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityOrganism, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateCohort(c Cohort) (Cohort, error) {
	if c.ID == "" {
		c.ID = tx.store.newID()
	}
	if _, exists := tx.state.cohorts[c.ID]; exists {
		return Cohort{Cohort: entitymodel.Cohort{}}, fmt.Errorf("cohort %q already exists", c.ID)
	}
	c.CreatedAt = tx.now
	c.UpdatedAt = tx.now
	tx.state.cohorts[c.ID] = cloneCohort(c)
	after, err := changePayloadFromValue(cloneCohort(c))
	if err != nil {
		return Cohort{Cohort: entitymodel.Cohort{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionCreate, After: after})
	return cloneCohort(c), nil
}
func (tx *transaction) UpdateCohort(id string, mutator func(*Cohort) error) (Cohort, error) {
	current, ok := tx.state.cohorts[id]
	if !ok {
		return Cohort{Cohort: entitymodel.Cohort{}}, fmt.Errorf("cohort %q not found", id)
	}
	before := cloneCohort(current)
	if err := mutator(&current); err != nil {
		return Cohort{Cohort: entitymodel.Cohort{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.cohorts[id] = cloneCohort(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Cohort{Cohort: entitymodel.Cohort{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneCohort(current))
	if err != nil {
		return Cohort{Cohort: entitymodel.Cohort{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneCohort(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityCohort, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateHousingUnit(h HousingUnit) (HousingUnit, error) {
	if h.ID == "" {
		h.ID = tx.store.newID()
	}
	if _, exists := tx.state.housing[h.ID]; exists {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, fmt.Errorf("housing unit %q already exists", h.ID)
	}
	if h.FacilityID == "" {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, errors.New("housing unit requires facility id")
	}
	if _, ok := tx.state.facilities[h.FacilityID]; !ok {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, fmt.Errorf("facility %q not found", h.FacilityID)
	}
	if h.Capacity <= 0 {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, errors.New("housing capacity must be positive")
	}
	if err := normalizeHousingUnit(&h); err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	h.CreatedAt = tx.now
	h.UpdatedAt = tx.now
	tx.state.housing[h.ID] = cloneHousing(h)
	after, err := changePayloadFromValue(cloneHousing(h))
	if err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionCreate, After: after})
	return cloneHousing(h), nil
}
func (tx *transaction) UpdateHousingUnit(id string, mutator func(*HousingUnit) error) (HousingUnit, error) {
	current, ok := tx.state.housing[id]
	if !ok {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, fmt.Errorf("housing unit %q not found", id)
	}
	before := cloneHousing(current)
	if err := mutator(&current); err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	if current.FacilityID == "" {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, errors.New("housing unit requires facility id")
	}
	if _, ok := tx.state.facilities[current.FacilityID]; !ok {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, fmt.Errorf("facility %q not found", current.FacilityID)
	}
	if current.Capacity <= 0 {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, errors.New("housing capacity must be positive")
	}
	if err := normalizeHousingUnit(&current); err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.housing[id] = cloneHousing(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneHousing(current))
	if err != nil {
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneHousing(current), nil
}
func (tx *transaction) DeleteHousingUnit(id string) error {
	current, ok := tx.state.housing[id]
	if !ok {
		return fmt.Errorf("housing unit %q not found", id)
	}
	delete(tx.state.housing, id)
	beforePayload, err := changePayloadFromValue(cloneHousing(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityHousingUnit, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateFacility(f Facility) (Facility, error) {
	if f.ID == "" {
		f.ID = tx.store.newID()
	}
	if _, exists := tx.state.facilities[f.ID]; exists {
		return Facility{Facility: entitymodel.Facility{}}, fmt.Errorf("facility %q already exists", f.ID)
	}
	f.CreatedAt = tx.now
	f.UpdatedAt = tx.now
	f.HousingUnitIDs = nil
	f.ProjectIDs = nil
	if baselines := f.EnvironmentBaselines(); baselines == nil {
		mustApply("apply facility baselines", f.ApplyEnvironmentBaselines(map[string]any{}))
	} else {
		mustApply("apply facility baselines", f.ApplyEnvironmentBaselines(baselines))
	}
	tx.state.facilities[f.ID] = cloneFacility(f)
	created := decorateFacility(&tx.state, f)
	after, err := changePayloadFromValue(cloneFacility(created))
	if err != nil {
		return Facility{Facility: entitymodel.Facility{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionCreate, After: after})
	return cloneFacility(created), nil
}
func (tx *transaction) UpdateFacility(id string, mutator func(*Facility) error) (Facility, error) {
	current, ok := tx.state.facilities[id]
	if !ok {
		return Facility{Facility: entitymodel.Facility{}}, fmt.Errorf("facility %q not found", id)
	}
	beforeDecorated := decorateFacility(&tx.state, current)
	before := cloneFacility(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Facility{Facility: entitymodel.Facility{}}, err
	}
	if baselines := current.EnvironmentBaselines(); baselines == nil {
		mustApply("apply facility baselines", current.ApplyEnvironmentBaselines(map[string]any{}))
	} else {
		mustApply("apply facility baselines", current.ApplyEnvironmentBaselines(baselines))
	}
	current.HousingUnitIDs = nil
	current.ProjectIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.facilities[id] = cloneFacility(current)
	afterDecorated := decorateFacility(&tx.state, current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Facility{Facility: entitymodel.Facility{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneFacility(afterDecorated))
	if err != nil {
		return Facility{Facility: entitymodel.Facility{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneFacility(decoratedCurrent))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityFacility, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateBreedingUnit(b BreedingUnit) (BreedingUnit, error) {
	if b.ID == "" {
		b.ID = tx.store.newID()
	}
	if _, exists := tx.state.breeding[b.ID]; exists {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, fmt.Errorf("breeding unit %q already exists", b.ID)
	}
	b.CreatedAt = tx.now
	b.UpdatedAt = tx.now
	tx.state.breeding[b.ID] = cloneBreeding(b)
	after, err := changePayloadFromValue(cloneBreeding(b))
	if err != nil {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionCreate, After: after})
	return cloneBreeding(b), nil
}
func (tx *transaction) UpdateBreedingUnit(id string, mutator func(*BreedingUnit) error) (BreedingUnit, error) {
	current, ok := tx.state.breeding[id]
	if !ok {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, fmt.Errorf("breeding unit %q not found", id)
	}
	before := cloneBreeding(current)
	if err := mutator(&current); err != nil {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.breeding[id] = cloneBreeding(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneBreeding(current))
	if err != nil {
		return BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneBreeding(current), nil
}
func (tx *transaction) DeleteBreedingUnit(id string) error {
	current, ok := tx.state.breeding[id]
	if !ok {
		return fmt.Errorf("breeding unit %q not found", id)
	}
	delete(tx.state.breeding, id)
	beforePayload, err := changePayloadFromValue(cloneBreeding(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityBreeding, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}

func (tx *transaction) CreateLine(l Line) (Line, error) {
	if l.ID == "" {
		l.ID = tx.store.newID()
	}
	if _, exists := tx.state.lines[l.ID]; exists {
		return Line{Line: entitymodel.Line{}}, fmt.Errorf("line %q already exists", l.ID)
	}
	if filtered, changed := filterIDs(l.GenotypeMarkerIDs, func(id string) bool { _, ok := tx.state.markers[id]; return ok }); changed {
		l.GenotypeMarkerIDs = filtered
	}
	if err := requireNonEmpty("line.genotype_marker_ids", l.GenotypeMarkerIDs); err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	if attrs := l.DefaultAttributes(); attrs == nil {
		mustApply("apply line default attributes", l.ApplyDefaultAttributes(map[string]any{}))
	} else {
		mustApply("apply line default attributes", l.ApplyDefaultAttributes(attrs))
	}
	if overrides := l.ExtensionOverrides(); overrides == nil {
		mustApply("apply line extension overrides", l.ApplyExtensionOverrides(map[string]any{}))
	} else {
		mustApply("apply line extension overrides", l.ApplyExtensionOverrides(overrides))
	}
	l.CreatedAt = tx.now
	l.UpdatedAt = tx.now
	tx.state.lines[l.ID] = cloneLine(l)
	after, err := changePayloadFromValue(cloneLine(l))
	if err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityLine, Action: domain.ActionCreate, After: after})
	return cloneLine(l), nil
}

func (tx *transaction) UpdateLine(id string, mutator func(*Line) error) (Line, error) {
	current, ok := tx.state.lines[id]
	if !ok {
		return Line{Line: entitymodel.Line{}}, fmt.Errorf("line %q not found", id)
	}
	before := cloneLine(current)
	if err := mutator(&current); err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	if filtered, changed := filterIDs(current.GenotypeMarkerIDs, func(markerID string) bool { _, ok := tx.state.markers[markerID]; return ok }); changed {
		current.GenotypeMarkerIDs = filtered
	}
	if err := requireNonEmpty("line.genotype_marker_ids", current.GenotypeMarkerIDs); err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	if attrs := current.DefaultAttributes(); attrs == nil {
		mustApply("apply line default attributes", current.ApplyDefaultAttributes(map[string]any{}))
	} else {
		mustApply("apply line default attributes", current.ApplyDefaultAttributes(attrs))
	}
	if overrides := current.ExtensionOverrides(); overrides == nil {
		mustApply("apply line extension overrides", current.ApplyExtensionOverrides(map[string]any{}))
	} else {
		mustApply("apply line extension overrides", current.ApplyExtensionOverrides(overrides))
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.lines[id] = cloneLine(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneLine(current))
	if err != nil {
		return Line{Line: entitymodel.Line{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityLine, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneLine(current), nil
}

func (tx *transaction) DeleteLine(id string) error {
	current, ok := tx.state.lines[id]
	if !ok {
		return fmt.Errorf("line %q not found", id)
	}
	for _, strain := range tx.state.strains {
		if strain.LineID == id {
			return fmt.Errorf("line %q still referenced by strain %q", id, strain.ID)
		}
	}
	for _, breeding := range tx.state.breeding {
		if breeding.LineID != nil && *breeding.LineID == id {
			return fmt.Errorf("line %q still referenced by breeding unit %q", id, breeding.ID)
		}
		if breeding.TargetLineID != nil && *breeding.TargetLineID == id {
			return fmt.Errorf("line %q still referenced by breeding unit %q", id, breeding.ID)
		}
	}
	for _, organism := range tx.state.organisms {
		if organism.LineID != nil && *organism.LineID == id {
			return fmt.Errorf("line %q still referenced by organism %q", id, organism.ID)
		}
	}
	delete(tx.state.lines, id)
	beforePayload, err := changePayloadFromValue(cloneLine(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityLine, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}

func (tx *transaction) CreateStrain(s Strain) (Strain, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.strains[s.ID]; exists {
		return Strain{Strain: entitymodel.Strain{}}, fmt.Errorf("strain %q already exists", s.ID)
	}
	if s.LineID == "" {
		return Strain{Strain: entitymodel.Strain{}}, errors.New("strain requires line id")
	}
	if _, ok := tx.state.lines[s.LineID]; !ok {
		return Strain{Strain: entitymodel.Strain{}}, fmt.Errorf("line %q not found for strain", s.LineID)
	}
	if filtered, changed := filterIDs(s.GenotypeMarkerIDs, func(markerID string) bool { _, ok := tx.state.markers[markerID]; return ok }); changed {
		s.GenotypeMarkerIDs = filtered
	}
	if attrs := s.StrainAttributesByPlugin(); attrs == nil {
		mustApply("apply strain attributes", s.ApplyStrainAttributes(map[string]any{}))
	} else {
		mustApply("apply strain attributes", s.ApplyStrainAttributes(attrs))
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	tx.state.strains[s.ID] = cloneStrain(s)
	after, err := changePayloadFromValue(cloneStrain(s))
	if err != nil {
		return Strain{Strain: entitymodel.Strain{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityStrain, Action: domain.ActionCreate, After: after})
	return cloneStrain(s), nil
}

func (tx *transaction) UpdateStrain(id string, mutator func(*Strain) error) (Strain, error) {
	current, ok := tx.state.strains[id]
	if !ok {
		return Strain{Strain: entitymodel.Strain{}}, fmt.Errorf("strain %q not found", id)
	}
	before := cloneStrain(current)
	if err := mutator(&current); err != nil {
		return Strain{Strain: entitymodel.Strain{}}, err
	}
	if current.LineID == "" {
		return Strain{Strain: entitymodel.Strain{}}, errors.New("strain requires line id")
	}
	if _, ok := tx.state.lines[current.LineID]; !ok {
		return Strain{Strain: entitymodel.Strain{}}, fmt.Errorf("line %q not found for strain", current.LineID)
	}
	if filtered, changed := filterIDs(current.GenotypeMarkerIDs, func(markerID string) bool { _, ok := tx.state.markers[markerID]; return ok }); changed {
		current.GenotypeMarkerIDs = filtered
	}
	if attrs := current.StrainAttributesByPlugin(); attrs == nil {
		mustApply("apply strain attributes", current.ApplyStrainAttributes(map[string]any{}))
	} else {
		mustApply("apply strain attributes", current.ApplyStrainAttributes(attrs))
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.strains[id] = cloneStrain(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Strain{Strain: entitymodel.Strain{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneStrain(current))
	if err != nil {
		return Strain{Strain: entitymodel.Strain{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityStrain, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneStrain(current), nil
}

func (tx *transaction) DeleteStrain(id string) error {
	current, ok := tx.state.strains[id]
	if !ok {
		return fmt.Errorf("strain %q not found", id)
	}
	for _, organism := range tx.state.organisms {
		if organism.StrainID != nil && *organism.StrainID == id {
			return fmt.Errorf("strain %q still referenced by organism %q", id, organism.ID)
		}
	}
	for _, breeding := range tx.state.breeding {
		if breeding.StrainID != nil && *breeding.StrainID == id {
			return fmt.Errorf("strain %q still referenced by breeding unit %q", id, breeding.ID)
		}
		if breeding.TargetStrainID != nil && *breeding.TargetStrainID == id {
			return fmt.Errorf("strain %q still referenced by breeding unit %q", id, breeding.ID)
		}
	}
	delete(tx.state.strains, id)
	beforePayload, err := changePayloadFromValue(cloneStrain(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityStrain, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}

func (tx *transaction) CreateGenotypeMarker(g GenotypeMarker) (GenotypeMarker, error) {
	if g.ID == "" {
		g.ID = tx.store.newID()
	}
	if _, exists := tx.state.markers[g.ID]; exists {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, fmt.Errorf("genotype marker %q already exists", g.ID)
	}
	if len(g.Alleles) > 0 {
		g.Alleles = dedupeStrings(g.Alleles)
	}
	if attrs := g.GenotypeMarkerAttributesByPlugin(); attrs == nil {
		mustApply("apply genotype marker attributes", g.ApplyGenotypeMarkerAttributes(map[string]any{}))
	} else {
		mustApply("apply genotype marker attributes", g.ApplyGenotypeMarkerAttributes(attrs))
	}
	g.CreatedAt = tx.now
	g.UpdatedAt = tx.now
	tx.state.markers[g.ID] = cloneGenotypeMarker(g)
	after, err := changePayloadFromValue(cloneGenotypeMarker(g))
	if err != nil {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityGenotypeMarker, Action: domain.ActionCreate, After: after})
	return cloneGenotypeMarker(g), nil
}

func (tx *transaction) UpdateGenotypeMarker(id string, mutator func(*GenotypeMarker) error) (GenotypeMarker, error) {
	current, ok := tx.state.markers[id]
	if !ok {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, fmt.Errorf("genotype marker %q not found", id)
	}
	before := cloneGenotypeMarker(current)
	if err := mutator(&current); err != nil {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, err
	}
	if len(current.Alleles) > 0 {
		current.Alleles = dedupeStrings(current.Alleles)
	}
	if attrs := current.GenotypeMarkerAttributesByPlugin(); attrs == nil {
		mustApply("apply genotype marker attributes", current.ApplyGenotypeMarkerAttributes(map[string]any{}))
	} else {
		mustApply("apply genotype marker attributes", current.ApplyGenotypeMarkerAttributes(attrs))
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.markers[id] = cloneGenotypeMarker(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneGenotypeMarker(current))
	if err != nil {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityGenotypeMarker, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneGenotypeMarker(current), nil
}

func (tx *transaction) DeleteGenotypeMarker(id string) error {
	current, ok := tx.state.markers[id]
	if !ok {
		return fmt.Errorf("genotype marker %q not found", id)
	}
	for _, line := range tx.state.lines {
		if containsString(line.GenotypeMarkerIDs, id) {
			return fmt.Errorf("genotype marker %q still referenced by line %q", id, line.ID)
		}
	}
	for _, strain := range tx.state.strains {
		if containsString(strain.GenotypeMarkerIDs, id) {
			return fmt.Errorf("genotype marker %q still referenced by strain %q", id, strain.ID)
		}
	}
	delete(tx.state.markers, id)
	beforePayload, err := changePayloadFromValue(cloneGenotypeMarker(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityGenotypeMarker, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}

func (tx *transaction) CreateProcedure(p Procedure) (Procedure, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.procedures[p.ID]; exists {
		return Procedure{Procedure: entitymodel.Procedure{}}, fmt.Errorf("procedure %q already exists", p.ID)
	}
	if err := normalizeProcedure(&p); err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	p.TreatmentIDs = nil
	p.ObservationIDs = nil
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.procedures[p.ID] = cloneProcedure(p)
	created := decorateProcedure(&tx.state, p)
	after, err := changePayloadFromValue(cloneProcedure(created))
	if err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionCreate, After: after})
	return cloneProcedure(created), nil
}
func (tx *transaction) UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error) {
	current, ok := tx.state.procedures[id]
	if !ok {
		return Procedure{Procedure: entitymodel.Procedure{}}, fmt.Errorf("procedure %q not found", id)
	}
	beforeDecorated := decorateProcedure(&tx.state, current)
	before := cloneProcedure(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	if err := normalizeProcedure(&current); err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	current.TreatmentIDs = nil
	current.ObservationIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.procedures[id] = cloneProcedure(current)
	afterDecorated := decorateProcedure(&tx.state, current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneProcedure(afterDecorated))
	if err != nil {
		return Procedure{Procedure: entitymodel.Procedure{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneProcedure(decoratedCurrent))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityProcedure, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateTreatment(t Treatment) (Treatment, error) {
	if t.ID == "" {
		t.ID = tx.store.newID()
	}
	if _, exists := tx.state.treatments[t.ID]; exists {
		return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("treatment %q already exists", t.ID)
	}
	if t.ProcedureID == "" {
		return Treatment{Treatment: entitymodel.Treatment{}}, errors.New("treatment requires procedure id")
	}
	if _, ok := tx.state.procedures[t.ProcedureID]; !ok {
		return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("procedure %q not found", t.ProcedureID)
	}
	if err := normalizeTreatment(&t); err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	t.OrganismIDs = dedupeStrings(t.OrganismIDs)
	for _, organismID := range t.OrganismIDs {
		if _, ok := tx.state.organisms[organismID]; !ok {
			return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("organism %q not found for treatment", organismID)
		}
	}
	t.CohortIDs = dedupeStrings(t.CohortIDs)
	for _, cohortID := range t.CohortIDs {
		if _, ok := tx.state.cohorts[cohortID]; !ok {
			return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("cohort %q not found for treatment", cohortID)
		}
	}
	t.CreatedAt = tx.now
	t.UpdatedAt = tx.now
	tx.state.treatments[t.ID] = cloneTreatment(t)
	after, err := changePayloadFromValue(cloneTreatment(t))
	if err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionCreate, After: after})
	return cloneTreatment(t), nil
}
func (tx *transaction) UpdateTreatment(id string, mutator func(*Treatment) error) (Treatment, error) {
	current, ok := tx.state.treatments[id]
	if !ok {
		return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("treatment %q not found", id)
	}
	before := cloneTreatment(current)
	if err := mutator(&current); err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	if current.ProcedureID == "" {
		return Treatment{Treatment: entitymodel.Treatment{}}, errors.New("treatment requires procedure id")
	}
	if _, ok := tx.state.procedures[current.ProcedureID]; !ok {
		return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("procedure %q not found", current.ProcedureID)
	}
	current.OrganismIDs = dedupeStrings(current.OrganismIDs)
	for _, organismID := range current.OrganismIDs {
		if _, ok := tx.state.organisms[organismID]; !ok {
			return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("organism %q not found for treatment", organismID)
		}
	}
	current.CohortIDs = dedupeStrings(current.CohortIDs)
	for _, cohortID := range current.CohortIDs {
		if _, ok := tx.state.cohorts[cohortID]; !ok {
			return Treatment{Treatment: entitymodel.Treatment{}}, fmt.Errorf("cohort %q not found for treatment", cohortID)
		}
	}
	if err := normalizeTreatment(&current); err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.treatments[id] = cloneTreatment(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneTreatment(current))
	if err != nil {
		return Treatment{Treatment: entitymodel.Treatment{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneTreatment(current), nil
}
func (tx *transaction) DeleteTreatment(id string) error {
	current, ok := tx.state.treatments[id]
	if !ok {
		return fmt.Errorf("treatment %q not found", id)
	}
	delete(tx.state.treatments, id)
	beforePayload, err := changePayloadFromValue(cloneTreatment(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityTreatment, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateObservation(o Observation) (Observation, error) {
	if o.ID == "" {
		o.ID = tx.store.newID()
	}
	if _, exists := tx.state.observations[o.ID]; exists {
		return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("observation %q already exists", o.ID)
	}
	if o.ProcedureID == nil && o.OrganismID == nil && o.CohortID == nil {
		return Observation{Observation: entitymodel.Observation{}}, errors.New("observation requires procedure, organism, or cohort reference")
	}
	if o.ProcedureID != nil {
		if _, ok := tx.state.procedures[*o.ProcedureID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("procedure %q not found for observation", *o.ProcedureID)
		}
	}
	if o.OrganismID != nil {
		if _, ok := tx.state.organisms[*o.OrganismID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("organism %q not found for observation", *o.OrganismID)
		}
	}
	if o.CohortID != nil {
		if _, ok := tx.state.cohorts[*o.CohortID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("cohort %q not found for observation", *o.CohortID)
		}
	}
	o.CreatedAt = tx.now
	o.UpdatedAt = tx.now
	if data := o.ObservationData(); data == nil {
		mustApply("apply observation data", o.ApplyObservationData(map[string]any{}))
	} else {
		mustApply("apply observation data", o.ApplyObservationData(data))
	}
	tx.state.observations[o.ID] = cloneObservation(o)
	after, err := changePayloadFromValue(cloneObservation(o))
	if err != nil {
		return Observation{Observation: entitymodel.Observation{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionCreate, After: after})
	return cloneObservation(o), nil
}
func (tx *transaction) UpdateObservation(id string, mutator func(*Observation) error) (Observation, error) {
	current, ok := tx.state.observations[id]
	if !ok {
		return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("observation %q not found", id)
	}
	before := cloneObservation(current)
	if err := mutator(&current); err != nil {
		return Observation{Observation: entitymodel.Observation{}}, err
	}
	if current.ProcedureID == nil && current.OrganismID == nil && current.CohortID == nil {
		return Observation{Observation: entitymodel.Observation{}}, errors.New("observation requires procedure, organism, or cohort reference")
	}
	if current.ProcedureID != nil {
		if _, ok := tx.state.procedures[*current.ProcedureID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("procedure %q not found for observation", *current.ProcedureID)
		}
	}
	if current.OrganismID != nil {
		if _, ok := tx.state.organisms[*current.OrganismID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("organism %q not found for observation", *current.OrganismID)
		}
	}
	if current.CohortID != nil {
		if _, ok := tx.state.cohorts[*current.CohortID]; !ok {
			return Observation{Observation: entitymodel.Observation{}}, fmt.Errorf("cohort %q not found for observation", *current.CohortID)
		}
	}
	if data := current.ObservationData(); data == nil {
		mustApply("apply observation data", current.ApplyObservationData(map[string]any{}))
	} else {
		mustApply("apply observation data", current.ApplyObservationData(data))
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.observations[id] = cloneObservation(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Observation{Observation: entitymodel.Observation{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneObservation(current))
	if err != nil {
		return Observation{Observation: entitymodel.Observation{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneObservation(current), nil
}
func (tx *transaction) DeleteObservation(id string) error {
	current, ok := tx.state.observations[id]
	if !ok {
		return fmt.Errorf("observation %q not found", id)
	}
	delete(tx.state.observations, id)
	beforePayload, err := changePayloadFromValue(cloneObservation(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityObservation, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateSample(s Sample) (Sample, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.samples[s.ID]; exists {
		return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("sample %q already exists", s.ID)
	}
	if s.FacilityID == "" {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires facility id")
	}
	if _, ok := tx.state.facilities[s.FacilityID]; !ok {
		return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("facility %q not found for sample", s.FacilityID)
	}
	if s.OrganismID == nil && s.CohortID == nil {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires organism or cohort reference")
	}
	if s.OrganismID != nil {
		if _, ok := tx.state.organisms[*s.OrganismID]; !ok {
			return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("organism %q not found for sample", *s.OrganismID)
		}
	}
	if s.CohortID != nil {
		if _, ok := tx.state.cohorts[*s.CohortID]; !ok {
			return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("cohort %q not found for sample", *s.CohortID)
		}
	}
	if len(s.ChainOfCustody) == 0 {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires chain of custody")
	}
	if err := normalizeSample(&s); err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if attrs := s.SampleAttributes(); attrs == nil {
		mustApply("apply sample attributes", s.ApplySampleAttributes(map[string]any{}))
	} else {
		mustApply("apply sample attributes", s.ApplySampleAttributes(attrs))
	}
	tx.state.samples[s.ID] = cloneSample(s)
	after, err := changePayloadFromValue(cloneSample(s))
	if err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionCreate, After: after})
	return cloneSample(s), nil
}
func (tx *transaction) UpdateSample(id string, mutator func(*Sample) error) (Sample, error) {
	current, ok := tx.state.samples[id]
	if !ok {
		return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("sample %q not found", id)
	}
	before := cloneSample(current)
	if err := mutator(&current); err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	if current.FacilityID == "" {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires facility id")
	}
	if _, ok := tx.state.facilities[current.FacilityID]; !ok {
		return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("facility %q not found for sample", current.FacilityID)
	}
	if current.OrganismID == nil && current.CohortID == nil {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires organism or cohort reference")
	}
	if current.OrganismID != nil {
		if _, ok := tx.state.organisms[*current.OrganismID]; !ok {
			return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("organism %q not found for sample", *current.OrganismID)
		}
	}
	if current.CohortID != nil {
		if _, ok := tx.state.cohorts[*current.CohortID]; !ok {
			return Sample{Sample: entitymodel.Sample{}}, fmt.Errorf("cohort %q not found for sample", *current.CohortID)
		}
	}
	if len(current.ChainOfCustody) == 0 {
		return Sample{Sample: entitymodel.Sample{}}, errors.New("sample requires chain of custody")
	}
	if err := normalizeSample(&current); err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	if attrs := current.SampleAttributes(); attrs == nil {
		mustApply("apply sample attributes", current.ApplySampleAttributes(map[string]any{}))
	} else {
		mustApply("apply sample attributes", current.ApplySampleAttributes(attrs))
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.samples[id] = cloneSample(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneSample(current))
	if err != nil {
		return Sample{Sample: entitymodel.Sample{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneSample(current), nil
}
func (tx *transaction) DeleteSample(id string) error {
	current, ok := tx.state.samples[id]
	if !ok {
		return fmt.Errorf("sample %q not found", id)
	}
	delete(tx.state.samples, id)
	beforePayload, err := changePayloadFromValue(cloneSample(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntitySample, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateProtocol(p Protocol) (Protocol, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.protocols[p.ID]; exists {
		return Protocol{Protocol: entitymodel.Protocol{}}, fmt.Errorf("protocol %q already exists", p.ID)
	}
	if err := normalizeProtocol(&p); err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.protocols[p.ID] = cloneProtocol(p)
	after, err := changePayloadFromValue(cloneProtocol(p))
	if err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionCreate, After: after})
	return cloneProtocol(p), nil
}
func (tx *transaction) UpdateProtocol(id string, mutator func(*Protocol) error) (Protocol, error) {
	current, ok := tx.state.protocols[id]
	if !ok {
		return Protocol{Protocol: entitymodel.Protocol{}}, fmt.Errorf("protocol %q not found", id)
	}
	before := cloneProtocol(current)
	if err := mutator(&current); err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	if err := normalizeProtocol(&current); err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.protocols[id] = cloneProtocol(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneProtocol(current))
	if err != nil {
		return Protocol{Protocol: entitymodel.Protocol{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneProtocol(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityProtocol, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreatePermit(p Permit) (Permit, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.permits[p.ID]; exists {
		return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("permit %q already exists", p.ID)
	}
	if err := requireNonEmpty("permit.allowed_activities", p.AllowedActivities); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	p.FacilityIDs = dedupeStrings(p.FacilityIDs)
	if err := requireNonEmpty("permit.facility_ids", p.FacilityIDs); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	for _, facilityID := range p.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("facility %q not found for permit", facilityID)
		}
	}
	p.ProtocolIDs = dedupeStrings(p.ProtocolIDs)
	if err := requireNonEmpty("permit.protocol_ids", p.ProtocolIDs); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	for _, protocolID := range p.ProtocolIDs {
		if _, ok := tx.state.protocols[protocolID]; !ok {
			return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("protocol %q not found for permit", protocolID)
		}
	}
	if err := normalizePermit(&p); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.permits[p.ID] = clonePermit(p)
	after, err := changePayloadFromValue(clonePermit(p))
	if err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionCreate, After: after})
	return clonePermit(p), nil
}
func (tx *transaction) UpdatePermit(id string, mutator func(*Permit) error) (Permit, error) {
	current, ok := tx.state.permits[id]
	if !ok {
		return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("permit %q not found", id)
	}
	before := clonePermit(current)
	if err := mutator(&current); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	if err := requireNonEmpty("permit.allowed_activities", current.AllowedActivities); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	if err := requireNonEmpty("permit.facility_ids", current.FacilityIDs); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("facility %q not found for permit", facilityID)
		}
	}
	current.ProtocolIDs = dedupeStrings(current.ProtocolIDs)
	if err := requireNonEmpty("permit.protocol_ids", current.ProtocolIDs); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	for _, protocolID := range current.ProtocolIDs {
		if _, ok := tx.state.protocols[protocolID]; !ok {
			return Permit{Permit: entitymodel.Permit{}}, fmt.Errorf("protocol %q not found for permit", protocolID)
		}
	}
	if err := normalizePermit(&current); err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.permits[id] = clonePermit(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	afterPayload, err := changePayloadFromValue(clonePermit(current))
	if err != nil {
		return Permit{Permit: entitymodel.Permit{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return clonePermit(current), nil
}
func (tx *transaction) DeletePermit(id string) error {
	current, ok := tx.state.permits[id]
	if !ok {
		return fmt.Errorf("permit %q not found", id)
	}
	delete(tx.state.permits, id)
	beforePayload, err := changePayloadFromValue(clonePermit(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityPermit, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateProject(p Project) (Project, error) {
	if p.ID == "" {
		p.ID = tx.store.newID()
	}
	if _, exists := tx.state.projects[p.ID]; exists {
		return Project{Project: entitymodel.Project{}}, fmt.Errorf("project %q already exists", p.ID)
	}
	p.FacilityIDs = dedupeStrings(p.FacilityIDs)
	if err := requireNonEmpty("project.facility_ids", p.FacilityIDs); err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	for _, facilityID := range p.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Project{Project: entitymodel.Project{}}, fmt.Errorf("facility %q not found for project", facilityID)
		}
	}
	p.OrganismIDs = nil
	p.ProcedureIDs = nil
	p.SupplyItemIDs = nil
	p.CreatedAt = tx.now
	p.UpdatedAt = tx.now
	tx.state.projects[p.ID] = cloneProject(p)
	created := decorateProject(&tx.state, p)
	after, err := changePayloadFromValue(cloneProject(created))
	if err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionCreate, After: after})
	return cloneProject(created), nil
}
func (tx *transaction) UpdateProject(id string, mutator func(*Project) error) (Project, error) {
	current, ok := tx.state.projects[id]
	if !ok {
		return Project{Project: entitymodel.Project{}}, fmt.Errorf("project %q not found", id)
	}
	beforeDecorated := decorateProject(&tx.state, current)
	before := cloneProject(beforeDecorated)
	if err := mutator(&current); err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	if err := requireNonEmpty("project.facility_ids", current.FacilityIDs); err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return Project{Project: entitymodel.Project{}}, fmt.Errorf("facility %q not found for project", facilityID)
		}
	}
	current.OrganismIDs = nil
	current.ProcedureIDs = nil
	current.SupplyItemIDs = nil
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.projects[id] = cloneProject(current)
	afterDecorated := decorateProject(&tx.state, current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneProject(afterDecorated))
	if err != nil {
		return Project{Project: entitymodel.Project{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
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
	beforePayload, err := changePayloadFromValue(cloneProject(decoratedCurrent))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntityProject, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (tx *transaction) CreateSupplyItem(s SupplyItem) (SupplyItem, error) {
	if s.ID == "" {
		s.ID = tx.store.newID()
	}
	if _, exists := tx.state.supplies[s.ID]; exists {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("supply item %q already exists", s.ID)
	}
	s.FacilityIDs = dedupeStrings(s.FacilityIDs)
	if err := requireNonEmpty("supply_item.facility_ids", s.FacilityIDs); err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	for _, facilityID := range s.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("facility %q not found for supply item", facilityID)
		}
	}
	s.ProjectIDs = dedupeStrings(s.ProjectIDs)
	if err := requireNonEmpty("supply_item.project_ids", s.ProjectIDs); err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	for _, projectID := range s.ProjectIDs {
		if _, ok := tx.state.projects[projectID]; !ok {
			return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("project %q not found for supply item", projectID)
		}
	}
	s.CreatedAt = tx.now
	s.UpdatedAt = tx.now
	if attrs := s.SupplyAttributes(); attrs == nil {
		mustApply("apply supply attributes", s.ApplySupplyAttributes(map[string]any{}))
	} else {
		mustApply("apply supply attributes", s.ApplySupplyAttributes(attrs))
	}
	tx.state.supplies[s.ID] = cloneSupplyItem(s)
	after, err := changePayloadFromValue(cloneSupplyItem(s))
	if err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionCreate, After: after})
	return cloneSupplyItem(s), nil
}
func (tx *transaction) UpdateSupplyItem(id string, mutator func(*SupplyItem) error) (SupplyItem, error) {
	current, ok := tx.state.supplies[id]
	if !ok {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("supply item %q not found", id)
	}
	before := cloneSupplyItem(current)
	if err := mutator(&current); err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	current.FacilityIDs = dedupeStrings(current.FacilityIDs)
	if err := requireNonEmpty("supply_item.facility_ids", current.FacilityIDs); err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	for _, facilityID := range current.FacilityIDs {
		if _, ok := tx.state.facilities[facilityID]; !ok {
			return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("facility %q not found for supply item", facilityID)
		}
	}
	current.ProjectIDs = dedupeStrings(current.ProjectIDs)
	if err := requireNonEmpty("supply_item.project_ids", current.ProjectIDs); err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	for _, projectID := range current.ProjectIDs {
		if _, ok := tx.state.projects[projectID]; !ok {
			return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, fmt.Errorf("project %q not found for supply item", projectID)
		}
	}
	if attrs := current.SupplyAttributes(); attrs == nil {
		mustApply("apply supply attributes", current.ApplySupplyAttributes(map[string]any{}))
	} else {
		mustApply("apply supply attributes", current.ApplySupplyAttributes(attrs))
	}
	if current.ExpiresAt != nil {
		t := *current.ExpiresAt
		current.ExpiresAt = &t
	}
	current.ID = id
	current.UpdatedAt = tx.now
	tx.state.supplies[id] = cloneSupplyItem(current)
	beforePayload, err := changePayloadFromValue(before)
	if err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	afterPayload, err := changePayloadFromValue(cloneSupplyItem(current))
	if err != nil {
		return SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, err
	}
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionUpdate, Before: beforePayload, After: afterPayload})
	return cloneSupplyItem(current), nil
}
func (tx *transaction) DeleteSupplyItem(id string) error {
	current, ok := tx.state.supplies[id]
	if !ok {
		return fmt.Errorf("supply item %q not found", id)
	}
	delete(tx.state.supplies, id)
	beforePayload, err := changePayloadFromValue(cloneSupplyItem(current))
	if err != nil {
		return err
	}
	tx.recordChange(Change{Entity: domain.EntitySupplyItem, Action: domain.ActionDelete, Before: beforePayload})
	return nil
}
func (s *memStore) GetOrganism(id string) (Organism, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.state.organisms[id]
	if !ok {
		return Organism{Organism: entitymodel.Organism{}}, false
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
		return HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, false
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
		return Facility{Facility: entitymodel.Facility{}}, false
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
func (s *memStore) GetLine(id string) (Line, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	line, ok := s.state.lines[id]
	if !ok {
		return Line{Line: entitymodel.Line{}}, false
	}
	return cloneLine(line), true
}
func (s *memStore) ListLines() []Line {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Line, 0, len(s.state.lines))
	for _, line := range s.state.lines {
		out = append(out, cloneLine(line))
	}
	return out
}
func (s *memStore) GetStrain(id string) (Strain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	strain, ok := s.state.strains[id]
	if !ok {
		return Strain{Strain: entitymodel.Strain{}}, false
	}
	return cloneStrain(strain), true
}
func (s *memStore) ListStrains() []Strain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Strain, 0, len(s.state.strains))
	for _, strain := range s.state.strains {
		out = append(out, cloneStrain(strain))
	}
	return out
}
func (s *memStore) GetGenotypeMarker(id string) (GenotypeMarker, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	marker, ok := s.state.markers[id]
	if !ok {
		return GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{}}, false
	}
	return cloneGenotypeMarker(marker), true
}
func (s *memStore) ListGenotypeMarkers() []GenotypeMarker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]GenotypeMarker, 0, len(s.state.markers))
	for _, marker := range s.state.markers {
		out = append(out, cloneGenotypeMarker(marker))
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
		return Permit{Permit: entitymodel.Permit{}}, false
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
