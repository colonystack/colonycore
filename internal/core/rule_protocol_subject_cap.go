package core

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
)

// NewProtocolSubjectCapRule ensures protocol subject counts stay within configured bounds.
func NewProtocolSubjectCapRule() domain.Rule {
	return protocolSubjectCapRule{}
}

type protocolSubjectCapRule struct{}

func (protocolSubjectCapRule) Name() string { return "protocol_subject_cap" }

func (protocolSubjectCapRule) Evaluate(_ context.Context, view domain.RuleView, _ []domain.Change) (domain.Result, error) {
	counts := make(map[string]int)
	for _, organism := range view.ListOrganisms() {
		if organism.ProtocolID == nil {
			continue
		}
		counts[*organism.ProtocolID]++
	}

	res := domain.Result{}
	for _, protocol := range view.ListProtocols() {
		if protocol.MaxSubjects <= 0 {
			continue
		}
		if counts[protocol.ID] > protocol.MaxSubjects {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "protocol_subject_cap",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("protocol %s (%s) over subject limit: %d/%d", protocol.Title, protocol.Code, counts[protocol.ID], protocol.MaxSubjects),
				Entity:   domain.EntityProtocol,
				EntityID: protocol.ID,
			})
		}
	}
	return res, nil
}
