package datasetapi

import "slices"

// ExtensionSet exposes read-only extension payloads grouped by hook and plugin.
// Use the contextual helper interfaces (ExtensionHookContext, ExtensionContributorContext)
// instead of relying on raw string identifiers.
type ExtensionSet interface {
	Hooks() []HookRef
	Plugins(h HookRef) []PluginRef
	Get(h HookRef, plugin PluginRef) (ExtensionPayload, bool)
	Core(h HookRef) (ExtensionPayload, bool)
	Raw() map[string]map[string]map[string]any
}

// NewExtensionSet constructs a defensive copy of the provided raw payload.
func NewExtensionSet(raw map[string]map[string]map[string]any) ExtensionSet {
	payload := cloneRaw(raw)
	return &extensionSet{payload: payload}
}

type extensionSet struct {
	payload map[string]map[string]map[string]any
}

func (s *extensionSet) Hooks() []HookRef {
	if len(s.payload) == 0 {
		return nil
	}
	keys := make([]hookRef, 0, len(s.payload))
	for hook := range s.payload {
		keys = append(keys, hookRef{identifier: hook})
	}
	slices.SortFunc(keys, func(a, b hookRef) int {
		switch {
		case a.identifier < b.identifier:
			return -1
		case a.identifier > b.identifier:
			return 1
		default:
			return 0
		}
	})
	result := make([]HookRef, len(keys))
	for i := range keys {
		result[i] = keys[i]
	}
	return result
}

func (s *extensionSet) Plugins(h HookRef) []PluginRef {
	if len(s.payload) == 0 {
		return nil
	}
	entries, ok := s.payload[h.value()]
	if !ok {
		return nil
	}
	plugins := make([]pluginRef, 0, len(entries))
	for plugin := range entries {
		plugins = append(plugins, pluginRef{identifier: plugin})
	}
	slices.SortFunc(plugins, func(a, b pluginRef) int {
		switch {
		case a.identifier < b.identifier:
			return -1
		case a.identifier > b.identifier:
			return 1
		default:
			return 0
		}
	})
	result := make([]PluginRef, len(plugins))
	for i := range plugins {
		result[i] = plugins[i]
	}
	return result
}

func (s *extensionSet) Get(h HookRef, plugin PluginRef) (ExtensionPayload, bool) {
	if len(s.payload) == 0 {
		return UndefinedExtensionPayload(), false
	}
	entries, ok := s.payload[h.value()]
	if !ok {
		return UndefinedExtensionPayload(), false
	}
	value, ok := entries[plugin.value()]
	if !ok {
		return UndefinedExtensionPayload(), false
	}
	return NewExtensionPayload(value), true
}

func (s *extensionSet) Core(h HookRef) (ExtensionPayload, bool) {
	return s.Get(h, extensionContributorContext{}.Core())
}

func (s *extensionSet) Raw() map[string]map[string]map[string]any {
	return cloneRaw(s.payload)
}

func cloneRaw(raw map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]map[string]map[string]any, len(raw))
	for hook, plugins := range raw {
		if len(plugins) == 0 {
			out[hook] = nil
			continue
		}
		cloned := make(map[string]map[string]any, len(plugins))
		for plugin, value := range plugins {
			if value == nil {
				cloned[plugin] = nil
				continue
			}
			clone, _ := cloneValue(value).(map[string]any)
			cloned[plugin] = clone
		}
		out[hook] = cloned
	}
	return out
}

func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return map[string]any{}
		}
		cloned := make(map[string]any, len(typed))
		for key, element := range typed {
			cloned[key] = cloneValue(element)
		}
		return cloned
	case []any:
		if len(typed) == 0 {
			return []any{}
		}
		cloned := make([]any, len(typed))
		for i, element := range typed {
			cloned[i] = cloneValue(element)
		}
		return cloned
	case []string:
		if len(typed) == 0 {
			return []string{}
		}
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	case []map[string]any:
		if len(typed) == 0 {
			return []map[string]any{}
		}
		cloned := make([]map[string]any, len(typed))
		for i, element := range typed {
			if element == nil {
				continue
			}
			if nested, ok := cloneValue(element).(map[string]any); ok {
				cloned[i] = nested
			}
		}
		return cloned
	default:
		return typed
	}
}
