package core

import (
	"context"
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestDatasetTestHelpers(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	t.Run("BindTemplateForTests handles nil template", func(_ *testing.T) {
		// Should not panic with nil template
		BindTemplateForTests(nil, nil)
	})

	t.Run("BindTemplateForTests binds runner successfully", func(t *testing.T) {
		template := &DatasetTemplate{
			Plugin: "test-plugin",
			Template: datasetapi.Template{
				Key:           "test",
				Title:         "Test Template",
				Version:       "1.0.0",
				Dialect:       dialectProvider.SQL(),
				Query:         "SELECT 1",
				Columns:       []datasetapi.Column{{Name: "test", Type: "integer"}},
				OutputFormats: []datasetapi.Format{formatProvider.JSON()},
			},
		}

		runner := func(_ context.Context, _ DatasetRunRequest) (DatasetRunResult, error) {
			return DatasetRunResult{}, nil
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BindTemplateForTests panicked: %v", r)
			}
		}()
		BindTemplateForTests(template, DatasetRunner(runner))

		// Template should now have a binder
		if template.Binder == nil {
			t.Error("Expected template to have binder after BindTemplateForTests")
		}
	})

	t.Run("RunnerForTests handles nil template", func(_ *testing.T) {
		var template *DatasetTemplate
		// Should not panic with nil template
		template.RunnerForTests(nil)
	})

	t.Run("RunnerForTests sets runner successfully", func(t *testing.T) {
		template := &DatasetTemplate{
			Plugin: "test-plugin",
			Template: datasetapi.Template{
				Key:           "test",
				Title:         "Test Template",
				Version:       "1.0.0",
				Dialect:       dialectProvider.SQL(),
				Query:         "SELECT 1",
				Columns:       []datasetapi.Column{{Name: "test", Type: "integer"}},
				OutputFormats: []datasetapi.Format{formatProvider.JSON()},
			},
		}

		runner := func(_ context.Context, _ DatasetRunRequest) (DatasetRunResult, error) {
			return DatasetRunResult{}, nil
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RunnerForTests panicked: %v", r)
			}
		}()
		template.RunnerForTests(runner)
	})

	t.Run("DatasetTemplateRuntimeForTests returns runtime", func(t *testing.T) {
		template := DatasetTemplate{
			Plugin: "test-plugin",
			Template: datasetapi.Template{
				Key:           "test",
				Title:         "Test Template",
				Version:       "1.0.0",
				Dialect:       dialectProvider.SQL(),
				Query:         "SELECT 1",
				Columns:       []datasetapi.Column{{Name: "test", Type: "integer"}},
				OutputFormats: []datasetapi.Format{formatProvider.JSON()},
			},
		}

		// Bind the template first like the other helpers do
		runner := func(_ context.Context, _ DatasetRunRequest) (DatasetRunResult, error) {
			return DatasetRunResult{}, nil
		}
		BindTemplateForTests(&template, DatasetRunner(runner))

		runtime := DatasetTemplateRuntimeForTests(template)
		if runtime == nil {
			t.Error("Expected DatasetTemplateRuntimeForTests to return non-nil runtime")
		}

		// Check that runtime has the correct descriptor
		descriptor := runtime.Descriptor()
		if descriptor.Key != "test" {
			t.Errorf("Expected descriptor key 'test', got '%s'", descriptor.Key)
		}
	})
}
