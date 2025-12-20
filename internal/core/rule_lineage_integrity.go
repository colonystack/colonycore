package core

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
)

// LineageIntegrityRule enforces parent/offspring and breeding lineage constraints.
func LineageIntegrityRule() domain.Rule {
	return lineageIntegrityRule{}
}

type lineageIntegrityRule struct{}

func (lineageIntegrityRule) Name() string { return "lineage_integrity" }

func (lineageIntegrityRule) Evaluate(_ context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	res := domain.Result{}

	organisms := view.ListOrganisms()
	orgIndex := make(map[string]domain.Organism, len(organisms))
	for _, org := range organisms {
		orgIndex[org.ID] = org
	}

	for _, child := range organisms {
		if len(child.ParentIDs) == 0 {
			continue
		}
		seen := make(map[string]struct{}, len(child.ParentIDs))
		for _, parentID := range child.ParentIDs {
			if parentID == "" {
				continue
			}
			if parentID == child.ID {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s references itself as a parent", child.ID)))
				continue
			}
			if _, dup := seen[parentID]; dup {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s lists parent %s multiple times", child.ID, parentID)))
				continue
			}
			seen[parentID] = struct{}{}

			parent, ok := orgIndex[parentID]
			if !ok {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s references missing parent %s", child.ID, parentID)))
				continue
			}
			if parent.Species != child.Species {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s parent %s has mismatched species", child.ID, parentID)))
			}
			if child.LineID != nil && parent.LineID != nil && *child.LineID != *parent.LineID {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s parent %s has mismatched line", child.ID, parentID)))
			}
			if child.StrainID != nil && parent.StrainID != nil && *child.StrainID != *parent.StrainID {
				res.Violations = append(res.Violations, lineageViolation(child.ID, fmt.Sprintf("organism %s parent %s has mismatched strain", child.ID, parentID)))
			}
		}
	}

	for _, change := range changes {
		if change.Entity != domain.EntityBreeding || change.After == nil {
			continue
		}
		breeding, ok := change.After.(domain.BreedingUnit)
		if !ok {
			continue
		}
		evaluateBreedingUnit(&res, breeding, view)
	}

	return res, nil
}

func lineageViolation(entityID, message string) domain.Violation {
	return domain.Violation{
		Rule:     "lineage_integrity",
		Severity: domain.SeverityBlock,
		Message:  message,
		Entity:   domain.EntityOrganism,
		EntityID: entityID,
	}
}

func evaluateBreedingUnit(res *domain.Result, breeding domain.BreedingUnit, view domain.RuleView) {
	seen := make(map[string]string)
	var speciesRef string

	checkOrganism := func(role, organismID string) {
		if organismID == "" {
			return
		}
		if prevRole, exists := seen[organismID]; exists {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lineage_integrity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("breeding unit %s reuses organism %s as both %s and %s", breeding.ID, organismID, prevRole, role),
				Entity:   domain.EntityBreeding,
				EntityID: breeding.ID,
			})
			return
		}
		seen[organismID] = role

		organism, ok := view.FindOrganism(organismID)
		if !ok {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lineage_integrity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("breeding unit %s references missing organism %s", breeding.ID, organismID),
				Entity:   domain.EntityBreeding,
				EntityID: breeding.ID,
			})
			return
		}
		if speciesRef == "" {
			speciesRef = organism.Species
		} else if organism.Species != speciesRef {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lineage_integrity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("breeding unit %s mixes species %s and %s", breeding.ID, speciesRef, organism.Species),
				Entity:   domain.EntityBreeding,
				EntityID: breeding.ID,
			})
		}
		if breeding.LineID != nil && organism.LineID != nil && *breeding.LineID != *organism.LineID {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lineage_integrity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("breeding unit %s expected line %s for organism %s", breeding.ID, *breeding.LineID, organismID),
				Entity:   domain.EntityBreeding,
				EntityID: breeding.ID,
			})
		}
		if breeding.StrainID != nil && organism.StrainID != nil && *breeding.StrainID != *organism.StrainID {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "lineage_integrity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("breeding unit %s expected strain %s for organism %s", breeding.ID, *breeding.StrainID, organismID),
				Entity:   domain.EntityBreeding,
				EntityID: breeding.ID,
			})
		}
	}

	for _, id := range breeding.FemaleIDs {
		checkOrganism("female", id)
	}
	for _, id := range breeding.MaleIDs {
		checkOrganism("male", id)
	}
}
