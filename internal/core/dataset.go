package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

type (
	// DatasetDialect mirrors datasetapi.Dialect for legacy core consumers.
	DatasetDialect = datasetapi.Dialect
	// DatasetFormat mirrors datasetapi.Format for legacy core consumers.
	DatasetFormat = datasetapi.Format
	// DatasetScope mirrors datasetapi.Scope for legacy core consumers.
	DatasetScope = datasetapi.Scope
	// DatasetParameter mirrors datasetapi.Parameter for legacy core consumers.
	DatasetParameter = datasetapi.Parameter
	// DatasetColumn mirrors datasetapi.Column for legacy core consumers.
	DatasetColumn = datasetapi.Column
	// DatasetTemplateMetadata mirrors datasetapi.Metadata for legacy core consumers.
	DatasetTemplateMetadata = datasetapi.Metadata
	// DatasetBinder mirrors datasetapi.Binder for legacy core consumers.
	DatasetBinder = datasetapi.Binder
	// DatasetRunner mirrors datasetapi.Runner for legacy core consumers.
	DatasetRunner = datasetapi.Runner
	// DatasetRunRequest mirrors datasetapi.RunRequest for legacy core consumers.
	DatasetRunRequest = datasetapi.RunRequest
	// DatasetRunResult mirrors datasetapi.RunResult for legacy core consumers.
	DatasetRunResult = datasetapi.RunResult
	// DatasetParameterError mirrors datasetapi.ParameterError for legacy core consumers.
	DatasetParameterError = datasetapi.ParameterError
	// DatasetTemplateDescriptor mirrors datasetapi.TemplateDescriptor for legacy core consumers.
	DatasetTemplateDescriptor = datasetapi.TemplateDescriptor
)

const (
	// DatasetDialectSQL exposes datasetapi.DialectSQL via the core package.
	DatasetDialectSQL DatasetDialect = datasetapi.DialectSQL
	// DatasetDialectDSL exposes datasetapi.DialectDSL via the core package.
	DatasetDialectDSL DatasetDialect = datasetapi.DialectDSL
)

const (
	// FormatJSON exposes datasetapi.FormatJSON via the core package.
	FormatJSON DatasetFormat = datasetapi.FormatJSON
	// FormatCSV exposes datasetapi.FormatCSV via the core package.
	FormatCSV DatasetFormat = datasetapi.FormatCSV
	// FormatParquet exposes datasetapi.FormatParquet via the core package.
	FormatParquet DatasetFormat = datasetapi.FormatParquet
	// FormatPNG exposes datasetapi.FormatPNG via the core package.
	FormatPNG DatasetFormat = datasetapi.FormatPNG
	// FormatHTML exposes datasetapi.FormatHTML via the core package.
	FormatHTML DatasetFormat = datasetapi.FormatHTML
)

// DatasetEnvironment provides runtime dependencies to binders within the core layer.
type DatasetEnvironment struct {
	Store domain.PersistentStore
	Now   func() time.Time
}

// DatasetTemplate wraps a dataset template contributed by plugins and manages host-side
// runtime state via pkg/datasetapi's HostTemplate implementation.
type DatasetTemplate struct {
	datasetapi.Template
	Plugin string

	host *datasetapi.HostTemplate
}

// Descriptor produces a descriptor snapshot for the template, cloning metadata to guard against mutation.
func (t DatasetTemplate) Descriptor() DatasetTemplateDescriptor {
	if host, err := t.hostOrNew(); err == nil {
		return host.Descriptor()
	}
	return DatasetTemplateDescriptor{
		Plugin:        t.Plugin,
		Key:           t.Key,
		Version:       t.Version,
		Title:         t.Title,
		Description:   t.Description,
		Dialect:       t.Dialect,
		Query:         t.Query,
		Parameters:    cloneParameters(t.Parameters),
		Columns:       cloneColumns(t.Columns),
		Metadata:      cloneMetadata(t.Metadata),
		OutputFormats: cloneFormats(t.OutputFormats),
		Slug:          datasetSlug(t.Plugin, t.Key, t.Version),
	}
}

// SupportsFormat reports whether the template declares the requested format.
func (t DatasetTemplate) SupportsFormat(format DatasetFormat) bool {
	if t.host != nil {
		return t.host.SupportsFormat(format)
	}
	for _, candidate := range t.OutputFormats {
		if candidate == format {
			return true
		}
	}
	return false
}

// ValidateParameters validates supplied parameters against the template definition.
func (t DatasetTemplate) ValidateParameters(params map[string]any) (map[string]any, []DatasetParameterError) {
	host, err := t.hostOrNew()
	if err != nil {
		return nil, []DatasetParameterError{{Name: "", Message: err.Error()}}
	}
	return host.ValidateParameters(params)
}

// Run executes the dataset template using the bound runner after validating parameters.
func (t DatasetTemplate) Run(ctx context.Context, params map[string]any, scope DatasetScope, format DatasetFormat) (DatasetRunResult, []DatasetParameterError, error) {
	host, err := t.boundHost()
	if err != nil {
		return DatasetRunResult{}, nil, err
	}
	return host.Run(ctx, params, scope, format)
}

// Bind attaches a runtime runner using the provided environment.
func (t *DatasetTemplate) bind(env DatasetEnvironment) error {
	if t == nil {
		return errors.New("dataset template nil")
	}
	host, err := datasetapi.NewHostTemplate(t.Plugin, t.Template)
	if err != nil {
		return err
	}
	apiEnv := datasetapi.Environment{Store: newDatasetPersistentStore(env.Store), Now: env.Now}
	if err := host.Bind(apiEnv); err != nil {
		return err
	}
	t.host = &host
	return nil
}

// validate ensures required fields are present and structurally sound.
func (t DatasetTemplate) validate() error {
	_, err := datasetapi.NewHostTemplate(t.Plugin, t.Template)
	return err
}

// slug returns the canonical identifier for the template.
func (t DatasetTemplate) slug() string {
	return datasetSlug(t.Plugin, t.Key, t.Version)
}

func (t DatasetTemplate) hostOrNew() (datasetapi.HostTemplate, error) {
	if t.host != nil {
		return *t.host, nil
	}
	return datasetapi.NewHostTemplate(t.Plugin, t.Template)
}

func (t DatasetTemplate) boundHost() (*datasetapi.HostTemplate, error) {
	if t.host == nil {
		return nil, errors.New("dataset template not bound")
	}
	return t.host, nil
}

func datasetSlug(plugin, key, version string) string {
	keyPart := strings.TrimSpace(key)
	versionPart := strings.TrimSpace(version)
	if plugin = strings.TrimSpace(plugin); plugin == "" {
		return fmt.Sprintf("%s@%s", keyPart, versionPart)
	}
	return fmt.Sprintf("%s/%s@%s", plugin, keyPart, versionPart)
}

func cloneParameters(params []DatasetParameter) []DatasetParameter {
	if len(params) == 0 {
		return nil
	}
	cloned := make([]DatasetParameter, len(params))
	copy(cloned, params)
	for i := range cloned {
		if len(cloned[i].Enum) > 0 {
			cloned[i].Enum = append([]string(nil), cloned[i].Enum...)
		}
	}
	return cloned
}

func cloneColumns(columns []DatasetColumn) []DatasetColumn {
	if len(columns) == 0 {
		return nil
	}
	cloned := make([]DatasetColumn, len(columns))
	copy(cloned, columns)
	return cloned
}

func cloneMetadata(metadata DatasetTemplateMetadata) DatasetTemplateMetadata {
	cloned := metadata
	if len(metadata.Tags) > 0 {
		cloned.Tags = append([]string(nil), metadata.Tags...)
	}
	if len(metadata.Annotations) > 0 {
		cloned.Annotations = make(map[string]string, len(metadata.Annotations))
		for k, v := range metadata.Annotations {
			cloned.Annotations[k] = v
		}
	}
	return cloned
}

func cloneFormats(formats []DatasetFormat) []DatasetFormat {
	if len(formats) == 0 {
		return nil
	}
	cloned := make([]DatasetFormat, len(formats))
	copy(cloned, formats)
	return cloned
}

func cloneTemplate(t datasetapi.Template) datasetapi.Template {
	cloned := t
	cloned.Parameters = cloneParameters(t.Parameters)
	cloned.Columns = cloneColumns(t.Columns)
	cloned.Metadata = cloneMetadata(t.Metadata)
	cloned.OutputFormats = cloneFormats(t.OutputFormats)
	return cloned
}
