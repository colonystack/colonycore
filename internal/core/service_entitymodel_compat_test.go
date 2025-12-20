package core

import (
	"colonycore/internal/entitymodel"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
	"context"
	"strings"
	"testing"
	"time"
)

func intPtr(v int) *int {
	return &v
}

type compatTestPlugin struct {
	name             string
	version          string
	entityModelMajor int
	templateMajor    int
}

func (p compatTestPlugin) Name() string    { return p.name }
func (p compatTestPlugin) Version() string { return p.version }

func (p compatTestPlugin) EntityModelMajor() int { return p.entityModelMajor }

func (p compatTestPlugin) Register(reg pluginapi.Registry) error {
	tpl := datasetapi.Template{
		Key:           "compat_dataset",
		Version:       "1.0.0",
		Title:         "compat dataset",
		Dialect:       datasetapi.GetDialectProvider().SQL(),
		Query:         "select 1",
		Columns:       []datasetapi.Column{{Name: "c", Type: "string"}},
		OutputFormats: []datasetapi.Format{datasetapi.GetFormatProvider().JSON()},
		Metadata: datasetapi.Metadata{
			EntityModelMajor: intPtr(p.templateMajor),
		},
		Binder: func(env datasetapi.Environment) (datasetapi.Runner, error) {
			return func(ctx context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
				if ctx == nil {
					ctx = context.Background()
				}
				select {
				case <-ctx.Done():
					return datasetapi.RunResult{}, ctx.Err()
				default:
				}
				return datasetapi.RunResult{
					Schema:      req.Template.Columns,
					Rows:        []datasetapi.Row{{"c": "ok"}},
					GeneratedAt: env.Now(),
					Format:      datasetapi.GetFormatProvider().JSON(),
				}, nil
			}, nil
		},
	}
	return reg.RegisterDatasetTemplate(tpl)
}

var _ pluginapi.Plugin = (*compatTestPlugin)(nil)
var _ pluginapi.EntityModelCompatibilityProvider = (*compatTestPlugin)(nil)

func requireEntityModelMajor(t *testing.T) int {
	t.Helper()
	major, ok := entitymodel.MajorVersion()
	if !ok {
		t.Skip("entity model major version unavailable")
	}
	return major
}

func TestInstallPluginRejectsIncompatiblePluginMajor(t *testing.T) {
	hostMajor := requireEntityModelMajor(t)
	svc := NewInMemoryService(NewDefaultRulesEngine())
	plugin := &compatTestPlugin{
		name:             "incompatible-plugin",
		version:          "0.0.1",
		entityModelMajor: hostMajor + 1,
	}
	if _, err := svc.InstallPlugin(plugin); err == nil || !strings.Contains(err.Error(), "entity model major mismatch") {
		t.Fatalf("expected entity model mismatch error, got %v", err)
	}
}

func TestInstallPluginRejectsIncompatibleTemplateMajor(t *testing.T) {
	hostMajor := requireEntityModelMajor(t)
	svc := NewInMemoryService(NewDefaultRulesEngine())
	plugin := &compatTestPlugin{
		name:             "template-incompatible",
		version:          "0.0.2",
		entityModelMajor: -1,
		templateMajor:    hostMajor + 1,
	}
	if _, err := svc.InstallPlugin(plugin); err == nil || !strings.Contains(err.Error(), "entity model major mismatch") {
		t.Fatalf("expected entity model mismatch error from template, got %v", err)
	}
}

func TestInstallPluginRejectsTemplatePluginMajorDisagreement(t *testing.T) {
	hostMajor := requireEntityModelMajor(t)
	svc := NewInMemoryService(NewDefaultRulesEngine())
	plugin := &compatTestPlugin{
		name:             "template-plugin-disagree",
		version:          "0.0.3",
		entityModelMajor: hostMajor,
		templateMajor:    hostMajor + 1,
	}
	if _, err := svc.InstallPlugin(plugin); err == nil || !strings.Contains(err.Error(), "declares entity model major") {
		t.Fatalf("expected disagreement error, got %v", err)
	}
}

func TestInstallPluginAcceptsCompatibleDeclarations(t *testing.T) {
	hostMajor := requireEntityModelMajor(t)
	svc := NewInMemoryService(NewDefaultRulesEngine(), WithClock(ClockFunc(func() time.Time {
		return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	})))
	plugin := &compatTestPlugin{
		name:             "compatible-plugin",
		version:          "0.0.4",
		entityModelMajor: hostMajor,
		templateMajor:    hostMajor,
	}
	meta, err := svc.InstallPlugin(plugin)
	if err != nil {
		t.Fatalf("expected plugin installation to succeed, got %v", err)
	}
	if len(meta.Datasets) != 1 {
		t.Fatalf("expected one dataset descriptor, got %d", len(meta.Datasets))
	}
	templates := svc.DatasetTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected one template, got %d", len(templates))
	}
	template, ok := svc.ResolveDatasetTemplate(templates[0].Slug)
	if !ok {
		t.Fatalf("expected template resolution to succeed")
	}
	formatProvider := datasetapi.GetFormatProvider()
	result, errs, err := template.Run(context.Background(), nil, datasetapi.Scope{}, formatProvider.JSON())
	if err != nil {
		t.Fatalf("expected run success, got %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %+v", errs)
	}
	if result.Format != formatProvider.JSON() {
		t.Fatalf("expected result format %s, got %s", formatProvider.JSON(), result.Format)
	}
}
