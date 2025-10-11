package datasets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

// TestHandleRunNotAcceptable requests unsupported format to trigger 406.
func TestHandleRunNotAcceptable(t *testing.T) {
	tpl := buildTemplate()
	tpl.OutputFormats = []datasetapi.Format{core.FormatJSON} // no CSV support
	core.BindTemplateForTests(&tpl, func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
		return datasetapi.RunResult{}, nil
	})
	h := NewHandler(testCatalog{tpl: tpl})
	d := tpl.Descriptor()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run?format=csv", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotAcceptable {
		t.Fatalf("expected 406 got %d", w.Code)
	}
}
