package core

import (
	"colonycore/pkg/domain"
	"encoding/json"
)

// decodeChangePayload decodes a domain.ChangePayload's JSON contents into a value of type T.
// It returns the decoded value and true on success. It returns the zero value and false if
// the payload is not defined, contains no data, or cannot be unmarshaled into T.
func decodeChangePayload[T any](payload domain.ChangePayload) (T, bool) {
	var out T
	if !payload.Defined() {
		return out, false
	}
	raw := payload.Raw()
	if len(raw) == 0 {
		return out, false
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, false
	}
	return out, true
}