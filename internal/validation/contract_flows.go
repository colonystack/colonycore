package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ValidateContractFlows ensures plugin-provided contract flow samples satisfy the canonical entity model.
// Flow files are optional; when absent, no errors are reported.
func ValidateContractFlows(pluginDir, schemaPath string) []Error {
	flowsDir := filepath.Join(pluginDir, "contract_flows")
	entries, err := os.ReadDir(flowsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return []Error{{
			File:    flowsDir,
			Message: fmt.Sprintf("read contract flows: %v", err),
		}}
	}
	schema, err := LoadContractMetadata(schemaPath)
	if err != nil {
		return []Error{{
			File:    schemaPath,
			Message: fmt.Sprintf("load contract metadata: %v", err),
		}}
	}
	var errs []Error
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(flowsDir, entry.Name())
		flow, err := readContractFlow(path)
		if err != nil {
			errs = append(errs, Error{File: path, Message: err.Error()})
			continue
		}
		entity, ok := schema.Entities[flow.Entity]
		if !ok {
			errs = append(errs, Error{File: path, Message: fmt.Sprintf("unknown entity %q", flow.Entity)})
			continue
		}
		if err := validateFlowAction(flow.Action); err != nil {
			errs = append(errs, Error{File: path, Message: err.Error()})
		}
		if len(flow.Payload) == 0 {
			errs = append(errs, Error{File: path, Message: "payload is required"})
			continue
		}
		missing := missingRequired(entity.Required, flow.Payload)
		if len(missing) > 0 {
			errs = append(errs, Error{File: path, Message: fmt.Sprintf("payload missing required fields: %s", strings.Join(missing, ", "))})
		}
		for field := range flow.Payload {
			if entity.HasProperty(field) {
				continue
			}
			if entity.IsExtensionHook(field) {
				continue
			}
			errs = append(errs, Error{File: path, Message: fmt.Sprintf("field %q is not declared for %s", field, flow.Entity)})
		}
	}
	return errs
}

type contractFlow struct {
	Entity  string         `json:"entity"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload"`
}

func readContractFlow(path string) (contractFlow, error) {
	// #nosec G304 -- flow fixtures live inside repo-controlled plugin directories
	data, err := os.ReadFile(path)
	if err != nil {
		return contractFlow{}, fmt.Errorf("read flow: %w", err)
	}
	var flow contractFlow
	if err := json.Unmarshal(data, &flow); err != nil {
		return contractFlow{}, fmt.Errorf("parse flow: %w", err)
	}
	if flow.Entity == "" {
		return contractFlow{}, errors.New("entity is required")
	}
	if flow.Payload == nil {
		flow.Payload = make(map[string]any)
	}
	return flow, nil
}

func validateFlowAction(action string) error {
	switch action {
	case "create", "update":
		return nil
	case "":
		return errors.New("action is required")
	default:
		return fmt.Errorf("unsupported action %q (expected create or update)", action)
	}
}

func missingRequired(required []string, payload map[string]any) []string {
	if len(required) == 0 {
		return nil
	}
	var missing []string
	for _, field := range required {
		if _, ok := payload[field]; !ok {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	return missing
}
