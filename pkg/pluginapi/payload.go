package pluginapi

// ObjectPayload exposes a defensive view over structured JSON-like payloads
// stored in extension hooks. Callers receive cloned maps so the underlying
// state cannot be mutated.
type ObjectPayload struct {
	defined bool
	values  map[string]any
}

// NewObjectPayload constructs a payload wrapper from the provided values. The
// input map is cloned to prevent callers from mutating shared state. Passing a
// nil map produces an initialised, empty payload.
func NewObjectPayload(values map[string]any) ObjectPayload {
	payload := ObjectPayload{defined: true}
	if values != nil {
		payload.values = cloneMap(values)
	}
	return payload
}

// UndefinedPayload returns a zero-value payload for internal use when no data
// was recorded.
func UndefinedPayload() ObjectPayload {
	return ObjectPayload{}
}

// Defined reports whether the payload is initialised.
func (p ObjectPayload) Defined() bool {
	return p.defined
}

// IsEmpty reports whether the payload carries any data.
func (p ObjectPayload) IsEmpty() bool {
	if !p.defined {
		return true
	}
	return len(p.values) == 0
}

// Map returns a cloned representation of the payload. Nil is returned when the
// payload is undefined or empty.
func (p ObjectPayload) Map() map[string]any {
	if !p.defined || p.values == nil {
		return nil
	}
	return cloneMap(p.values)
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned, _ := cloneValue(values).(map[string]any)
	return cloned
}
