package core

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
)

// ProtocolCoverageRule enforces that procedures and treatments operate under approved protocols.
func ProtocolCoverageRule() domain.Rule {
	return protocolCoverageRule{}
}

type protocolCoverageRule struct{}

func (protocolCoverageRule) Name() string { return "protocol_coverage" }

func (protocolCoverageRule) Evaluate(_ context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	res := domain.Result{}
	protocols := make(map[string]domain.Protocol)
	for _, proto := range view.ListProtocols() {
		protocols[proto.ID] = proto
	}

	for _, change := range changes {
		switch change.Entity {
		case domain.EntityProcedure:
			if change.After == nil {
				continue
			}
			proc, ok := change.After.(domain.Procedure)
			if !ok {
				continue
			}
			validateProcedureCoverage(&res, proc, protocols, view)
		case domain.EntityTreatment:
			if change.After == nil {
				continue
			}
			treatment, ok := change.After.(domain.Treatment)
			if !ok {
				continue
			}
			validateTreatmentCoverage(&res, treatment, protocols, view)
		}
	}

	return res, nil
}

func validateProcedureCoverage(res *domain.Result, proc domain.Procedure, protocols map[string]domain.Protocol, view domain.RuleView) {
	if proc.ProtocolID == "" {
		res.Violations = append(res.Violations, protocolViolation(proc.ID, "procedure is missing required protocol", domain.EntityProcedure))
		return
	}
	proto, ok := protocols[proc.ProtocolID]
	if !ok {
		res.Violations = append(res.Violations, protocolViolation(proc.ID, fmt.Sprintf("procedure references unknown protocol %s", proc.ProtocolID), domain.EntityProcedure))
		return
	}
	if proto.Status != domain.ProtocolStatusApproved {
		res.Violations = append(res.Violations, protocolViolation(proc.ID, fmt.Sprintf("procedure protocol %s is not approved", proto.ID), domain.EntityProcedure))
	}
	for _, organismID := range proc.OrganismIDs {
		organism, ok := view.FindOrganism(organismID)
		if !ok {
			res.Violations = append(res.Violations, protocolViolation(proc.ID, fmt.Sprintf("procedure references unknown organism %s", organismID), domain.EntityProcedure))
			continue
		}
		if organism.ProtocolID == nil || *organism.ProtocolID != proc.ProtocolID {
			res.Violations = append(res.Violations, protocolViolation(proc.ID, fmt.Sprintf("organism %s is not covered by protocol %s", organismID, proc.ProtocolID), domain.EntityProcedure))
		}
	}
}

func validateTreatmentCoverage(res *domain.Result, treatment domain.Treatment, protocols map[string]domain.Protocol, view domain.RuleView) {
	if treatment.ProcedureID == "" {
		res.Violations = append(res.Violations, protocolViolation(treatment.ID, "treatment is missing procedure reference", domain.EntityTreatment))
		return
	}
	procedure, ok := view.FindProcedure(treatment.ProcedureID)
	if !ok {
		res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("treatment references unknown procedure %s", treatment.ProcedureID), domain.EntityTreatment))
		return
	}
	if procedure.ProtocolID == "" {
		res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("procedure %s lacks protocol for treatment", procedure.ID), domain.EntityTreatment))
		return
	}
	proto, ok := protocols[procedure.ProtocolID]
	if !ok {
		res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("treatment references procedure %s with unknown protocol %s", procedure.ID, procedure.ProtocolID), domain.EntityTreatment))
		return
	}
	if proto.Status != domain.ProtocolStatusApproved {
		res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("procedure %s protocol %s is not approved", procedure.ID, proto.ID), domain.EntityTreatment))
	}
	for _, organismID := range treatment.OrganismIDs {
		organism, ok := view.FindOrganism(organismID)
		if !ok {
			res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("treatment references unknown organism %s", organismID), domain.EntityTreatment))
			continue
		}
		if organism.ProtocolID == nil || *organism.ProtocolID != procedure.ProtocolID {
			res.Violations = append(res.Violations, protocolViolation(treatment.ID, fmt.Sprintf("organism %s is not covered by protocol %s", organismID, procedure.ProtocolID), domain.EntityTreatment))
		}
	}
}

func protocolViolation(entityID, message string, entity domain.EntityType) domain.Violation {
	return domain.Violation{
		Rule:     "protocol_coverage",
		Severity: domain.SeverityBlock,
		Message:  message,
		Entity:   entity,
		EntityID: entityID,
	}
}
