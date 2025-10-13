package core

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

func baseDataFromDomain(base domain.Base) datasetapi.BaseData {
	return datasetapi.BaseData{
		ID:        base.ID,
		CreatedAt: base.CreatedAt,
		UpdatedAt: base.UpdatedAt,
	}
}

func facadeOrganismFromDomain(org domain.Organism) datasetapi.Organism {
	return datasetapi.NewOrganism(datasetapi.OrganismData{
		Base:       baseDataFromDomain(org.Base),
		Name:       org.Name,
		Species:    org.Species,
		Line:       org.Line,
		Stage:      datasetapi.LifecycleStage(org.Stage),
		CohortID:   org.CohortID,
		HousingID:  org.HousingID,
		ProtocolID: org.ProtocolID,
		ProjectID:  org.ProjectID,
		Attributes: org.Attributes,
	})
}

func facadeOrganismsFromDomain(orgs []domain.Organism) []datasetapi.Organism {
	if len(orgs) == 0 {
		return nil
	}
	out := make([]datasetapi.Organism, len(orgs))
	for i := range orgs {
		out[i] = facadeOrganismFromDomain(orgs[i])
	}
	return out
}

func facadeHousingUnitFromDomain(unit domain.HousingUnit) datasetapi.HousingUnit {
	return datasetapi.NewHousingUnit(datasetapi.HousingUnitData{
		Base:        baseDataFromDomain(unit.Base),
		Name:        unit.Name,
		FacilityID:  unit.FacilityID,
		Capacity:    unit.Capacity,
		Environment: unit.Environment,
	})
}

func facadeHousingUnitsFromDomain(units []domain.HousingUnit) []datasetapi.HousingUnit {
	if len(units) == 0 {
		return nil
	}
	out := make([]datasetapi.HousingUnit, len(units))
	for i := range units {
		out[i] = facadeHousingUnitFromDomain(units[i])
	}
	return out
}

func facadeProtocolFromDomain(protocol domain.Protocol) datasetapi.Protocol {
	return datasetapi.NewProtocol(datasetapi.ProtocolData{
		Base:        baseDataFromDomain(protocol.Base),
		Code:        protocol.Code,
		Title:       protocol.Title,
		Description: protocol.Description,
		MaxSubjects: protocol.MaxSubjects,
	})
}

func facadeProtocolsFromDomain(protocols []domain.Protocol) []datasetapi.Protocol {
	if len(protocols) == 0 {
		return nil
	}
	out := make([]datasetapi.Protocol, len(protocols))
	for i := range protocols {
		out[i] = facadeProtocolFromDomain(protocols[i])
	}
	return out
}

func facadeProjectFromDomain(project domain.Project) datasetapi.Project {
	return datasetapi.NewProject(datasetapi.ProjectData{
		Base:        baseDataFromDomain(project.Base),
		Code:        project.Code,
		Title:       project.Title,
		Description: project.Description,
	})
}

func facadeProjectsFromDomain(projects []domain.Project) []datasetapi.Project {
	if len(projects) == 0 {
		return nil
	}
	out := make([]datasetapi.Project, len(projects))
	for i := range projects {
		out[i] = facadeProjectFromDomain(projects[i])
	}
	return out
}

func facadeCohortFromDomain(cohort domain.Cohort) datasetapi.Cohort {
	return datasetapi.NewCohort(datasetapi.CohortData{
		Base:       baseDataFromDomain(cohort.Base),
		Name:       cohort.Name,
		Purpose:    cohort.Purpose,
		ProjectID:  cohort.ProjectID,
		HousingID:  cohort.HousingID,
		ProtocolID: cohort.ProtocolID,
	})
}

func facadeCohortsFromDomain(cohorts []domain.Cohort) []datasetapi.Cohort {
	if len(cohorts) == 0 {
		return nil
	}
	out := make([]datasetapi.Cohort, len(cohorts))
	for i := range cohorts {
		out[i] = facadeCohortFromDomain(cohorts[i])
	}
	return out
}

func facadeBreedingUnitFromDomain(unit domain.BreedingUnit) datasetapi.BreedingUnit {
	return datasetapi.NewBreedingUnit(datasetapi.BreedingUnitData{
		Base:       baseDataFromDomain(unit.Base),
		Name:       unit.Name,
		Strategy:   unit.Strategy,
		HousingID:  unit.HousingID,
		ProtocolID: unit.ProtocolID,
		FemaleIDs:  unit.FemaleIDs,
		MaleIDs:    unit.MaleIDs,
	})
}

func facadeBreedingUnitsFromDomain(units []domain.BreedingUnit) []datasetapi.BreedingUnit {
	if len(units) == 0 {
		return nil
	}
	out := make([]datasetapi.BreedingUnit, len(units))
	for i := range units {
		out[i] = facadeBreedingUnitFromDomain(units[i])
	}
	return out
}

func facadeProcedureFromDomain(proc domain.Procedure) datasetapi.Procedure {
	return datasetapi.NewProcedure(datasetapi.ProcedureData{
		Base:        baseDataFromDomain(proc.Base),
		Name:        proc.Name,
		Status:      proc.Status,
		ScheduledAt: proc.ScheduledAt,
		ProtocolID:  proc.ProtocolID,
		CohortID:    proc.CohortID,
		OrganismIDs: proc.OrganismIDs,
	})
}

func facadeProceduresFromDomain(procs []domain.Procedure) []datasetapi.Procedure {
	if len(procs) == 0 {
		return nil
	}
	out := make([]datasetapi.Procedure, len(procs))
	for i := range procs {
		out[i] = facadeProcedureFromDomain(procs[i])
	}
	return out
}
