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
		LineID:     org.LineID,
		StrainID:   org.StrainID,
		ParentIDs:  append([]string(nil), org.ParentIDs...),
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
		Status:      protocol.Status,
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
		Base:          baseDataFromDomain(project.Base),
		Code:          project.Code,
		Title:         project.Title,
		Description:   project.Description,
		FacilityIDs:   project.FacilityIDs,
		ProtocolIDs:   project.ProtocolIDs,
		OrganismIDs:   project.OrganismIDs,
		ProcedureIDs:  project.ProcedureIDs,
		SupplyItemIDs: project.SupplyItemIDs,
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
		Base:              baseDataFromDomain(unit.Base),
		Name:              unit.Name,
		Strategy:          unit.Strategy,
		HousingID:         unit.HousingID,
		ProtocolID:        unit.ProtocolID,
		LineID:            unit.LineID,
		StrainID:          unit.StrainID,
		TargetLineID:      unit.TargetLineID,
		TargetStrainID:    unit.TargetStrainID,
		PairingIntent:     unit.PairingIntent,
		PairingNotes:      unit.PairingNotes,
		PairingAttributes: unit.PairingAttributes,
		FemaleIDs:         unit.FemaleIDs,
		MaleIDs:           unit.MaleIDs,
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
		Base:           baseDataFromDomain(proc.Base),
		Name:           proc.Name,
		Status:         proc.Status,
		ScheduledAt:    proc.ScheduledAt,
		ProtocolID:     proc.ProtocolID,
		ProjectID:      proc.ProjectID,
		CohortID:       proc.CohortID,
		OrganismIDs:    proc.OrganismIDs,
		TreatmentIDs:   proc.TreatmentIDs,
		ObservationIDs: proc.ObservationIDs,
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

func facadeFacilityFromDomain(facility domain.Facility) datasetapi.Facility {
	return datasetapi.NewFacility(datasetapi.FacilityData{
		Base:                 baseDataFromDomain(facility.Base),
		Code:                 facility.Code,
		Name:                 facility.Name,
		Zone:                 facility.Zone,
		AccessPolicy:         facility.AccessPolicy,
		EnvironmentBaselines: facility.EnvironmentBaselines,
		HousingUnitIDs:       facility.HousingUnitIDs,
		ProjectIDs:           facility.ProjectIDs,
	})
}

func facadeFacilitiesFromDomain(facilities []domain.Facility) []datasetapi.Facility {
	if len(facilities) == 0 {
		return nil
	}
	out := make([]datasetapi.Facility, len(facilities))
	for i := range facilities {
		out[i] = facadeFacilityFromDomain(facilities[i])
	}
	return out
}

func facadeTreatmentFromDomain(treatment domain.Treatment) datasetapi.Treatment {
	return datasetapi.NewTreatment(datasetapi.TreatmentData{
		Base:              baseDataFromDomain(treatment.Base),
		Name:              treatment.Name,
		ProcedureID:       treatment.ProcedureID,
		OrganismIDs:       treatment.OrganismIDs,
		CohortIDs:         treatment.CohortIDs,
		DosagePlan:        treatment.DosagePlan,
		AdministrationLog: treatment.AdministrationLog,
		AdverseEvents:     treatment.AdverseEvents,
	})
}

func facadeTreatmentsFromDomain(treatments []domain.Treatment) []datasetapi.Treatment {
	if len(treatments) == 0 {
		return nil
	}
	out := make([]datasetapi.Treatment, len(treatments))
	for i := range treatments {
		out[i] = facadeTreatmentFromDomain(treatments[i])
	}
	return out
}

func facadeObservationFromDomain(observation domain.Observation) datasetapi.Observation {
	return datasetapi.NewObservation(datasetapi.ObservationData{
		Base:        baseDataFromDomain(observation.Base),
		ProcedureID: observation.ProcedureID,
		OrganismID:  observation.OrganismID,
		CohortID:    observation.CohortID,
		RecordedAt:  observation.RecordedAt,
		Observer:    observation.Observer,
		Data:        observation.Data,
		Notes:       observation.Notes,
	})
}

func facadeObservationsFromDomain(observations []domain.Observation) []datasetapi.Observation {
	if len(observations) == 0 {
		return nil
	}
	out := make([]datasetapi.Observation, len(observations))
	for i := range observations {
		out[i] = facadeObservationFromDomain(observations[i])
	}
	return out
}

func facadeSampleFromDomain(sample domain.Sample) datasetapi.Sample {
	return datasetapi.NewSample(datasetapi.SampleData{
		Base:            baseDataFromDomain(sample.Base),
		Identifier:      sample.Identifier,
		SourceType:      sample.SourceType,
		OrganismID:      sample.OrganismID,
		CohortID:        sample.CohortID,
		FacilityID:      sample.FacilityID,
		CollectedAt:     sample.CollectedAt,
		Status:          sample.Status,
		StorageLocation: sample.StorageLocation,
		AssayType:       sample.AssayType,
		ChainOfCustody:  custodyEventsToData(sample.ChainOfCustody),
		Attributes:      sample.Attributes,
	})
}

func facadeSamplesFromDomain(samples []domain.Sample) []datasetapi.Sample {
	if len(samples) == 0 {
		return nil
	}
	out := make([]datasetapi.Sample, len(samples))
	for i := range samples {
		out[i] = facadeSampleFromDomain(samples[i])
	}
	return out
}

func facadePermitFromDomain(permit domain.Permit) datasetapi.Permit {
	return datasetapi.NewPermit(datasetapi.PermitData{
		Base:              baseDataFromDomain(permit.Base),
		PermitNumber:      permit.PermitNumber,
		Authority:         permit.Authority,
		ValidFrom:         permit.ValidFrom,
		ValidUntil:        permit.ValidUntil,
		AllowedActivities: permit.AllowedActivities,
		FacilityIDs:       permit.FacilityIDs,
		ProtocolIDs:       permit.ProtocolIDs,
		Notes:             permit.Notes,
	})
}

func facadePermitsFromDomain(permits []domain.Permit) []datasetapi.Permit {
	if len(permits) == 0 {
		return nil
	}
	out := make([]datasetapi.Permit, len(permits))
	for i := range permits {
		out[i] = facadePermitFromDomain(permits[i])
	}
	return out
}

func facadeSupplyItemFromDomain(item domain.SupplyItem) datasetapi.SupplyItem {
	return datasetapi.NewSupplyItem(datasetapi.SupplyItemData{
		Base:           baseDataFromDomain(item.Base),
		SKU:            item.SKU,
		Name:           item.Name,
		Description:    item.Description,
		QuantityOnHand: item.QuantityOnHand,
		Unit:           item.Unit,
		LotNumber:      item.LotNumber,
		ExpiresAt:      item.ExpiresAt,
		FacilityIDs:    item.FacilityIDs,
		ProjectIDs:     item.ProjectIDs,
		ReorderLevel:   item.ReorderLevel,
		Attributes:     item.Attributes,
	})
}

func facadeSupplyItemsFromDomain(items []domain.SupplyItem) []datasetapi.SupplyItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]datasetapi.SupplyItem, len(items))
	for i := range items {
		out[i] = facadeSupplyItemFromDomain(items[i])
	}
	return out
}

func custodyEventsToData(events []domain.SampleCustodyEvent) []datasetapi.SampleCustodyEventData {
	if len(events) == 0 {
		return nil
	}
	out := make([]datasetapi.SampleCustodyEventData, len(events))
	for i, event := range events {
		out[i] = datasetapi.SampleCustodyEventData{
			Actor:     event.Actor,
			Location:  event.Location,
			Timestamp: event.Timestamp,
			Notes:     event.Notes,
		}
	}
	return out
}
