package extension_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"colonycore/pkg/domain"
	domainext "colonycore/pkg/domain/extension"
)

// TestHookSpecsReferenceExistingMembers verifies that every hook spec points to
// a real, exported field or method on the domain entities.
func TestHookSpecsReferenceExistingMembers(t *testing.T) {
	entityTypes := map[string]struct {
		value reflect.Type
		ptr   reflect.Type
	}{
		"Organism":       {value: reflect.TypeOf(domain.Organism{}), ptr: reflect.TypeOf(&domain.Organism{})},
		"Cohort":         {value: reflect.TypeOf(domain.Cohort{}), ptr: reflect.TypeOf(&domain.Cohort{})},
		"HousingUnit":    {value: reflect.TypeOf(domain.HousingUnit{}), ptr: reflect.TypeOf(&domain.HousingUnit{})},
		"Facility":       {value: reflect.TypeOf(domain.Facility{}), ptr: reflect.TypeOf(&domain.Facility{})},
		"BreedingUnit":   {value: reflect.TypeOf(domain.BreedingUnit{}), ptr: reflect.TypeOf(&domain.BreedingUnit{})},
		"Line":           {value: reflect.TypeOf(domain.Line{}), ptr: reflect.TypeOf(&domain.Line{})},
		"Strain":         {value: reflect.TypeOf(domain.Strain{}), ptr: reflect.TypeOf(&domain.Strain{})},
		"GenotypeMarker": {value: reflect.TypeOf(domain.GenotypeMarker{}), ptr: reflect.TypeOf(&domain.GenotypeMarker{})},
		"Procedure":      {value: reflect.TypeOf(domain.Procedure{}), ptr: reflect.TypeOf(&domain.Procedure{})},
		"Treatment":      {value: reflect.TypeOf(domain.Treatment{}), ptr: reflect.TypeOf(&domain.Treatment{})},
		"Observation":    {value: reflect.TypeOf(domain.Observation{}), ptr: reflect.TypeOf(&domain.Observation{})},
		"Sample":         {value: reflect.TypeOf(domain.Sample{}), ptr: reflect.TypeOf(&domain.Sample{})},
		"Protocol":       {value: reflect.TypeOf(domain.Protocol{}), ptr: reflect.TypeOf(&domain.Protocol{})},
		"Permit":         {value: reflect.TypeOf(domain.Permit{}), ptr: reflect.TypeOf(&domain.Permit{})},
		"Project":        {value: reflect.TypeOf(domain.Project{}), ptr: reflect.TypeOf(&domain.Project{})},
		"SupplyItem":     {value: reflect.TypeOf(domain.SupplyItem{}), ptr: reflect.TypeOf(&domain.SupplyItem{})},
	}

	for _, hook := range domainext.KnownHooks() {
		spec, ok := domainext.Spec(hook)
		if !ok {
			t.Fatalf("spec for hook %s not found", hook)
		}
		if spec.DomainMember == "" {
			t.Fatalf("spec for hook %s missing DomainMember metadata", hook)
		}
		parts := strings.Split(spec.DomainMember, ".")
		if len(parts) != 3 || parts[0] != "domain" {
			t.Fatalf("unexpected DomainMember format %q (hook=%s)", spec.DomainMember, hook)
		}
		entityName := parts[1]
		member := parts[2]
		types, ok := entityTypes[entityName]
		if !ok {
			t.Fatalf("hook %s references unknown entity %q", hook, entityName)
		}
		if _, ok := types.value.FieldByName(member); ok {
			continue
		}
		if _, ok := types.value.MethodByName(member); ok {
			continue
		}
		if _, ok := types.ptr.MethodByName(member); ok {
			continue
		}
		t.Fatalf("hook %s references missing domain member %s", hook, spec.DomainMember)
	}
}

// TestExtensionPackageDoesNotImportInternal enforces that the extension
// sub-package does not import any internal layers, mirroring the safeguard on
// the root domain package.
func TestExtensionPackageDoesNotImportInternal(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working dir: %v", err)
	}

	entries, err := os.ReadDir(wd)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}

	var violations []string

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(wd, name)
		// #nosec G304 -- path is derived from directory entries within the same package
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(data), "\n")
		inBlock := false
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if !inBlock {
				if strings.HasPrefix(line, "import (") {
					inBlock = true
					continue
				}
				if strings.HasPrefix(line, "import ") {
					checkImport(line, name, &violations)
				}
				continue
			}
			if line == ")" {
				inBlock = false
				continue
			}
			checkImport(line, name, &violations)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("extension package must not import internal packages: %s", strings.Join(violations, ", "))
	}
}

func checkImport(line, file string, violations *[]string) {
	if path := extractQuoted(line); path != "" && strings.Contains(path, "/internal/") {
		*violations = append(*violations, file+":"+path)
	}
}

func extractQuoted(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}
