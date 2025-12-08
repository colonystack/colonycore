// Program entitymodeldiff validates schema fingerprints to prevent breaking changes.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
)

type enumSpec struct {
	Values      []string `json:"values"`
	Description string   `json:"description"`
	Initial     string   `json:"initial"`
	Terminal    []string `json:"terminal"`
}

type relationshipSpec struct {
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
	Storage     string `json:"storage"`
}

type entitySpec struct {
	Required      []string                    `json:"required"`
	Properties    map[string]json.RawMessage  `json:"properties"`
	Relationships map[string]relationshipSpec `json:"relationships"`
	States        *stateSpec                  `json:"states"`
	Invariants    []string                    `json:"invariants"`
}

type stateSpec struct {
	Enum     string   `json:"enum"`
	Initial  string   `json:"initial"`
	Terminal []string `json:"terminal"`
}

type schemaDoc struct {
	Version  string                `json:"version"`
	Enums    map[string]enumSpec   `json:"enums"`
	Entities map[string]entitySpec `json:"entities"`
}

type fingerprintDoc struct {
	Version  string                       `json:"version"`
	Enums    map[string][]string          `json:"enums"`
	Entities map[string]entityFingerprint `json:"entities"`
}

type entityFingerprint struct {
	Properties    []string                           `json:"properties"`
	Required      []string                           `json:"required"`
	Invariants    []string                           `json:"invariants"`
	Relationships map[string]relationshipFingerprint `json:"relationships"`
	States        *stateSpec                         `json:"states,omitempty"`
}

type relationshipFingerprint struct {
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
	Storage     string `json:"storage"`
}

var exitFunc = os.Exit

func main() {
	schemaPath := flag.String("schema", "docs/schema/entity-model.json", "path to the entity model schema")
	fingerprintPath := flag.String("fingerprint", "docs/schema/entity-model.fingerprint.json", "path to the fingerprint file")
	write := flag.Bool("write", false, "rewrite the fingerprint file instead of diffing")
	flag.Parse()

	doc, err := loadSchema(*schemaPath)
	if err != nil {
		exitErr(err)
	}

	current := computeFingerprint(doc)

	if *write {
		if err := writeFingerprint(*fingerprintPath, current); err != nil {
			exitErr(err)
		}
		fmt.Printf("wrote fingerprint to %s\n", *fingerprintPath)
		return
	}

	baseline, err := loadFingerprint(*fingerprintPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			exitErr(fmt.Errorf("fingerprint missing (%s); run with -write", *fingerprintPath))
		}
		exitErr(err)
	}

	issues := diffFingerprints(baseline, current)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		exitFunc(1)
	}

	fmt.Println("entity-model fingerprint matches")
}

func loadSchema(path string) (schemaDoc, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // schema path stays within the repo workspace
	if err != nil {
		return schemaDoc{}, fmt.Errorf("read schema: %w", err)
	}
	var doc schemaDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return schemaDoc{}, fmt.Errorf("parse schema: %w", err)
	}
	return doc, nil
}

func computeFingerprint(doc schemaDoc) fingerprintDoc {
	fp := fingerprintDoc{
		Version:  doc.Version,
		Enums:    make(map[string][]string, len(doc.Enums)),
		Entities: make(map[string]entityFingerprint, len(doc.Entities)),
	}

	for name, enum := range doc.Enums {
		values := append([]string(nil), enum.Values...)
		sort.Strings(values)
		fp.Enums[name] = values
	}

	for name, ent := range doc.Entities {
		props := sortedKeys(ent.Properties)
		req := append([]string(nil), ent.Required...)
		sort.Strings(req)
		invariants := append([]string(nil), ent.Invariants...)
		sort.Strings(invariants)

		rels := make(map[string]relationshipFingerprint, len(ent.Relationships))
		for relName, rel := range ent.Relationships {
			rels[relName] = relationshipFingerprint(rel)
		}

		fp.Entities[name] = entityFingerprint{
			Properties:    props,
			Required:      req,
			Invariants:    invariants,
			Relationships: rels,
			States:        ent.States,
		}
	}

	return fp
}

func loadFingerprint(path string) (fingerprintDoc, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // fingerprint originates from the repo and is user-controlled
	if err != nil {
		return fingerprintDoc{}, err
	}
	var fp fingerprintDoc
	if err := json.Unmarshal(raw, &fp); err != nil {
		return fingerprintDoc{}, fmt.Errorf("parse fingerprint: %w", err)
	}
	return fp, nil
}

func writeFingerprint(path string, fp fingerprintDoc) error {
	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fingerprint: %w", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write fingerprint: %w", err)
	}
	return nil
}

func diffFingerprints(old, updated fingerprintDoc) []string {
	var issues []string

	for name, oldEnt := range old.Entities {
		newEnt, ok := updated.Entities[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("entity removed: %s", name))
			continue
		}
		issues = append(issues, diffList(fmt.Sprintf("entity %s", name), "property", oldEnt.Properties, newEnt.Properties)...)
		issues = append(issues, diffList(fmt.Sprintf("entity %s", name), "required field", oldEnt.Required, newEnt.Required)...)
		issues = append(issues, diffList(fmt.Sprintf("entity %s", name), "invariant", oldEnt.Invariants, newEnt.Invariants)...)

		for relName, oldRel := range oldEnt.Relationships {
			newRel, ok := newEnt.Relationships[relName]
			if !ok {
				issues = append(issues, fmt.Sprintf("entity %s relationship removed: %s", name, relName))
				continue
			}
			if oldRel.Target != newRel.Target || oldRel.Cardinality != newRel.Cardinality || oldRel.Storage != newRel.Storage {
				issues = append(issues, fmt.Sprintf("entity %s relationship changed: %s", name, relName))
			}
		}

		if issue := diffStates(name, oldEnt.States, newEnt.States); issue != "" {
			issues = append(issues, issue)
		}
	}

	for enumName, oldValues := range old.Enums {
		newValues, ok := updated.Enums[enumName]
		if !ok {
			issues = append(issues, fmt.Sprintf("enum removed: %s", enumName))
			continue
		}
		issues = append(issues, diffList(fmt.Sprintf("enum %s", enumName), "value", oldValues, newValues)...)
	}

	if old.Version != "" && updated.Version != old.Version {
		issues = append(issues, fmt.Sprintf("schema version changed from %s to %s", old.Version, updated.Version))
	}

	sort.Strings(issues)
	return issues
}

func diffList(scope, label string, oldVals, newVals []string) []string {
	var issues []string
	newSet := make(map[string]struct{}, len(newVals))
	for _, v := range newVals {
		newSet[v] = struct{}{}
	}
	for _, v := range oldVals {
		if _, ok := newSet[v]; !ok {
			issues = append(issues, fmt.Sprintf("%s %s removed: %s", scope, label, v))
		}
	}
	return issues
}

func diffStates(entity string, oldState, newState *stateSpec) string {
	if oldState == nil {
		return ""
	}
	if newState == nil {
		return fmt.Sprintf("entity %s states removed", entity)
	}
	if oldState.Initial != newState.Initial {
		return fmt.Sprintf("entity %s initial state changed: %s -> %s", entity, oldState.Initial, newState.Initial)
	}
	oldTerms := append([]string(nil), oldState.Terminal...)
	sort.Strings(oldTerms)
	newTerms := append([]string(nil), newState.Terminal...)
	sort.Strings(newTerms)
	if len(oldTerms) != len(newTerms) {
		return fmt.Sprintf("entity %s terminal states changed", entity)
	}
	for i := range oldTerms {
		if oldTerms[i] != newTerms[i] {
			return fmt.Sprintf("entity %s terminal states changed", entity)
		}
	}
	if oldState.Enum != newState.Enum {
		return fmt.Sprintf("entity %s state enum changed: %s -> %s", entity, oldState.Enum, newState.Enum)
	}
	return ""
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	exitFunc(1)
}
