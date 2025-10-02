package datasets

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
)

// testDatasetPlugin is a lightweight plugin used in adapter tests.
type testDatasetPlugin struct{ dataset datasetapi.Template }

func (p testDatasetPlugin) Name() string    { return "test-dataset" }
func (p testDatasetPlugin) Version() string { return "0.0.1" }
func (p testDatasetPlugin) Register(registry pluginapi.Registry) error {
	return registry.RegisterDatasetTemplate(p.dataset)
}
