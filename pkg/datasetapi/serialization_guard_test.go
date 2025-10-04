package datasetapi

// This test enforces that the public transport / serialization shapes remain
// free of embedded internal metadata structs (e.g. domain.Base, BaseData, the
// unexported base facade type) and only contain the explicitly allowed fields.
// It protects the guarantee that these types are stable, minimal projections
// and do not accidentally leak persistence / internal concerns.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSerializationTypeShapes guards EntityRef, Row, and TemplateDescriptor.
func TestSerializationTypeShapes(t *testing.T) {
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

	// We only care about the datasetapi package in this directory.
	pkg := pkgs["datasetapi"]
	if pkg == nil {
		t.Fatalf("datasetapi package not found at %s", pkgDir)
	}

	var (
		foundEntityRef    bool
		foundRow          bool
		foundTemplateDesc bool
	)

	// Allowed field sets for guarded struct types.
	allowedEntityRefFields := map[string]struct{}{"Entity": {}, "ID": {}}
	allowedTemplateDescriptorFields := map[string]struct{}{
		"Plugin":        {},
		"Key":           {},
		"Version":       {},
		"Title":         {},
		"Description":   {},
		"Dialect":       {},
		"Query":         {},
		"Parameters":    {},
		"Columns":       {},
		"Metadata":      {},
		"OutputFormats": {},
		"Slug":          {},
	}

	// Expected JSON tag map (strict) for guarded struct types.
	expectedEntityRefJSON := map[string]string{
		"Entity": "entity",
		"ID":     "id",
	}
	expectedTemplateDescriptorJSON := map[string]string{
		"Plugin":        "plugin",
		"Key":           "key",
		"Version":       "version",
		"Title":         "title",
		"Description":   "description",
		"Dialect":       "dialect",
		"Query":         "query",
		"Parameters":    "parameters",
		"Columns":       "columns",
		"Metadata":      "metadata",
		"OutputFormats": "output_formats",
		"Slug":          "slug",
	}

	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				switch ts.Name.Name {
				case "EntityRef":
					foundEntityRef = true
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						t.Fatalf("EntityRef must remain a struct, found %T", ts.Type)
					}
					for _, field := range st.Fields.List {
						// Disallow embedded / anonymous fields (field.Names == nil)
						if len(field.Names) == 0 {
							t.Fatalf("EntityRef must not embed other types (found anonymous field of type %T)", field.Type)
						}
						for _, name := range field.Names {
							if _, allowed := allowedEntityRefFields[name.Name]; !allowed {
								t.Fatalf("EntityRef contains disallowed field %q; update guard test intentionally if this is an approved change", name.Name)
							}
							// JSON tag enforcement
							if field.Tag == nil {
								t.Fatalf("EntityRef field %q missing json tag", name.Name)
							}
							rawTag := strings.Trim(field.Tag.Value, "`")
							jsonVal := extractJSONName(rawTag)
							if jsonVal == "" {
								t.Fatalf("EntityRef field %q missing json tag value", name.Name)
							}
							expected := expectedEntityRefJSON[name.Name]
							if jsonVal != expected {
								t.Fatalf("EntityRef field %q json tag = %q, want %q", name.Name, jsonVal, expected)
							}
						}
					}
				case "Row":
					foundRow = true
					// Row must remain a map[string]any (or interface{} alias) not a struct.
					mt, ok := ts.Type.(*ast.MapType)
					if !ok {
						t.Fatalf("Row must remain a map type, found %T", ts.Type)
					}
					// Key must be ident 'string'
					if ident, ok := mt.Key.(*ast.Ident); !ok || ident.Name != "string" {
						t.Fatalf("Row map key must be string, found %#v", mt.Key)
					}
					// Value can be 'any' (go1.18 alias) or 'interface{}'
					switch vt := mt.Value.(type) {
					case *ast.Ident:
						if vt.Name != "any" && vt.Name != "interface{}" { // defensively allow interface{}
							t.Fatalf("Row map value must be any/interface{}, found %s", vt.Name)
						}
					case *ast.InterfaceType:
						// acceptable (interface{})
					default:
						t.Fatalf("Row map value must be any/interface{}, found %T", vt)
					}
				case "TemplateDescriptor":
					foundTemplateDesc = true
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						t.Fatalf("TemplateDescriptor must remain a struct, found %T", ts.Type)
					}
					for _, field := range st.Fields.List {
						if len(field.Names) == 0 { // embedded field
							t.Fatalf("TemplateDescriptor must not embed other types (found anonymous field of type %T)", field.Type)
						}
						for _, name := range field.Names {
							if _, allowed := allowedTemplateDescriptorFields[name.Name]; !allowed {
								t.Fatalf("TemplateDescriptor contains disallowed field %q; update guard test intentionally if this is an approved change", name.Name)
							}
							if field.Tag == nil {
								t.Fatalf("TemplateDescriptor field %q missing json tag", name.Name)
							}
							rawTag := strings.Trim(field.Tag.Value, "`")
							jsonVal := extractJSONName(rawTag)
							if jsonVal == "" {
								t.Fatalf("TemplateDescriptor field %q missing json tag value", name.Name)
							}
							expected := expectedTemplateDescriptorJSON[name.Name]
							if jsonVal != expected {
								t.Fatalf("TemplateDescriptor field %q json tag = %q, want %q", name.Name, jsonVal, expected)
							}
						}
					}
				}
			}
		}
	}

	if !foundEntityRef {
		t.Fatalf("EntityRef type not found; guard needs update if it was renamed/removed")
	}
	if !foundRow {
		t.Fatalf("Row type not found; guard needs update if it was renamed/removed")
	}
	if !foundTemplateDesc {
		t.Fatalf("TemplateDescriptor type not found; guard needs update if it was renamed/removed")
	}
}

// extractJSONName extracts the base json field name (before any comma options)
// from a raw struct tag string (without surrounding backticks). Returns empty
// string if not present.
func extractJSONName(raw string) string {
	// Tags are in form: key:"value" key2:"value2" ... We look for json:"...".
	// We avoid reflect.StructTag to satisfy depguard.
	fields := strings.Split(raw, " ")
	for _, f := range fields {
		if !strings.HasPrefix(f, "json:") {
			continue
		}
		// Expect json:"name[,opts]". Trim prefix json:" then trim trailing quotes.
		val := strings.TrimPrefix(f, "json:\"")
		if i := strings.Index(val, "\""); i >= 0 { // end quote
			val = val[:i]
		}
		if val == "" || val == "-" { // disallow omitted or ignored
			return ""
		}
		// remove options after comma
		if c := strings.Index(val, ","); c >= 0 {
			val = val[:c]
		}
		return val
	}
	return ""
}
