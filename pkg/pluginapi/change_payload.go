package pluginapi

import "encoding/json"

// ChangePayload wraps a JSON snapshot of a change's before/after state.
// Callers should unmarshal the raw bytes into typed structures as needed.
type ChangePayload struct {
	defined bool
	raw     json.RawMessage
}

// NewChangePayload builds a payload wrapper from raw JSON. The bytes are cloned
// to prevent callers from mutating shared state. Passing a nil slice yields a
// NewChangePayload creates a ChangePayload marked as defined from the provided JSON snapshot.
// It makes a defensive copy of `raw` to prevent external mutation. If `raw` is nil, the result
// is a defined-but-empty payload (use UndefinedChangePayload to represent "not set").
func NewChangePayload(raw json.RawMessage) ChangePayload {
	payload := ChangePayload{defined: true}
	if raw != nil {
		payload.raw = cloneRawMessage(raw)
	}
	return payload
}

// UndefinedChangePayload returns an uninitialized ChangePayload that represents a payload that was not provided.
// The returned value is the zero value; its Defined method reports false.
func UndefinedChangePayload() ChangePayload {
	return ChangePayload{}
}

// Defined reports whether the payload was provided, including a defined-but-empty
// payload. Use this to distinguish "undefined" from "defined but empty".
func (p ChangePayload) Defined() bool {
	return p.defined
}

// IsEmpty reports whether the payload contains no bytes. This returns true for
// both undefined payloads and defined-but-empty payloads; use Defined to check
// whether a payload was provided.
func (p ChangePayload) IsEmpty() bool {
	if !p.defined {
		return true
	}
	return len(p.raw) == 0
}

// Raw returns a cloned copy of the underlying JSON bytes. Nil is returned when
// the payload is undefined or empty. Callers that need to distinguish undefined
// from defined-but-empty should check Defined first; internal callers that need a
// clone even when empty should use cloneRawMessage directly.
func (p ChangePayload) Raw() json.RawMessage {
	if !p.defined || len(p.raw) == 0 {
		return nil
	}
	return cloneRawMessage(p.raw)
}

// cloneRawMessage creates a copy of the provided json.RawMessage.
// If raw is nil, it returns nil; otherwise it returns a newly allocated json.RawMessage
// containing the same bytes.
func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

// cloneChangePayload returns a copy of payload with its raw bytes deep-cloned.
// If payload is undefined, it returns an undefined (zero) ChangePayload.
// If payload is defined, the returned payload has defined set to true and raw set to a cloned copy of the original.
func cloneChangePayload(payload ChangePayload) ChangePayload {
	if !payload.defined {
		return ChangePayload{}
	}
	return ChangePayload{defined: true, raw: cloneRawMessage(payload.raw)}
}