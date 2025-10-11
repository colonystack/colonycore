package core

import (
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestSortTemplateDescriptors(t *testing.T) {
	collection := []datasetapi.TemplateDescriptor{
		{Plugin: "b", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "beta", Version: "1"},
		{Plugin: "a", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "alpha", Version: "1"},
	}
	datasetapi.SortTemplateDescriptors(collection)
	expected := []datasetapi.TemplateDescriptor{
		{Plugin: "a", Key: "alpha", Version: "1"},
		{Plugin: "a", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "beta", Version: "1"},
		{Plugin: "b", Key: "alpha", Version: "2"},
	}
	for i, want := range expected {
		got := collection[i]
		if got.Plugin != want.Plugin || got.Key != want.Key || got.Version != want.Version {
			t.Fatalf("unexpected ordering at %d: %+v (want %+v)", i, got, want)
		}
	}
}

func TestNoopLoggerImplementsMethods(t *testing.T) {
	t.Helper()
	var logger noopLogger
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
	// No panics or side-effects expected; test ensures coverage of methods.
}
