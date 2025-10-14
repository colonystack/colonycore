// Package datasetapi provides a stable facade for dataset plugins exposing
// read-only store contracts and lightweight entity projections decoupled from
// the internal domain model.
package datasetapi

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	// Procedure status constants
	statusInProgress = "in_progress"
	statusCompleted  = "completed"
	statusCancelled  = "cancelled"
)

// TransactionView offers read-only access to a snapshot of core entities for
// dataset and rule execution within an ambient transaction.
type TransactionView interface {
	ListOrganisms() []Organism
	ListHousingUnits() []HousingUnit
	ListFacilities() []Facility
	ListTreatments() []Treatment
	ListObservations() []Observation
	ListSamples() []Sample
	ListProtocols() []Protocol
	ListPermits() []Permit
	ListProjects() []Project
	ListSupplyItems() []SupplyItem
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
	FindFacility(id string) (Facility, bool)
	FindTreatment(id string) (Treatment, bool)
	FindObservation(id string) (Observation, bool)
	FindSample(id string) (Sample, bool)
	FindPermit(id string) (Permit, bool)
	FindSupplyItem(id string) (SupplyItem, bool)
}

// PersistentStore exposes read-only query helpers used by dataset binders.
type PersistentStore interface {
	View(ctx context.Context, fn func(TransactionView) error) error
	GetOrganism(id string) (Organism, bool)
	ListOrganisms() []Organism
	GetHousingUnit(id string) (HousingUnit, bool)
	ListHousingUnits() []HousingUnit
	GetFacility(id string) (Facility, bool)
	ListFacilities() []Facility
	ListCohorts() []Cohort
	ListTreatments() []Treatment
	ListObservations() []Observation
	ListSamples() []Sample
	ListProtocols() []Protocol
	GetPermit(id string) (Permit, bool)
	ListPermits() []Permit
	ListProjects() []Project
	ListBreedingUnits() []BreedingUnit
	ListProcedures() []Procedure
	ListSupplyItems() []SupplyItem
}

// BaseData captures shared entity metadata for constructing facade objects.
type BaseData struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LifecycleStage represents canonical organism lifecycle identifiers.
type LifecycleStage string

// Canonical lifecycle stage constants - now internal, accessed via contextual interfaces.
const (
	stagePlanned  LifecycleStage = "planned"
	stageLarva    LifecycleStage = "embryo_larva"
	stageJuvenile LifecycleStage = "juvenile"
	stageAdult    LifecycleStage = "adult"
	stageRetired  LifecycleStage = "retired"
	stageDeceased LifecycleStage = "deceased"
)

// OrganismData describes the fields required to construct an Organism facade.
type OrganismData struct {
	Base       BaseData
	Name       string
	Species    string
	Line       string
	Stage      LifecycleStage
	CohortID   *string
	HousingID  *string
	ProtocolID *string
	ProjectID  *string
	Attributes map[string]any
}

// CohortData describes the fields required to construct a Cohort facade.
type CohortData struct {
	Base       BaseData
	Name       string
	Purpose    string
	ProjectID  *string
	HousingID  *string
	ProtocolID *string
}

// HousingUnitData describes the fields required to construct a HousingUnit facade.
type HousingUnitData struct {
	Base        BaseData
	Name        string
	FacilityID  string
	Capacity    int
	Environment string
}

// FacilityData describes the fields required to construct a Facility facade.
type FacilityData struct {
	Base                 BaseData
	Name                 string
	Zone                 string
	AccessPolicy         string
	EnvironmentBaselines map[string]any
	HousingUnitIDs       []string
	ProjectIDs           []string
}

// BreedingUnitData describes the fields required to construct a BreedingUnit facade.
type BreedingUnitData struct {
	Base       BaseData
	Name       string
	Strategy   string
	HousingID  *string
	ProtocolID *string
	FemaleIDs  []string
	MaleIDs    []string
}

// ProcedureData describes the fields required to construct a Procedure facade.
type ProcedureData struct {
	Base        BaseData
	Name        string
	Status      string
	ScheduledAt time.Time
	ProtocolID  string
	CohortID    *string
	OrganismIDs []string
}

// TreatmentData describes the fields required to construct a Treatment facade.
type TreatmentData struct {
	Base              BaseData
	Name              string
	ProcedureID       string
	OrganismIDs       []string
	CohortIDs         []string
	DosagePlan        string
	AdministrationLog []string
	AdverseEvents     []string
}

// ObservationData describes the fields required to construct an Observation facade.
type ObservationData struct {
	Base        BaseData
	ProcedureID *string
	OrganismID  *string
	CohortID    *string
	RecordedAt  time.Time
	Observer    string
	Data        map[string]any
	Notes       string
}

// SampleData describes the fields required to construct a Sample facade.
type SampleData struct {
	Base            BaseData
	Identifier      string
	SourceType      string
	OrganismID      *string
	CohortID        *string
	FacilityID      string
	CollectedAt     time.Time
	Status          string
	StorageLocation string
	AssayType       string
	ChainOfCustody  []SampleCustodyEventData
	Attributes      map[string]any
}

// SampleCustodyEventData represents an entry in a sample custody chain.
type SampleCustodyEventData struct {
	Actor     string
	Location  string
	Timestamp time.Time
	Notes     string
}

// ProtocolData describes the fields required to construct a Protocol facade.
type ProtocolData struct {
	Base        BaseData
	Code        string
	Title       string
	Description string
	MaxSubjects int
	Status      string
}

// PermitData describes the fields required to construct a Permit facade.
type PermitData struct {
	Base              BaseData
	PermitNumber      string
	Authority         string
	ValidFrom         time.Time
	ValidUntil        time.Time
	AllowedActivities []string
	FacilityIDs       []string
	ProtocolIDs       []string
	Notes             string
}

// ProjectData describes the fields required to construct a Project facade.
type ProjectData struct {
	Base        BaseData
	Code        string
	Title       string
	Description string
	FacilityIDs []string
}

// SupplyItemData describes the fields required to construct a SupplyItem facade.
type SupplyItemData struct {
	Base           BaseData
	SKU            string
	Name           string
	Description    string
	QuantityOnHand int
	Unit           string
	LotNumber      string
	ExpiresAt      *time.Time
	FacilityIDs    []string
	ProjectIDs     []string
	ReorderLevel   int
	Attributes     map[string]any
}

// Organism exposes read-only organism metadata to dataset plugins.
type Organism interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Species() string
	Line() string
	Stage() LifecycleStage // Legacy - prefer GetCurrentStage() for new code
	CohortID() (string, bool)
	HousingID() (string, bool)
	ProtocolID() (string, bool)
	ProjectID() (string, bool)
	Attributes() map[string]any

	// Contextual lifecycle stage accessors
	GetCurrentStage() LifecycleStageRef
	IsActive() bool
	IsRetired() bool
	IsDeceased() bool
}

// Cohort exposes read-only cohort metadata to dataset plugins.
type Cohort interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Purpose() string
	ProjectID() (string, bool)
	HousingID() (string, bool)
	ProtocolID() (string, bool)

	// Contextual purpose accessors
	GetPurpose() CohortPurposeRef
	IsResearchCohort() bool
	RequiresProtocol() bool
}

// HousingUnit exposes read-only housing metadata to dataset plugins.
type HousingUnit interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	FacilityID() string
	Capacity() int
	Environment() string

	// Contextual environment accessors
	GetEnvironmentType() EnvironmentTypeRef
	IsAquaticEnvironment() bool
	IsHumidEnvironment() bool
	SupportsSpecies(species string) bool
}

// Facility exposes read-only facility metadata to dataset plugins.
type Facility interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Zone() string
	AccessPolicy() string
	EnvironmentBaselines() map[string]any
	HousingUnitIDs() []string
	ProjectIDs() []string

	// Contextual zone & access policy accessors
	GetZone() FacilityZoneRef
	GetAccessPolicy() FacilityAccessPolicyRef
	SupportsHousingUnit(id string) bool
}

// BreedingUnit exposes read-only breeding metadata to dataset plugins.
type BreedingUnit interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Strategy() string
	HousingID() (string, bool)
	ProtocolID() (string, bool)
	FemaleIDs() []string
	MaleIDs() []string

	// Contextual strategy accessors
	GetBreedingStrategy() BreedingStrategyRef
	IsNaturalBreeding() bool
	RequiresIntervention() bool
}

// Procedure exposes read-only procedure metadata to dataset plugins.
type Procedure interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Status() string
	ScheduledAt() time.Time
	ProtocolID() string
	CohortID() (string, bool)
	OrganismIDs() []string

	// Contextual status accessors
	GetCurrentStatus() ProcedureStatusRef
	IsActiveProcedure() bool
	IsTerminalStatus() bool
	IsSuccessful() bool
}

// Treatment exposes read-only treatment metadata to dataset plugins.
type Treatment interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	ProcedureID() string
	OrganismIDs() []string
	CohortIDs() []string
	DosagePlan() string
	AdministrationLog() []string
	AdverseEvents() []string

	// Contextual workflow accessors
	GetCurrentStatus() TreatmentStatusRef
	IsCompleted() bool
	HasAdverseEvents() bool
}

// Observation exposes read-only observation metadata to dataset plugins.
type Observation interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	ProcedureID() (string, bool)
	OrganismID() (string, bool)
	CohortID() (string, bool)
	RecordedAt() time.Time
	Observer() string
	Data() map[string]any
	Notes() string

	// Contextual data shape accessors
	GetDataShape() ObservationShapeRef
	HasStructuredPayload() bool
	HasNarrativeNotes() bool
}

// Sample exposes read-only sample metadata to dataset plugins.
type Sample interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Identifier() string
	SourceType() string
	OrganismID() (string, bool)
	CohortID() (string, bool)
	FacilityID() string
	CollectedAt() time.Time
	Status() string
	StorageLocation() string
	AssayType() string
	ChainOfCustody() []SampleCustodyEvent
	Attributes() map[string]any

	// Contextual sample accessors
	GetSource() SampleSourceRef
	GetStatus() SampleStatusRef
	IsAvailable() bool
}

// SampleCustodyEvent represents an immutable custody transition.
type SampleCustodyEvent interface {
	Actor() string
	Location() string
	Timestamp() time.Time
	Notes() string
}

// Protocol exposes read-only protocol metadata to dataset plugins.
type Protocol interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Code() string
	Title() string
	Description() string
	MaxSubjects() int

	// Contextual status accessors
	GetCurrentStatus() ProtocolStatusRef
	IsActiveProtocol() bool
	IsTerminalStatus() bool
	CanAcceptNewSubjects() bool
}

// Permit exposes read-only permit metadata to dataset plugins.
type Permit interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	PermitNumber() string
	Authority() string
	ValidFrom() time.Time
	ValidUntil() time.Time
	AllowedActivities() []string
	FacilityIDs() []string
	ProtocolIDs() []string
	Notes() string

	// Contextual validity accessors
	GetStatus(reference time.Time) PermitStatusRef
	IsActive(reference time.Time) bool
	IsExpired(reference time.Time) bool
}

// Project exposes read-only project metadata to dataset plugins.
type Project interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Code() string
	Title() string
	Description() string
	FacilityIDs() []string
}

// SupplyItem exposes read-only supply metadata to dataset plugins.
type SupplyItem interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	SKU() string
	Name() string
	Description() string
	QuantityOnHand() int
	Unit() string
	LotNumber() string
	ExpiresAt() (*time.Time, bool)
	FacilityIDs() []string
	ProjectIDs() []string
	ReorderLevel() int
	Attributes() map[string]any

	// Contextual inventory accessors
	GetInventoryStatus(reference time.Time) SupplyStatusRef
	RequiresReorder(reference time.Time) bool
	IsExpired(reference time.Time) bool
}

type base struct {
	id        string
	createdAt time.Time
	updatedAt time.Time
}

func newBase(data BaseData) base {
	return base{
		id:        data.ID,
		createdAt: data.CreatedAt,
		updatedAt: data.UpdatedAt,
	}
}

func (b base) ID() string           { return b.id }
func (b base) CreatedAt() time.Time { return b.createdAt }
func (b base) UpdatedAt() time.Time { return b.updatedAt }

type organism struct {
	base
	name       string
	species    string
	line       string
	stage      LifecycleStage
	cohortID   *string
	housingID  *string
	protocolID *string
	projectID  *string
	attributes map[string]any
}

// NewOrganism constructs a read-only Organism facade.
func NewOrganism(data OrganismData) Organism {
	return organism{
		base:       newBase(data.Base),
		name:       data.Name,
		species:    data.Species,
		line:       data.Line,
		stage:      data.Stage,
		cohortID:   cloneOptionalString(data.CohortID),
		housingID:  cloneOptionalString(data.HousingID),
		protocolID: cloneOptionalString(data.ProtocolID),
		projectID:  cloneOptionalString(data.ProjectID),
		attributes: cloneAttributes(data.Attributes),
	}
}

func (o organism) Name() string    { return o.name }
func (o organism) Species() string { return o.species }
func (o organism) Line() string    { return o.line }
func (o organism) Stage() LifecycleStage {
	return o.stage
}
func (o organism) CohortID() (string, bool) {
	return derefString(o.cohortID)
}
func (o organism) HousingID() (string, bool) {
	return derefString(o.housingID)
}
func (o organism) ProtocolID() (string, bool) {
	return derefString(o.protocolID)
}
func (o organism) ProjectID() (string, bool) {
	return derefString(o.projectID)
}
func (o organism) Attributes() map[string]any {
	return cloneAttributes(o.attributes)
}

// Contextual lifecycle stage accessors
func (o organism) GetCurrentStage() LifecycleStageRef {
	stages := NewLifecycleStageContext()
	switch o.stage {
	case stagePlanned:
		return stages.Planned()
	case stageLarva:
		return stages.Larva()
	case stageJuvenile:
		return stages.Juvenile()
	case stageAdult:
		return stages.Adult()
	case stageRetired:
		return stages.Retired()
	case stageDeceased:
		return stages.Deceased()
	default:
		// Fallback for unknown stages - should not happen in normal operation
		return stages.Adult()
	}
}

func (o organism) IsActive() bool {
	return o.GetCurrentStage().IsActive()
}

func (o organism) IsRetired() bool {
	return o.stage == stageRetired
}

func (o organism) IsDeceased() bool {
	return o.stage == stageDeceased
}

func (o organism) MarshalJSON() ([]byte, error) {
	type organismJSON struct {
		ID         string         `json:"id"`
		CreatedAt  time.Time      `json:"created_at"`
		UpdatedAt  time.Time      `json:"updated_at"`
		Name       string         `json:"name"`
		Species    string         `json:"species"`
		Line       string         `json:"line"`
		Stage      LifecycleStage `json:"stage"`
		CohortID   *string        `json:"cohort_id,omitempty"`
		HousingID  *string        `json:"housing_id,omitempty"`
		ProtocolID *string        `json:"protocol_id,omitempty"`
		ProjectID  *string        `json:"project_id,omitempty"`
		Attributes map[string]any `json:"attributes,omitempty"`
	}
	return json.Marshal(organismJSON{
		ID:         o.ID(),
		CreatedAt:  o.CreatedAt(),
		UpdatedAt:  o.UpdatedAt(),
		Name:       o.name,
		Species:    o.species,
		Line:       o.line,
		Stage:      o.stage,
		CohortID:   cloneOptionalString(o.cohortID),
		HousingID:  cloneOptionalString(o.housingID),
		ProtocolID: cloneOptionalString(o.protocolID),
		ProjectID:  cloneOptionalString(o.projectID),
		Attributes: cloneAttributes(o.attributes),
	})
}

type cohort struct {
	base
	name       string
	purpose    string
	projectID  *string
	housingID  *string
	protocolID *string
}

// NewCohort constructs a read-only Cohort facade.
func NewCohort(data CohortData) Cohort {
	return cohort{
		base:       newBase(data.Base),
		name:       data.Name,
		purpose:    data.Purpose,
		projectID:  cloneOptionalString(data.ProjectID),
		housingID:  cloneOptionalString(data.HousingID),
		protocolID: cloneOptionalString(data.ProtocolID),
	}
}

func (c cohort) Name() string    { return c.name }
func (c cohort) Purpose() string { return c.purpose }
func (c cohort) ProjectID() (string, bool) {
	return derefString(c.projectID)
}
func (c cohort) HousingID() (string, bool) {
	return derefString(c.housingID)
}
func (c cohort) ProtocolID() (string, bool) {
	return derefString(c.protocolID)
}

// Contextual purpose accessors
func (c cohort) GetPurpose() CohortPurposeRef {
	ctx := NewCohortContext()
	switch strings.ToLower(c.purpose) {
	case "research":
		return ctx.Research()
	case "breeding":
		return ctx.Breeding()
	case "teaching":
		return ctx.Teaching()
	case "conservation":
		return ctx.Conservation()
	case "production":
		return ctx.Production()
	default:
		// Default to research for unknown purposes
		return ctx.Research()
	}
}

func (c cohort) IsResearchCohort() bool {
	return c.GetPurpose().IsResearch()
}

func (c cohort) RequiresProtocol() bool {
	return c.GetPurpose().RequiresProtocol()
}

func (c cohort) MarshalJSON() ([]byte, error) {
	type cohortJSON struct {
		ID         string    `json:"id"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
		Name       string    `json:"name"`
		Purpose    string    `json:"purpose"`
		ProjectID  *string   `json:"project_id,omitempty"`
		HousingID  *string   `json:"housing_id,omitempty"`
		ProtocolID *string   `json:"protocol_id,omitempty"`
	}
	return json.Marshal(cohortJSON{
		ID:         c.ID(),
		CreatedAt:  c.CreatedAt(),
		UpdatedAt:  c.UpdatedAt(),
		Name:       c.name,
		Purpose:    c.purpose,
		ProjectID:  cloneOptionalString(c.projectID),
		HousingID:  cloneOptionalString(c.housingID),
		ProtocolID: cloneOptionalString(c.protocolID),
	})
}

type housingUnit struct {
	base
	name        string
	facilityID  string
	capacity    int
	environment string
}

// NewHousingUnit constructs a read-only HousingUnit facade.
func NewHousingUnit(data HousingUnitData) HousingUnit {
	return housingUnit{
		base:        newBase(data.Base),
		name:        data.Name,
		facilityID:  data.FacilityID,
		capacity:    data.Capacity,
		environment: data.Environment,
	}
}

func (h housingUnit) Name() string        { return h.name }
func (h housingUnit) FacilityID() string  { return h.facilityID }
func (h housingUnit) Capacity() int       { return h.capacity }
func (h housingUnit) Environment() string { return h.environment }

// Contextual environment accessors
func (h housingUnit) GetEnvironmentType() EnvironmentTypeRef {
	ctx := NewHousingContext()
	switch strings.ToLower(h.environment) {
	case "aquatic":
		return ctx.Aquatic()
	case "terrestrial":
		return ctx.Terrestrial()
	case "arboreal":
		return ctx.Arboreal()
	case "humid":
		return ctx.Humid()
	default:
		// Default to terrestrial for unknown environments
		return ctx.Terrestrial()
	}
}

func (h housingUnit) IsAquaticEnvironment() bool {
	return h.GetEnvironmentType().IsAquatic()
}

func (h housingUnit) IsHumidEnvironment() bool {
	return h.GetEnvironmentType().IsHumid()
}

func (h housingUnit) SupportsSpecies(species string) bool {
	speciesLower := strings.ToLower(species)
	envType := h.GetEnvironmentType()

	// Basic species-environment compatibility logic
	if strings.Contains(speciesLower, "frog") || strings.Contains(speciesLower, "amphibian") {
		return envType.IsAquatic() || envType.IsHumid()
	}

	if strings.Contains(speciesLower, "fish") {
		return envType.IsAquatic()
	}

	// Default: terrestrial animals can live in terrestrial environments
	return !envType.IsAquatic() || envType.String() == "terrestrial"
}

func (h housingUnit) MarshalJSON() ([]byte, error) {
	type housingJSON struct {
		ID          string    `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Name        string    `json:"name"`
		FacilityID  string    `json:"facility_id"`
		Capacity    int       `json:"capacity"`
		Environment string    `json:"environment"`
	}
	return json.Marshal(housingJSON{
		ID:          h.ID(),
		CreatedAt:   h.CreatedAt(),
		UpdatedAt:   h.UpdatedAt(),
		Name:        h.name,
		FacilityID:  h.facilityID,
		Capacity:    h.capacity,
		Environment: h.environment,
	})
}

type facility struct {
	base
	name                 string
	zone                 string
	accessPolicy         string
	environmentBaselines map[string]any
	housingUnitIDs       []string
	projectIDs           []string
}

// NewFacility constructs a read-only Facility facade.
func NewFacility(data FacilityData) Facility {
	return facility{
		base:                 newBase(data.Base),
		name:                 data.Name,
		zone:                 data.Zone,
		accessPolicy:         data.AccessPolicy,
		environmentBaselines: cloneAttributes(data.EnvironmentBaselines),
		housingUnitIDs:       cloneStringSlice(data.HousingUnitIDs),
		projectIDs:           cloneStringSlice(data.ProjectIDs),
	}
}

func (f facility) Name() string         { return f.name }
func (f facility) Zone() string         { return f.zone }
func (f facility) AccessPolicy() string { return f.accessPolicy }
func (f facility) EnvironmentBaselines() map[string]any {
	return cloneAttributes(f.environmentBaselines)
}
func (f facility) HousingUnitIDs() []string { return cloneStringSlice(f.housingUnitIDs) }
func (f facility) ProjectIDs() []string     { return cloneStringSlice(f.projectIDs) }

// Contextual zone & access policy accessors
func (f facility) GetZone() FacilityZoneRef {
	ctx := NewFacilityContext().Zones()
	zone := strings.ToLower(strings.TrimSpace(f.zone))
	switch {
	case zone == "":
		return ctx.General()
	case strings.Contains(zone, "bio") || strings.Contains(zone, "bsl"):
		return ctx.Biosecure()
	case strings.Contains(zone, "quarantine") || strings.Contains(zone, "isolation"):
		return ctx.Quarantine()
	default:
		return ctx.General()
	}
}

func (f facility) GetAccessPolicy() FacilityAccessPolicyRef {
	ctx := NewFacilityContext().AccessPolicies()
	policy := strings.ToLower(strings.TrimSpace(f.accessPolicy))
	switch {
	case policy == "":
		return ctx.Open()
	case strings.Contains(policy, "restricted") || strings.Contains(policy, "secure"):
		return ctx.Restricted()
	case strings.Contains(policy, "staff"):
		return ctx.StaffOnly()
	default:
		return ctx.Open()
	}
}

func (f facility) SupportsHousingUnit(id string) bool {
	for _, housingID := range f.housingUnitIDs {
		if housingID == id {
			return true
		}
	}
	return false
}

func (f facility) MarshalJSON() ([]byte, error) {
	type facilityJSON struct {
		ID                   string         `json:"id"`
		CreatedAt            time.Time      `json:"created_at"`
		UpdatedAt            time.Time      `json:"updated_at"`
		Name                 string         `json:"name"`
		Zone                 string         `json:"zone"`
		AccessPolicy         string         `json:"access_policy"`
		EnvironmentBaselines map[string]any `json:"environment_baselines,omitempty"`
		HousingUnitIDs       []string       `json:"housing_unit_ids,omitempty"`
		ProjectIDs           []string       `json:"project_ids,omitempty"`
	}
	return json.Marshal(facilityJSON{
		ID:                   f.ID(),
		CreatedAt:            f.CreatedAt(),
		UpdatedAt:            f.UpdatedAt(),
		Name:                 f.name,
		Zone:                 f.zone,
		AccessPolicy:         f.accessPolicy,
		EnvironmentBaselines: cloneAttributes(f.environmentBaselines),
		HousingUnitIDs:       cloneStringSlice(f.housingUnitIDs),
		ProjectIDs:           cloneStringSlice(f.projectIDs),
	})
}

type breedingUnit struct {
	base
	name       string
	strategy   string
	housingID  *string
	protocolID *string
	femaleIDs  []string
	maleIDs    []string
}

// NewBreedingUnit constructs a read-only BreedingUnit facade.
func NewBreedingUnit(data BreedingUnitData) BreedingUnit {
	return breedingUnit{
		base:       newBase(data.Base),
		name:       data.Name,
		strategy:   data.Strategy,
		housingID:  cloneOptionalString(data.HousingID),
		protocolID: cloneOptionalString(data.ProtocolID),
		femaleIDs:  cloneStringSlice(data.FemaleIDs),
		maleIDs:    cloneStringSlice(data.MaleIDs),
	}
}

func (b breedingUnit) Name() string     { return b.name }
func (b breedingUnit) Strategy() string { return b.strategy }
func (b breedingUnit) HousingID() (string, bool) {
	return derefString(b.housingID)
}
func (b breedingUnit) ProtocolID() (string, bool) {
	return derefString(b.protocolID)
}
func (b breedingUnit) FemaleIDs() []string {
	return cloneStringSlice(b.femaleIDs)
}
func (b breedingUnit) MaleIDs() []string {
	return cloneStringSlice(b.maleIDs)
}

// Contextual strategy accessors
func (b breedingUnit) GetBreedingStrategy() BreedingStrategyRef {
	ctx := NewBreedingContext()
	switch strings.ToLower(b.strategy) {
	case "natural":
		return ctx.Natural()
	case "artificial":
		return ctx.Artificial()
	case "controlled":
		return ctx.Controlled()
	case "selective":
		return ctx.Selective()
	default:
		// Default to natural for unknown strategies
		return ctx.Natural()
	}
}

func (b breedingUnit) IsNaturalBreeding() bool {
	return b.GetBreedingStrategy().IsNatural()
}

func (b breedingUnit) RequiresIntervention() bool {
	return b.GetBreedingStrategy().RequiresIntervention()
}

func (b breedingUnit) MarshalJSON() ([]byte, error) {
	type breedingJSON struct {
		ID         string    `json:"id"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
		Name       string    `json:"name"`
		Strategy   string    `json:"strategy"`
		HousingID  *string   `json:"housing_id,omitempty"`
		ProtocolID *string   `json:"protocol_id,omitempty"`
		FemaleIDs  []string  `json:"female_ids"`
		MaleIDs    []string  `json:"male_ids"`
	}
	return json.Marshal(breedingJSON{
		ID:         b.ID(),
		CreatedAt:  b.CreatedAt(),
		UpdatedAt:  b.UpdatedAt(),
		Name:       b.name,
		Strategy:   b.strategy,
		HousingID:  cloneOptionalString(b.housingID),
		ProtocolID: cloneOptionalString(b.protocolID),
		FemaleIDs:  cloneStringSlice(b.femaleIDs),
		MaleIDs:    cloneStringSlice(b.maleIDs),
	})
}

type procedure struct {
	base
	name        string
	status      string
	scheduledAt time.Time
	protocolID  string
	cohortID    *string
	organismIDs []string
}

// NewProcedure constructs a read-only Procedure facade.
func NewProcedure(data ProcedureData) Procedure {
	return procedure{
		base:        newBase(data.Base),
		name:        data.Name,
		status:      data.Status,
		scheduledAt: data.ScheduledAt,
		protocolID:  data.ProtocolID,
		cohortID:    cloneOptionalString(data.CohortID),
		organismIDs: cloneStringSlice(data.OrganismIDs),
	}
}

func (p procedure) Name() string           { return p.name }
func (p procedure) Status() string         { return p.status }
func (p procedure) ScheduledAt() time.Time { return p.scheduledAt }
func (p procedure) ProtocolID() string     { return p.protocolID }
func (p procedure) CohortID() (string, bool) {
	return derefString(p.cohortID)
}
func (p procedure) OrganismIDs() []string {
	return cloneStringSlice(p.organismIDs)
}

// Contextual status accessors
func (p procedure) GetCurrentStatus() ProcedureStatusRef {
	ctx := NewProcedureContext()
	switch strings.ToLower(p.status) {
	case "scheduled":
		return ctx.Scheduled()
	case statusInProgress, "inprogress", "running":
		return ctx.InProgress()
	case statusCompleted, "done":
		return ctx.Completed()
	case statusCancelled:
		return ctx.Cancelled()
	case "failed", "error":
		return ctx.Failed()
	default:
		// Default to scheduled for unknown statuses
		return ctx.Scheduled()
	}
}

func (p procedure) IsActiveProcedure() bool {
	return p.GetCurrentStatus().IsActive()
}

func (p procedure) IsTerminalStatus() bool {
	return p.GetCurrentStatus().IsTerminal()
}

func (p procedure) IsSuccessful() bool {
	return p.GetCurrentStatus().IsSuccessful()
}

func (p procedure) MarshalJSON() ([]byte, error) {
	type procedureJSON struct {
		ID          string    `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Name        string    `json:"name"`
		Status      string    `json:"status"`
		ScheduledAt time.Time `json:"scheduled_at"`
		ProtocolID  string    `json:"protocol_id"`
		CohortID    *string   `json:"cohort_id,omitempty"`
		OrganismIDs []string  `json:"organism_ids"`
	}
	return json.Marshal(procedureJSON{
		ID:          p.ID(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
		Name:        p.name,
		Status:      p.status,
		ScheduledAt: p.scheduledAt,
		ProtocolID:  p.protocolID,
		CohortID:    cloneOptionalString(p.cohortID),
		OrganismIDs: cloneStringSlice(p.organismIDs),
	})
}

type treatment struct {
	base
	name              string
	procedureID       string
	organismIDs       []string
	cohortIDs         []string
	dosagePlan        string
	administrationLog []string
	adverseEvents     []string
}

// NewTreatment constructs a read-only Treatment facade.
func NewTreatment(data TreatmentData) Treatment {
	return treatment{
		base:              newBase(data.Base),
		name:              data.Name,
		procedureID:       data.ProcedureID,
		organismIDs:       cloneStringSlice(data.OrganismIDs),
		cohortIDs:         cloneStringSlice(data.CohortIDs),
		dosagePlan:        data.DosagePlan,
		administrationLog: cloneStringSlice(data.AdministrationLog),
		adverseEvents:     cloneStringSlice(data.AdverseEvents),
	}
}

func (t treatment) Name() string          { return t.name }
func (t treatment) ProcedureID() string   { return t.procedureID }
func (t treatment) OrganismIDs() []string { return cloneStringSlice(t.organismIDs) }
func (t treatment) CohortIDs() []string   { return cloneStringSlice(t.cohortIDs) }
func (t treatment) DosagePlan() string    { return t.dosagePlan }
func (t treatment) AdministrationLog() []string {
	return cloneStringSlice(t.administrationLog)
}
func (t treatment) AdverseEvents() []string {
	return cloneStringSlice(t.adverseEvents)
}

// Contextual workflow accessors
func (t treatment) GetCurrentStatus() TreatmentStatusRef {
	return deriveTreatmentStatus(t.administrationLog, t.adverseEvents)
}

func (t treatment) IsCompleted() bool {
	status := t.GetCurrentStatus()
	return status.IsCompleted() || status.IsFlagged()
}

func (t treatment) HasAdverseEvents() bool {
	return len(t.adverseEvents) > 0
}

func (t treatment) MarshalJSON() ([]byte, error) {
	type treatmentJSON struct {
		ID                string    `json:"id"`
		CreatedAt         time.Time `json:"created_at"`
		UpdatedAt         time.Time `json:"updated_at"`
		Name              string    `json:"name"`
		ProcedureID       string    `json:"procedure_id"`
		OrganismIDs       []string  `json:"organism_ids,omitempty"`
		CohortIDs         []string  `json:"cohort_ids,omitempty"`
		DosagePlan        string    `json:"dosage_plan"`
		AdministrationLog []string  `json:"administration_log,omitempty"`
		AdverseEvents     []string  `json:"adverse_events,omitempty"`
	}
	return json.Marshal(treatmentJSON{
		ID:                t.ID(),
		CreatedAt:         t.CreatedAt(),
		UpdatedAt:         t.UpdatedAt(),
		Name:              t.name,
		ProcedureID:       t.procedureID,
		OrganismIDs:       cloneStringSlice(t.organismIDs),
		CohortIDs:         cloneStringSlice(t.cohortIDs),
		DosagePlan:        t.dosagePlan,
		AdministrationLog: cloneStringSlice(t.administrationLog),
		AdverseEvents:     cloneStringSlice(t.adverseEvents),
	})
}

type observation struct {
	base
	procedureID *string
	organismID  *string
	cohortID    *string
	recordedAt  time.Time
	observer    string
	data        map[string]any
	notes       string
}

// NewObservation constructs a read-only Observation facade.
func NewObservation(data ObservationData) Observation {
	return observation{
		base:        newBase(data.Base),
		procedureID: cloneOptionalString(data.ProcedureID),
		organismID:  cloneOptionalString(data.OrganismID),
		cohortID:    cloneOptionalString(data.CohortID),
		recordedAt:  data.RecordedAt,
		observer:    data.Observer,
		data:        cloneAttributes(data.Data),
		notes:       data.Notes,
	}
}

func (o observation) ProcedureID() (string, bool) {
	return derefString(o.procedureID)
}

func (o observation) OrganismID() (string, bool) {
	return derefString(o.organismID)
}

func (o observation) CohortID() (string, bool) {
	return derefString(o.cohortID)
}

func (o observation) RecordedAt() time.Time { return o.recordedAt }
func (o observation) Observer() string      { return o.observer }
func (o observation) Data() map[string]any  { return cloneAttributes(o.data) }
func (o observation) Notes() string         { return o.notes }

// Contextual data shape accessors
func (o observation) GetDataShape() ObservationShapeRef {
	return inferObservationShape(len(o.data) > 0, strings.TrimSpace(o.notes) != "")
}

func (o observation) HasStructuredPayload() bool {
	return o.GetDataShape().HasStructuredPayload()
}

func (o observation) HasNarrativeNotes() bool {
	return o.GetDataShape().HasNarrativeNotes()
}

func (o observation) MarshalJSON() ([]byte, error) {
	type observationJSON struct {
		ID          string         `json:"id"`
		CreatedAt   time.Time      `json:"created_at"`
		UpdatedAt   time.Time      `json:"updated_at"`
		ProcedureID *string        `json:"procedure_id,omitempty"`
		OrganismID  *string        `json:"organism_id,omitempty"`
		CohortID    *string        `json:"cohort_id,omitempty"`
		RecordedAt  time.Time      `json:"recorded_at"`
		Observer    string         `json:"observer"`
		Data        map[string]any `json:"data,omitempty"`
		Notes       string         `json:"notes,omitempty"`
	}
	return json.Marshal(observationJSON{
		ID:          o.ID(),
		CreatedAt:   o.CreatedAt(),
		UpdatedAt:   o.UpdatedAt(),
		ProcedureID: cloneOptionalString(o.procedureID),
		OrganismID:  cloneOptionalString(o.organismID),
		CohortID:    cloneOptionalString(o.cohortID),
		RecordedAt:  o.recordedAt,
		Observer:    o.observer,
		Data:        cloneAttributes(o.data),
		Notes:       o.notes,
	})
}

type sample struct {
	base
	identifier      string
	sourceType      string
	organismID      *string
	cohortID        *string
	facilityID      string
	collectedAt     time.Time
	status          string
	storageLocation string
	assayType       string
	chainOfCustody  []custodyEvent
	attributes      map[string]any
}

// NewSample constructs a read-only Sample facade.
func NewSample(data SampleData) Sample {
	return sample{
		base:            newBase(data.Base),
		identifier:      data.Identifier,
		sourceType:      data.SourceType,
		organismID:      cloneOptionalString(data.OrganismID),
		cohortID:        cloneOptionalString(data.CohortID),
		facilityID:      data.FacilityID,
		collectedAt:     data.CollectedAt,
		status:          data.Status,
		storageLocation: data.StorageLocation,
		assayType:       data.AssayType,
		chainOfCustody:  buildCustodyEvents(data.ChainOfCustody),
		attributes:      cloneAttributes(data.Attributes),
	}
}

func (s sample) Identifier() string      { return s.identifier }
func (s sample) SourceType() string      { return s.sourceType }
func (s sample) FacilityID() string      { return s.facilityID }
func (s sample) CollectedAt() time.Time  { return s.collectedAt }
func (s sample) Status() string          { return s.status }
func (s sample) StorageLocation() string { return s.storageLocation }
func (s sample) AssayType() string       { return s.assayType }
func (s sample) Attributes() map[string]any {
	return cloneAttributes(s.attributes)
}

func (s sample) OrganismID() (string, bool) {
	return derefString(s.organismID)
}

func (s sample) CohortID() (string, bool) {
	return derefString(s.cohortID)
}

func (s sample) ChainOfCustody() []SampleCustodyEvent {
	if len(s.chainOfCustody) == 0 {
		return nil
	}
	out := make([]SampleCustodyEvent, len(s.chainOfCustody))
	for i := range s.chainOfCustody {
		out[i] = s.chainOfCustody[i]
	}
	return out
}

// Contextual sample accessors
func (s sample) GetSource() SampleSourceRef {
	ctx := NewSampleContext().Sources()
	source := strings.ToLower(strings.TrimSpace(s.sourceType))
	switch source {
	case "organism":
		return ctx.Organism()
	case "cohort":
		return ctx.Cohort()
	case "environment", "environmental":
		return ctx.Environmental()
	default:
		return ctx.Unknown()
	}
}

func (s sample) GetStatus() SampleStatusRef {
	ctx := NewSampleContext().Statuses()
	status := strings.ToLower(strings.TrimSpace(s.status))
	switch status {
	case "stored":
		return ctx.Stored()
	case "in_transit", "in-transit", "transit":
		return ctx.InTransit()
	case "consumed":
		return ctx.Consumed()
	case "disposed":
		return ctx.Disposed()
	default:
		return ctx.Stored()
	}
}

func (s sample) IsAvailable() bool {
	return s.GetStatus().IsAvailable()
}

func (s sample) MarshalJSON() ([]byte, error) {
	type sampleJSON struct {
		ID              string           `json:"id"`
		CreatedAt       time.Time        `json:"created_at"`
		UpdatedAt       time.Time        `json:"updated_at"`
		Identifier      string           `json:"identifier"`
		SourceType      string           `json:"source_type"`
		OrganismID      *string          `json:"organism_id,omitempty"`
		CohortID        *string          `json:"cohort_id,omitempty"`
		FacilityID      string           `json:"facility_id"`
		CollectedAt     time.Time        `json:"collected_at"`
		Status          string           `json:"status"`
		StorageLocation string           `json:"storage_location"`
		AssayType       string           `json:"assay_type"`
		ChainOfCustody  []map[string]any `json:"chain_of_custody,omitempty"`
		Attributes      map[string]any   `json:"attributes,omitempty"`
	}
	return json.Marshal(sampleJSON{
		ID:              s.ID(),
		CreatedAt:       s.CreatedAt(),
		UpdatedAt:       s.UpdatedAt(),
		Identifier:      s.identifier,
		SourceType:      s.sourceType,
		OrganismID:      cloneOptionalString(s.organismID),
		CohortID:        cloneOptionalString(s.cohortID),
		FacilityID:      s.facilityID,
		CollectedAt:     s.collectedAt,
		Status:          s.status,
		StorageLocation: s.storageLocation,
		AssayType:       s.assayType,
		ChainOfCustody:  serializeCustodyEvents(s.chainOfCustody),
		Attributes:      cloneAttributes(s.attributes),
	})
}

type custodyEvent struct {
	actor     string
	location  string
	timestamp time.Time
	notes     string
}

func (c custodyEvent) Actor() string        { return c.actor }
func (c custodyEvent) Location() string     { return c.location }
func (c custodyEvent) Timestamp() time.Time { return c.timestamp }
func (c custodyEvent) Notes() string        { return c.notes }

type protocol struct {
	base
	code        string
	title       string
	description string
	maxSubjects int
	status      string
}

// NewProtocol constructs a read-only Protocol facade.
func NewProtocol(data ProtocolData) Protocol {
	return protocol{
		base:        newBase(data.Base),
		code:        data.Code,
		title:       data.Title,
		description: data.Description,
		maxSubjects: data.MaxSubjects,
		status:      data.Status,
	}
}

func (p protocol) Code() string        { return p.code }
func (p protocol) Title() string       { return p.title }
func (p protocol) Description() string { return p.description }
func (p protocol) MaxSubjects() int    { return p.maxSubjects }

// Contextual status accessors
func (p protocol) GetCurrentStatus() ProtocolStatusRef {
	ctx := NewProtocolContext()
	switch strings.ToLower(p.status) {
	case "draft":
		return ctx.Draft()
	case "active":
		return ctx.Active()
	case "suspended", "paused":
		return ctx.Suspended()
	case "completed", "done":
		return ctx.Completed()
	case "cancelled":
		return ctx.Cancelled()
	default:
		// Default to draft for unknown statuses
		return ctx.Draft()
	}
}

func (p protocol) IsActiveProtocol() bool {
	return p.GetCurrentStatus().IsActive()
}

func (p protocol) IsTerminalStatus() bool {
	return p.GetCurrentStatus().IsTerminal()
}

func (p protocol) CanAcceptNewSubjects() bool {
	return p.GetCurrentStatus().IsActive() && p.maxSubjects > 0
}

func (p protocol) MarshalJSON() ([]byte, error) {
	type protocolJSON struct {
		ID          string    `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Code        string    `json:"code"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		MaxSubjects int       `json:"max_subjects"`
		Status      string    `json:"status"`
	}
	return json.Marshal(protocolJSON{
		ID:          p.ID(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
		Code:        p.code,
		Title:       p.title,
		Description: p.description,
		MaxSubjects: p.maxSubjects,
		Status:      p.status,
	})
}

type permit struct {
	base
	permitNumber      string
	authority         string
	validFrom         time.Time
	validUntil        time.Time
	allowedActivities []string
	facilityIDs       []string
	protocolIDs       []string
	notes             string
}

// NewPermit constructs a read-only Permit facade.
func NewPermit(data PermitData) Permit {
	return permit{
		base:              newBase(data.Base),
		permitNumber:      data.PermitNumber,
		authority:         data.Authority,
		validFrom:         data.ValidFrom,
		validUntil:        data.ValidUntil,
		allowedActivities: cloneStringSlice(data.AllowedActivities),
		facilityIDs:       cloneStringSlice(data.FacilityIDs),
		protocolIDs:       cloneStringSlice(data.ProtocolIDs),
		notes:             data.Notes,
	}
}

func (p permit) PermitNumber() string  { return p.permitNumber }
func (p permit) Authority() string     { return p.authority }
func (p permit) ValidFrom() time.Time  { return p.validFrom }
func (p permit) ValidUntil() time.Time { return p.validUntil }
func (p permit) AllowedActivities() []string {
	return cloneStringSlice(p.allowedActivities)
}
func (p permit) FacilityIDs() []string { return cloneStringSlice(p.facilityIDs) }
func (p permit) ProtocolIDs() []string { return cloneStringSlice(p.protocolIDs) }
func (p permit) Notes() string         { return p.notes }

// Contextual validity accessors
func (p permit) GetStatus(reference time.Time) PermitStatusRef {
	statuses := NewPermitContext().Statuses()
	switch {
	case reference.Before(p.validFrom):
		return statuses.Pending()
	case !p.validUntil.IsZero() && reference.After(p.validUntil):
		return statuses.Expired()
	default:
		return statuses.Active()
	}
}

func (p permit) IsActive(reference time.Time) bool {
	return p.GetStatus(reference).IsActive()
}

func (p permit) IsExpired(reference time.Time) bool {
	return p.GetStatus(reference).IsExpired()
}

func (p permit) MarshalJSON() ([]byte, error) {
	type permitJSON struct {
		ID                string    `json:"id"`
		CreatedAt         time.Time `json:"created_at"`
		UpdatedAt         time.Time `json:"updated_at"`
		PermitNumber      string    `json:"permit_number"`
		Authority         string    `json:"authority"`
		ValidFrom         time.Time `json:"valid_from"`
		ValidUntil        time.Time `json:"valid_until"`
		AllowedActivities []string  `json:"allowed_activities,omitempty"`
		FacilityIDs       []string  `json:"facility_ids,omitempty"`
		ProtocolIDs       []string  `json:"protocol_ids,omitempty"`
		Notes             string    `json:"notes,omitempty"`
	}
	return json.Marshal(permitJSON{
		ID:                p.ID(),
		CreatedAt:         p.CreatedAt(),
		UpdatedAt:         p.UpdatedAt(),
		PermitNumber:      p.permitNumber,
		Authority:         p.authority,
		ValidFrom:         p.validFrom,
		ValidUntil:        p.validUntil,
		AllowedActivities: cloneStringSlice(p.allowedActivities),
		FacilityIDs:       cloneStringSlice(p.facilityIDs),
		ProtocolIDs:       cloneStringSlice(p.protocolIDs),
		Notes:             p.notes,
	})
}

type project struct {
	base
	code        string
	title       string
	description string
	facilityIDs []string
}

// NewProject constructs a read-only Project facade.
func NewProject(data ProjectData) Project {
	return project{
		base:        newBase(data.Base),
		code:        data.Code,
		title:       data.Title,
		description: data.Description,
		facilityIDs: cloneStringSlice(data.FacilityIDs),
	}
}

func (p project) Code() string        { return p.code }
func (p project) Title() string       { return p.title }
func (p project) Description() string { return p.description }
func (p project) FacilityIDs() []string {
	return cloneStringSlice(p.facilityIDs)
}

func (p project) MarshalJSON() ([]byte, error) {
	type projectJSON struct {
		ID          string    `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Code        string    `json:"code"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		FacilityIDs []string  `json:"facility_ids,omitempty"`
	}
	return json.Marshal(projectJSON{
		ID:          p.ID(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
		Code:        p.code,
		Title:       p.title,
		Description: p.description,
		FacilityIDs: cloneStringSlice(p.facilityIDs),
	})
}

type supplyItem struct {
	base
	sku            string
	name           string
	description    string
	quantityOnHand int
	unit           string
	lotNumber      string
	expiresAt      *time.Time
	facilityIDs    []string
	projectIDs     []string
	reorderLevel   int
	attributes     map[string]any
}

// NewSupplyItem constructs a read-only SupplyItem facade.
func NewSupplyItem(data SupplyItemData) SupplyItem {
	return supplyItem{
		base:           newBase(data.Base),
		sku:            data.SKU,
		name:           data.Name,
		description:    data.Description,
		quantityOnHand: data.QuantityOnHand,
		unit:           data.Unit,
		lotNumber:      data.LotNumber,
		expiresAt:      cloneTimePtr(data.ExpiresAt),
		facilityIDs:    cloneStringSlice(data.FacilityIDs),
		projectIDs:     cloneStringSlice(data.ProjectIDs),
		reorderLevel:   data.ReorderLevel,
		attributes:     cloneAttributes(data.Attributes),
	}
}

func (s supplyItem) SKU() string         { return s.sku }
func (s supplyItem) Name() string        { return s.name }
func (s supplyItem) Description() string { return s.description }
func (s supplyItem) QuantityOnHand() int { return s.quantityOnHand }
func (s supplyItem) Unit() string        { return s.unit }
func (s supplyItem) LotNumber() string   { return s.lotNumber }
func (s supplyItem) FacilityIDs() []string {
	return cloneStringSlice(s.facilityIDs)
}
func (s supplyItem) ProjectIDs() []string {
	return cloneStringSlice(s.projectIDs)
}
func (s supplyItem) ReorderLevel() int { return s.reorderLevel }
func (s supplyItem) Attributes() map[string]any {
	return cloneAttributes(s.attributes)
}

func (s supplyItem) ExpiresAt() (*time.Time, bool) {
	return derefTime(s.expiresAt)
}

// Contextual inventory accessors
func (s supplyItem) GetInventoryStatus(reference time.Time) SupplyStatusRef {
	return computeSupplyStatus(s.quantityOnHand, s.reorderLevel, s.expiresAt, reference)
}

func (s supplyItem) RequiresReorder(reference time.Time) bool {
	return s.GetInventoryStatus(reference).RequiresReorder()
}

func (s supplyItem) IsExpired(reference time.Time) bool {
	return s.GetInventoryStatus(reference).IsExpired()
}

func (s supplyItem) MarshalJSON() ([]byte, error) {
	type supplyJSON struct {
		ID             string         `json:"id"`
		CreatedAt      time.Time      `json:"created_at"`
		UpdatedAt      time.Time      `json:"updated_at"`
		SKU            string         `json:"sku"`
		Name           string         `json:"name"`
		Description    string         `json:"description"`
		QuantityOnHand int            `json:"quantity_on_hand"`
		Unit           string         `json:"unit"`
		LotNumber      string         `json:"lot_number"`
		ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
		FacilityIDs    []string       `json:"facility_ids,omitempty"`
		ProjectIDs     []string       `json:"project_ids,omitempty"`
		ReorderLevel   int            `json:"reorder_level"`
		Attributes     map[string]any `json:"attributes,omitempty"`
	}
	return json.Marshal(supplyJSON{
		ID:             s.ID(),
		CreatedAt:      s.CreatedAt(),
		UpdatedAt:      s.UpdatedAt(),
		SKU:            s.sku,
		Name:           s.name,
		Description:    s.description,
		QuantityOnHand: s.quantityOnHand,
		Unit:           s.unit,
		LotNumber:      s.lotNumber,
		ExpiresAt:      cloneTimePtr(s.expiresAt),
		FacilityIDs:    cloneStringSlice(s.facilityIDs),
		ProjectIDs:     cloneStringSlice(s.projectIDs),
		ReorderLevel:   s.reorderLevel,
		Attributes:     cloneAttributes(s.attributes),
	})
}

func derefString(ptr *string) (string, bool) {
	if ptr == nil {
		return "", false
	}
	return *ptr, true
}

func cloneOptionalString(ptr *string) *string {
	if ptr == nil {
		return nil
	}
	value := *ptr
	return &value
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func derefTime(src *time.Time) (*time.Time, bool) {
	if src == nil {
		return nil, false
	}
	value := *src
	return &value, true
}

func buildCustodyEvents(events []SampleCustodyEventData) []custodyEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]custodyEvent, len(events))
	for i, event := range events {
		out[i] = custodyEvent{
			actor:     event.Actor,
			location:  event.Location,
			timestamp: event.Timestamp,
			notes:     event.Notes,
		}
	}
	return out
}

func serializeCustodyEvents(events []custodyEvent) []map[string]any {
	if len(events) == 0 {
		return nil
	}
	out := make([]map[string]any, len(events))
	for i, event := range events {
		out[i] = map[string]any{
			"actor":     event.actor,
			"location":  event.location,
			"timestamp": event.timestamp,
			"notes":     event.notes,
		}
	}
	return out
}

func cloneAttributes(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = deepClone(v)
	}
	return out
}

// deepClone performs a best-effort recursive clone of common container shapes
// (map[string]any, []any, []string, []map[string]any) to harden immutability of
// facade projections returned to plugins. Values that are not recognized
// containers (numbers, bools, strings, time.Time, etc.) are returned as-is.
// The cloning intentionally does not try to handle cycles; the data structures
// in the domain model used for attributes are expected to be acyclic.
func deepClone(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		if len(tv) == 0 {
			return map[string]any{}
		}
		m := make(map[string]any, len(tv))
		for k, vv := range tv {
			m[k] = deepClone(vv)
		}
		return m
	case []any:
		if len(tv) == 0 {
			return []any{}
		}
		s := make([]any, len(tv))
		for i, vv := range tv {
			s[i] = deepClone(vv)
		}
		return s
	case []string:
		if len(tv) == 0 {
			return []string{}
		}
		s := make([]string, len(tv))
		copy(s, tv)
		return s
	case []map[string]any:
		if len(tv) == 0 {
			return []map[string]any{}
		}
		s := make([]map[string]any, len(tv))
		for i, mv := range tv {
			if mv == nil {
				continue
			}
			s[i] = cloneAttributes(mv)
		}
		return s
	default:
		return v
	}
}
