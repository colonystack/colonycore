package core

import "context"

// BindTemplateForTests attaches a runner to the template for use in cross-package tests.
// It is intentionally exported only for test scenarios; production code should
// rely on plugin registration and internal binding logic.
func BindTemplateForTests(t *DatasetTemplate, runner DatasetRunner) {
	if t != nil {
		t.runner = runner
	}
}

// RunnerForTests is an internal helper to allow external test helper packages to bind runners without exposing dataset internals broadly.
func (t *DatasetTemplate) RunnerForTests(r func(context.Context, DatasetRunRequest) (DatasetRunResult, error)) {
	if t != nil {
		t.runner = r
	}
}
