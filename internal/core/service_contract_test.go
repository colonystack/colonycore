package core

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestServiceStructContract(t *testing.T) {
	pkg := loadCorePackage(t)

	obj := pkg.Types.Scope().Lookup("Service")
	if obj == nil {
		t.Fatalf("Service type not found in package")
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		t.Fatalf("Service is not a named type")
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		t.Fatalf("Service is not a struct")
	}

	qualifier := func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Path()
	}

	fields := make(map[string]string, structType.NumFields())
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		fields[field.Name()] = types.TypeString(field.Type(), qualifier)
	}

	required := map[string]string{
		"store":    "colonycore/pkg/domain.PersistentStore",
		"engine":   "*colonycore/pkg/domain.RulesEngine",
		"clock":    "colonycore/internal/core.Clock",
		"now":      "func() time.Time",
		"logger":   "colonycore/internal/core.Logger",
		"plugins":  "map[string]colonycore/internal/core.PluginMetadata",
		"datasets": "map[string]colonycore/internal/core.DatasetTemplate",
		"mu":       "sync.RWMutex",
	}

	var missing []string
	var mismatched []string
	for name, want := range required {
		got, ok := fields[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if got != want {
			mismatched = append(mismatched, fmt.Sprintf("%s: want %s, got %s", name, want, got))
		}
	}

	if len(missing) > 0 || len(mismatched) > 0 {
		_, file, line, _ := runtime.Caller(0)
		var details []string
		if len(missing) > 0 {
			details = append(details, "missing fields: "+strings.Join(missing, ", "))
		}
		if len(mismatched) > 0 {
			details = append(details, "type mismatches: "+strings.Join(mismatched, "; "))
		}
		t.Fatalf("service struct contract violated (%s:%d): %s", filepath.Base(file), line, strings.Join(details, "; "))
	}
}

func TestServiceTransactionalMethodsUseRun(t *testing.T) {
	pkg := loadCorePackage(t)

	serviceFile := findFile(t, pkg, "service.go")

	var violations []string

	for _, decl := range serviceFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Body == nil {
			continue
		}
		recvName, isService := serviceReceiverName(fn)
		if !isService {
			continue
		}
		if !ast.IsExported(fn.Name.Name) {
			continue
		}
		if !methodReturnsResult(fn) {
			continue
		}
		if methodUsesRun(fn, recvName) {
			continue
		}
		pos := pkg.Fset.Position(fn.Pos())
		violations = append(violations, fmt.Sprintf("%s:%d %s", filepath.Base(pos.Filename), pos.Line, fn.Name.Name))
	}

	if len(violations) > 0 {
		t.Fatalf("service methods returning Result must delegate to run:\n%s", strings.Join(violations, "\n"))
	}
}

func TestServiceInstallPluginContract(t *testing.T) {
	pkg := loadCorePackage(t)
	serviceFile := findFile(t, pkg, "service.go")

	fnDecl := findFuncDecl(t, serviceFile, "InstallPlugin")
	if fnDecl.Body == nil {
		t.Fatalf("InstallPlugin has no body")
	}

	if !containsRegisterRules(fnDecl.Body) {
		t.Fatalf("InstallPlugin no longer registers plugin rules with the service engine")
	}
	if !containsDatasetBinding(fnDecl.Body) {
		t.Fatalf("InstallPlugin no longer binds dataset templates before installation")
	}
	if !containsDatasetDescriptorAppend(fnDecl.Body) {
		t.Fatalf("InstallPlugin no longer persists dataset descriptors onto metadata")
	}
}

var (
	corePkgOnce sync.Once
	corePkg     *packages.Package
	corePkgErr  error
)

func loadCorePackage(t *testing.T) *packages.Package {
	t.Helper()

	corePkgOnce.Do(func() {
		cfg := &packages.Config{
			Mode:  packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedCompiledGoFiles | packages.NeedFiles,
			Tests: true,
		}
		pkgs, err := packages.Load(cfg, "colonycore/internal/core")
		if err != nil {
			corePkgErr = fmt.Errorf("load core package: %w", err)
			return
		}
		if len(pkgs) == 0 {
			corePkgErr = fmt.Errorf("no packages returned when loading core")
			return
		}
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				corePkgErr = fmt.Errorf("package load errors: %v", pkg.Errors)
				return
			}
			if pkg.PkgPath == "colonycore/internal/core" {
				corePkg = pkg
				return
			}
		}
		corePkgErr = fmt.Errorf("core package not found in load results")
	})

	if corePkgErr != nil {
		t.Fatalf("core package load: %v", corePkgErr)
	}
	return corePkg
}

func findFile(t *testing.T, pkg *packages.Package, target string) *ast.File {
	t.Helper()
	for _, file := range pkg.Syntax {
		pos := pkg.Fset.Position(file.Pos())
		if filepath.Base(pos.Filename) == target {
			return file
		}
	}
	t.Fatalf("failed to locate %s in package", target)
	return nil
}

func findFuncDecl(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	t.Fatalf("failed to locate %s function", name)
	return nil
}

func serviceReceiverName(fn *ast.FuncDecl) (string, bool) {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return "", false
	}
	recv := fn.Recv.List[0]
	var ident *ast.Ident
	switch expr := recv.Type.(type) {
	case *ast.StarExpr:
		switch inner := expr.X.(type) {
		case *ast.Ident:
			ident = inner
		case *ast.SelectorExpr:
			ident = inner.Sel
		}
	case *ast.Ident:
		ident = expr
	case *ast.SelectorExpr:
		ident = expr.Sel
	}
	if ident == nil || ident.Name != "Service" {
		return "", false
	}
	if len(recv.Names) == 0 {
		return "", false
	}
	return recv.Names[0].Name, true
}

func methodReturnsResult(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, res := range fn.Type.Results.List {
		switch expr := res.Type.(type) {
		case *ast.Ident:
			if expr.Name == "Result" {
				return true
			}
		case *ast.SelectorExpr:
			if expr.Sel.Name == "Result" {
				return true
			}
		}
	}
	return false
}

func methodUsesRun(fn *ast.FuncDecl, receiver string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == receiver && sel.Sel.Name == "run" {
			found = true
			return false
		}
		return true
	})
	return found
}

func containsRegisterRules(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Register" {
			return true
		}
		engineSel, ok := sel.X.(*ast.SelectorExpr)
		if !ok || engineSel.Sel.Name != "engine" {
			return true
		}
		if recv, ok := engineSel.X.(*ast.Ident); ok && recv.Name == "s" {
			found = true
			return false
		}
		return true
	})
	return found
}

func containsDatasetBinding(body *ast.BlockStmt) bool {
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		if !iteratesDatasetTemplates(rng.X) {
			return true
		}
		if loopHasBindCall(rng.Body) {
			found = true
			return false
		}
		return true
	})
	return found
}

func iteratesDatasetTemplates(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "DatasetTemplates"
}

func loopHasBindCall(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "bind" {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == datasetIdentName {
			found = true
			return false
		}
		return true
	})
	return found
}

const datasetIdentName = "dataset"

func containsDatasetDescriptorAppend(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		lhsSel, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if lhsSel.Sel.Name != "Datasets" {
			return true
		}
		if recv, ok := lhsSel.X.(*ast.Ident); !ok || recv.Name != "meta" {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		funIdent, ok := call.Fun.(*ast.Ident)
		if !ok || funIdent.Name != "append" || len(call.Args) < 2 {
			return true
		}
		if !selectorMatches(call.Args[0], "meta", "Datasets") {
			return true
		}
		if hasDatasetDescriptor(call.Args[1:]) {
			found = true
			return false
		}
		return true
	})
	return found
}

func selectorMatches(expr ast.Expr, recvName, selName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != selName {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == recvName
}

func hasDatasetDescriptor(args []ast.Expr) bool {
	for _, arg := range args {
		if call, ok := arg.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Descriptor" {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == datasetIdentName {
						return true
					}
				}
			}
		}
	}
	return false
}
