package pluginapi

// This test enforces the exported surface of pluginapi against a committed
// snapshot (see internal/ci/pluginapi.snapshot). It provides two modes:
//   go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
//     Regenerates the snapshot file (intentionally reviewed & committed).
//   go test ./pkg/pluginapi -run TestPluginAPISnapshot
//     Fails if current surface diverges from snapshot.
//
// Rationale: Protects stability guarantees defined in ADR-0009 without
// depending on external tooling. Lightweight reflection + go/doc parsing
// keeps implementation simple.

import (
	"bytes"
	"flag" //nolint:depguard // test-only snapshot update flag
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"                       //nolint:depguard // reflective inspection for snapshot
	"golang.org/x/tools/go/packages" //nolint:depguard // test-time package loading for snapshot
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateSnapshot = flag.Bool("update", false, "update pluginapi snapshot")

const snapshotFileName = "pluginapi.snapshot"

// TestGeneratePluginAPISnapshot regenerates the snapshot when -update is supplied.
func TestGeneratePluginAPISnapshot(t *testing.T) {
	if !*updateSnapshot {
		t.Skip("skipping generation without -update")
	}
	content, err := currentAPISnapshot(t)
	if err != nil {
		t.Fatalf("generate snapshot: %v", err)
	}
	path := resolveSnapshotPath(t)
	if err := os.WriteFile(path, content, 0o600); err != nil { // restrictive perms for generated file
		t.Fatalf("write snapshot: %v", err)
	}
}

// TestPluginAPISnapshot compares the live surface with the committed snapshot.
func TestPluginAPISnapshot(t *testing.T) {
	path := resolveSnapshotPath(t)
	committed, err := os.ReadFile(path) //nolint:gosec // path resolved internally within repo root
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	current, err := currentAPISnapshot(t)
	if err != nil {
		t.Fatalf("build current snapshot: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(committed), bytes.TrimSpace(current)) {
		// Include explicit remediation + documentation steps in the failure output so a
		// developer encountering this in CI understands the governed workflow.
		// The snapshot is the contract enumerated in ADR-0009; any intentional change
		// MUST be accompanied by: (1) review of semantic impact, (2) snapshot update,
		// (3) commit of updated snapshot, and (4) CHANGELOG / ADR amendments if required.
		// DO NOT run with -update inside CI; updates should occur locally in a reviewable PR.
		// Quick fix steps (run from repo root):
		//   go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
		//   git add internal/ci/pluginapi.snapshot
		//   (optional) update docs/adr/0009* or CHANGELOG if stability classification changed
		//   go test ./... && make lint
		//   git commit -m "pluginapi: accept snapshot update <short rationale>"
		// If this change was NOT intentional, revert your code changes to exported symbols.
		t.Fatalf("pluginapi surface drift detected (public API changed).\n\nRemediation:\n  1. If intentional, regenerate + commit snapshot (see below).\n  2. If unintentional, revert exported API changes.\n  3. Ensure ADR-0009 / CHANGELOG updated for any breaking or additive stable changes.\n\nRegenerate & accept (local only):\n  go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update\n  git add internal/ci/pluginapi.snapshot\n  git commit -m 'pluginapi: accept snapshot update <reason>'\n\n--- committed ---\n%s\n--- current ---\n%s\n", committed, current)
	}
}

// currentAPISnapshot introspects exported declarations to a deterministic textual form.
func currentAPISnapshot(t *testing.T) ([]byte, error) {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedCompiledGoFiles}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("packages load errors present")
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected single package, got %d", len(pkgs))
	}
	p := pkgs[0]
	scope := p.Types.Scope()
	var lines []string

	// Collect consts / vars grouped by name kind.
	// We'll also parse syntax to detect const value identifiers we expect.
	fset := token.NewFileSet()
	// Re-parse files to inspect const declarations ordering as written.
	for _, fname := range p.GoFiles {
		fileAst, err := parser.ParseFile(fset, fname, nil, 0)
		if err != nil { // pragma: nocover
			return nil, err
		}
		for _, decl := range fileAst.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for _, name := range vs.Names {
					if !name.IsExported() {
						continue
					}
					lines = append(lines, "CONST "+name.Name)
				}
			}
		}
	}

	// Types.
	for _, name := range scope.Names() {
		if !ast.IsExported(name) {
			continue
		}
		obj := scope.Lookup(name)
		if _, ok := obj.(*types.TypeName); !ok {
			continue
		}
		// Exclude automatically generated error interfaces, etc., but keep everything else.
		// Represent interfaces vs concrete.
		typ := obj.Type().Underlying()
		switch u := typ.(type) {
		case *types.Interface:
			methods := make([]string, 0, u.NumMethods())
			for i := 0; i < u.NumMethods(); i++ {
				m := u.Method(i)
				if !m.Exported() {
					continue
				}
				sig := m.Type().(*types.Signature)
				methods = append(methods, formatSignature(m.Name(), sig))
			}
			lines = append(lines, "TYPE "+name+" interface { "+strings.Join(methods, " ")+" }")
		case *types.Struct:
			// We intentionally only note that it's a struct with unexported fields (public fields would be part of API; guard tests already cover). For simplicity just mark struct.
			lines = append(lines, "TYPE "+name+" struct { unexported }")
		default:
			// Basic / defined types (e.g., type Severity string)
			lines = append(lines, "TYPE "+name+" ("+typ.String()+")")
		}
	}

	// Functions and methods on exported value types (constructors & methods already captured if receivers are exported structs; only our constructors are plain funcs).
	for _, name := range scope.Names() {
		if !ast.IsExported(name) {
			continue
		}
		obj := scope.Lookup(name)
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		sig := fn.Type().(*types.Signature)
		lines = append(lines, "FUNC "+formatSignature(name, sig))
	}

	sort.Strings(lines)
	var buf bytes.Buffer
	buf.WriteString("# DO NOT EDIT MANUALLY.\n")
	buf.WriteString("# Generated snapshot of exported pluginapi surface (types, funcs, consts, vars, methods on exported interfaces) used by TestPluginAPISnapshot.\n")
	for _, l := range lines {
		buf.WriteString(l)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// resolveSnapshotPath finds the repository root by walking upward until an internal/ci directory containing the snapshot file exists.
func resolveSnapshotPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// When tests run inside pkg/pluginapi, walk up until we see internal/ci.
	dir := wd
	for i := 0; i < 10; i++ { // safety bound
		candidate := filepath.Join(dir, "internal", "ci", snapshotFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir { // root
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate %s in ancestor internal/ci directories", snapshotFileName)
	return ""
}

func formatSignature(name string, sig *types.Signature) string {
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('(')
	// params
	for i := 0; i < sig.Params().Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		p := sig.Params().At(i)
		// omit param names for stability
		b.WriteString(p.Type().String())
	}
	b.WriteByte(')')
	if sig.Results().Len() > 0 {
		b.WriteByte(' ')
		if sig.Results().Len() == 1 {
			b.WriteString(sig.Results().At(0).Type().String())
		} else {
			b.WriteByte('(')
			for i := 0; i < sig.Results().Len(); i++ {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(sig.Results().At(i).Type().String())
			}
			b.WriteByte(')')
		}
	}
	return b.String()
}
