package domain

import "encoding/json"

// ChangePayload wraps a JSON snapshot of a change's before/after state.
// Callers should unmarshal the raw bytes into typed structures as needed.
type ChangePayload struct {
	defined bool
	raw     json.RawMessage
}

// NewChangePayload builds a payload wrapper from raw JSON. The bytes are cloned
// to prevent callers from mutating shared state. Passing a nil slice yields a
// defined but empty payload; use UndefinedChangePayload for "not set".
func NewChangePayload(raw json.RawMessage) ChangePayload {
	payload := ChangePayload{defined: true}
	if raw != nil {
		payload.raw = cloneRawMessage(raw)
	}
	return payload
}

// NewChangePayloadFromValue marshals a typed value into a ChangePayload.
func NewChangePayloadFromValue[T any](value T) (ChangePayload, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return ChangePayload{}, err
	}
	return NewChangePayload(raw), nil
}

// UndefinedChangePayload returns an uninitialized payload wrapper.
func UndefinedChangePayload() ChangePayload {
	return ChangePayload{}
}

// Defined reports whether the payload has been initialized.
func (p ChangePayload) Defined() bool {
	return p.defined
}

// IsEmpty reports whether the payload contains no bytes.
func (p ChangePayload) IsEmpty() bool {
	if !p.defined {
		return true
	}
	return len(p.raw) == 0
}

// Raw returns a cloned copy of the underlying JSON bytes. Nil is returned when
// the payload is undefined or empty.
func (p ChangePayload) Raw() json.RawMessage {
	if !p.defined || len(p.raw) == 0 {
		return nil
	}
	return cloneRawMessage(p.raw)
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}
