package core

import (
	"colonycore/pkg/domain"
	"encoding/json"
)

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
