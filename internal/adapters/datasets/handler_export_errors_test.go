package datasets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

// TestHandleExportCreateErrorBranches exercises missing slug and unsupported format paths.
func TestHandleExportCreateErrorBranches(t *testing.T) {
	tpl := buildTemplate()
	cat := testCatalog{tpl: tpl}
	wkr := NewWorker(cat, NewMemoryObjectStore(), &memAudit{})
	wkr.Start()
	defer func() { _ = wkr.Stop(context.Background()) }()
	h := &Handler{Catalog: cat, Exports: wkr}

	cases := []struct{ body, want string }{
		{`{}`, "template slug or plugin/key/version required"},
		{`{"template":{"slug":"` + tpl.Descriptor().Slug + `"},"formats":["weird"]}`, "unsupported export format"},
	}
	for i, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", strings.NewReader(c.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), c.want) {
			t.Fatalf("case %d expected 400 containing %q got %d body=%s", i, c.want, w.Code, w.Body.String())
		}
	}
}

// TestHandleExportCreateDedupFormats ensures duplicate formats accepted and deduplicated.
func TestHandleExportCreateDedupFormats(t *testing.T) {
	tpl := buildTemplate()
	tpl.OutputFormats = []datasetapi.Format{core.FormatJSON, core.FormatCSV}
	cat := testCatalog{tpl: tpl}
	wkr := NewWorker(cat, NewMemoryObjectStore(), &memAudit{})
	wkr.Start()
	defer func() { _ = wkr.Stop(context.Background()) }()
	h := &Handler{Catalog: cat, Exports: wkr}
	body := `{"template":{"slug":"` + tpl.Descriptor().Slug + `"},"formats":["json","JSON","csv","csv"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Export struct {
			Formats []datasetapi.Format `json:"formats"`
			ID      string              `json:"id"`
		} `json:"export"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || len(resp.Export.Formats) != 2 {
		t.Fatalf("unexpected formats resp=%s err=%v", w.Body.String(), err)
	}
	// wait for completion
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, ok := wkr.GetExport(resp.Export.ID)
		if ok && (rec.Status == ExportStatusSucceeded || rec.Status == ExportStatusFailed) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("export did not finish")
}

// TestHandleValidateInvalidJSON ensures decode error path.
func TestHandleValidateInvalidJSON(t *testing.T) {
	tpl := buildTemplate()
	h := NewHandler(testCatalog{tpl: tpl})
	d := tpl.Descriptor()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/validate", strings.NewReader("{"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

// TestHandleRunParameterValidationError triggers parameter type coercion failure path.
func TestHandleRunParameterValidationError(t *testing.T) {
	tpl := buildTemplate()
	tpl.OutputFormats = []datasetapi.Format{core.FormatJSON}
	h := NewHandler(testCatalog{tpl: tpl})
	d := tpl.Descriptor()
	body := `{"parameters":{"limit":"not-number"}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d body=%s", w.Code, w.Body.String())
	}
}
