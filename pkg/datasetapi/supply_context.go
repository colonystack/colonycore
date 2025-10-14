package datasetapi

import "time"

const (
	datasetSupplyStatusHealthy  = "healthy"
	datasetSupplyStatusReorder  = "reorder"
	datasetSupplyStatusCritical = "critical"
	datasetSupplyStatusExpired  = "expired"
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

// SupplyStatusRef represents an opaque supply inventory status reference.
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
	return supplyStatusRef{value: datasetSupplyStatusHealthy}
}

func (supplyStatusProvider) Reorder() SupplyStatusRef {
	return supplyStatusRef{value: datasetSupplyStatusReorder}
}

func (supplyStatusProvider) Critical() SupplyStatusRef {
	return supplyStatusRef{value: datasetSupplyStatusCritical}
}

func (supplyStatusProvider) Expired() SupplyStatusRef {
	return supplyStatusRef{value: datasetSupplyStatusExpired}
}

type supplyStatusRef struct {
	value string
}

func (s supplyStatusRef) String() string {
	return s.value
}

func (s supplyStatusRef) RequiresReorder() bool {
	return s.value == datasetSupplyStatusReorder || s.value == datasetSupplyStatusCritical
}

func (s supplyStatusRef) IsExpired() bool {
	return s.value == datasetSupplyStatusExpired
}

func (s supplyStatusRef) Equals(other SupplyStatusRef) bool {
	if otherRef, ok := other.(supplyStatusRef); ok {
		return s.value == otherRef.value
	}
	return false
}

func (s supplyStatusRef) isSupplyStatusRef() {}

// computeSupplyStatus derives an inventory status given quantity thresholds and expiration.
func computeSupplyStatus(quantity, reorderLevel int, expiresAt *time.Time, now time.Time) SupplyStatusRef {
	statuses := supplyStatusProvider{}
	if expiresAt != nil && !expiresAt.IsZero() && expiresAt.Before(now) {
		return statuses.Expired()
	}
	switch {
	case quantity <= 0:
		return statuses.Critical()
	case reorderLevel > 0 && quantity <= reorderLevel:
		return statuses.Reorder()
	default:
		return statuses.Healthy()
	}
}
