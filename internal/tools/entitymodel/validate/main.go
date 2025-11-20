// Program entitymodelvalidate ensures the entity-model schema stays structurally valid.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type enumSpec struct {
	Values []string `json:"values"`
}

type stateSpec struct {
	Enum string `json:"enum"`
}

type relationshipSpec struct {
	Target string `json:"target"`
}

type entitySpec struct {
	Description   string                      `json:"description"`
	Required      []string                    `json:"required"`
	Properties    map[string]json.RawMessage  `json:"properties"`
	Relationships map[string]relationshipSpec `json:"relationships"`
	States        *stateSpec                  `json:"states"`
}

type schemaDoc struct {
	Version  string                `json:"version"`
	Enums    map[string]enumSpec   `json:"enums"`
	Entities map[string]entitySpec `json:"entities"`
}

func main() {
	path := "docs/schema/entity-model.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	if err := validate(path); err != nil {
		exitErr(err.Error())
	}

	fmt.Println("entity-model validation: OK")
}

func validate(path string) error {
	//nolint:gosec // path is provided by the caller; validator is intended to read the specified schema file.
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	var doc schemaDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse schema JSON: %w", err)
	}

	var errs []string

	if doc.Version == "" {
		errs = append(errs, "version must be set (semver expected)")
	}
	if len(doc.Enums) == 0 {
		errs = append(errs, "enums must not be empty")
	}
	for name, spec := range doc.Enums {
		if len(spec.Values) == 0 {
			errs = append(errs, fmt.Sprintf("enum %q must include at least one value", name))
		}
	}

	if len(doc.Entities) == 0 {
		errs = append(errs, "entities section must not be empty")
	}

	baseRequired := []string{"id", "created_at", "updated_at"}

	for name, ent := range doc.Entities {
		if len(ent.Required) == 0 {
			errs = append(errs, fmt.Sprintf("entity %q must declare required fields", name))
		}
		if len(ent.Properties) == 0 {
			errs = append(errs, fmt.Sprintf("entity %q must declare properties", name))
		}

		for _, base := range baseRequired {
			if !contains(ent.Required, base) {
				errs = append(errs, fmt.Sprintf("entity %q must require base field %q", name, base))
			}
		}

		for _, field := range ent.Required {
			if _, ok := ent.Properties[field]; !ok {
				errs = append(errs, fmt.Sprintf("entity %q required field %q missing from properties", name, field))
			}
		}

		if ent.States != nil {
			if ent.States.Enum == "" {
				errs = append(errs, fmt.Sprintf("entity %q states.enum must reference an enum name", name))
			} else if _, ok := doc.Enums[ent.States.Enum]; !ok {
				errs = append(errs, fmt.Sprintf("entity %q states.enum %q not found in enums", name, ent.States.Enum))
			}
		}

		for relName, rel := range ent.Relationships {
			if rel.Target == "" {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q missing target", name, relName))
				continue
			}
			if _, ok := doc.Entities[rel.Target]; !ok {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q targets unknown entity %q", name, relName, rel.Target))
			}
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func contains(list []string, needle string) bool {
	for _, candidate := range list {
		if strings.EqualFold(candidate, needle) {
			return true
		}
	}
	return false
}

func exitErr(msg string) {
	fmt.Fprintf(os.Stderr, "entity-model validation failed: %s\n", msg)
	os.Exit(1)
}
