package memory

import (
	"errors"
	"testing"
)

type failingPayload struct{}

func (failingPayload) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failure")
}

func TestChangePayloadFromValueSuccess(t *testing.T) {
	tx := &transaction{}
	payload := changePayloadFromValue(tx, map[string]string{"id": "o1"})
	if tx.err != nil {
		t.Fatalf("expected no transaction error, got %v", tx.err)
	}
	if !payload.Defined() {
		t.Fatalf("expected payload to be defined")
	}
	if payload.Raw() == nil {
		t.Fatalf("expected payload raw bytes to be set")
	}
}

func TestChangePayloadFromValueFailure(t *testing.T) {
	tx := &transaction{}
	payload := changePayloadFromValue(tx, failingPayload{})
	if tx.err == nil {
		t.Fatalf("expected transaction error to be set")
	}
	if payload.Defined() {
		t.Fatalf("expected payload to be undefined on marshal failure")
	}
	if payload.Raw() != nil {
		t.Fatalf("expected payload raw bytes to be nil on marshal failure")
	}
}
