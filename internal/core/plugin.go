package core

// Plugin describes a species module or extension that contributes rules and schema.
type Plugin interface {
	Name() string
	Version() string
	Register(registry *PluginRegistry) error
}

// PluginRegistry accumulates plugin contributions during registration.
type PluginRegistry struct {
	rules   []Rule
	schemas map[string]map[string]any
}

// NewPluginRegistry constructs a plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		schemas: make(map[string]map[string]any),
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

// PluginMetadata stores metadata describing an installed plugin.
type PluginMetadata struct {
	Name    string
	Version string
	Schemas map[string]map[string]any
}
