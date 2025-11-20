// Program entitymodelvalidate ensures the entity-model schema stays structurally valid.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type naturalKeySpec struct {
	Fields      []string `json:"fields"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
}

type entitySpec struct {
	Description   string                      `json:"description"`
	NaturalKeys   []naturalKeySpec            `json:"natural_keys"`
	Required      []string                    `json:"required"`
	Properties    map[string]json.RawMessage  `json:"properties"`
	Relationships map[string]relationshipSpec `json:"relationships"`
	States        *stateSpec                  `json:"states"`
	Invariants    []string                    `json:"invariants"`
}

type schemaDoc struct {
	Version  string                `json:"version"`
	Enums    map[string]enumSpec   `json:"enums"`
	Entities map[string]entitySpec `json:"entities"`
}

var (
	exitFn              = os.Exit
	errWriter io.Writer = os.Stderr
)

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
		if ent.NaturalKeys == nil {
			errs = append(errs, fmt.Sprintf("entity %q must declare natural_keys (empty array allowed)", name))
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

		for i, nk := range ent.NaturalKeys {
			if len(nk.Fields) == 0 {
				errs = append(errs, fmt.Sprintf("entity %q natural key #%d must declare at least one field", name, i))
			}
			for _, field := range nk.Fields {
				if _, ok := ent.Properties[field]; !ok {
					errs = append(errs, fmt.Sprintf("entity %q natural key field %q missing from properties", name, field))
				}
			}
			if nk.Scope == "" {
				fieldLabel := strings.Join(nk.Fields, ",")
				if fieldLabel == "" {
					fieldLabel = "<unset>"
				}
				errs = append(errs, fmt.Sprintf("entity %q natural key [%s] must declare scope", name, fieldLabel))
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
			if _, ok := ent.Properties[relName]; !ok {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q missing property definition", name, relName))
			}
		}

		for i, invariant := range ent.Invariants {
			if strings.TrimSpace(invariant) == "" {
				errs = append(errs, fmt.Sprintf("entity %q invariants[%d] must not be empty", name, i))
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
	if _, err := fmt.Fprintf(errWriter, "entity-model validation failed: %s\n", msg); err != nil {
		// Fallback to stderr if the configured writer fails.
		//nolint:errcheck // best-effort secondary logging; exiting regardless.
		fmt.Fprintf(os.Stderr, "entity-model validation failed (write error: %v)\n", err)
	}
	exitFn(1)
}
