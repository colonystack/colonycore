package datasetapi

// ExtensionPayload wraps a cloned map payload so dataset consumers access
// immutable extension data.
type ExtensionPayload struct {
	defined bool
	values  map[string]any
}

// NewExtensionPayload builds a payload wrapper from the provided map. The map
// is defensively cloned to prevent callers from mutating shared state. Passing
// nil results in an initialised, empty payload.
func NewExtensionPayload(values map[string]any) ExtensionPayload {
	payload := ExtensionPayload{defined: true}
	if len(values) > 0 {
		payload.values = cloneAttributes(values)
	}
	return payload
}

// UndefinedExtensionPayload returns an uninitialised payload wrapper.
func UndefinedExtensionPayload() ExtensionPayload {
	return ExtensionPayload{}
}

// Defined reports whether the payload has been initialised.
func (p ExtensionPayload) Defined() bool {
	return p.defined
}

// IsEmpty reports whether the payload contains any values.
func (p ExtensionPayload) IsEmpty() bool {
	if !p.defined {
		return true
	}
	return len(p.values) == 0
}

// Map returns a cloned representation of the payload contents.
func (p ExtensionPayload) Map() map[string]any {
	if !p.defined || len(p.values) == 0 {
		return nil
	}
	return cloneAttributes(p.values)
}
