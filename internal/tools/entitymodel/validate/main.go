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
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
	Storage     string `json:"storage"`
}

type naturalKeySpec struct {
	Fields      []string `json:"fields"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
}

type idSemanticsSpec struct {
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
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
	ID       *idSemanticsSpec      `json:"id_semantics"`
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
		for i, v := range spec.Values {
			if strings.TrimSpace(v) == "" {
				errs = append(errs, fmt.Sprintf("enum %q value #%d must not be empty", name, i))
			}
		}
		if dup := firstDuplicate(spec.Values); dup != "" {
			errs = append(errs, fmt.Sprintf("enum %q has duplicate value %q", name, dup))
		}
	}

	if len(doc.Entities) == 0 {
		errs = append(errs, "entities section must not be empty")
	}

	if doc.ID == nil {
		errs = append(errs, "id_semantics must be declared")
	} else {
		if strings.TrimSpace(doc.ID.Type) == "" {
			errs = append(errs, "id_semantics.type must be set")
		}
		if strings.TrimSpace(doc.ID.Scope) == "" {
			errs = append(errs, "id_semantics.scope must be set")
		}
		if !doc.ID.Required {
			errs = append(errs, "id_semantics.required must be true")
		}
		if strings.TrimSpace(doc.ID.Description) == "" {
			errs = append(errs, "id_semantics.description must be set")
		}
	}

	allowedInvariants := map[string]struct{}{
		"housing_capacity":     {},
		"lineage_integrity":    {},
		"lifecycle_transition": {},
		"protocol_coverage":    {},
		"protocol_subject_cap": {},
	}

	usedEnums := make(map[string]struct{}, len(doc.Enums))

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
		if ent.Relationships == nil {
			errs = append(errs, fmt.Sprintf("entity %q must declare relationships (empty object allowed)", name))
		}
		if ent.Invariants == nil {
			errs = append(errs, fmt.Sprintf("entity %q must declare invariants (empty array allowed)", name))
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
				usedEnums[ent.States.Enum] = struct{}{}
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
			if strings.TrimSpace(rel.Cardinality) == "" {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q missing cardinality", name, relName))
			} else if !isValidCardinality(rel.Cardinality) {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q has invalid cardinality %q", name, relName, rel.Cardinality))
			}
			if storage := strings.TrimSpace(rel.Storage); storage != "" && !isValidStorage(storage) {
				errs = append(errs, fmt.Sprintf("entity %q relationship %q has invalid storage %q", name, relName, storage))
			}
		}

		for i, invariant := range ent.Invariants {
			if strings.TrimSpace(invariant) == "" {
				errs = append(errs, fmt.Sprintf("entity %q invariants[%d] must not be empty", name, i))
				continue
			}
			if _, ok := allowedInvariants[invariant]; !ok {
				errs = append(errs, fmt.Sprintf("entity %q invariants[%d] %q is not in the allowed invariants list", name, i, invariant))
			}
		}
		if dup := firstDuplicate(ent.Invariants); dup != "" {
			errs = append(errs, fmt.Sprintf("entity %q invariants has duplicate entry %q", name, dup))
		}

		for propName, prop := range ent.Properties {
			meta, err := extractPropertyMeta(prop)
			if err != nil {
				errs = append(errs, fmt.Sprintf("entity %q property %q invalid JSON: %v", name, propName, err))
				continue
			}
			if !meta.hasType && !meta.hasRef {
				errs = append(errs, fmt.Sprintf("entity %q property %q must declare a type or $ref", name, propName))
			}
			for _, enumName := range meta.enums {
				if _, ok := doc.Enums[enumName]; !ok {
					errs = append(errs, fmt.Sprintf("entity %q property %q references unknown enum %q", name, propName, enumName))
					continue
				}
				usedEnums[enumName] = struct{}{}
			}
		}
	}

	for enumName := range doc.Enums {
		if _, ok := usedEnums[enumName]; !ok {
			errs = append(errs, fmt.Sprintf("enum %q is defined but not referenced by any entity states or properties", enumName))
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

func isValidCardinality(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0..1", "1..1", "0..n", "1..n":
		return true
	default:
		return false
	}
}

func isValidStorage(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fk", "join", "derived", "json":
		return true
	default:
		return false
	}
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

type propertyMeta struct {
	enums   []string
	hasType bool
	hasRef  bool
}

func extractPropertyMeta(raw json.RawMessage) (propertyMeta, error) {
	var prop map[string]any
	if err := json.Unmarshal(raw, &prop); err != nil {
		return propertyMeta{}, err
	}

	return propertyMeta{
		enums:   enumRefs(prop),
		hasType: strings.TrimSpace(asString(prop["type"])) != "",
		hasRef:  strings.TrimSpace(asString(prop["$ref"])) != "",
	}, nil
}

func enumRefs(prop map[string]any) []string {
	var enums []string
	ref := asString(prop["$ref"])
	if strings.HasPrefix(ref, "#/enums/") {
		enums = append(enums, strings.TrimPrefix(ref, "#/enums/"))
	}
	return enums
}

func asString(candidate any) string {
	value, _ := candidate.(string)
	return value
}

func exitErr(msg string) {
	if _, err := fmt.Fprintf(errWriter, "entity-model validation failed: %s\n", msg); err != nil {
		// Fallback to stderr if the configured writer fails.
		//nolint:errcheck // best-effort secondary logging; exiting regardless.
		fmt.Fprintf(os.Stderr, "entity-model validation failed (write error: %v)\n", err)
	}
	exitFn(1)
}
