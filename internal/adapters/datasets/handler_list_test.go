package datasets

import (
	"encoding/json"
	"fmt"
	"math"
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
		Templates  []datasetapi.TemplateDescriptor `json:"templates"`
		Pagination templatePagination              `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Pagination.TotalItems != 4 {
		t.Fatalf("expected pagination total_items=4, got %+v", payload.Pagination)
	}
	if payload.Pagination.Page != datasetListDefaultPage || payload.Pagination.PageSize != datasetListDefaultPageSize {
		t.Fatalf("unexpected pagination defaults: %+v", payload.Pagination)
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

func TestHandleListTemplatesFiltersByScopeAndPaginates(t *testing.T) {
	h := &Handler{
		Catalog: stubCatalog{
			templates: []datasetapi.TemplateDescriptor{
				{Plugin: "a", Key: "public", Version: "1.0.0"},
				{
					Plugin:  "a",
					Key:     "project",
					Version: "1.0.0",
					Metadata: datasetapi.Metadata{
						Annotations: map[string]string{templateRBACProjectsAnnotation: wildcardScopeValue},
					},
				},
				{
					Plugin:  "a",
					Key:     "restricted",
					Version: "1.0.0",
					Metadata: datasetapi.Metadata{
						Annotations: map[string]string{templateRBACProjectsAnnotation: "project-2"},
					},
				},
			},
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates?page=2&page_size=1", nil)
	req.Header.Set(datasetScopeProjectIDsHeader, "project-1")
	h.handleListTemplates(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Templates  []datasetapi.TemplateDescriptor `json:"templates"`
		Pagination templatePagination              `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Pagination.Page != 2 || payload.Pagination.PageSize != 1 {
		t.Fatalf("unexpected pagination page metadata: %+v", payload.Pagination)
	}
	if payload.Pagination.TotalItems != 2 || payload.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected pagination totals: %+v", payload.Pagination)
	}
	if !payload.Pagination.HasPrev || payload.Pagination.HasNext {
		t.Fatalf("unexpected pagination navigation flags: %+v", payload.Pagination)
	}
	if len(payload.Templates) != 1 || payload.Templates[0].Key != "public" {
		t.Fatalf("expected second filtered template page to return public template, got %+v", payload.Templates)
	}
}

func TestHandleListTemplatesWithoutProjectScopeOmitsProjectTemplates(t *testing.T) {
	h := &Handler{
		Catalog: stubCatalog{
			templates: []datasetapi.TemplateDescriptor{
				{Plugin: "a", Key: "public", Version: "1.0.0"},
				{
					Plugin:  "a",
					Key:     "project",
					Version: "1.0.0",
					Metadata: datasetapi.Metadata{
						Annotations: map[string]string{templateRBACProjectsAnnotation: wildcardScopeValue},
					},
				},
			},
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	h.handleListTemplates(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Templates  []datasetapi.TemplateDescriptor `json:"templates"`
		Pagination templatePagination              `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Pagination.TotalItems != 1 {
		t.Fatalf("expected one visible template, got %+v", payload.Pagination)
	}
	if len(payload.Templates) != 1 || payload.Templates[0].Key != "public" {
		t.Fatalf("expected only public template, got %+v", payload.Templates)
	}
}

func TestHandleListTemplatesRejectsInvalidPagination(t *testing.T) {
	h := &Handler{Catalog: stubCatalog{}}

	cases := []string{
		"/api/v1/datasets/templates?page=0",
		"/api/v1/datasets/templates?page=-1",
		"/api/v1/datasets/templates?page=foo",
		"/api/v1/datasets/templates?page_size=0",
		"/api/v1/datasets/templates?page_size=-5",
		"/api/v1/datasets/templates?page_size=bar",
		"/api/v1/datasets/templates?page_size=201",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			h.handleListTemplates(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d body=%s", path, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPaginateTemplatesHandlesOverflowingPageOffsets(t *testing.T) {
	templates := []datasetapi.TemplateDescriptor{
		{Plugin: "a", Key: "one", Version: "1.0.0"},
		{Plugin: "a", Key: "two", Version: "1.0.0"},
	}

	paged, pagination := paginateTemplates(templates, math.MaxInt, datasetListMaxPageSize)

	if len(paged) != 0 {
		t.Fatalf("expected empty page for overflowing offset, got %+v", paged)
	}
	if pagination.Page != math.MaxInt || pagination.PageSize != datasetListMaxPageSize {
		t.Fatalf("unexpected pagination metadata: %+v", pagination)
	}
	if pagination.TotalItems != len(templates) {
		t.Fatalf("expected total_items=%d, got %+v", len(templates), pagination)
	}
}
