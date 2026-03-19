package datasets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

func TestHandlerRunCSVUsesChunkedTransferAndProgressTrailer(t *testing.T) {
	tpl := buildTemplate()
	h := NewHandler(testCatalog{tpl: tpl})

	server := httptest.NewServer(h)
	defer server.Close()

	desc := tpl.Descriptor()
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/v1/datasets/templates/"+desc.Plugin+"/"+desc.Key+"/"+desc.Version+"/run?format=csv",
		strings.NewReader(`{"parameters":{}}`),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept", "text/csv")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(resp.TransferEncoding) == 0 || resp.TransferEncoding[0] != "chunked" {
		t.Fatalf("expected chunked transfer encoding, got %+v", resp.TransferEncoding)
	}

	expectedResult, _, err := tpl.Run(context.Background(), nil, datasetapi.Scope{}, core.FormatCSV)
	if err != nil {
		t.Fatalf("run template for expected progress: %v", err)
	}
	expectedInitialProgress := fmt.Sprintf(
		"bytes=0/%d",
		estimateCSVSize(desc.Columns, expectedResult.Rows),
	)
	if got := resp.Header.Get(streamProgressHeader); got != expectedInitialProgress {
		t.Fatalf("expected initial progress %q, got %q", expectedInitialProgress, got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	expectedFinalProgress := fmt.Sprintf("bytes=%d/%d", len(body), len(body))
	if got := resp.Trailer.Get(streamProgressHeader); got != expectedFinalProgress {
		t.Fatalf("expected final progress trailer %q, got %q", expectedFinalProgress, got)
	}
	if got := resp.Trailer.Get(streamErrorTrailer); got != "" {
		t.Fatalf("expected empty stream error trailer, got %q", got)
	}
	if !strings.Contains(string(body), "value") {
		t.Fatalf("expected csv body to contain header row, got %q", string(body))
	}
}
