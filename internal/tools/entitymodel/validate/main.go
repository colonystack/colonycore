// Program entitymodelvalidate ensures the entity-model schema stays structurally valid.
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
)

type enumSpec struct {
	Values []string `json:"values"`
}

type stateSpec struct {
	Enum     string   `json:"enum"`
	Initial  string   `json:"initial"`
	Terminal []string `json:"terminal"`
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

type metadataSpec struct {
	Status string `json:"status"`
}

type schemaDoc struct {
	Version  string                `json:"version"`
	Metadata metadataSpec          `json:"metadata"`
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

	if !isSemver(doc.Version) {
		errs = append(errs, "version must be set (semver expected)")
	}
	if strings.TrimSpace(doc.Metadata.Status) == "" {
		errs = append(errs, "metadata.status must be set")
	}
	if len(doc.Enums) == 0 {
		errs = append(errs, "enums must not be empty")
	}
	for name, spec := range doc.Enums {
		if len(spec.Values) == 0 {
			errs = append(errs, fmt.Sprintf("enum %q must include at least one value", name))
			continue
		}
		if dup := firstDuplicate(spec.Values); dup != "" {
			errs = append(errs, fmt.Sprintf("enum %q has duplicate value %q", name, dup))
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
			} else {
				enumValues := doc.Enums[ent.States.Enum].Values
				if ent.States.Initial == "" {
					errs = append(errs, fmt.Sprintf("entity %q states.initial must reference a value in enum %q", name, ent.States.Enum))
				} else if !contains(enumValues, ent.States.Initial) {
					errs = append(errs, fmt.Sprintf("entity %q states.initial %q not found in enum %q", name, ent.States.Initial, ent.States.Enum))
				}
				if len(ent.States.Terminal) == 0 {
					errs = append(errs, fmt.Sprintf("entity %q states.terminal must include at least one value", name))
				}
				for _, term := range ent.States.Terminal {
					if !contains(enumValues, term) {
						errs = append(errs, fmt.Sprintf("entity %q states.terminal value %q not found in enum %q", name, term, ent.States.Enum))
					}
				}
				if dup := firstDuplicate(ent.States.Terminal); dup != "" {
					errs = append(errs, fmt.Sprintf("entity %q states.terminal has duplicate value %q", name, dup))
				}
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
		if dup := firstDuplicate(ent.Invariants); dup != "" {
			errs = append(errs, fmt.Sprintf("entity %q invariants has duplicate entry %q", name, dup))
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

func isSemver(version string) bool {
	semverRe := regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	return semverRe.MatchString(strings.TrimSpace(version))
}

func firstDuplicate(values []string) string {
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			return v
		}
		seen[v] = struct{}{}
	}
	return ""
}

func exitErr(msg string) {
	if _, err := fmt.Fprintf(errWriter, "entity-model validation failed: %s\n", msg); err != nil {
		// Fallback to stderr if the configured writer fails.
		//nolint:errcheck // best-effort secondary logging; exiting regardless.
		fmt.Fprintf(os.Stderr, "entity-model validation failed (write error: %v)\n", err)
	}
	exitFn(1)
}
