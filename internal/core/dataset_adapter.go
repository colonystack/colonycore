package core

import "colonycore/pkg/datasetapi"

func newDatasetTemplateFromAPI(template datasetapi.Template) (DatasetTemplate, error) {
	host, err := datasetapi.NewHostTemplate("", template)
	if err != nil {
		return DatasetTemplate{}, err
	}
	return DatasetTemplate{Template: host.Template()}, nil
}

func newDatasetTemplateRuntime(template DatasetTemplate) datasetapi.TemplateRuntime {
	return template.host
}
