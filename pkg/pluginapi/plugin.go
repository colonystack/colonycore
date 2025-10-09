// Package pluginapi contains interfaces implemented by runtime extensions
// (plugins) which can register schemas, rules, and dataset templates.
package pluginapi

import "colonycore/pkg/datasetapi"

// Registry is implemented by the host to allow plugins to register resources.
type Registry interface {
	RegisterSchema(entity string, schema map[string]any)
	RegisterRule(rule Rule)
	RegisterDatasetTemplate(template datasetapi.Template) error
}

// Plugin represents a runtime extension that can register its capabilities.
type Plugin interface {
	Name() string
	Version() string
	Register(Registry) error
}

// VersionProvider defines the interface for providing API version information.
type VersionProvider interface {
	// APIVersion returns the semantic version of the plugin host API supported.
	APIVersion() string
}

// DefaultVersionProvider provides the default version implementation.
type DefaultVersionProvider struct{}

// APIVersion returns the current API version.
func (DefaultVersionProvider) APIVersion() string {
	return "v1"
}

// GetVersionProvider returns the default version provider instance.
func GetVersionProvider() VersionProvider {
	return DefaultVersionProvider{}
}
