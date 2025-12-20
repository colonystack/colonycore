// Package domain defines the core persistent entities, value types, and
// rule evaluation primitives used by colonycore.
package domain

import (
	"colonycore/pkg/domain/entitymodel"
	"colonycore/pkg/domain/extension"
	"encoding/json"
)

// EntityType identifies the type of record stored in the core domain.
type EntityType string

// Supported entity type identifiers used in Change records and persistence buckets.
const (
	// EntityOrganism identifies an individual organism record.
	EntityOrganism EntityType = "organism"
	// EntityCohort identifies a cohort record.
	EntityCohort EntityType = "cohort"
	// EntityHousingUnit identifies a housing unit record.
	EntityHousingUnit EntityType = "housing_unit"
	// EntityFacility identifies a facility record.
	EntityFacility EntityType = "facility"
	// EntityBreeding identifies a breeding unit record.
	EntityBreeding EntityType = "breeding_unit"
	// EntityProcedure identifies a procedure record.
	EntityProcedure EntityType = "procedure"
	// EntityTreatment identifies a treatment record.
	EntityTreatment EntityType = "treatment"
	// EntityObservation identifies an observation record.
	EntityObservation EntityType = "observation"
	// EntitySample identifies a sample record.
	EntitySample EntityType = "sample"
	// EntityProtocol identifies a protocol record.
	EntityProtocol EntityType = "protocol"
	// EntityProject identifies a project record.
	EntityProject EntityType = "project"
	// EntityPermit identifies a permit record.
	EntityPermit EntityType = "permit"
	// EntitySupplyItem identifies a supply item record.
	EntitySupplyItem EntityType = "supply_item"
	// EntityLine identifies a genetic line record.
	EntityLine EntityType = "line"
	// EntityStrain identifies a managed strain record derived from a line.
	EntityStrain EntityType = "strain"
	// EntityGenotypeMarker identifies a genotype marker definition record.
	EntityGenotypeMarker EntityType = "genotype_marker"
)

// LifecycleStage represents the canonical organism lifecycle states described in the RFC.
type LifecycleStage = entitymodel.LifecycleStage

// Canonical organism lifecycle stages used for housing and capacity rule evaluation.
const (
	// StagePlanned indicates a planned organism not yet created in lab.
	StagePlanned LifecycleStage = entitymodel.LifecycleStagePlanned
	// StageLarva indicates embryonic or larval stage.
	StageLarva    LifecycleStage = entitymodel.LifecycleStageEmbryoLarva
	StageJuvenile LifecycleStage = entitymodel.LifecycleStageJuvenile
	StageAdult    LifecycleStage = entitymodel.LifecycleStageAdult
	StageRetired  LifecycleStage = entitymodel.LifecycleStageRetired
	StageDeceased LifecycleStage = entitymodel.LifecycleStageDeceased
)

// ProtocolStatus enumerates compliance protocol lifecycle states (RFC-0001 §5.3).
type ProtocolStatus = entitymodel.ProtocolStatus

// Canonical protocol statuses aligned to Entity Model v0.
const (
	ProtocolStatusDraft     ProtocolStatus = entitymodel.ProtocolStatusDraft
	ProtocolStatusSubmitted ProtocolStatus = entitymodel.ProtocolStatusSubmitted
	ProtocolStatusApproved  ProtocolStatus = entitymodel.ProtocolStatusApproved
	ProtocolStatusOnHold    ProtocolStatus = entitymodel.ProtocolStatusOnHold
	ProtocolStatusExpired   ProtocolStatus = entitymodel.ProtocolStatusExpired
	ProtocolStatusArchived  ProtocolStatus = entitymodel.ProtocolStatusArchived
)

// ProcedureStatus enumerates canonical procedure workflow states (RFC-0001 §5.4).
type ProcedureStatus = entitymodel.ProcedureStatus

// Canonical procedure statuses used for scheduling and validation.
const (
	ProcedureStatusScheduled  ProcedureStatus = entitymodel.ProcedureStatusScheduled
	ProcedureStatusInProgress ProcedureStatus = entitymodel.ProcedureStatusInProgress
	ProcedureStatusCompleted  ProcedureStatus = entitymodel.ProcedureStatusCompleted
	ProcedureStatusCancelled  ProcedureStatus = entitymodel.ProcedureStatusCancelled
	ProcedureStatusFailed     ProcedureStatus = entitymodel.ProcedureStatusFailed
)

// TreatmentStatus enumerates treatment lifecycle states enforced by the plugin contract.
type TreatmentStatus = entitymodel.TreatmentStatus

// Canonical treatment statuses recognised by rule and dataset adapters.
const (
	TreatmentStatusPlanned    TreatmentStatus = entitymodel.TreatmentStatusPlanned
	TreatmentStatusInProgress TreatmentStatus = entitymodel.TreatmentStatusInProgress
	TreatmentStatusCompleted  TreatmentStatus = entitymodel.TreatmentStatusCompleted
	TreatmentStatusFlagged    TreatmentStatus = entitymodel.TreatmentStatusFlagged
)

// SampleStatus enumerates sample custody states (stored, in transit, consumed, disposed).
type SampleStatus = entitymodel.SampleStatus

// Canonical sample statuses used for chain-of-custody validation.
const (
	SampleStatusStored    SampleStatus = entitymodel.SampleStatusStored
	SampleStatusInTransit SampleStatus = entitymodel.SampleStatusInTransit
	SampleStatusConsumed  SampleStatus = entitymodel.SampleStatusConsumed
	SampleStatusDisposed  SampleStatus = entitymodel.SampleStatusDisposed
)

// PermitStatus enumerates permit validity states consumed by compliance workflows.
type PermitStatus = entitymodel.PermitStatus

// Canonical permit statuses describing regulatory validity.
const (
	PermitStatusDraft     PermitStatus = entitymodel.PermitStatusDraft
	PermitStatusSubmitted PermitStatus = entitymodel.PermitStatusSubmitted
	PermitStatusApproved  PermitStatus = entitymodel.PermitStatusApproved
	PermitStatusOnHold    PermitStatus = entitymodel.PermitStatusOnHold
	PermitStatusExpired   PermitStatus = entitymodel.PermitStatusExpired
	PermitStatusArchived  PermitStatus = entitymodel.PermitStatusArchived
)

// HousingState enumerates lifecycle states for housing units (RFC-0001 §5.2).
type HousingState = entitymodel.HousingState

// Canonical housing lifecycle states aligned to Entity Model v0.
const (
	HousingStateQuarantine     HousingState = entitymodel.HousingStateQuarantine
	HousingStateActive         HousingState = entitymodel.HousingStateActive
	HousingStateCleaning       HousingState = entitymodel.HousingStateCleaning
	HousingStateDecommissioned HousingState = entitymodel.HousingStateDecommissioned
)

// HousingEnvironment enumerates canonical housing environments.
type HousingEnvironment = entitymodel.HousingEnvironment

// Canonical housing environments aligned to Entity Model v0.
const (
	HousingEnvironmentAquatic     HousingEnvironment = entitymodel.HousingEnvironmentAquatic
	HousingEnvironmentTerrestrial HousingEnvironment = entitymodel.HousingEnvironmentTerrestrial
	HousingEnvironmentArboreal    HousingEnvironment = entitymodel.HousingEnvironmentArboreal
	HousingEnvironmentHumid       HousingEnvironment = entitymodel.HousingEnvironmentHumid
)

// Severity captures rule outcomes.
type Severity string

// Rule evaluation severities determine commit behavior and logging.
const (
	// SeverityBlock blocks transaction commit.
	SeverityBlock Severity = "block"
	// SeverityWarn logs a warning but allows commit.
	SeverityWarn Severity = "warn"
	SeverityLog  Severity = "log"
)

// Organism represents an individual animal tracked by the system.
type Organism struct {
	entitymodel.Organism
	extensions *extension.Container `json:"-"`
}

// Cohort represents a managed group of organisms.
type Cohort struct {
	entitymodel.Cohort
}

// HousingUnit captures physical housing metadata.
type HousingUnit struct {
	entitymodel.HousingUnit
}

// Facility aggregates housing units with shared biosecurity controls.
type Facility struct {
	entitymodel.Facility
	extensions *extension.Container `json:"-"`
}

// BreedingUnit tracks configured pairings or groups intended for reproduction.
type BreedingUnit struct {
	entitymodel.BreedingUnit
	extensions *extension.Container `json:"-"`
}

// Line represents a genetic lineage with shared inheritance characteristics.
type Line struct {
	entitymodel.Line
	defaultAttributesSlot  *extension.Slot      `json:"-"` // cache avoids rehydrating the container; container remains canonical
	extensionOverridesSlot *extension.Slot      `json:"-"` // cache avoids rehydrating the container; container remains canonical
	extensions             *extension.Container `json:"-"`
}

// Strain represents a managed sub-population derived from a line.
type Strain struct {
	entitymodel.Strain
	extensions *extension.Container `json:"-"`
}

// GenotypeMarker captures assay metadata for genetic markers used in lineage tracking.
type GenotypeMarker struct {
	entitymodel.GenotypeMarker
	extensions *extension.Container `json:"-"`
}

// Procedure captures scheduled or completed animal procedures.
type Procedure struct {
	entitymodel.Procedure
}

// Treatment captures therapeutic interventions and their outcomes.
type Treatment struct {
	entitymodel.Treatment
}

// Observation records structured or free-form notes captured during workflows.
type Observation struct {
	entitymodel.Observation
	extensions *extension.Container `json:"-"`
}

// Sample tracks material derived from organisms or cohorts.
type Sample struct {
	entitymodel.Sample
	extensions *extension.Container `json:"-"`
}

// SampleCustodyEvent logs a change in possession or storage for a sample.
type SampleCustodyEvent = entitymodel.SampleCustodyEvent

// Protocol represents compliance agreements.
type Protocol struct {
	entitymodel.Protocol
}

// Permit represents external authorizations needed for compliance.
type Permit struct {
	entitymodel.Permit
}

// Project captures cost center allocations.
type Project struct {
	entitymodel.Project
}

// SupplyItem models inventory resources consumed by projects or facilities.
type SupplyItem struct {
	entitymodel.SupplyItem
	extensions *extension.Container `json:"-"`
}

type organismAlias entitymodel.Organism

// MarshalJSON ensures organism attributes are serialised via the core plugin payload.
func (o Organism) MarshalJSON() ([]byte, error) {
	container, err := o.OrganismExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		organismAlias
		Attributes map[string]any            `json:"attributes,omitempty"`
		Extensions map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		organismAlias: organismAlias(o.Organism),
		Attributes:    (&o).CoreAttributes(),
		Extensions:    extensions,
	})
}

// UnmarshalJSON hydrates organism extension slots from the JSON payload.
func (o *Organism) UnmarshalJSON(data []byte) error {
	type payload struct {
		organismAlias
		Attributes map[string]any            `json:"attributes"`
		Extensions map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.Organism = entitymodel.Organism(aux.organismAlias)
	if len(aux.Extensions) != 0 {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if err := o.SetOrganismExtensions(container); err != nil {
			return err
		}
	}
	if aux.Attributes == nil {
		return nil
	}
	return o.SetCoreAttributes(aux.Attributes)
}

type facilityAlias entitymodel.Facility

// MarshalJSON ensures facility environment baselines are serialised via the core plugin payload.
func (f Facility) MarshalJSON() ([]byte, error) {
	container, err := f.FacilityExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		facilityAlias
		EnvironmentBaselines map[string]any            `json:"environment_baselines,omitempty"`
		Extensions           map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		facilityAlias:        facilityAlias(f.Facility),
		EnvironmentBaselines: (&f).EnvironmentBaselines(),
		Extensions:           extensions,
	})
}

// UnmarshalJSON hydrates facility extension slots from the JSON payload.
func (f *Facility) UnmarshalJSON(data []byte) error {
	type payload struct {
		facilityAlias
		EnvironmentBaselines map[string]any            `json:"environment_baselines"`
		Extensions           map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	f.Facility = entitymodel.Facility(aux.facilityAlias)
	if len(aux.Extensions) != 0 {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if err := f.SetFacilityExtensions(container); err != nil {
			return err
		}
	}
	return f.ApplyEnvironmentBaselines(aux.EnvironmentBaselines)
}

type breedingUnitAlias entitymodel.BreedingUnit

// MarshalJSON ensures breeding unit pairing attributes are serialised via the core plugin payload.
func (b BreedingUnit) MarshalJSON() ([]byte, error) {
	container, err := b.BreedingUnitExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		breedingUnitAlias
		PairingAttributes map[string]any            `json:"pairing_attributes,omitempty"`
		Extensions        map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		breedingUnitAlias: breedingUnitAlias(b.BreedingUnit),
		PairingAttributes: (&b).PairingAttributes(),
		Extensions:        extensions,
	})
}

// UnmarshalJSON hydrates breeding unit extension slots from the JSON payload.
func (b *BreedingUnit) UnmarshalJSON(data []byte) error {
	type payload struct {
		breedingUnitAlias
		PairingAttributes map[string]any            `json:"pairing_attributes"`
		Extensions        map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.BreedingUnit = entitymodel.BreedingUnit(aux.breedingUnitAlias)
	if len(aux.Extensions) != 0 {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if err := b.SetBreedingUnitExtensions(container); err != nil {
			return err
		}
	}
	return b.ApplyPairingAttributes(aux.PairingAttributes)
}

type observationAlias entitymodel.Observation

// MarshalJSON ensures observation data are serialised via the core plugin payload.
func (o Observation) MarshalJSON() ([]byte, error) {
	container, err := o.ObservationExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		observationAlias
		Data       map[string]any            `json:"data,omitempty"`
		Extensions map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		observationAlias: observationAlias(o.Observation),
		Data:             (&o).ObservationData(),
		Extensions:       extensions,
	})
}

// UnmarshalJSON hydrates observation extension slots from the JSON payload.
func (o *Observation) UnmarshalJSON(data []byte) error {
	type payload struct {
		observationAlias
		Data       map[string]any            `json:"data"`
		Extensions map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.Observation = entitymodel.Observation(aux.observationAlias)
	if aux.Extensions != nil {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if _, ok := container.Get(extension.HookObservationData, extension.PluginCore); !ok && aux.Data != nil {
			if err := container.Set(extension.HookObservationData, extension.PluginCore, aux.Data); err != nil {
				return err
			}
		}
		return o.SetObservationExtensions(container)
	}
	return o.ApplyObservationData(aux.Data)
}

type sampleAlias entitymodel.Sample

// MarshalJSON ensures sample attributes are serialised via the core plugin payload.
func (s Sample) MarshalJSON() ([]byte, error) {
	container, err := s.SampleExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		sampleAlias
		Attributes map[string]any            `json:"attributes,omitempty"`
		Extensions map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		sampleAlias: sampleAlias(s.Sample),
		Attributes:  (&s).SampleAttributes(),
		Extensions:  extensions,
	})
}

// UnmarshalJSON hydrates sample extension slots from the JSON payload.
func (s *Sample) UnmarshalJSON(data []byte) error {
	type payload struct {
		sampleAlias
		Attributes map[string]any            `json:"attributes"`
		Extensions map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Sample = entitymodel.Sample(aux.sampleAlias)
	if len(aux.Extensions) != 0 {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if err := s.SetSampleExtensions(container); err != nil {
			return err
		}
	}
	if aux.Attributes == nil {
		return nil
	}
	return s.ApplySampleAttributes(aux.Attributes)
}

type supplyAlias entitymodel.SupplyItem

// MarshalJSON ensures supply item attributes are serialised via the core plugin payload.
func (s SupplyItem) MarshalJSON() ([]byte, error) {
	container, err := s.SupplyItemExtensions()
	if err != nil {
		return nil, err
	}
	extensions := container.Raw()
	if len(extensions) == 0 {
		extensions = nil
	}
	type payload struct {
		supplyAlias
		Attributes map[string]any            `json:"attributes,omitempty"`
		Extensions map[string]map[string]any `json:"extensions,omitempty"`
	}
	return json.Marshal(payload{
		supplyAlias: supplyAlias(s.SupplyItem),
		Attributes:  (&s).SupplyAttributes(),
		Extensions:  extensions,
	})
}

// UnmarshalJSON hydrates supply item extension slots from the JSON payload.
func (s *SupplyItem) UnmarshalJSON(data []byte) error {
	type payload struct {
		supplyAlias
		Attributes map[string]any            `json:"attributes"`
		Extensions map[string]map[string]any `json:"extensions"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.SupplyItem = entitymodel.SupplyItem(aux.supplyAlias)
	if len(aux.Extensions) != 0 {
		container, err := extension.FromRaw(aux.Extensions)
		if err != nil {
			return err
		}
		if err := s.SetSupplyItemExtensions(container); err != nil {
			return err
		}
	}
	if aux.Attributes == nil {
		return nil
	}
	return s.ApplySupplyAttributes(aux.Attributes)
}

type lineAlias entitymodel.Line

// MarshalJSON ensures line extension slots are serialised as hook-indexed maps.
func (l Line) MarshalJSON() ([]byte, error) {
	type payload struct {
		lineAlias
		DefaultAttributes  map[string]any `json:"default_attributes"`
		ExtensionOverrides map[string]any `json:"extension_overrides"`
	}

	return json.Marshal(payload{
		lineAlias:          lineAlias(l.Line),
		DefaultAttributes:  (&l).DefaultAttributes(),
		ExtensionOverrides: (&l).ExtensionOverrides(),
	})
}

// UnmarshalJSON hydrates line extension slots from the JSON payload.
func (l *Line) UnmarshalJSON(data []byte) error {
	type payload struct {
		lineAlias
		DefaultAttributes  map[string]any `json:"default_attributes"`
		ExtensionOverrides map[string]any `json:"extension_overrides"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	l.Line = entitymodel.Line(aux.lineAlias)
	if err := l.ApplyDefaultAttributes(aux.DefaultAttributes); err != nil {
		return err
	}
	return l.ApplyExtensionOverrides(aux.ExtensionOverrides)
}

type strainAlias entitymodel.Strain

// MarshalJSON ensures strain attributes slot is serialised via hook-indexed map.
func (s Strain) MarshalJSON() ([]byte, error) {
	type payload struct {
		strainAlias
		Attributes map[string]any `json:"attributes"`
	}

	return json.Marshal(payload{
		strainAlias: strainAlias(s.Strain),
		Attributes:  (&s).StrainAttributesByPlugin(),
	})
}

// UnmarshalJSON hydrates strain extension slot from the payload.
func (s *Strain) UnmarshalJSON(data []byte) error {
	type payload struct {
		strainAlias
		Attributes map[string]any `json:"attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Strain = entitymodel.Strain(aux.strainAlias)
	return s.ApplyStrainAttributes(aux.Attributes)
}

type genotypeMarkerAlias entitymodel.GenotypeMarker

// MarshalJSON ensures genotype marker attributes are serialised via hook-indexed map.
func (g GenotypeMarker) MarshalJSON() ([]byte, error) {
	type payload struct {
		genotypeMarkerAlias
		Attributes map[string]any `json:"attributes"`
	}

	return json.Marshal(payload{
		genotypeMarkerAlias: genotypeMarkerAlias(g.GenotypeMarker),
		Attributes:          (&g).GenotypeMarkerAttributesByPlugin(),
	})
}

// UnmarshalJSON hydrates genotype marker attributes from the JSON payload.
func (g *GenotypeMarker) UnmarshalJSON(data []byte) error {
	type payload struct {
		genotypeMarkerAlias
		Attributes map[string]any `json:"attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.GenotypeMarker = entitymodel.GenotypeMarker(aux.genotypeMarkerAlias)
	return g.ApplyGenotypeMarkerAttributes(aux.Attributes)
}

// Change describes a mutation applied to an entity during a transaction.
type Change struct {
	Entity EntityType
	Action Action
	Before any
	After  any
}

// Action indicates the type of modification performed.
type Action string

// Change actions enumerate supported CRUD operations captured in audit trail.
const (
	// ActionCreate indicates an entity was created.
	ActionCreate Action = "create"
	// ActionUpdate indicates an entity was updated.
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

// Violation reports a failed rule evaluation.
type Violation struct {
	Rule     string
	Severity Severity
	Message  string
	Entity   EntityType
	EntityID string
}

// Result aggregates violations from the rules engine.
type Result struct {
	Violations []Violation
}

// Merge appends violations from another result.
func (r *Result) Merge(other Result) {
	if len(other.Violations) == 0 {
		return
	}
	r.Violations = append(r.Violations, other.Violations...)
}

// HasBlocking returns true if the result contains blocking violations.
func (r Result) HasBlocking() bool {
	for _, v := range r.Violations {
		if v.Severity == SeverityBlock {
			return true
		}
	}
	return false
}

// RuleViolationError is returned when blocking violations are present.
type RuleViolationError struct {
	Result Result
}

func (e RuleViolationError) Error() string {
	return "transaction blocked by rules"
}
