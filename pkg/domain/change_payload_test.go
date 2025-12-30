package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

type failingPayload struct{}

func (failingPayload) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failure")
}

func TestChangePayloadDefinedAndEmpty(t *testing.T) {
	undefined := UndefinedChangePayload()
	if undefined.Defined() {
		t.Fatalf("expected undefined payload to be not defined")
	}
	if !undefined.IsEmpty() {
		t.Fatalf("expected undefined payload to be empty")
	}
	if undefined.Raw() != nil {
		t.Fatalf("expected undefined payload to return nil raw bytes")
	}

	empty := NewChangePayload(nil)
	if !empty.Defined() {
		t.Fatalf("expected empty payload to be defined")
	}
	if !empty.IsEmpty() {
		t.Fatalf("expected empty payload to be empty")
	}
	if empty.Raw() != nil {
		t.Fatalf("expected empty payload to return nil raw bytes")
	}

	raw := json.RawMessage(`{"id":"123"}`)
	defined := NewChangePayload(raw)
	if !defined.Defined() {
		t.Fatalf("expected raw payload to be defined")
	}
	if defined.IsEmpty() {
		t.Fatalf("expected raw payload to be non-empty")
	}
	if got := defined.Raw(); string(got) != string(raw) {
		t.Fatalf("expected raw payload %s, got %s", raw, got)
	}
}

func TestChangePayloadRawIsCloned(t *testing.T) {
	raw := json.RawMessage(`{"id":"cloned"}`)
	payload := NewChangePayload(raw)
	raw[2] = 'X'

	first := payload.Raw()
	first[2] = 'Y'
	second := payload.Raw()
	if string(first) == string(second) {
		t.Fatalf("expected raw payload to be cloned per call")
	}
	if string(second) != `{"id":"cloned"}` {
		t.Fatalf("expected stored payload to remain unchanged, got %s", second)
	}
}

func TestNewChangePayloadFromValue(t *testing.T) {
	payload, err := NewChangePayloadFromValue(map[string]string{"id": "123"})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	if !payload.Defined() {
		t.Fatalf("expected payload to be defined")
	}
	if payload.IsEmpty() {
		t.Fatalf("expected payload to be non-empty")
	}
	var out map[string]string
	if err := json.Unmarshal(payload.Raw(), &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if out["id"] != "123" {
		t.Fatalf("expected id 123, got %s", out["id"])
	}

	if _, err := NewChangePayloadFromValue(failingPayload{}); err == nil {
		t.Fatalf("expected marshal error for failing payload")
	}
}
