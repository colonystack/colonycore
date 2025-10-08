// Package testhelper hosts plugin fixture builders that can reference domain
// types without tripping plugin import restrictions enforced by import-boss.
package testhelper

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"time"
)

// LifecycleStages provides access to lifecycle stage values for testing without exposing constants
func LifecycleStages() struct {
	Planned  datasetapi.LifecycleStage
	Larva    datasetapi.LifecycleStage
	Juvenile datasetapi.LifecycleStage
	Adult    datasetapi.LifecycleStage
	Retired  datasetapi.LifecycleStage
	Deceased datasetapi.LifecycleStage
} {
	stages := datasetapi.NewLifecycleStageContext()
	return struct {
		Planned  datasetapi.LifecycleStage
		Larva    datasetapi.LifecycleStage
		Juvenile datasetapi.LifecycleStage
		Adult    datasetapi.LifecycleStage
		Retired  datasetapi.LifecycleStage
		Deceased datasetapi.LifecycleStage
	}{
		Planned:  datasetapi.LifecycleStage(stages.Planned().String()),
		Larva:    datasetapi.LifecycleStage(stages.Larva().String()),
		Juvenile: datasetapi.LifecycleStage(stages.Juvenile().String()),
		Adult:    datasetapi.LifecycleStage(stages.Adult().String()),
		Retired:  datasetapi.LifecycleStage(stages.Retired().String()),
		Deceased: datasetapi.LifecycleStage(stages.Deceased().String()),
	}
}

// BaseFixture captures shared metadata for entity fixtures.
type BaseFixture struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrganismFixtureConfig describes an organism projection used in plugin tests.
type OrganismFixtureConfig struct {
	BaseFixture
	Name       string
	Species    string
	Line       string
	Stage      datasetapi.LifecycleStage
	CohortID   *string
	HousingID  *string
	ProtocolID *string
	ProjectID  *string
	Attributes map[string]any
}

// HousingUnitFixtureConfig describes a housing unit projection used in plugin tests.
type HousingUnitFixtureConfig struct {
	BaseFixture
	Name        string
	Facility    string
	Capacity    int
	Environment string
}

// Organism constructs a datasetapi.Organism using domain conversion helpers to avoid
// leaking domain imports into plugin test packages constrained by import-boss rules.
func Organism(cfg OrganismFixtureConfig) datasetapi.Organism {
	domainOrganism := domain.Organism{
		Base:       cfg.baseDomain(),
		Name:       cfg.Name,
		Species:    cfg.Species,
		Line:       cfg.Line,
		Stage:      domain.LifecycleStage(cfg.Stage),
		CohortID:   cloneOptionalString(cfg.CohortID),
		HousingID:  cloneOptionalString(cfg.HousingID),
		ProtocolID: cloneOptionalString(cfg.ProtocolID),
		ProjectID:  cloneOptionalString(cfg.ProjectID),
		Attributes: cloneAttributes(cfg.Attributes),
	}

	return datasetapi.NewOrganism(datasetapi.OrganismData{
		Base:       baseDataFromDomain(domainOrganism.Base),
		Name:       domainOrganism.Name,
		Species:    domainOrganism.Species,
		Line:       domainOrganism.Line,
		Stage:      datasetapi.LifecycleStage(domainOrganism.Stage),
		CohortID:   domainOrganism.CohortID,
		HousingID:  domainOrganism.HousingID,
		ProtocolID: domainOrganism.ProtocolID,
		ProjectID:  domainOrganism.ProjectID,
		Attributes: domainOrganism.Attributes,
	})
}

// Organisms convenience helper to build a slice of dataset organisms.
func Organisms(cfgs ...OrganismFixtureConfig) []datasetapi.Organism {
	if len(cfgs) == 0 {
		return nil
	}
	out := make([]datasetapi.Organism, len(cfgs))
	for i := range cfgs {
		out[i] = Organism(cfgs[i])
	}
	return out
}

// HousingUnit constructs a datasetapi.HousingUnit using domain conversion helpers.
func HousingUnit(cfg HousingUnitFixtureConfig) datasetapi.HousingUnit {
	domainUnit := domain.HousingUnit{
		Base:        cfg.baseDomain(),
		Name:        cfg.Name,
		Facility:    cfg.Facility,
		Capacity:    cfg.Capacity,
		Environment: cfg.Environment,
	}

	return datasetapi.NewHousingUnit(datasetapi.HousingUnitData{
		Base:        baseDataFromDomain(domainUnit.Base),
		Name:        domainUnit.Name,
		Facility:    domainUnit.Facility,
		Capacity:    domainUnit.Capacity,
		Environment: domainUnit.Environment,
	})
}

func (cfg BaseFixture) baseDomain() domain.Base {
	return domain.Base{
		ID:        cfg.ID,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	}
}

func baseDataFromDomain(base domain.Base) datasetapi.BaseData {
	return datasetapi.BaseData{
		ID:        base.ID,
		CreatedAt: base.CreatedAt,
		UpdatedAt: base.UpdatedAt,
	}
}

func cloneOptionalString(ptr *string) *string {
	if ptr == nil {
		return nil
	}
	value := *ptr
	return &value
}

func cloneAttributes(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = deepCloneAttr(v)
	}
	return out
}

// deepCloneAttr mirrors the deep cloning strategy used in production code to
// ensure test fixtures reflect immutability guarantees.
func deepCloneAttr(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		if len(tv) == 0 {
			return map[string]any{}
		}
		m := make(map[string]any, len(tv))
		for k, vv := range tv {
			m[k] = deepCloneAttr(vv)
		}
		return m
	case []any:
		if len(tv) == 0 {
			return []any{}
		}
		s := make([]any, len(tv))
		for i, vv := range tv {
			s[i] = deepCloneAttr(vv)
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
