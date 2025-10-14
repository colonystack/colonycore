package datasets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"colonycore/pkg/datasetapi"
)

type stubCatalog struct {
	templates []datasetapi.TemplateDescriptor
}

func (s stubCatalog) DatasetTemplates() []datasetapi.TemplateDescriptor {
	out := make([]datasetapi.TemplateDescriptor, len(s.templates))
	copy(out, s.templates)
	return out
}

func (stubCatalog) ResolveDatasetTemplate(string) (datasetapi.TemplateRuntime, bool) {
	return nil, false
}

func TestHandleListTemplatesSortsByPluginKeyVersion(t *testing.T) {
	h := &Handler{
		Catalog: stubCatalog{
			templates: []datasetapi.TemplateDescriptor{
				{Plugin: "b", Key: "alpha", Version: "1.0.0"},
				{Plugin: "a", Key: "beta", Version: "1.0.0"},
				{Plugin: "a", Key: "alpha", Version: "2.0.0"},
				{Plugin: "a", Key: "alpha", Version: "1.0.0"},
			},
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	h.handleListTemplates(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var payload struct {
		Templates []datasetapi.TemplateDescriptor `json:"templates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedOrder := []string{
		"a/alpha@1.0.0",
		"a/alpha@2.0.0",
		"a/beta@1.0.0",
		"b/alpha@1.0.0",
	}
	var gotOrder []string
	for _, tpl := range payload.Templates {
		gotOrder = append(gotOrder, fmt.Sprintf("%s/%s@%s", tpl.Plugin, tpl.Key, tpl.Version))
	}
	if len(gotOrder) != len(expectedOrder) {
		t.Fatalf("expected %d templates, got %d", len(expectedOrder), len(gotOrder))
	}
	for i, want := range expectedOrder {
		if gotOrder[i] != want {
			t.Fatalf("unexpected order at index %d: want %s, got %s", i, want, gotOrder[i])
		}
	}
}
