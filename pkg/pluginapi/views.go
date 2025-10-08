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
	Facility() string
	Capacity() int
	Environment() string
}

// ProtocolView is a read-only projection of a protocol record.
type ProtocolView interface {
	BaseView
	Code() string
	Title() string
	Description() string
	MaxSubjects() int
}
