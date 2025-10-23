package pluginapi

import "time"

// BaseView exposes shared metadata available on all core entities.
type BaseView interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// OrganismView is a read-only projection of an organism record provided to rules.
type OrganismView interface {
	BaseView
	Name() string
	Species() string
	Line() string
	LineID() (string, bool)
	StrainID() (string, bool)
	ParentIDs() []string
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

// HousingUnitView is a read-only projection of a housing unit record.
type HousingUnitView interface {
	BaseView
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

// FacilityView is a read-only projection of a facility record.
type FacilityView interface {
	BaseView
	Code() string
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

// TreatmentView is a read-only projection of a treatment record.
type TreatmentView interface {
	BaseView
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

// ObservationView is a read-only projection of an observation record.
type ObservationView interface {
	BaseView
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

// SampleView is a read-only projection of a sample record.
type SampleView interface {
	BaseView
	Identifier() string
	SourceType() string
	OrganismID() (string, bool)
	CohortID() (string, bool)
	FacilityID() string
	CollectedAt() time.Time
	Status() string
	StorageLocation() string
	AssayType() string
	ChainOfCustody() []map[string]any
	Attributes() map[string]any

	// Contextual sample accessors
	GetSource() SampleSourceRef
	GetStatus() SampleStatusRef
	IsAvailable() bool
}

// ProtocolView is a read-only projection of a protocol record.
type ProtocolView interface {
	BaseView
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

// PermitView is a read-only projection of a permit record.
type PermitView interface {
	BaseView
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

// ProjectView is a read-only projection of a project record.
type ProjectView interface {
	BaseView
	Code() string
	Title() string
	Description() string
	FacilityIDs() []string
}

// SupplyItemView is a read-only projection of a supply item record.
type SupplyItemView interface {
	BaseView
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
