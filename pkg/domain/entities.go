// Package domain defines the core persistent entities, value types, and
// rule evaluation primitives used by colonycore.
package domain

import "time"

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
	// EntityBreeding identifies a breeding unit record.
	EntityBreeding EntityType = "breeding_unit"
	// EntityProcedure identifies a procedure record.
	EntityProcedure EntityType = "procedure"
	// EntityProtocol identifies a protocol record.
	EntityProtocol EntityType = "protocol"
	// EntityProject identifies a project record.
	EntityProject EntityType = "project"
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
	Name       string         `json:"name"`
	Species    string         `json:"species"`
	Line       string         `json:"line"`
	Stage      LifecycleStage `json:"stage"`
	CohortID   *string        `json:"cohort_id"`
	HousingID  *string        `json:"housing_id"`
	ProtocolID *string        `json:"protocol_id"`
	ProjectID  *string        `json:"project_id"`
	Attributes map[string]any `json:"attributes"`
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
	Facility    string `json:"facility"`
	Capacity    int    `json:"capacity"`
	Environment string `json:"environment"`
}

// BreedingUnit tracks configured pairings or groups intended for reproduction.
type BreedingUnit struct {
	Base
	Name       string   `json:"name"`
	Strategy   string   `json:"strategy"`
	HousingID  *string  `json:"housing_id"`
	ProtocolID *string  `json:"protocol_id"`
	FemaleIDs  []string `json:"female_ids"`
	MaleIDs    []string `json:"male_ids"`
}

// Procedure captures scheduled or completed animal procedures.
type Procedure struct {
	Base
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
	ProtocolID  string    `json:"protocol_id"`
	CohortID    *string   `json:"cohort_id"`
	OrganismIDs []string  `json:"organism_ids"`
}

// Protocol represents compliance agreements.
type Protocol struct {
	Base
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	MaxSubjects int    `json:"max_subjects"`
	Status      string `json:"status"`
}

// Project captures cost center allocations.
type Project struct {
	Base
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
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
