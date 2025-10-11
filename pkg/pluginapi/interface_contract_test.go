package pluginapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	apiRuleName     = "Rule"
	apiRuleViewName = "RuleView"
)

// TestInterfaceContract ensures that the public rule/view surface remains interface-only
// and that the plugin API does not import internal packages directly.
// Guard conditions:
// 1. Exported types named Rule or RuleView must be interface type declarations (not aliases, not structs).
// 2. Any exported type whose name ends with View must be an interface type declaration (not alias, not struct).
// 3. No import path in this package may contain "/internal/".
func TestInterfaceContract(t *testing.T) {
	pkgDir := "." // current directory when tests run inside pkg/pluginapi
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	fset := token.NewFileSet()

	// Track findings
	foundRule := false
	foundRuleView := false

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(pkgDir, name)
		fileAst, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		// Check imports for forbidden internal usage.
		for _, imp := range fileAst.Imports {
			impPath := strings.Trim(imp.Path.Value, "\"")
			if strings.Contains(impPath, "/internal/") {
				t.Errorf("forbidden import of internal package: %s", impPath)
			}
		}

		for _, decl := range fileAst.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !ts.Name.IsExported() {
					continue
				}
				exportedName := ts.Name.Name
				// Disallow aliasing for guarded names (Assign != 0 indicates alias: type X = Y)
				if ts.Assign != 0 && (exportedName == apiRuleName || exportedName == apiRuleViewName || strings.HasSuffix(exportedName, "View")) {
					t.Errorf("%s must not be a type alias; keep it a direct interface declaration", exportedName)
					continue
				}
				switch ut := ts.Type.(type) {
				case *ast.InterfaceType:
					if exportedName == apiRuleName {
						foundRule = true
					}
					if exportedName == apiRuleViewName {
						foundRuleView = true
					}
				case *ast.StructType:
					if exportedName == apiRuleName || exportedName == apiRuleViewName || strings.HasSuffix(exportedName, "View") {
						t.Errorf("exported concrete struct %s not allowed; must remain an interface", exportedName)
					}
				default:
					if exportedName == apiRuleName || exportedName == apiRuleViewName || strings.HasSuffix(exportedName, "View") {
						t.Errorf("%s must be an interface; found %T", exportedName, ut)
					}
				}
			}
		}
	}

	if !foundRule {
		t.Error("Rule interface not found (or no longer an interface)")
	}
	if !foundRuleView {
		t.Error("RuleView interface not found (or no longer an interface)")
	}
}
