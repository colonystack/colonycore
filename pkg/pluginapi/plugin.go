package pluginapi

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

type Registry interface {
	RegisterSchema(entity string, schema map[string]any)
	RegisterRule(rule domain.Rule)
	RegisterDatasetTemplate(template datasetapi.Template) error
}

type Plugin interface {
	Name() string
	Version() string
	Register(Registry) error
}

const Version = "v1"
