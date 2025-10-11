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

type testCatalog struct{ tpl core.DatasetTemplate }

func (c testCatalog) DatasetTemplates() []datasetapi.TemplateDescriptor {
	runtime := core.DatasetTemplateRuntimeForTests(c.tpl)
	return []datasetapi.TemplateDescriptor{runtime.Descriptor()}
}
func (c testCatalog) ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool) {
	runtime := core.DatasetTemplateRuntimeForTests(c.tpl)
	if runtime.Descriptor().Slug == slug {
		return runtime, true
	}
	return nil, false
}

func TestHandlerListAndGetTemplate(t *testing.T) {
	tpl := buildTemplate()
	h := NewHandler(testCatalog{tpl: tpl})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("list templates status: %d", w.Code)
	}
	var listResp struct {
		Templates []datasetapi.TemplateDescriptor `json:"templates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil || len(listResp.Templates) != 1 {
		t.Fatalf("unexpected list body: %s err=%v", w.Body.String(), err)
	}
	desc := tpl.Descriptor()
	// handler expects path /plugin/key/version (version separated, not slug with @)
	path := "/api/v1/datasets/templates/" + desc.Plugin + "/" + desc.Key + "/" + desc.Version
	r2 := httptest.NewRequest(http.MethodGet, path, nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get template status: %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestHandlerTemplateNotFound(t *testing.T) {
	h := NewHandler(testCatalog{tpl: buildTemplate()})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/missing@1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandlerCatalogMissing(t *testing.T) {
	h := &Handler{}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// exercise export endpoints minimal path (queue + get)
func TestHandlerExportsLifecycle(t *testing.T) {
	tpl := buildTemplate()
	cat := testCatalog{tpl: tpl}
	store := NewMemoryObjectStore()
	wkr := NewWorker(cat, store, &memAudit{})
	wkr.Start()
	defer func() { _ = wkr.Stop(context.Background()) }()
	h := &Handler{Catalog: cat, Exports: wkr}
	// enqueue with proper request schema {"template":{"slug":...},"formats":["json"]}
	body := `{"template":{"slug":"` + tpl.Descriptor().Slug + `"},"formats":["json"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("enqueue status %d body=%s", w.Code, w.Body.String())
	}
	var enqueueResp struct {
		Export struct {
			ID string `json:"id"`
		} `json:"export"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &enqueueResp); err != nil || enqueueResp.Export.ID == "" {
		t.Fatalf("unexpected enqueue response: %s err=%v", w.Body.String(), err)
	}
	id := enqueueResp.Export.ID
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/"+id, nil)
		getW := httptest.NewRecorder()
		h.ServeHTTP(getW, getReq)
		if getW.Code == http.StatusOK && strings.Contains(getW.Body.String(), "succeeded") {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("export did not succeed in time")
}

// TestHandlerRunCSV exercises format negotiation and CSV streaming path.
func TestHandlerRunCSV(t *testing.T) {
	tpl := buildTemplate()
	// restrict formats to JSON+CSV so negotiation picks CSV via query param
	tpl.OutputFormats = []datasetapi.Format{core.FormatJSON, core.FormatCSV}
	h := NewHandler(testCatalog{tpl: tpl})
	// run endpoint path: /api/v1/datasets/templates/{plugin}/{key}/{version}/run?format=csv
	d := tpl.Descriptor()
	body := strings.NewReader(`{"parameters":{}}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run?format=csv", body)
	r.Header.Set("Accept", "text/csv")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected csv content type, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "value") {
		t.Fatalf("expected header row in csv output: %s", w.Body.String())
	}
}

func buildTemplate() core.DatasetTemplate {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	tpl := core.DatasetTemplate{
		Plugin: "frog",
		Template: datasetapi.Template{
			Key:           "fixture",
			Version:       "1",
			Title:         "Fixture",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()},
		},
	}
	core.BindTemplateForTests(&tpl, func(_ context.Context, _ datasetapi.RunRequest) (datasetapi.RunResult, error) {
		return datasetapi.RunResult{
			Schema:      tpl.Columns,
			Rows:        []datasetapi.Row{{"value": "ok"}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      core.FormatJSON,
		}, nil
	})
	return tpl
}
