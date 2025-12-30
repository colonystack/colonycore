// Package testhelper hosts plugin fixture builders that can reference domain
// types without tripping plugin import restrictions enforced by import-boss.
package testhelper

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
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
	LineID     *string
	StrainID   *string
	ParentIDs  []string
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
	FacilityID  string
	Capacity    int
	Environment string
}

// Organism constructs a datasetapi.Organism using domain conversion helpers to avoid
// Organism builds a datasetapi.Organism from the provided OrganismFixtureConfig for use in tests.
// 
// The returned organism includes base metadata (ID, CreatedAt, UpdatedAt), standard fields
// (Name, Species, Line, LineID, StrainID, ParentIDs, Stage, CohortID, HousingID, ProtocolID, ProjectID)
// and an ExtensionSet containing core attributes when present. Panics if assigning core attributes fails.
func Organism(cfg OrganismFixtureConfig) datasetapi.Organism {
	domainOrganism := domain.Organism{
		Organism: entitymodel.Organism{
			ID:         cfg.ID,
			CreatedAt:  cfg.CreatedAt,
			UpdatedAt:  cfg.UpdatedAt,
			Name:       cfg.Name,
			Species:    cfg.Species,
			Line:       cfg.Line,
			LineID:     cloneOptionalString(cfg.LineID),
			StrainID:   cloneOptionalString(cfg.StrainID),
			ParentIDs:  append([]string(nil), cfg.ParentIDs...),
			Stage:      domain.LifecycleStage(cfg.Stage),
			CohortID:   cloneOptionalString(cfg.CohortID),
			HousingID:  cloneOptionalString(cfg.HousingID),
			ProtocolID: cloneOptionalString(cfg.ProtocolID),
			ProjectID:  cloneOptionalString(cfg.ProjectID),
		},
	}
	if err := domainOrganism.SetCoreAttributes(cloneAttributes(cfg.Attributes)); err != nil {
		panic(err)
	}

	coreExtensions := domainOrganism.CoreAttributes()
	var extensionSet datasetapi.ExtensionSet
	if len(coreExtensions) > 0 {
		hook := datasetapi.NewExtensionHookContext().OrganismAttributes()
		contributor := datasetapi.NewExtensionContributorContext().Core()
		extensionSet = datasetapi.NewExtensionSet(map[string]map[string]map[string]any{
			hook.String(): {
				contributor.String(): cloneAttributes(coreExtensions),
			},
		})
	} else {
		extensionSet = datasetapi.NewExtensionSet(nil)
	}

	return datasetapi.NewOrganism(datasetapi.OrganismData{
		Base:       cfg.baseData(),
		Name:       domainOrganism.Name,
		Species:    domainOrganism.Species,
		Line:       domainOrganism.Line,
		LineID:     domainOrganism.LineID,
		StrainID:   domainOrganism.StrainID,
		ParentIDs:  append([]string(nil), domainOrganism.ParentIDs...),
		Stage:      datasetapi.LifecycleStage(domainOrganism.Stage),
		CohortID:   domainOrganism.CohortID,
		HousingID:  domainOrganism.HousingID,
		ProtocolID: domainOrganism.ProtocolID,
		ProjectID:  domainOrganism.ProjectID,
		Extensions: extensionSet,
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
		HousingUnit: entitymodel.HousingUnit{
			ID:          cfg.ID,
			CreatedAt:   cfg.CreatedAt,
			UpdatedAt:   cfg.UpdatedAt,
			Name:        cfg.Name,
			FacilityID:  cfg.FacilityID,
			Capacity:    cfg.Capacity,
			Environment: domain.HousingEnvironment(cfg.Environment),
		},
	}

	return datasetapi.NewHousingUnit(datasetapi.HousingUnitData{
		Base:        cfg.baseData(),
		Name:        domainUnit.Name,
		FacilityID:  domainUnit.FacilityID,
		Capacity:    domainUnit.Capacity,
		Environment: string(domainUnit.Environment),
	})
}

func (cfg BaseFixture) baseData() datasetapi.BaseData {
	return datasetapi.BaseData{
		ID:        cfg.ID,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
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