// Package core contains dataset and service integration tests along with guard
// rails that enforce architectural constraints within the core module.
package core

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoTypeAliases ensures the core package never reintroduces type aliases.
func TestNoTypeAliases(t *testing.T) {
	pkg := loadCorePackage(t)
	var aliases []string

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !ts.Assign.IsValid() {
					continue
				}
				if sel, ok := ts.Type.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "datasetapi" {
						continue
					}
				}
				pos := pkg.Fset.Position(ts.Pos())
				aliases = append(aliases, fmt.Sprintf("%s:%d type %s", filepath.Base(pos.Filename), pos.Line, ts.Name.Name))
			}
		}
	}

	if len(aliases) > 0 {
		t.Fatalf("type aliases are forbidden in internal/core; found %d:\n%s", len(aliases), strings.Join(aliases, "\n"))
	}
}
