package core

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
)

// LifecycleTransitionRule blocks illegal state transitions on stateful entities.
func LifecycleTransitionRule() domain.Rule {
	return lifecycleTransitionRule{}
}

type lifecycleTransitionRule struct{}

type lifecycleMachine struct {
	entity    domain.EntityType
	label     string
	terminal  map[string]struct{}
	valid     map[string]struct{}
	extractor func(model any) (id string, state string, ok bool)
}

var lifecycleMachines = map[domain.EntityType]lifecycleMachine{
	domain.EntityOrganism: {
		entity:   domain.EntityOrganism,
		label:    "organism",
		terminal: toSet(string(domain.StageRetired), string(domain.StageDeceased)),
		valid: toSet(
			string(domain.StagePlanned),
			string(domain.StageLarva),
			string(domain.StageJuvenile),
			string(domain.StageAdult),
			string(domain.StageRetired),
			string(domain.StageDeceased),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Organism)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Stage), true
		},
	},
	domain.EntityHousingUnit: {
		entity:   domain.EntityHousingUnit,
		label:    "housing unit",
		terminal: toSet(string(domain.HousingStateDecommissioned)),
		valid: toSet(
			string(domain.HousingStateQuarantine),
			string(domain.HousingStateActive),
			string(domain.HousingStateCleaning),
			string(domain.HousingStateDecommissioned),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.HousingUnit)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.State), true
		},
	},
	domain.EntityProcedure: {
		entity:   domain.EntityProcedure,
		label:    "procedure",
		terminal: toSet(string(domain.ProcedureStatusCompleted), string(domain.ProcedureStatusCancelled), string(domain.ProcedureStatusFailed)),
		valid: toSet(
			string(domain.ProcedureStatusScheduled),
			string(domain.ProcedureStatusInProgress),
			string(domain.ProcedureStatusCompleted),
			string(domain.ProcedureStatusCancelled),
			string(domain.ProcedureStatusFailed),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Procedure)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Status), true
		},
	},
	domain.EntityTreatment: {
		entity:   domain.EntityTreatment,
		label:    "treatment",
		terminal: toSet(string(domain.TreatmentStatusCompleted), string(domain.TreatmentStatusFlagged)),
		valid: toSet(
			string(domain.TreatmentStatusPlanned),
			string(domain.TreatmentStatusInProgress),
			string(domain.TreatmentStatusCompleted),
			string(domain.TreatmentStatusFlagged),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Treatment)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Status), true
		},
	},
	domain.EntityProtocol: {
		entity:   domain.EntityProtocol,
		label:    "protocol",
		terminal: toSet(string(domain.ProtocolStatusExpired), string(domain.ProtocolStatusArchived)),
		valid: toSet(
			string(domain.ProtocolStatusDraft),
			string(domain.ProtocolStatusSubmitted),
			string(domain.ProtocolStatusApproved),
			string(domain.ProtocolStatusOnHold),
			string(domain.ProtocolStatusExpired),
			string(domain.ProtocolStatusArchived),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Protocol)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Status), true
		},
	},
	domain.EntityPermit: {
		entity:   domain.EntityPermit,
		label:    "permit",
		terminal: toSet(string(domain.PermitStatusExpired), string(domain.PermitStatusArchived)),
		valid: toSet(
			string(domain.PermitStatusDraft),
			string(domain.PermitStatusSubmitted),
			string(domain.PermitStatusApproved),
			string(domain.PermitStatusOnHold),
			string(domain.PermitStatusExpired),
			string(domain.PermitStatusArchived),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Permit)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Status), true
		},
	},
	domain.EntitySample: {
		entity:   domain.EntitySample,
		label:    "sample",
		terminal: toSet(string(domain.SampleStatusConsumed), string(domain.SampleStatusDisposed)),
		valid: toSet(
			string(domain.SampleStatusStored),
			string(domain.SampleStatusInTransit),
			string(domain.SampleStatusConsumed),
			string(domain.SampleStatusDisposed),
		),
		extractor: func(model any) (string, string, bool) {
			cast, ok := model.(domain.Sample)
			if !ok {
				return "", "", false
			}
			return cast.ID, string(cast.Status), true
		},
	},
}

func (lifecycleTransitionRule) Name() string { return "lifecycle_transition" }

func (lifecycleTransitionRule) Evaluate(_ context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	_ = view // view not needed for lifecycle evaluation today
	res := domain.Result{}
	for _, change := range changes {
		machine, ok := lifecycleMachines[change.Entity]
		if !ok {
			continue
		}

		if change.After != nil {
			if _, newState, ok := machine.extractor(change.After); ok {
				if _, valid := machine.valid[newState]; !valid {
					id := entityID(change.After)
					res.Violations = append(res.Violations, domain.Violation{
						Rule:     "lifecycle_transition",
						Severity: domain.SeverityBlock,
						Message:  fmt.Sprintf("%s %s is set to invalid state %s", machine.label, id, newState),
						Entity:   machine.entity,
						EntityID: id,
					})
					continue
				}
			}
		}

		if change.Before == nil || change.After == nil {
			continue
		}
		beforeID, beforeState, ok := machine.extractor(change.Before)
		if !ok {
			continue
		}
		if _, ok := machine.terminal[beforeState]; !ok {
			continue
		}
		afterID, afterState, ok := machine.extractor(change.After)
		if !ok {
			continue
		}
		if afterState != beforeState {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lifecycle_transition",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("cannot move %s %s from terminal state %s to %s", machine.label, beforeID, beforeState, afterState),
				Entity:   machine.entity,
				EntityID: afterID,
			})
		}
	}
	return res, nil
}

func toSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

func entityID(model any) string {
	switch v := model.(type) {
	case domain.Organism:
		return v.ID
	case domain.HousingUnit:
		return v.ID
	case domain.Procedure:
		return v.ID
	case domain.Treatment:
		return v.ID
	case domain.Protocol:
		return v.ID
	case domain.Permit:
		return v.ID
	case domain.Sample:
		return v.ID
	default:
		return ""
	}
}
