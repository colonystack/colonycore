// validate_plugin_patterns.go provides compile-time validation of plugin code
// to ensure adherence to hexagonal architecture and contextual accessor patterns.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"colonycore/internal/validation"
)

var (
	defaultContractPath = "docs/annex/plugin-contract.md"
	entityModelPath     = "docs/schema/entity-model.json"
)

func main() {
	os.Exit(run(os.Args, os.Stderr, validation.ValidatePluginDirectory, validateContract))
}

func run(args []string, stderr io.Writer, validate func(string) []validation.Error, enforceContract func(string) error) int {
	if len(args) < 2 {
		progName := "validate_plugin_patterns"
		if len(args) > 0 {
			progName = args[0]
		}
		if _, err := fmt.Fprintf(stderr, "Usage: %s <plugin-directory> [plugin-contract.md]\n", progName); err != nil {
			return 1
		}
		return 1
	}

	pluginDir := args[1]
	contractPath := defaultContractPath
	if len(args) >= 3 && strings.TrimSpace(args[2]) != "" {
		contractPath = args[2]
	}

	if err := enforceContract(contractPath); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "Plugin contract enforcement failed: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}

	errors := validate(pluginDir)

	if len(errors) > 0 {
		if _, err := fmt.Fprintf(stderr, "❌ Found %d hexagonal architecture violations:\n\n", len(errors)); err != nil {
			return 1
		}
		for _, err := range errors {
			if _, writeErr := fmt.Fprintf(stderr, "🚨 %s:%d\n", err.File, err.Line); writeErr != nil {
				return 1
			}
			if _, writeErr := fmt.Fprintf(stderr, "   %s\n", err.Message); writeErr != nil {
				return 1
			}
			if _, writeErr := fmt.Fprintf(stderr, "   Code: %s\n\n", err.Code); writeErr != nil {
				return 1
			}
		}
		return 1
	}
	return 0
}

type contractMetadata struct {
	Version  string                            `json:"version"`
	Entities map[string]contractEntityMetadata `json:"entities"`
}

type contractEntityMetadata struct {
	Required       []string `json:"required"`
	ExtensionHooks []string `json:"extension_hooks"`
}

var contractMetadataRegex = regexp.MustCompile(`(?s)CONTRACT-METADATA\s*(\{.*?\})\s*-->`)

func validateContract(contractPath string) error {
	// #nosec G304 -- contract path is provided by repo tooling during linting
	content, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("read contract: %w", err)
	}
	contractMeta, err := extractContractMetadata(content)
	if err != nil {
		return err
	}
	schemaMeta, err := loadSchemaMetadata(entityModelPath)
	if err != nil {
		return err
	}
	return compareContractMetadata(contractMeta, schemaMeta)
}

func extractContractMetadata(content []byte) (contractMetadata, error) {
	matches := contractMetadataRegex.FindSubmatch(content)
	if len(matches) < 2 {
		return contractMetadata{}, errors.New("CONTRACT-METADATA block missing; regenerate plugin contract")
	}
	var meta contractMetadata
	if err := json.Unmarshal(matches[1], &meta); err != nil {
		return contractMetadata{}, fmt.Errorf("parse contract metadata: %w", err)
	}
	return meta, nil
}

type schemaDoc struct {
	Version  string                `json:"version"`
	Entities map[string]entitySpec `json:"entities"`
}

type entitySpec struct {
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
}

type propertyRef struct {
	Ref string `json:"$ref"`
}

func loadSchemaMetadata(path string) (contractMetadata, error) {
	// #nosec G304 -- schema path is fixed within the repository
	content, err := os.ReadFile(path)
	if err != nil {
		return contractMetadata{}, fmt.Errorf("read entity-model schema: %w", err)
	}
	var doc schemaDoc
	if err := json.Unmarshal(content, &doc); err != nil {
		return contractMetadata{}, fmt.Errorf("parse entity-model schema: %w", err)
	}
	entities := make(map[string]contractEntityMetadata, len(doc.Entities))
	for name, ent := range doc.Entities {
		entities[name] = contractEntityMetadata{
			Required:       sortedCopy(ent.Required),
			ExtensionHooks: sortedHooks(ent.Properties),
		}
	}
	return contractMetadata{Version: strings.TrimSpace(doc.Version), Entities: entities}, nil
}

func sortedCopy(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedHooks(props map[string]json.RawMessage) []string {
	if len(props) == 0 {
		return nil
	}
	var hooks []string
	for name, raw := range props {
		var ref propertyRef
		if err := json.Unmarshal(raw, &ref); err != nil {
			continue
		}
		if ref.Ref == "#/definitions/extension_attributes" {
			hooks = append(hooks, name)
		}
	}
	sort.Strings(hooks)
	return hooks
}

func compareContractMetadata(contractMeta, schemaMeta contractMetadata) error {
	if contractMeta.Version != schemaMeta.Version {
		return fmt.Errorf("contract version %q does not match schema version %q", contractMeta.Version, schemaMeta.Version)
	}
	contractEntities := sortedKeys(contractMeta.Entities)
	schemaEntities := sortedKeys(schemaMeta.Entities)
	if strings.Join(contractEntities, ",") != strings.Join(schemaEntities, ",") {
		return fmt.Errorf("contract entities %v do not match schema entities %v", contractEntities, schemaEntities)
	}
	for _, name := range contractEntities {
		contractEntry := contractMeta.Entities[name]
		schemaEntry := schemaMeta.Entities[name]
		if !equalSlices(contractEntry.Required, schemaEntry.Required) {
			return fmt.Errorf("entity %s required fields mismatch between contract (%v) and schema (%v)", name, contractEntry.Required, schemaEntry.Required)
		}
		if !equalSlices(contractEntry.ExtensionHooks, schemaEntry.ExtensionHooks) {
			return fmt.Errorf("entity %s extension hooks mismatch between contract (%v) and schema (%v)", name, contractEntry.ExtensionHooks, schemaEntry.ExtensionHooks)
		}
	}
	return nil
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
