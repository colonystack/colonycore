package core

import (
	"context"
	"fmt"

	"colonycore/pkg/datasetapi"
)

// BindTemplateForTests attaches a runner to the template for use in cross-package tests.
// It is intentionally exported only for test scenarios; production code should
// rely on plugin registration and internal binding logic.
func BindTemplateForTests(t *DatasetTemplate, runner DatasetRunner) {
	if t == nil {
		return
	}
	t.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return runner, nil
	}
	if err := t.bind(DatasetEnvironment{}); err != nil {
		panic(fmt.Sprintf("bind template for tests: %v", err))
	}
}

// RunnerForTests is an internal helper to allow external test helper packages to bind runners without exposing dataset internals broadly.
func (t *DatasetTemplate) RunnerForTests(r func(context.Context, DatasetRunRequest) (DatasetRunResult, error)) {
	if t == nil {
		return
	}
	BindTemplateForTests(t, DatasetRunner(r))
}

// DatasetTemplateRuntimeForTests adapts a DatasetTemplate into the dataset API runtime facade for external tests.
func DatasetTemplateRuntimeForTests(t DatasetTemplate) datasetapi.TemplateRuntime {
	return newDatasetTemplateRuntime(t)
}
