// Package domain defines the core persistent entities, value types, and
// rule evaluation primitives used by colonycore.
package domain

import (
	"colonycore/pkg/domain/extension"
	"encoding/json"
	"time"
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
type LifecycleStage string

// Canonical organism lifecycle stages used for housing and capacity rule evaluation.
const (
	// StagePlanned indicates a planned organism not yet created in lab.
	StagePlanned LifecycleStage = "planned"
	// StageLarva indicates embryonic or larval stage.
	StageLarva    LifecycleStage = "embryo_larva"
	StageJuvenile LifecycleStage = "juvenile"
	StageAdult    LifecycleStage = "adult"
	StageRetired  LifecycleStage = "retired"
	StageDeceased LifecycleStage = "deceased"
)

// ProcedureStatus enumerates canonical procedure workflow states (RFC-0001 §5.4).
type ProcedureStatus string

// Canonical procedure statuses used for scheduling and validation.
const (
	ProcedureStatusScheduled  ProcedureStatus = "scheduled"
	ProcedureStatusInProgress ProcedureStatus = "in_progress"
	ProcedureStatusCompleted  ProcedureStatus = "completed"
	ProcedureStatusCancelled  ProcedureStatus = "cancelled"
	ProcedureStatusFailed     ProcedureStatus = "failed"
)

// TreatmentStatus enumerates treatment lifecycle states enforced by the plugin contract.
type TreatmentStatus string

// Canonical treatment statuses recognised by rule and dataset adapters.
const (
	TreatmentStatusPlanned    TreatmentStatus = "planned"
	TreatmentStatusInProgress TreatmentStatus = "in_progress"
	TreatmentStatusCompleted  TreatmentStatus = "completed"
	TreatmentStatusFlagged    TreatmentStatus = "flagged"
)

// SampleStatus enumerates sample custody states (stored, in transit, consumed, disposed).
type SampleStatus string

// Canonical sample statuses used for chain-of-custody validation.
const (
	SampleStatusStored    SampleStatus = "stored"
	SampleStatusInTransit SampleStatus = "in_transit"
	SampleStatusConsumed  SampleStatus = "consumed"
	SampleStatusDisposed  SampleStatus = "disposed"
)

// PermitStatus enumerates permit validity states consumed by compliance workflows.
type PermitStatus string

// Canonical permit statuses describing regulatory validity.
const (
	PermitStatusPending PermitStatus = "pending"
	PermitStatusActive  PermitStatus = "active"
	PermitStatusExpired PermitStatus = "expired"
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

// Base contains common fields for all domain records.
type Base struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Organism represents an individual animal tracked by the system.
type Organism struct {
	Base
	Name           string               `json:"name"`
	Species        string               `json:"species"`
	Line           string               `json:"line"`
	LineID         *string              `json:"line_id"`
	StrainID       *string              `json:"strain_id"`
	ParentIDs      []string             `json:"parent_ids"`
	Stage          LifecycleStage       `json:"stage"`
	CohortID       *string              `json:"cohort_id"`
	HousingID      *string              `json:"housing_id"`
	ProtocolID     *string              `json:"protocol_id"`
	ProjectID      *string              `json:"project_id"`
	attributesSlot *extension.Slot      `json:"-"`
	extensions     *extension.Container `json:"-"`
}

// Cohort represents a managed group of organisms.
type Cohort struct {
	Base
	Name       string  `json:"name"`
	Purpose    string  `json:"purpose"`
	ProjectID  *string `json:"project_id"`
	HousingID  *string `json:"housing_id"`
	ProtocolID *string `json:"protocol_id"`
}

// HousingUnit captures physical housing metadata.
type HousingUnit struct {
	Base
	Name        string `json:"name"`
	FacilityID  string `json:"facility_id"`
	Capacity    int    `json:"capacity"`
	Environment string `json:"environment"`
}

// Facility aggregates housing units with shared biosecurity controls.
type Facility struct {
	Base
	Code                     string               `json:"code"`
	Name                     string               `json:"name"`
	Zone                     string               `json:"zone"`
	AccessPolicy             string               `json:"access_policy"`
	HousingUnitIDs           []string             `json:"housing_unit_ids"`
	ProjectIDs               []string             `json:"project_ids"`
	environmentBaselinesSlot *extension.Slot      `json:"-"`
	extensions               *extension.Container `json:"-"`
}

// BreedingUnit tracks configured pairings or groups intended for reproduction.
type BreedingUnit struct {
	Base
	Name                  string               `json:"name"`
	Strategy              string               `json:"strategy"`
	HousingID             *string              `json:"housing_id"`
	ProtocolID            *string              `json:"protocol_id"`
	LineID                *string              `json:"line_id"`
	StrainID              *string              `json:"strain_id"`
	TargetLineID          *string              `json:"target_line_id"`
	TargetStrainID        *string              `json:"target_strain_id"`
	PairingIntent         *string              `json:"pairing_intent,omitempty"`
	PairingNotes          *string              `json:"pairing_notes,omitempty"`
	FemaleIDs             []string             `json:"female_ids"`
	MaleIDs               []string             `json:"male_ids"`
	pairingAttributesSlot *extension.Slot      `json:"-"`
	extensions            *extension.Container `json:"-"`
}

// Line represents a genetic lineage with shared inheritance characteristics.
type Line struct {
	Base
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	Description        *string         `json:"description,omitempty"`
	Origin             string          `json:"origin"`
	GenotypeMarkerIDs  []string        `json:"genotype_marker_ids"`
	DefaultAttributes  *extension.Slot `json:"default_attributes"`
	DeprecatedAt       *time.Time      `json:"deprecated_at"`
	DeprecationReason  *string         `json:"deprecation_reason,omitempty"`
	ExtensionOverrides *extension.Slot `json:"extension_overrides"`
}

// Strain represents a managed sub-population derived from a line.
type Strain struct {
	Base
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	LineID            string          `json:"line_id"`
	Description       *string         `json:"description,omitempty"`
	Generation        *string         `json:"generation,omitempty"`
	GenotypeMarkerIDs []string        `json:"genotype_marker_ids"`
	Attributes        *extension.Slot `json:"attributes"`
	RetiredAt         *time.Time      `json:"retired_at"`
	RetirementReason  *string         `json:"retirement_reason,omitempty"`
}

// GenotypeMarker captures assay metadata for genetic markers used in lineage tracking.
type GenotypeMarker struct {
	Base
	Name           string          `json:"name"`
	Locus          string          `json:"locus"`
	Alleles        []string        `json:"alleles"`
	AssayMethod    string          `json:"assay_method"`
	Interpretation string          `json:"interpretation"`
	Version        string          `json:"version"`
	Attributes     *extension.Slot `json:"attributes"`
}

// Procedure captures scheduled or completed animal procedures.
type Procedure struct {
	Base
	Name           string          `json:"name"`
	Status         ProcedureStatus `json:"status"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	ProtocolID     string          `json:"protocol_id"`
	ProjectID      *string         `json:"project_id"`
	CohortID       *string         `json:"cohort_id"`
	OrganismIDs    []string        `json:"organism_ids"`
	TreatmentIDs   []string        `json:"treatment_ids"`
	ObservationIDs []string        `json:"observation_ids"`
}

// Treatment captures therapeutic interventions and their outcomes.
type Treatment struct {
	Base
	Name              string          `json:"name"`
	Status            TreatmentStatus `json:"status"`
	ProcedureID       string          `json:"procedure_id"`
	OrganismIDs       []string        `json:"organism_ids"`
	CohortIDs         []string        `json:"cohort_ids"`
	DosagePlan        string          `json:"dosage_plan"`
	AdministrationLog []string        `json:"administration_log"`
	AdverseEvents     []string        `json:"adverse_events"`
}

// Observation records structured or free-form notes captured during workflows.
type Observation struct {
	Base
	ProcedureID *string              `json:"procedure_id"`
	OrganismID  *string              `json:"organism_id"`
	CohortID    *string              `json:"cohort_id"`
	RecordedAt  time.Time            `json:"recorded_at"`
	Observer    string               `json:"observer"`
	Notes       *string              `json:"notes,omitempty"`
	dataSlot    *extension.Slot      `json:"-"`
	extensions  *extension.Container `json:"-"`
}

// Sample tracks material derived from organisms or cohorts.
type Sample struct {
	Base
	Identifier      string               `json:"identifier"`
	SourceType      string               `json:"source_type"`
	OrganismID      *string              `json:"organism_id"`
	CohortID        *string              `json:"cohort_id"`
	FacilityID      string               `json:"facility_id"`
	CollectedAt     time.Time            `json:"collected_at"`
	Status          SampleStatus         `json:"status"`
	StorageLocation string               `json:"storage_location"`
	AssayType       string               `json:"assay_type"`
	ChainOfCustody  []SampleCustodyEvent `json:"chain_of_custody"`
	attributesSlot  *extension.Slot      `json:"-"`
	extensions      *extension.Container `json:"-"`
}

// SampleCustodyEvent logs a change in possession or storage for a sample.
type SampleCustodyEvent struct {
	Actor     string    `json:"actor"`
	Location  string    `json:"location"`
	Timestamp time.Time `json:"timestamp"`
	Notes     *string   `json:"notes,omitempty"`
}

// Protocol represents compliance agreements.
type Protocol struct {
	Base
	Code        string  `json:"code"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	MaxSubjects int     `json:"max_subjects"`
	Status      string  `json:"status"`
}

// Permit represents external authorizations needed for compliance.
type Permit struct {
	Base
	PermitNumber      string       `json:"permit_number"`
	Authority         string       `json:"authority"`
	Status            PermitStatus `json:"status"`
	ValidFrom         time.Time    `json:"valid_from"`
	ValidUntil        time.Time    `json:"valid_until"`
	AllowedActivities []string     `json:"allowed_activities"`
	FacilityIDs       []string     `json:"facility_ids"`
	ProtocolIDs       []string     `json:"protocol_ids"`
	Notes             *string      `json:"notes,omitempty"`
}

// Project captures cost center allocations.
type Project struct {
	Base
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	Description   *string  `json:"description,omitempty"`
	FacilityIDs   []string `json:"facility_ids"`
	ProtocolIDs   []string `json:"protocol_ids"`
	OrganismIDs   []string `json:"organism_ids"`
	ProcedureIDs  []string `json:"procedure_ids"`
	SupplyItemIDs []string `json:"supply_item_ids"`
}

// SupplyItem models inventory resources consumed by projects or facilities.
type SupplyItem struct {
	Base
	SKU            string               `json:"sku"`
	Name           string               `json:"name"`
	Description    *string              `json:"description,omitempty"`
	QuantityOnHand int                  `json:"quantity_on_hand"`
	Unit           string               `json:"unit"`
	LotNumber      *string              `json:"lot_number,omitempty"`
	ExpiresAt      *time.Time           `json:"expires_at"`
	FacilityIDs    []string             `json:"facility_ids"`
	ProjectIDs     []string             `json:"project_ids"`
	ReorderLevel   int                  `json:"reorder_level"`
	attributesSlot *extension.Slot      `json:"-"`
	extensions     *extension.Container `json:"-"`
}

type organismAlias Organism

// MarshalJSON ensures organism attributes are serialised via the core plugin payload.
func (o Organism) MarshalJSON() ([]byte, error) {
	type payload struct {
		organismAlias
		Attributes map[string]any `json:"attributes,omitempty"`
	}
	return json.Marshal(payload{
		organismAlias: organismAlias(o),
		Attributes:    o.AttributesMap(),
	})
}

// UnmarshalJSON hydrates organism extension slots from the JSON payload.
func (o *Organism) UnmarshalJSON(data []byte) error {
	type payload struct {
		organismAlias
		Attributes map[string]any `json:"attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*o = Organism(aux.organismAlias)
	o.SetAttributes(aux.Attributes)
	return nil
}

type facilityAlias Facility

// MarshalJSON ensures facility environment baselines are serialised via the core plugin payload.
func (f Facility) MarshalJSON() ([]byte, error) {
	type payload struct {
		facilityAlias
		EnvironmentBaselines map[string]any `json:"environment_baselines,omitempty"`
	}
	return json.Marshal(payload{
		facilityAlias:        facilityAlias(f),
		EnvironmentBaselines: f.EnvironmentBaselinesMap(),
	})
}

// UnmarshalJSON hydrates facility extension slots from the JSON payload.
func (f *Facility) UnmarshalJSON(data []byte) error {
	type payload struct {
		facilityAlias
		EnvironmentBaselines map[string]any `json:"environment_baselines"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*f = Facility(aux.facilityAlias)
	f.SetEnvironmentBaselines(aux.EnvironmentBaselines)
	return nil
}

type breedingUnitAlias BreedingUnit

// MarshalJSON ensures breeding unit pairing attributes are serialised via the core plugin payload.
func (b BreedingUnit) MarshalJSON() ([]byte, error) {
	type payload struct {
		breedingUnitAlias
		PairingAttributes map[string]any `json:"pairing_attributes,omitempty"`
	}
	return json.Marshal(payload{
		breedingUnitAlias: breedingUnitAlias(b),
		PairingAttributes: b.PairingAttributesMap(),
	})
}

// UnmarshalJSON hydrates breeding unit extension slots from the JSON payload.
func (b *BreedingUnit) UnmarshalJSON(data []byte) error {
	type payload struct {
		breedingUnitAlias
		PairingAttributes map[string]any `json:"pairing_attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*b = BreedingUnit(aux.breedingUnitAlias)
	b.SetPairingAttributes(aux.PairingAttributes)
	return nil
}

type observationAlias Observation

// MarshalJSON ensures observation data are serialised via the core plugin payload.
func (o Observation) MarshalJSON() ([]byte, error) {
	type payload struct {
		observationAlias
		Data map[string]any `json:"data,omitempty"`
	}
	return json.Marshal(payload{
		observationAlias: observationAlias(o),
		Data:             o.DataMap(),
	})
}

// UnmarshalJSON hydrates observation extension slots from the JSON payload.
func (o *Observation) UnmarshalJSON(data []byte) error {
	type payload struct {
		observationAlias
		Data map[string]any `json:"data"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*o = Observation(aux.observationAlias)
	o.SetData(aux.Data)
	return nil
}

type sampleAlias Sample

// MarshalJSON ensures sample attributes are serialised via the core plugin payload.
func (s Sample) MarshalJSON() ([]byte, error) {
	type payload struct {
		sampleAlias
		Attributes map[string]any `json:"attributes,omitempty"`
	}
	return json.Marshal(payload{
		sampleAlias: sampleAlias(s),
		Attributes:  s.AttributesMap(),
	})
}

// UnmarshalJSON hydrates sample extension slots from the JSON payload.
func (s *Sample) UnmarshalJSON(data []byte) error {
	type payload struct {
		sampleAlias
		Attributes map[string]any `json:"attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = Sample(aux.sampleAlias)
	s.SetAttributes(aux.Attributes)
	return nil
}

type supplyAlias SupplyItem

// MarshalJSON ensures supply item attributes are serialised via the core plugin payload.
func (s SupplyItem) MarshalJSON() ([]byte, error) {
	type payload struct {
		supplyAlias
		Attributes map[string]any `json:"attributes,omitempty"`
	}
	return json.Marshal(payload{
		supplyAlias: supplyAlias(s),
		Attributes:  s.AttributesMap(),
	})
}

// UnmarshalJSON hydrates supply item extension slots from the JSON payload.
func (s *SupplyItem) UnmarshalJSON(data []byte) error {
	type payload struct {
		supplyAlias
		Attributes map[string]any `json:"attributes"`
	}
	var aux payload
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = SupplyItem(aux.supplyAlias)
	s.SetAttributes(aux.Attributes)
	return nil
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
