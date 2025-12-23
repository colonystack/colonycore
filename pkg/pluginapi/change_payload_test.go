package pluginapi

import (
	"bytes"
	"encoding/json"
	"testing"
)

func mustMarshal(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return payload
}

func newPayload(t *testing.T, value any) ChangePayload {
	t.Helper()
	if value == nil {
		return UndefinedChangePayload()
	}
	return NewChangePayload(mustMarshal(t, value))
}

func unmarshalPayload(t *testing.T, payload ChangePayload, target any) {
	t.Helper()
	raw := payload.Raw()
	if raw == nil {
		t.Fatalf("payload is undefined or empty")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
}

func TestChangePayloadClone(t *testing.T) {
	expected := []byte(`{"a":1}`)
	source := json.RawMessage(append([]byte(nil), expected...))
	payload := NewChangePayload(source)

	source[0] = '['
	if got := payload.Raw(); !bytes.Equal(got, expected) {
		t.Fatalf("expected payload to clone input bytes, got %s", string(got))
	}

	got := payload.Raw()
	got[0] = '['
	if next := payload.Raw(); !bytes.Equal(next, expected) {
		t.Fatalf("expected payload to clone raw bytes, got %s", string(next))
	}
}

func TestChangePayloadDefinedAndEmpty(t *testing.T) {
	undefined := UndefinedChangePayload()
	if undefined.Defined() {
		t.Fatalf("undefined payload should not be defined")
	}
	if !undefined.IsEmpty() {
		t.Fatalf("undefined payload should be empty")
	}
	if undefined.Raw() != nil {
		t.Fatalf("undefined payload should return nil raw")
	}

	definedEmpty := NewChangePayload(nil)
	if !definedEmpty.Defined() {
		t.Fatalf("defined payload should be defined")
	}
	if !definedEmpty.IsEmpty() {
		t.Fatalf("defined empty payload should be empty")
	}
	if definedEmpty.Raw() != nil {
		t.Fatalf("defined empty payload should return nil raw")
	}
}
