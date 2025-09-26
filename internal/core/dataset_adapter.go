package core

import (
	"context"
	"fmt"

	"colonycore/pkg/datasetapi"
)

func newDatasetTemplateFromAPI(template datasetapi.Template) (DatasetTemplate, error) {
	converted := DatasetTemplate{
		Key:           template.Key,
		Version:       template.Version,
		Title:         template.Title,
		Description:   template.Description,
		Dialect:       DatasetDialect(template.Dialect),
		Query:         template.Query,
		Parameters:    copyParametersFromAPI(template.Parameters),
		Columns:       copyColumnsFromAPI(template.Columns),
		Metadata:      copyMetadataFromAPI(template.Metadata),
		OutputFormats: copyFormatsFromAPI(template.OutputFormats),
	}
	converted.Binder = adaptDatasetBinder(template)
	if err := converted.validate(); err != nil {
		return DatasetTemplate{}, err
	}
	return converted, nil
}

func adaptDatasetBinder(template datasetapi.Template) DatasetBinder {
	if template.Binder == nil {
		return nil
	}
	return func(env DatasetEnvironment) (DatasetRunner, error) {
		runner, err := template.Binder(datasetapi.Environment{Store: env.Store, Now: env.Now})
		if err != nil {
			return nil, err
		}
		if runner == nil {
			return nil, fmt.Errorf("dataset binder returned nil runner")
		}
		return func(ctx context.Context, req DatasetRunRequest) (DatasetRunResult, error) {
			apiReq := datasetapi.RunRequest{
				Template:   convertDescriptorToAPI(req.Template),
				Parameters: req.Parameters,
				Scope:      convertScopeToAPI(req.Scope),
			}
			apiResult, err := runner(ctx, apiReq)
			if err != nil {
				return DatasetRunResult{}, err
			}
			return DatasetRunResult{
				Schema:      copyColumnsFromAPI(apiResult.Schema),
				Rows:        apiResult.Rows,
				Metadata:    apiResult.Metadata,
				GeneratedAt: apiResult.GeneratedAt,
				Format:      DatasetFormat(apiResult.Format),
			}, nil
		}, nil
	}
}

func convertDescriptorToAPI(descriptor DatasetTemplateDescriptor) datasetapi.TemplateDescriptor {
	return datasetapi.TemplateDescriptor{
		Plugin:        descriptor.Plugin,
		Key:           descriptor.Key,
		Version:       descriptor.Version,
		Title:         descriptor.Title,
		Description:   descriptor.Description,
		Dialect:       datasetapi.Dialect(descriptor.Dialect),
		Query:         descriptor.Query,
		Parameters:    copyParametersToAPI(descriptor.Parameters),
		Columns:       copyColumnsToAPI(descriptor.Columns),
		Metadata:      copyMetadataToAPI(descriptor.Metadata),
		OutputFormats: copyFormatsToAPI(descriptor.OutputFormats),
		Slug:          descriptor.Slug,
	}
}

func convertScopeToAPI(scope DatasetScope) datasetapi.Scope {
	apiScope := datasetapi.Scope{Requestor: scope.Requestor}
	if len(scope.Roles) > 0 {
		apiScope.Roles = append([]string(nil), scope.Roles...)
	}
	if len(scope.ProjectIDs) > 0 {
		apiScope.ProjectIDs = append([]string(nil), scope.ProjectIDs...)
	}
	if len(scope.ProtocolIDs) > 0 {
		apiScope.ProtocolIDs = append([]string(nil), scope.ProtocolIDs...)
	}
	return apiScope
}

func copyParametersFromAPI(params []datasetapi.Parameter) []DatasetParameter {
	if len(params) == 0 {
		return nil
	}
	converted := make([]DatasetParameter, len(params))
	for i, param := range params {
		converted[i] = DatasetParameter{
			Name:        param.Name,
			Type:        param.Type,
			Required:    param.Required,
			Description: param.Description,
			Unit:        param.Unit,
			Enum:        append([]string(nil), param.Enum...),
			Example:     param.Example,
			Default:     param.Default,
		}
	}
	return converted
}

func copyParametersToAPI(params []DatasetParameter) []datasetapi.Parameter {
	if len(params) == 0 {
		return nil
	}
	converted := make([]datasetapi.Parameter, len(params))
	for i, param := range params {
		converted[i] = datasetapi.Parameter{
			Name:        param.Name,
			Type:        param.Type,
			Required:    param.Required,
			Description: param.Description,
			Unit:        param.Unit,
			Enum:        append([]string(nil), param.Enum...),
			Example:     param.Example,
			Default:     param.Default,
		}
	}
	return converted
}

func copyColumnsFromAPI(columns []datasetapi.Column) []DatasetColumn {
	if len(columns) == 0 {
		return nil
	}
	converted := make([]DatasetColumn, len(columns))
	for i, column := range columns {
		converted[i] = DatasetColumn{
			Name:        column.Name,
			Type:        column.Type,
			Unit:        column.Unit,
			Description: column.Description,
			Format:      column.Format,
		}
	}
	return converted
}

func copyColumnsToAPI(columns []DatasetColumn) []datasetapi.Column {
	if len(columns) == 0 {
		return nil
	}
	converted := make([]datasetapi.Column, len(columns))
	for i, column := range columns {
		converted[i] = datasetapi.Column{
			Name:        column.Name,
			Type:        column.Type,
			Unit:        column.Unit,
			Description: column.Description,
			Format:      column.Format,
		}
	}
	return converted
}

func copyMetadataFromAPI(metadata datasetapi.Metadata) DatasetTemplateMetadata {
	converted := DatasetTemplateMetadata{
		Source:          metadata.Source,
		Documentation:   metadata.Documentation,
		RefreshInterval: metadata.RefreshInterval,
	}
	if len(metadata.Tags) > 0 {
		converted.Tags = append([]string(nil), metadata.Tags...)
	}
	if len(metadata.Annotations) > 0 {
		converted.Annotations = make(map[string]string, len(metadata.Annotations))
		for k, v := range metadata.Annotations {
			converted.Annotations[k] = v
		}
	}
	return converted
}

func copyMetadataToAPI(metadata DatasetTemplateMetadata) datasetapi.Metadata {
	converted := datasetapi.Metadata{
		Source:          metadata.Source,
		Documentation:   metadata.Documentation,
		RefreshInterval: metadata.RefreshInterval,
	}
	if len(metadata.Tags) > 0 {
		converted.Tags = append([]string(nil), metadata.Tags...)
	}
	if len(metadata.Annotations) > 0 {
		converted.Annotations = make(map[string]string, len(metadata.Annotations))
		for k, v := range metadata.Annotations {
			converted.Annotations[k] = v
		}
	}
	return converted
}

func copyFormatsFromAPI(formats []datasetapi.Format) []DatasetFormat {
	if len(formats) == 0 {
		return nil
	}
	converted := make([]DatasetFormat, len(formats))
	for i, format := range formats {
		converted[i] = DatasetFormat(format)
	}
	return converted
}

func copyFormatsToAPI(formats []DatasetFormat) []datasetapi.Format {
	if len(formats) == 0 {
		return nil
	}
	converted := make([]datasetapi.Format, len(formats))
	for i, format := range formats {
		converted[i] = datasetapi.Format(format)
	}
	return converted
}
