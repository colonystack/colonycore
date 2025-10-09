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
	ListProtocols() []Protocol
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
}

// PersistentStore exposes read-only query helpers used by dataset binders.
type PersistentStore interface {
	View(ctx context.Context, fn func(TransactionView) error) error
	GetOrganism(id string) (Organism, bool)
	ListOrganisms() []Organism
	GetHousingUnit(id string) (HousingUnit, bool)
	ListHousingUnits() []HousingUnit
	ListCohorts() []Cohort
	ListProtocols() []Protocol
	ListProjects() []Project
	ListBreedingUnits() []BreedingUnit
	ListProcedures() []Procedure
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
	Facility    string
	Capacity    int
	Environment string
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

// ProtocolData describes the fields required to construct a Protocol facade.
type ProtocolData struct {
	Base        BaseData
	Code        string
	Title       string
	Description string
	MaxSubjects int
	Status      string
}

// ProjectData describes the fields required to construct a Project facade.
type ProjectData struct {
	Base        BaseData
	Code        string
	Title       string
	Description string
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
}

// HousingUnit exposes read-only housing metadata to dataset plugins.
type HousingUnit interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Name() string
	Facility() string
	Capacity() int
	Environment() string

	// Contextual environment accessors
	GetEnvironmentType() EnvironmentTypeRef
	IsAquaticEnvironment() bool
	IsHumidEnvironment() bool
	SupportsSpecies(species string) bool
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

// Project exposes read-only project metadata to dataset plugins.
type Project interface {
	ID() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Code() string
	Title() string
	Description() string
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
	facility    string
	capacity    int
	environment string
}

// NewHousingUnit constructs a read-only HousingUnit facade.
func NewHousingUnit(data HousingUnitData) HousingUnit {
	return housingUnit{
		base:        newBase(data.Base),
		name:        data.Name,
		facility:    data.Facility,
		capacity:    data.Capacity,
		environment: data.Environment,
	}
}

func (h housingUnit) Name() string        { return h.name }
func (h housingUnit) Facility() string    { return h.facility }
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
		Facility    string    `json:"facility"`
		Capacity    int       `json:"capacity"`
		Environment string    `json:"environment"`
	}
	return json.Marshal(housingJSON{
		ID:          h.ID(),
		CreatedAt:   h.CreatedAt(),
		UpdatedAt:   h.UpdatedAt(),
		Name:        h.name,
		Facility:    h.facility,
		Capacity:    h.capacity,
		Environment: h.environment,
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

type project struct {
	base
	code        string
	title       string
	description string
}

// NewProject constructs a read-only Project facade.
func NewProject(data ProjectData) Project {
	return project{
		base:        newBase(data.Base),
		code:        data.Code,
		title:       data.Title,
		description: data.Description,
	}
}

func (p project) Code() string        { return p.code }
func (p project) Title() string       { return p.title }
func (p project) Description() string { return p.description }

func (p project) MarshalJSON() ([]byte, error) {
	type projectJSON struct {
		ID          string    `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Code        string    `json:"code"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
	}
	return json.Marshal(projectJSON{
		ID:          p.ID(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
		Code:        p.code,
		Title:       p.title,
		Description: p.description,
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
