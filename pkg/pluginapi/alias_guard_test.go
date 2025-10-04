package pluginapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNoDomainTypeAliases ensures:
// 1. No type alias (type X = domain.Y)
// 2. No wrapper redeclaration (type X domain.Y)
// This keeps pluginapi fully decoupled from domain concrete types.
func TestNoDomainTypeAliases(t *testing.T) {
	// Determine module root by walking up two levels from this file location.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller path")
	}
	pkgDir := filepath.Dir(file)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, 0)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}

	var violations []string
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			// Skip test files to reduce noise (aliases there would still compile; include them if wanted).
			if strings.HasSuffix(fname, "_test.go") && !strings.HasSuffix(fname, "alias_guard_test.go") {
				continue
			}
			// Build import alias map for this file.
			importAliases := map[string]string{}
			for _, imp := range f.Imports {
				pathVal := strings.Trim(imp.Path.Value, "\"")
				alias := ""
				if imp.Name != nil {
					alias = imp.Name.Name
				}
				if alias == "" { // derive default alias
					comps := strings.Split(pathVal, "/")
					alias = comps[len(comps)-1]
				}
				importAliases[alias] = pathVal
			}
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts := spec.(*ast.TypeSpec)
					// Only care about selector expressions referencing imported packages.
					sel, isSel := ts.Type.(*ast.SelectorExpr)
					if !isSel {
						continue
					}
					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok {
						continue
					}
					impPath, ok := importAliases[pkgIdent.Name]
					if !ok || !strings.HasSuffix(impPath, "/pkg/domain") {
						continue
					}
					if ts.Assign.IsValid() {
						violations = append(violations, "alias to domain type: "+ts.Name.Name+" = "+impPath+"."+sel.Sel.Name)
					} else {
						violations = append(violations, "wrapper of domain type: "+ts.Name.Name+" wraps "+impPath+"."+sel.Sel.Name)
					}
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("forbidden domain type aliases found:\n%s", strings.Join(violations, "\n"))
	}
}
