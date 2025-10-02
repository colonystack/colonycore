// Package pluginapi contains interfaces implemented by runtime extensions
// (plugins) which can register schemas, rules, and dataset templates.
package pluginapi

import (
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

// Registry is implemented by the host to allow plugins to register resources.
type Registry interface {
	RegisterSchema(entity string, schema map[string]any)
	RegisterRule(rule domain.Rule)
	RegisterDatasetTemplate(template datasetapi.Template) error
}

// Plugin represents a runtime extension that can register its capabilities.
type Plugin interface {
	Name() string
	Version() string
	Register(Registry) error
}

// Version is the semantic version of the plugin host API supported.
const Version = "v1"
