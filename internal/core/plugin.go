package core

import (
	"fmt"
	"sort"
)

// Plugin describes a species module or extension that contributes rules and schema.
type Plugin interface {
	Name() string
	Version() string
	Register(registry *PluginRegistry) error
}

// PluginRegistry accumulates plugin contributions during registration.
type PluginRegistry struct {
	rules    []Rule
	schemas  map[string]map[string]any
	datasets map[string]DatasetTemplate
}

// NewPluginRegistry constructs a plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		schemas:  make(map[string]map[string]any),
		datasets: make(map[string]DatasetTemplate),
	}
}

// RegisterRule adds an in-transaction rule contributed by the plugin.
func (r *PluginRegistry) RegisterRule(rule Rule) {
	if rule == nil {
		return
	}
	r.rules = append(r.rules, rule)
}

// RegisterSchema stores a JSON Schema fragment for an entity type.
func (r *PluginRegistry) RegisterSchema(entity string, schema map[string]any) {
	if entity == "" || schema == nil {
		return
	}
	cp := make(map[string]any, len(schema))
	for k, v := range schema {
		cp[k] = v
	}
	r.schemas[entity] = cp
}

// RegisterDatasetTemplate stores a dataset template manifest contributed by the plugin.
func (r *PluginRegistry) RegisterDatasetTemplate(template DatasetTemplate) error {
	if err := template.validate(); err != nil {
		return err
	}
	key := fmt.Sprintf("%s@%s", template.Key, template.Version)
	if _, exists := r.datasets[key]; exists {
		return fmt.Errorf("dataset template %s already registered", key)
	}
	r.datasets[key] = template
	return nil
}

// Rules returns a copy of registered rules.
func (r *PluginRegistry) Rules() []Rule {
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Schemas returns a copy of registered schema fragments keyed by entity type.
func (r *PluginRegistry) Schemas() map[string]map[string]any {
	out := make(map[string]map[string]any, len(r.schemas))
	for entity, schema := range r.schemas {
		cp := make(map[string]any, len(schema))
		for k, v := range schema {
			cp[k] = v
		}
		out[entity] = cp
	}
	return out
}

// DatasetTemplates returns registered dataset templates.
func (r *PluginRegistry) DatasetTemplates() []DatasetTemplate {
	out := make([]DatasetTemplate, 0, len(r.datasets))
	for _, template := range r.datasets {
		copy := template
		copy.Parameters = cloneParameters(template.Parameters)
		copy.Columns = cloneColumns(template.Columns)
		copy.Metadata = cloneMetadata(template.Metadata)
		copy.OutputFormats = append([]DatasetFormat(nil), template.OutputFormats...)
		out = append(out, copy)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key == out[j].Key {
			return out[i].Version < out[j].Version
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// PluginMetadata stores metadata describing an installed plugin.
type PluginMetadata struct {
	Name     string
	Version  string
	Schemas  map[string]map[string]any
	Datasets []DatasetTemplateDescriptor
}
