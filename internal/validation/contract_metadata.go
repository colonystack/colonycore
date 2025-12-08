package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ContractMetadata captures the subset of entity-model schema fields needed for plugin enforcement.
type ContractMetadata struct {
	Version  string
	Entities map[string]ContractEntity
}

// ContractEntity describes mandatory fields and extension hooks for an entity plus its declared properties.
type ContractEntity struct {
	Required       []string
	ExtensionHooks []string
	Properties     map[string]struct{}
}

// LoadContractMetadata reads the entity-model schema and returns the metadata required for enforcement workflows.
func LoadContractMetadata(schemaPath string) (ContractMetadata, error) {
	// #nosec G304 -- schema path is controlled by repository tooling
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return ContractMetadata{}, fmt.Errorf("read entity-model schema: %w", err)
	}
	var doc schemaDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return ContractMetadata{}, fmt.Errorf("parse entity-model schema: %w", err)
	}
	entities := make(map[string]ContractEntity, len(doc.Entities))
	for name, entity := range doc.Entities {
		properties := make(map[string]struct{}, len(entity.Properties))
		var extensionHooks []string
		for prop := range entity.Properties {
			properties[prop] = struct{}{}
			if ref := entity.Properties[prop].Ref; ref == "#/definitions/extension_attributes" {
				extensionHooks = append(extensionHooks, prop)
			}
		}
		sort.Strings(entity.Required)
		sort.Strings(extensionHooks)
		entities[name] = ContractEntity{
			Required:       append([]string(nil), entity.Required...),
			ExtensionHooks: extensionHooks,
			Properties:     properties,
		}
	}
	return ContractMetadata{Version: doc.Version, Entities: entities}, nil
}

type schemaDocument struct {
	Version  string                          `json:"version"`
	Entities map[string]schemaEntityDocument `json:"entities"`
}

type schemaEntityDocument struct {
	Required   []string                           `json:"required"`
	Properties map[string]schemaPropertyReference `json:"properties"`
}

type schemaPropertyReference struct {
	Ref string `json:"$ref"`
}

// HasProperty determines whether a field is declared in the canonical entity model.
func (e ContractEntity) HasProperty(name string) bool {
	if len(e.Properties) == 0 {
		return false
	}
	_, ok := e.Properties[name]
	return ok
}

// IsExtensionHook reports whether the provided field name corresponds to an extension hook.
func (e ContractEntity) IsExtensionHook(name string) bool {
	if len(e.ExtensionHooks) == 0 {
		return false
	}
	i := sort.SearchStrings(e.ExtensionHooks, name)
	return i < len(e.ExtensionHooks) && e.ExtensionHooks[i] == name
}
