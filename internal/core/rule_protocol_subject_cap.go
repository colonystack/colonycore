package core

import (
	"context"
	"fmt"
)

// NewProtocolSubjectCapRule ensures protocol subject counts stay within configured bounds.
func NewProtocolSubjectCapRule() Rule {
	return protocolSubjectCapRule{}
}

type protocolSubjectCapRule struct{}

func (protocolSubjectCapRule) Name() string { return "protocol_subject_cap" }

func (protocolSubjectCapRule) Evaluate(_ context.Context, view RuleView, _ []Change) (Result, error) {
	counts := make(map[string]int)
	for _, organism := range view.ListOrganisms() {
		if organism.ProtocolID == nil {
			continue
		}
		counts[*organism.ProtocolID]++
	}

	res := Result{}
	for _, protocol := range view.ListProtocols() {
		if protocol.MaxSubjects <= 0 {
			continue
		}
		if counts[protocol.ID] > protocol.MaxSubjects {
			res.Violations = append(res.Violations, Violation{
				Rule:     "protocol_subject_cap",
				Severity: SeverityBlock,
				Message:  fmt.Sprintf("protocol %s (%s) over subject limit: %d/%d", protocol.Title, protocol.Code, counts[protocol.ID], protocol.MaxSubjects),
				Entity:   EntityProtocol,
				EntityID: protocol.ID,
			})
		}
	}
	return res, nil
}
