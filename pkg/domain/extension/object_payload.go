package extension

import (
	"encoding/json"
	"fmt"
)

// ObjectPayload wraps a JSON object payload stored under a specific hook. It
// defensively clones data on construction and retrieval so callers cannot
// mutate the underlying container state. Instances are considered initialised
// only when constructed through NewObjectPayload or ObjectFromContainer.
type ObjectPayload struct {
	hook        Hook
	values      map[string]any
	initialised bool
}

// NewObjectPayload validates the supplied value against the hook definition
// and returns a cloned payload. Nil values clear the payload for the hook.
func NewObjectPayload(h Hook, value any) (ObjectPayload, error) {
	if err := validateHookPayload(h, value); err != nil {
		return ObjectPayload{}, err
	}
	payload := ObjectPayload{
		hook:        h,
		initialised: true,
	}
	if value == nil {
		return payload, nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return ObjectPayload{}, fmt.Errorf("extension: hook %s expects object payload, got %T", h, value)
	}
	payload.values = CloneMap(m)
	return payload, nil
}

// ObjectFromContainer reads the payload for the given hook/plugin combination
// and returns it as an ObjectPayload. Missing payloads are treated as an
// initialised hook with a nil value.
func ObjectFromContainer(container *Container, hook Hook, plugin PluginID) (ObjectPayload, error) {
	if container == nil {
		return NewObjectPayload(hook, nil)
	}
	value, ok := container.Get(hook, plugin)
	if !ok {
		return NewObjectPayload(hook, nil)
	}
	return NewObjectPayload(hook, value)
}

// Hook returns the hook identifier the payload is bound to.
func (p ObjectPayload) Hook() Hook {
	return p.hook
}

// Defined reports whether the payload was initialised through NewObjectPayload.
func (p ObjectPayload) Defined() bool {
	return p.initialised
}

// IsEmpty reports whether the payload carries no data (nil or empty map). The
// hook must still be considered initialised for the return value to be
// meaningful.
func (p ObjectPayload) IsEmpty() bool {
	if !p.initialised {
		return true
	}
	return len(p.values) == 0
}

// Map returns a deep copy of the payload contents. Nil is returned when the
// payload is empty or uninitialised.
func (p ObjectPayload) Map() map[string]any {
	if !p.initialised || p.values == nil {
		return nil
	}
	return CloneMap(p.values)
}

// MarshalJSON ensures the payload serialises as the underlying map so existing
// wire formats remain stable.
func (p ObjectPayload) MarshalJSON() ([]byte, error) {
	if !p.initialised {
		return []byte("null"), nil
	}
	return json.Marshal(p.values)
}

// ExpectHook verifies the payload was initialised for the provided hook.
func (p ObjectPayload) ExpectHook(expected Hook) error {
	if !p.initialised {
		return fmt.Errorf("extension: payload for hook %s is not initialised", expected)
	}
	if expected == "" {
		return fmt.Errorf("extension: expected hook must be provided")
	}
	if p.hook != expected {
		return fmt.Errorf("extension: payload hook %s does not match expected %s", p.hook, expected)
	}
	return nil
}
