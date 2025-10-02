package core

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
	"context"
	"fmt"
	"testing"
)

// simplePlugin implements pluginapi.Plugin for testing InstallPlugin branches.
type simplePlugin struct {
	name, version string
	register      func(reg *PluginRegistry) error
}

func (p simplePlugin) Name() string    { return p.name }
func (p simplePlugin) Version() string { return p.version }
func (p simplePlugin) Register(reg pluginapi.Registry) error {
	if p.register != nil {
		return p.register(reg.(*PluginRegistry))
	}
	return nil
}

// TestServiceInstallPluginNil covers nil plugin guard.
func TestServiceInstallPluginNil(t *testing.T) {
	svc := NewInMemoryService(NewRulesEngine())
	if _, err := svc.InstallPlugin(nil); err == nil {
		t.Fatalf("expected error for nil plugin")
	}
}

// TestServiceInstallPluginDuplicatePlugin covers duplicate plugin name guard.
func TestServiceInstallPluginDuplicatePlugin(t *testing.T) {
	svc := NewInMemoryService(NewRulesEngine())
	p := simplePlugin{name: "dup", version: "1.0.0"}
	if _, err := svc.InstallPlugin(p); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := svc.InstallPlugin(p); err == nil {
		t.Fatalf("expected duplicate plugin error")
	}
}

// TestServiceInstallPluginDuplicateDatasetSlug covers dataset slug already installed branch.
func TestServiceInstallPluginDuplicateDatasetSlug(t *testing.T) {
	svc := NewInMemoryService(NewRulesEngine())
	binder := func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}
	// register function that registers same dataset twice to trigger duplicate dataset template registration error inside registry
	regFuncDuplicate := func(reg *PluginRegistry) error {
		tmpl := datasetapi.Template{Key: "k", Version: "1", Title: "T", Dialect: datasetapi.DialectSQL, Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "c", Type: "string"}}, OutputFormats: []datasetapi.Format{datasetapi.FormatJSON}, Binder: binder}
		if err := reg.RegisterDatasetTemplate(tmpl); err != nil {
			return err
		}
		if err := reg.RegisterDatasetTemplate(tmpl); err == nil {
			return fmt.Errorf("expected duplicate inside registry")
		}
		return nil
	}
	if _, err := svc.InstallPlugin(simplePlugin{name: "pdup", version: "1", register: regFuncDuplicate}); err != nil {
		t.Fatalf("install plugin expected internal duplicate handling but got error: %v", err)
	}
}
