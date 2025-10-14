package pluginapi

const (
	supplyStatusHealthy  = "healthy"
	supplyStatusReorder  = "reorder"
	supplyStatusCritical = "critical"
	supplyStatusExpired  = "expired"
)

// SupplyContext provides contextual access to supply inventory status references.
type SupplyContext interface {
	Statuses() SupplyStatusProvider
}

// SupplyStatusProvider exposes canonical supply inventory statuses.
type SupplyStatusProvider interface {
	Healthy() SupplyStatusRef
	Reorder() SupplyStatusRef
	Critical() SupplyStatusRef
	Expired() SupplyStatusRef
}

// SupplyStatusRef represents an opaque supply inventory status.
type SupplyStatusRef interface {
	String() string
	RequiresReorder() bool
	IsExpired() bool
	Equals(other SupplyStatusRef) bool
	isSupplyStatusRef()
}

type supplyContext struct{}

// NewSupplyContext constructs the default supply context provider.
func NewSupplyContext() SupplyContext {
	return supplyContext{}
}

func (supplyContext) Statuses() SupplyStatusProvider {
	return supplyStatusProvider{}
}

type supplyStatusProvider struct{}

func (supplyStatusProvider) Healthy() SupplyStatusRef {
	return supplyStatusRef{value: supplyStatusHealthy}
}

func (supplyStatusProvider) Reorder() SupplyStatusRef {
	return supplyStatusRef{value: supplyStatusReorder}
}

func (supplyStatusProvider) Critical() SupplyStatusRef {
	return supplyStatusRef{value: supplyStatusCritical}
}

func (supplyStatusProvider) Expired() SupplyStatusRef {
	return supplyStatusRef{value: supplyStatusExpired}
}

type supplyStatusRef struct {
	value string
}

func (s supplyStatusRef) String() string {
	return s.value
}

func (s supplyStatusRef) RequiresReorder() bool {
	return s.value == supplyStatusReorder || s.value == supplyStatusCritical
}

func (s supplyStatusRef) IsExpired() bool {
	return s.value == supplyStatusExpired
}

func (s supplyStatusRef) Equals(other SupplyStatusRef) bool {
	if otherRef, ok := other.(supplyStatusRef); ok {
		return s.value == otherRef.value
	}
	return false
}

func (s supplyStatusRef) isSupplyStatusRef() {}
