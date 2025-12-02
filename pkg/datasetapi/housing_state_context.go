package datasetapi

// HousingState represents lifecycle states for housing units.
type HousingState string

const (
	housingStateQuarantine     HousingState = "quarantine"
	housingStateActive         HousingState = "active"
	housingStateCleaning       HousingState = "cleaning"
	housingStateDecommissioned HousingState = "decommissioned"
)

// HousingStateContext provides contextual access to housing lifecycle states.
type HousingStateContext interface {
	Quarantine() HousingStateRef
	Active() HousingStateRef
	Cleaning() HousingStateRef
	Decommissioned() HousingStateRef
}

// HousingStateRef represents an opaque reference to a housing lifecycle state.
type HousingStateRef interface {
	String() string
	Equals(other HousingStateRef) bool
	IsActive() bool
	IsDecommissioned() bool
	isHousingStateRef()
}

type housingStateContext struct{}

// NewHousingStateContext creates a new housing lifecycle context.
func NewHousingStateContext() HousingStateContext {
	return housingStateContext{}
}

func (housingStateContext) Quarantine() HousingStateRef {
	return housingStateRef{value: housingStateQuarantine}
}
func (housingStateContext) Active() HousingStateRef {
	return housingStateRef{value: housingStateActive}
}
func (housingStateContext) Cleaning() HousingStateRef {
	return housingStateRef{value: housingStateCleaning}
}
func (housingStateContext) Decommissioned() HousingStateRef {
	return housingStateRef{value: housingStateDecommissioned}
}

type housingStateRef struct {
	value HousingState
}

func (h housingStateRef) String() string {
	return string(h.value)
}

func (h housingStateRef) Equals(other HousingStateRef) bool {
	otherRef, ok := other.(housingStateRef)
	return ok && h.value == otherRef.value
}

func (h housingStateRef) IsActive() bool {
	return h.value == housingStateActive
}

func (h housingStateRef) IsDecommissioned() bool {
	return h.value == housingStateDecommissioned
}

func (h housingStateRef) isHousingStateRef() {}
