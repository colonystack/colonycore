package pluginapi

import "strings"

const (
	zoneBiosecure  = "biosecure"
	zoneQuarantine = "quarantine"
	zoneGeneral    = "general"

	accessRestricted = "restricted"
	accessStaffOnly  = "staff_only"
	accessOpen       = "open"
)

// FacilityContext provides contextual access to facility zoning and access policies.
type FacilityContext interface {
	Zones() FacilityZoneProvider
	AccessPolicies() FacilityAccessPolicyProvider
}

// FacilityZoneProvider exposes canonical facility zone references.
type FacilityZoneProvider interface {
	Biosecure() FacilityZoneRef
	Quarantine() FacilityZoneRef
	General() FacilityZoneRef
}

// FacilityZoneRef represents an opaque facility zone reference.
type FacilityZoneRef interface {
	String() string
	IsBiosecure() bool
	IsQuarantine() bool
	Equals(other FacilityZoneRef) bool
	isFacilityZoneRef()
}

// FacilityAccessPolicyProvider exposes canonical facility access policies.
type FacilityAccessPolicyProvider interface {
	Restricted() FacilityAccessPolicyRef
	StaffOnly() FacilityAccessPolicyRef
	Open() FacilityAccessPolicyRef
}

// FacilityAccessPolicyRef represents an opaque facility access policy reference.
type FacilityAccessPolicyRef interface {
	String() string
	IsRestricted() bool
	AllowsVisitors() bool
	Equals(other FacilityAccessPolicyRef) bool
	isFacilityAccessPolicyRef()
}

type facilityContext struct{}

// NewFacilityContext constructs the default facility context provider.
func NewFacilityContext() FacilityContext {
	return facilityContext{}
}

func (facilityContext) Zones() FacilityZoneProvider {
	return facilityZoneProvider{}
}

func (facilityContext) AccessPolicies() FacilityAccessPolicyProvider {
	return facilityAccessPolicyProvider{}
}

type facilityZoneProvider struct{}

func (facilityZoneProvider) Biosecure() FacilityZoneRef {
	return facilityZoneRef{value: zoneBiosecure}
}

func (facilityZoneProvider) Quarantine() FacilityZoneRef {
	return facilityZoneRef{value: zoneQuarantine}
}

func (facilityZoneProvider) General() FacilityZoneRef {
	return facilityZoneRef{value: zoneGeneral}
}

type facilityZoneRef struct {
	value string
}

func (f facilityZoneRef) String() string {
	return f.value
}

func (f facilityZoneRef) IsBiosecure() bool {
	val := strings.ToLower(f.value)
	return strings.Contains(val, "bsl") || strings.Contains(val, "biosecure")
}

func (f facilityZoneRef) IsQuarantine() bool {
	val := strings.ToLower(f.value)
	return strings.Contains(val, "quarantine") || strings.Contains(val, "isolation")
}

func (f facilityZoneRef) Equals(other FacilityZoneRef) bool {
	if otherRef, ok := other.(facilityZoneRef); ok {
		return strings.EqualFold(f.value, otherRef.value)
	}
	return false
}

func (f facilityZoneRef) isFacilityZoneRef() {}

type facilityAccessPolicyProvider struct{}

func (facilityAccessPolicyProvider) Restricted() FacilityAccessPolicyRef {
	return facilityAccessPolicyRef{value: accessRestricted}
}

func (facilityAccessPolicyProvider) StaffOnly() FacilityAccessPolicyRef {
	return facilityAccessPolicyRef{value: accessStaffOnly}
}

func (facilityAccessPolicyProvider) Open() FacilityAccessPolicyRef {
	return facilityAccessPolicyRef{value: accessOpen}
}

type facilityAccessPolicyRef struct {
	value string
}

func (f facilityAccessPolicyRef) String() string {
	return f.value
}

func (f facilityAccessPolicyRef) IsRestricted() bool {
	val := strings.ToLower(f.value)
	return strings.Contains(val, "restricted") || strings.Contains(val, "secure")
}

func (f facilityAccessPolicyRef) AllowsVisitors() bool {
	val := strings.ToLower(f.value)
	return strings.Contains(val, "open") || strings.Contains(val, "visitor")
}

func (f facilityAccessPolicyRef) Equals(other FacilityAccessPolicyRef) bool {
	if otherRef, ok := other.(facilityAccessPolicyRef); ok {
		return strings.EqualFold(f.value, otherRef.value)
	}
	return false
}

func (f facilityAccessPolicyRef) isFacilityAccessPolicyRef() {}
