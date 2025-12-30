package core

// mapExtensionPayloads transforms a two-level nested extension map (hook -> plugin -> value)
// into a three-level nested structure (hook -> plugin -> payload map) suitable for ExtensionSet construction.
// It returns nil for empty input. For each hook/plugin pair:
//   - explicit nil values are preserved as nil
//   - non-map values are converted to nil
//   - map[string]any values are retained as payload maps
//   - a value that is a map[string]any is retained as the payload.
func mapExtensionPayloads(raw map[string]map[string]any) map[string]map[string]map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]map[string]map[string]any, len(raw))
	for hook, plugins := range raw {
		if len(plugins) == 0 {
			out[hook] = nil
			continue
		}
		mapped := make(map[string]map[string]any, len(plugins))
		for plugin, value := range plugins {
			if value == nil {
				mapped[plugin] = nil
				continue
			}
			payload, ok := value.(map[string]any)
			if !ok {
				mapped[plugin] = nil
				continue
			}
			mapped[plugin] = payload
		}
		out[hook] = mapped
	}
	return out
}