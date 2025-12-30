package core

import (
	"fmt"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

func baseDataFromDomain(id string, createdAt, updatedAt time.Time) datasetapi.BaseData {
	return datasetapi.BaseData{
		ID:        id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// facadeOrganismFromDomain converts a domain.Organism into a datasetapi.Organism.
// It populates base metadata, core organism fields (Name, Species, Line, LineID, StrainID, ParentIDs, Stage, CohortID, HousingID, ProtocolID, ProjectID) and builds an ExtensionSet from the organism's extension container; it will panic if retrieving extensions returns an error.
func facadeOrganismFromDomain(org domain.Organism) datasetapi.Organism {
	container, err := org.OrganismExtensions()
	if err != nil {
		panic(fmt.Errorf("core: organism extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewOrganism(datasetapi.OrganismData{
		Base:       baseDataFromDomain(org.ID, org.CreatedAt, org.UpdatedAt),
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
		Extensions: extSet,
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
		Base:        baseDataFromDomain(unit.ID, unit.CreatedAt, unit.UpdatedAt),
		Name:        unit.Name,
		FacilityID:  unit.FacilityID,
		Capacity:    unit.Capacity,
		Environment: string(unit.Environment),
		State:       string(unit.State),
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
		Base:        baseDataFromDomain(protocol.ID, protocol.CreatedAt, protocol.UpdatedAt),
		Code:        protocol.Code,
		Title:       protocol.Title,
		Description: protocol.Description,
		MaxSubjects: protocol.MaxSubjects,
		Status:      string(protocol.Status),
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
		Base:          baseDataFromDomain(project.ID, project.CreatedAt, project.UpdatedAt),
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
		Base:       baseDataFromDomain(cohort.ID, cohort.CreatedAt, cohort.UpdatedAt),
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

// facadeBreedingUnitFromDomain converts a domain.BreedingUnit into a datasetapi.BreedingUnit.
// It copies base metadata and fields (name, strategy, housing/protocol/line/strain IDs, pairing details, and member IDs)
// and constructs the Extensions set from the unit's extension payloads. It panics if retrieving the unit's extensions fails.
func facadeBreedingUnitFromDomain(unit domain.BreedingUnit) datasetapi.BreedingUnit {
	container, err := unit.BreedingUnitExtensions()
	if err != nil {
		panic(fmt.Errorf("core: breeding unit extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewBreedingUnit(datasetapi.BreedingUnitData{
		Base:           baseDataFromDomain(unit.ID, unit.CreatedAt, unit.UpdatedAt),
		Name:           unit.Name,
		Strategy:       unit.Strategy,
		HousingID:      unit.HousingID,
		ProtocolID:     unit.ProtocolID,
		LineID:         unit.LineID,
		StrainID:       unit.StrainID,
		TargetLineID:   unit.TargetLineID,
		TargetStrainID: unit.TargetStrainID,
		PairingIntent:  unit.PairingIntent,
		PairingNotes:   unit.PairingNotes,
		Extensions:     extSet,
		FemaleIDs:      unit.FemaleIDs,
		MaleIDs:        unit.MaleIDs,
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
		Base:           baseDataFromDomain(proc.ID, proc.CreatedAt, proc.UpdatedAt),
		Name:           proc.Name,
		Status:         string(proc.Status),
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

// facadeFacilityFromDomain converts a domain.Facility into a datasetapi.Facility.
//
// The returned Facility contains base metadata (ID, CreatedAt, UpdatedAt), code,
// name, zone, access policy, housing unit and project references, and an
// extension set built from the facility's extension payloads. This function
// panics if retrieving the facility's extensions fails.
func facadeFacilityFromDomain(facility domain.Facility) datasetapi.Facility {
	container, err := facility.FacilityExtensions()
	if err != nil {
		panic(fmt.Errorf("core: facility extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewFacility(datasetapi.FacilityData{
		Base:           baseDataFromDomain(facility.ID, facility.CreatedAt, facility.UpdatedAt),
		Code:           facility.Code,
		Name:           facility.Name,
		Zone:           facility.Zone,
		AccessPolicy:   facility.AccessPolicy,
		Extensions:     extSet,
		HousingUnitIDs: facility.HousingUnitIDs,
		ProjectIDs:     facility.ProjectIDs,
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
		Base:              baseDataFromDomain(treatment.ID, treatment.CreatedAt, treatment.UpdatedAt),
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

// facadeObservationFromDomain converts a domain.Observation into a datasetapi.Observation.
// It maps base metadata and observation fields and attaches an ExtensionSet built from the observation's extension payloads.
// The function panics if retrieving the observation's extensions fails.
func facadeObservationFromDomain(observation domain.Observation) datasetapi.Observation {
	container, err := observation.ObservationExtensions()
	if err != nil {
		panic(fmt.Errorf("core: observation extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewObservation(datasetapi.ObservationData{
		Base:        baseDataFromDomain(observation.ID, observation.CreatedAt, observation.UpdatedAt),
		ProcedureID: observation.ProcedureID,
		OrganismID:  observation.OrganismID,
		CohortID:    observation.CohortID,
		RecordedAt:  observation.RecordedAt,
		Observer:    observation.Observer,
		Extensions:  extSet,
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

// facadeSampleFromDomain converts a domain.Sample into a datasetapi.Sample, mapping its base data,
// identifiers, timestamps, custody events, and building an ExtensionSet from the sample's extension payloads.
// It panics if retrieving the sample's extensions returns an error.
func facadeSampleFromDomain(sample domain.Sample) datasetapi.Sample {
	container, err := sample.SampleExtensions()
	if err != nil {
		panic(fmt.Errorf("core: sample extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewSample(datasetapi.SampleData{
		Base:            baseDataFromDomain(sample.ID, sample.CreatedAt, sample.UpdatedAt),
		Identifier:      sample.Identifier,
		SourceType:      sample.SourceType,
		OrganismID:      sample.OrganismID,
		CohortID:        sample.CohortID,
		FacilityID:      sample.FacilityID,
		CollectedAt:     sample.CollectedAt,
		Status:          string(sample.Status),
		StorageLocation: sample.StorageLocation,
		AssayType:       sample.AssayType,
		ChainOfCustody:  custodyEventsToData(sample.ChainOfCustody),
		Extensions:      extSet,
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
		Base:              baseDataFromDomain(permit.ID, permit.CreatedAt, permit.UpdatedAt),
		PermitNumber:      permit.PermitNumber,
		Authority:         permit.Authority,
		Status:            string(permit.Status),
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

// facadeSupplyItemFromDomain converts a domain.SupplyItem into a datasetapi.SupplyItem.
// The returned value contains the item's base metadata and fields (SKU, name, description,
// quantity, unit, lot number, expiration, facility/project associations, reorder level)
// and an Extensions set built from the domain item's extension payloads.
// It panics if retrieving the supply item extensions fails.
func facadeSupplyItemFromDomain(item domain.SupplyItem) datasetapi.SupplyItem {
	container, err := item.SupplyItemExtensions()
	if err != nil {
		panic(fmt.Errorf("core: supply item extensions: %w", err))
	}
	extSet := datasetapi.NewExtensionSet(mapExtensionPayloads(container.Raw()))
	return datasetapi.NewSupplyItem(datasetapi.SupplyItemData{
		Base:           baseDataFromDomain(item.ID, item.CreatedAt, item.UpdatedAt),
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
		Extensions:     extSet,
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