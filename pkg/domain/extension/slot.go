package extension

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Slot stores plugin-provided payloads for a single extension hook. It wraps
// JSON object semantics so domain entities can expose typed extension fields
// without leaking raw maps.
type Slot struct {
	hook   Hook
	values map[string]any
}

// NewSlot constructs a slot bound to the provided hook identifier.
func NewSlot(h Hook) *Slot {
	return &Slot{
		hook:   h,
		values: make(map[string]any),
	}
}

// BindHook assigns the hook identifier for the slot. It can be used to recover
// hook metadata after JSON unmarshalling.
func (s *Slot) BindHook(h Hook) error {
	if h == "" {
		return fmt.Errorf("extension: hook identifier must not be empty")
	}
	if !IsKnownHook(h) {
		return fmt.Errorf("%w: %s", ErrUnknownHook, h)
	}
	s.hook = h
	return nil
}

// Hook returns the hook identifier associated with the slot.
func (s *Slot) Hook() Hook {
	return s.hook
}

func (s *Slot) ensureMap() {
	if s.values == nil {
		s.values = make(map[string]any)
	}
}

// Set stores a payload for the given plugin identifier.
func (s *Slot) Set(plugin PluginID, payload any) error {
	if s.hook == "" {
		return ErrUnboundSlot
	}
	if plugin == "" {
		return ErrEmptyPlugin
	}
	if err := validateHookPayload(s.hook, payload); err != nil {
		return err
	}
	s.ensureMap()
	s.values[plugin.String()] = cloneValue(payload)
	return nil
}

// Get retrieves a deep copy of the payload stored for the plugin.
func (s *Slot) Get(plugin PluginID) (any, bool) {
	if s.values == nil {
		return nil, false
	}
	value, ok := s.values[plugin.String()]
	if !ok {
		return nil, false
	}
	return cloneValue(value), true
}

// Remove deletes the payload registered for the plugin.
func (s *Slot) Remove(plugin PluginID) {
	if s.values == nil {
		return
	}
	delete(s.values, plugin.String())
	if len(s.values) == 0 {
		s.values = nil
	}
}

// Plugins returns the set of plugin identifiers registered in the slot.
func (s *Slot) Plugins() []PluginID {
	if s.values == nil {
		return nil
	}
	result := make([]PluginID, 0, len(s.values))
	for plugin := range s.values {
		result = append(result, PluginID(plugin))
	}
	slices.Sort(result)
	return result
}

// Raw returns a JSON-compatible copy of the slot payload.
func (s *Slot) Raw() map[string]any {
	if s.values == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(s.values))
	for plugin, value := range s.values {
		out[plugin] = cloneValue(value)
	}
	return out
}

// Clone creates a deep copy of the slot.
func (s *Slot) Clone() *Slot {
	if s == nil {
		return nil
	}
	clone := &Slot{
		hook:   s.hook,
		values: nil,
	}
	if len(s.values) > 0 {
		clone.values = make(map[string]any, len(s.values))
		for plugin, value := range s.values {
			clone.values[plugin] = cloneValue(value)
		}
	}
	return clone
}

// MarshalJSON serialises the slot to the wire representation (map[plugin]payload).
func (s *Slot) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(s.Raw())
}

// UnmarshalJSON populates the slot from a wire representation.
func (s *Slot) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.values = nil
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		s.values = map[string]any{}
		return nil
	}
	if s.hook == "" {
		return ErrUnboundSlot
	}
	s.values = make(map[string]any, len(raw))
	for plugin, value := range raw {
		if plugin == "" {
			return ErrEmptyPlugin
		}
		if err := validateHookPayload(s.hook, value); err != nil {
			return err
		}
		s.values[plugin] = cloneValue(value)
	}
	return nil
}
