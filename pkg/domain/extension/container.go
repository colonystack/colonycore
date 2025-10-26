// Package extension provides typed containers for JSON-schema backed plugin
// extension slots defined in RFC-0001 (sections 6.2/11.3) and ADR-0003.
// Core entities expose a constrained set of hooks where species plugins may
// attach data; this package codifies those hooks and supplies helpers to clone
// and marshal payloads without leaking shared state between callers.
package extension

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
)

// Hook identifies a JSON-schema extension slot for a core entity field.
type Hook string

// Known hook identifiers. The values map 1:1 with the extension slots called
// out in the entity model planning docs so the JSON representation remains
// stable across generators and runtime adapters.
const (
	// HookOrganismAttributes aligns with domain.Organism.Attributes and captures
	// species-agnostic plugin fields declared in RFC-0001 §4 and ADR-0003.
	HookOrganismAttributes Hook = "entity.organism.attributes"
	// HookFacilityEnvironmentBaselines mirrors domain.Facility.EnvironmentBaselines
	// for plugins that publish environmental defaults and monitoring metadata.
	HookFacilityEnvironmentBaselines Hook = "entity.facility.environment_baselines"
	// HookBreedingUnitPairingAttributes maps to domain.BreedingUnit.PairingAttributes
	// and stores plugin-provided pairing metadata.
	HookBreedingUnitPairingAttributes Hook = "entity.breeding_unit.pairing_attributes"
	// HookLineDefaultAttributes aligns with domain.Line.DefaultAttributes to seed
	// plugin-defined defaults for descendants.
	HookLineDefaultAttributes Hook = "entity.line.default_attributes"
	// HookLineExtensionOverrides mirrors domain.Line.ExtensionOverrides, enabling
	// plugins to override upstream defaults for downstream hooks.
	HookLineExtensionOverrides Hook = "entity.line.extension_overrides"
	// HookStrainAttributes captures domain.Strain.Attributes for plugin-managed
	// strain metadata extensions.
	HookStrainAttributes Hook = "entity.strain.attributes"
	// HookGenotypeMarkerAttributes links to domain.GenotypeMarker.Attributes and
	// stores plugin-specific assay payloads.
	HookGenotypeMarkerAttributes Hook = "entity.genotype_marker.attributes"
	// HookObservationData corresponds to domain.Observation.Data for structured
	// measurements emitted by plugins.
	HookObservationData Hook = "entity.observation.data"
	// HookSampleAttributes mirrors domain.Sample.Attributes for chain-of-custody
	// and assay metadata extensions.
	HookSampleAttributes Hook = "entity.sample.attributes"
	// HookSupplyItemAttributes maps to domain.SupplyItem.Attributes for inventory
	// extensions contributed by plugins.
	HookSupplyItemAttributes Hook = "entity.supply_item.attributes"
)

// DataShape describes the top-level JSON shape expected for a hook payload.
type DataShape string

const (
	// ShapeObject indicates the payload is expected to be a JSON object.
	ShapeObject DataShape = "object"
	// ShapeArray indicates the payload is expected to be a JSON array.
	ShapeArray DataShape = "array"
	// ShapeScalar indicates the payload is expected to be a scalar JSON value.
	ShapeScalar DataShape = "scalar"
)

// HookSpec documents the entity contract surfaced through a hook.
type HookSpec struct {
	Entity      string
	Field       string
	Description string
	Shape       DataShape
}

var hookRegistry = map[Hook]HookSpec{
	HookOrganismAttributes: {
		Entity:      "organism",
		Field:       "attributes",
		Description: "Species-agnostic extension bag for organism fields (RFC-0001 §4, ADR-0003).",
		Shape:       ShapeObject,
	},
	HookFacilityEnvironmentBaselines: {
		Entity:      "facility",
		Field:       "environment_baselines",
		Description: "Environmental defaults and monitoring metadata emitted by plugins.",
		Shape:       ShapeObject,
	},
	HookBreedingUnitPairingAttributes: {
		Entity:      "breeding_unit",
		Field:       "pairing_attributes",
		Description: "Pairing metadata (fertility notes, lineage context) supplied by plugins.",
		Shape:       ShapeObject,
	},
	HookLineDefaultAttributes: {
		Entity:      "line",
		Field:       "default_attributes",
		Description: "Default lineage attributes inherited by downstream strains or organisms.",
		Shape:       ShapeObject,
	},
	HookLineExtensionOverrides: {
		Entity:      "line",
		Field:       "extension_overrides",
		Description: "Override values applied to downstream extension hooks for this line.",
		Shape:       ShapeObject,
	},
	HookStrainAttributes: {
		Entity:      "strain",
		Field:       "attributes",
		Description: "Strain-level metadata extensions (versions, husbandry qualities).",
		Shape:       ShapeObject,
	},
	HookGenotypeMarkerAttributes: {
		Entity:      "genotype_marker",
		Field:       "attributes",
		Description: "Assay-specific attributes for genotype markers (interpretation, thresholds).",
		Shape:       ShapeObject,
	},
	HookObservationData: {
		Entity:      "observation",
		Field:       "data",
		Description: "Structured measurement payloads recorded during procedures or husbandry.",
		Shape:       ShapeObject,
	},
	HookSampleAttributes: {
		Entity:      "sample",
		Field:       "attributes",
		Description: "Chain-of-custody and assay metadata supplied by plugins.",
		Shape:       ShapeObject,
	},
	HookSupplyItemAttributes: {
		Entity:      "supply_item",
		Field:       "attributes",
		Description: "Inventory metadata extensions for supply items.",
		Shape:       ShapeObject,
	},
}

// PluginID captures the logical plugin contributing to a hook payload.
type PluginID string

func (id PluginID) String() string {
	return string(id)
}

// ErrUnknownHook indicates an extension payload referenced a hook that is not
// part of the sanctioned schema slots.
var ErrUnknownHook = errors.New("extension: unknown hook identifier")

// ErrEmptyPlugin indicates an empty plugin identifier was supplied.
var ErrEmptyPlugin = errors.New("extension: plugin identifier must not be empty")

// KnownHooks returns the sorted list of registered hook identifiers.
func KnownHooks() []Hook {
	keys := slices.Collect(maps.Keys(hookRegistry))
	slices.Sort(keys)
	return keys
}

// IsKnownHook reports whether the provided hook identifier is registered.
func IsKnownHook(h Hook) bool {
	_, ok := hookRegistry[h]
	return ok
}

// ParseHook validates a string identifier and returns the typed Hook.
func ParseHook(value string) (Hook, error) {
	h := Hook(value)
	if !IsKnownHook(h) {
		return "", fmt.Errorf("%w: %s", ErrUnknownHook, value)
	}
	return h, nil
}

// Spec returns metadata describing the hook contract.
func Spec(h Hook) (HookSpec, bool) {
	spec, ok := hookRegistry[h]
	return spec, ok
}

// Container stores plugin-provided payloads keyed by hook and plugin.
// It centralises cloning and JSON marshalling so domain structs can replace the
// raw map[string]any fields incrementally without changing their wire shape.
type Container struct {
	payload map[Hook]map[string]any
}

// NewContainer initialises an empty extension container.
func NewContainer() Container {
	return Container{
		payload: make(map[Hook]map[string]any),
	}
}

// FromRaw builds a container from the JSON-compatible wire representation.
// Unknown hooks trigger an error to prevent accidental schema drift.
func FromRaw(raw map[string]map[string]any) (Container, error) {
	c := NewContainer()
	for hookStr, plugins := range raw {
		hook, err := ParseHook(hookStr)
		if err != nil {
			return Container{}, err
		}
		if err := c.mergeHookPayload(hook, plugins); err != nil {
			return Container{}, err
		}
	}
	return c, nil
}

// ensurePayload lazily initialises the payload map.
func (c *Container) ensurePayload() {
	if c.payload == nil {
		c.payload = make(map[Hook]map[string]any)
	}
}

// mergeHookPayload inserts a collection of plugin payloads for a hook.
func (c *Container) mergeHookPayload(hook Hook, plugins map[string]any) error {
	if plugins == nil {
		return nil
	}
	for plugin, value := range plugins {
		if err := c.Set(hook, PluginID(plugin), value); err != nil {
			return err
		}
	}
	return nil
}

// Set stores a payload for the given hook and plugin combination.
// Payloads are deep-copied to shield the container from external mutation.
func (c *Container) Set(hook Hook, plugin PluginID, value any) error {
	if !IsKnownHook(hook) {
		return fmt.Errorf("%w: %s", ErrUnknownHook, hook)
	}
	if plugin == "" {
		return ErrEmptyPlugin
	}
	c.ensurePayload()
	if _, exists := c.payload[hook]; !exists {
		c.payload[hook] = make(map[string]any)
	}
	c.payload[hook][plugin.String()] = cloneValue(value)
	return nil
}

// Remove deletes a payload for the given hook and plugin combination.
func (c *Container) Remove(hook Hook, plugin PluginID) {
	if c.payload == nil {
		return
	}
	entries, ok := c.payload[hook]
	if !ok {
		return
	}
	delete(entries, plugin.String())
	if len(entries) == 0 {
		delete(c.payload, hook)
	}
}

// Get retrieves a deep copy of the payload for the given hook and plugin.
func (c Container) Get(hook Hook, plugin PluginID) (any, bool) {
	if c.payload == nil {
		return nil, false
	}
	entries, ok := c.payload[hook]
	if !ok {
		return nil, false
	}
	value, ok := entries[plugin.String()]
	if !ok {
		return nil, false
	}
	return cloneValue(value), true
}

// Plugins returns the registered plugin identifiers for a hook.
func (c Container) Plugins(hook Hook) []PluginID {
	if c.payload == nil {
		return nil
	}
	entries, ok := c.payload[hook]
	if !ok {
		return nil
	}
	result := make([]PluginID, 0, len(entries))
	for plugin := range entries {
		result = append(result, PluginID(plugin))
	}
	slices.Sort(result)
	return result
}

// Hooks reports the hooks populated in the container.
func (c Container) Hooks() []Hook {
	if c.payload == nil {
		return nil
	}
	hooks := slices.Collect(maps.Keys(c.payload))
	slices.Sort(hooks)
	return hooks
}

// Clone produces a deep copy of the container, including all nested payloads.
func (c Container) Clone() (Container, error) {
	if c.payload == nil {
		return NewContainer(), nil
	}
	bytes, err := json.Marshal(c)
	if err != nil {
		return Container{}, err
	}
	var clone Container
	if err := json.Unmarshal(bytes, &clone); err != nil {
		return Container{}, err
	}
	return clone, nil
}

// MarshalJSON implements json.Marshaler to ensure the wire shape remains
// map[hook]map[plugin]any and all nested values are cloned to avoid aliasing.
func (c Container) MarshalJSON() ([]byte, error) {
	wire := make(map[string]map[string]any, len(c.payload))
	for hook, entries := range c.payload {
		inner := make(map[string]any, len(entries))
		for plugin, value := range entries {
			inner[plugin] = cloneValue(value)
		}
		wire[string(hook)] = inner
	}
	return json.Marshal(wire)
}

// UnmarshalJSON validates hook identifiers and populates the container.
func (c *Container) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*c = Container{}
		return nil
	}
	var wire map[string]map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*c = NewContainer()
	for hookStr, entries := range wire {
		hook, err := ParseHook(hookStr)
		if err != nil {
			return err
		}
		c.payload[hook] = make(map[string]any, len(entries))
		for plugin, value := range entries {
			if plugin == "" {
				return ErrEmptyPlugin
			}
			c.payload[hook][plugin] = cloneValue(value)
		}
	}
	return nil
}

// Raw exposes a JSON-compatible copy of the container payload.
func (c Container) Raw() map[string]map[string]any {
	if c.payload == nil {
		return map[string]map[string]any{}
	}
	wire := make(map[string]map[string]any, len(c.payload))
	for hook, entries := range c.payload {
		inner := make(map[string]any, len(entries))
		for plugin, value := range entries {
			inner[plugin] = cloneValue(value)
		}
		wire[string(hook)] = inner
	}
	return wire
}

// cloneValue deep copies supported JSON-compatible values to prevent shared
// references between callers.
func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		json.Number:
		return typed
	}

	source := reflect.ValueOf(value)

	switch source.Kind() {
	case reflect.Map:
		if source.IsNil() || source.Type().Key().Kind() != reflect.String {
			return value
		}
		clone := reflect.MakeMapWithSize(source.Type(), source.Len())
		iter := source.MapRange()
		for iter.Next() {
			key := iter.Key()
			clone.SetMapIndex(key, cloneIntoType(iter.Value(), source.Type().Elem()))
		}
		return clone.Interface()
	case reflect.Slice:
		if source.IsNil() {
			return value
		}
		clone := reflect.MakeSlice(source.Type(), source.Len(), source.Len())
		for i := 0; i < source.Len(); i++ {
			clone.Index(i).Set(cloneIntoType(source.Index(i), source.Type().Elem()))
		}
		return clone.Interface()
	case reflect.Array:
		clone := reflect.New(source.Type()).Elem()
		for i := 0; i < source.Len(); i++ {
			clone.Index(i).Set(cloneIntoType(source.Index(i), source.Type().Elem()))
		}
		return clone.Interface()
	default:
		return value
	}
}

// cloneIntoType deep copies the provided value and converts it to the target type.
func cloneIntoType(value reflect.Value, target reflect.Type) reflect.Value {
	if !value.IsValid() || (value.Kind() == reflect.Interface && value.IsNil()) {
		return reflect.Zero(target)
	}

	cloned := cloneValue(value.Interface())
	if cloned == nil {
		return reflect.Zero(target)
	}

	clonedValue := reflect.ValueOf(cloned)
	if !clonedValue.Type().AssignableTo(target) {
		if clonedValue.Type().ConvertibleTo(target) {
			clonedValue = clonedValue.Convert(target)
		} else {
			return value
		}
	}
	return clonedValue
}
