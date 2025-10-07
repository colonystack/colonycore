// Package pluginapi provides a stable surface for plugin authors by defining
// rule evaluation primitives and canonical identifiers independent of the core
// domain package.
package pluginapi

import "encoding/json"

// Severity captures rule outcomes exposed to plugins.
type Severity string

// Rule evaluation severities determine commit behavior and logging.
const (
	SeverityBlock Severity = "block"
	SeverityWarn  Severity = "warn"
	SeverityLog   Severity = "log"
)

// LifecycleStage represents canonical organism lifecycle identifiers available to plugins.
type LifecycleStage string

// Canonical organism lifecycle stages mirrored from the domain package.
const (
	StagePlanned  LifecycleStage = "planned"
	StageLarva    LifecycleStage = "embryo_larva"
	StageJuvenile LifecycleStage = "juvenile"
	StageAdult    LifecycleStage = "adult"
	StageRetired  LifecycleStage = "retired"
	StageDeceased LifecycleStage = "deceased"
)

// EntityType identifies the type of record referenced by rule changes.
type EntityType string

// Supported entity type identifiers used in Change records and persistence buckets.
const (
	EntityOrganism    EntityType = "organism"
	EntityCohort      EntityType = "cohort"
	EntityHousingUnit EntityType = "housing_unit"
	EntityBreeding    EntityType = "breeding_unit"
	EntityProcedure   EntityType = "procedure"
	EntityProtocol    EntityType = "protocol"
	EntityProject     EntityType = "project"
)

// Action indicates the type of modification performed.
type Action string

// Change actions enumerate supported CRUD operations captured in audit trail.
const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

// Change describes a mutation applied to an entity during a transaction. It is
// immutable to plugin authors (fields unexported) and accessed via getter
// methods to prevent accidental mutation of underlying state. Before/After are
// snapshots (shallow copies) of the prior and new values.
type Change struct {
	entity EntityType
	action Action
	before any
	after  any
}

// NewChange constructs an immutable Change for use by the host when adapting
// domain transactions for plugin rule evaluation.
func NewChange(entity EntityType, action Action, before, after any) Change {
	return Change{entity: entity, action: action, before: snapshotValue(before), after: snapshotValue(after)}
}

// Entity returns the entity type affected.
func (c Change) Entity() EntityType { return c.entity }

// Action returns the CRUD action performed.
func (c Change) Action() Action { return c.action }

// Before returns the previous value (if any) of the entity. Treat as read-only.
func (c Change) Before() any { return snapshotValue(c.before) }

// After returns the new value (if any) of the entity. Treat as read-only.
func (c Change) After() any { return snapshotValue(c.after) }

// snapshotValue performs a best-effort defensive copy of common container types
// to prevent plugin authors from mutating internal state through Change.Before/After.
// It shallow-copies maps and slices and JSON round-trips other JSON-serializable
// values (structs, basic types). Fallback returns the original value when copy
// would be lossy or unsupported.
func snapshotValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, vv := range val {
			cp[k] = snapshotValue(vv)
		}
		return cp
	case []string:
		cp := make([]string, len(val))
		copy(cp, val)
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, vv := range val {
			cp[i] = snapshotValue(vv)
		}
		return cp
	case []map[string]any:
		cp := make([]map[string]any, len(val))
		for i, m := range val {
			if m == nil {
				continue
			}
			cp[i] = snapshotValue(m).(map[string]any)
		}
		return cp
	// Primitive scalars are safe to return directly (they are immutable values).
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return val
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

// Violation reports a failed rule evaluation.
type Violation struct {
	rule     string
	severity Severity
	message  string
	entity   EntityType
	entityID string
}

// NewViolation constructs an immutable violation.
func NewViolation(rule string, severity Severity, message string, entity EntityType, entityID string) Violation {
	return Violation{rule: rule, severity: severity, message: message, entity: entity, entityID: entityID}
}

// NewViolationWithEntityRef constructs an immutable violation using contextual interface references.
// This promotes hexagonal architecture by allowing violation creation without direct constant access.
func NewViolationWithEntityRef(rule string, severity SeverityRef, message string, entity EntityTypeRef, entityID string) Violation {
	// Extract underlying values from contextual interfaces
	rawSeverity := extractSeverity(severity)
	rawEntity := extractEntityType(entity)
	return Violation{rule: rule, severity: rawSeverity, message: message, entity: rawEntity, entityID: entityID}
}

// extractSeverity safely extracts the underlying Severity from a SeverityRef.
func extractSeverity(ref SeverityRef) Severity {
	if severityRef, ok := ref.(severityRef); ok {
		return severityRef.value
	}
	// Fallback for unexpected implementations - should not happen in normal usage
	return Severity(ref.String())
}

// extractEntityType safely extracts the underlying EntityType from an EntityTypeRef.
func extractEntityType(ref EntityTypeRef) EntityType {
	if entityRef, ok := ref.(entityTypeRef); ok {
		return entityRef.value
	}
	// Fallback for unexpected implementations - should not happen in normal usage
	return EntityType(ref.String())
}

// Rule returns the identifier of the rule that produced the violation.
func (v Violation) Rule() string { return v.rule }

// Severity returns the severity classification of the violation.
func (v Violation) Severity() Severity { return v.severity }

// Message returns the human-readable description of the violation.
func (v Violation) Message() string { return v.message }

// Entity returns the entity type associated with the violation.
func (v Violation) Entity() EntityType { return v.entity }

// EntityID returns the identifier of the entity instance associated with the violation.
func (v Violation) EntityID() string { return v.entityID }

// Result aggregates violations from the rules engine in an immutable slice.
type Result struct {
	violations []Violation
}

// NewResult constructs a Result from provided violations (defensively copied).
func NewResult(violations ...Violation) Result {
	if len(violations) == 0 {
		return Result{}
	}
	cp := make([]Violation, len(violations))
	copy(cp, violations)
	return Result{violations: cp}
}

// Violations returns a defensive copy of contained violations.
func (r Result) Violations() []Violation {
	if len(r.violations) == 0 {
		return nil
	}
	cp := make([]Violation, len(r.violations))
	copy(cp, r.violations)
	return cp
}

// AddViolation returns a new Result with the additional violation appended.
func (r Result) AddViolation(v Violation) Result {
	if r.violations == nil {
		return Result{violations: []Violation{v}}
	}
	out := make([]Violation, len(r.violations)+1)
	copy(out, r.violations)
	out[len(out)-1] = v
	return Result{violations: out}
}

// Merge returns a new Result containing violations from both results.
func (r Result) Merge(other Result) Result {
	ov := other.violations
	if len(ov) == 0 {
		return r
	}
	if len(r.violations) == 0 {
		return other
	}
	out := make([]Violation, 0, len(r.violations)+len(ov))
	out = append(out, r.violations...)
	out = append(out, ov...)
	return Result{violations: out}
}

// HasBlocking returns true if the result contains blocking violations.
func (r Result) HasBlocking() bool {
	for _, v := range r.violations {
		if v.severity == SeverityBlock {
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
