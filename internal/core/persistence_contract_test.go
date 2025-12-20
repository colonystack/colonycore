package core

import (
	"go/types"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestPersistentStoreImplementationsHardening ensures only sanctioned persistence packages
// provide concrete implementations of the domain.PersistentStore interface. This guards
// architectural drift from introducing additional backends outside the vetted locations
// (memory + sqlite + optional postgres) without an explicit test update.
func TestPersistentStoreImplementationsHardening(t *testing.T) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes, Tests: true}
	pkgs, err := packages.Load(cfg, "colonycore/...")
	if err != nil {
		// If we cannot load packages, fail fast – this should never happen in CI.
		t.Fatalf("load packages: %v", err)
	}
	// Locate the PersistentStore interface type from the domain package.
	var persistentStore *types.Interface
	for _, p := range pkgs {
		if p.PkgPath == "colonycore/pkg/domain" {
			obj := p.Types.Scope().Lookup("PersistentStore")
			if obj == nil {
				t.Fatalf("domain.PersistentStore not found")
			}
			iface, ok := obj.Type().Underlying().(*types.Interface)
			if !ok {
				t.Fatalf("domain.PersistentStore is not an interface")
			}
			persistentStore = iface
		}
	}
	if persistentStore == nil {
		t.Fatalf("failed to resolve PersistentStore interface")
	}
	allowed := map[string]struct{}{
		"colonycore/internal/infra/persistence/memory":   {},
		"colonycore/internal/infra/persistence/sqlite":   {},
		"colonycore/internal/infra/persistence/postgres": {},
		"colonycore/internal/core":                       {}, // postgres wrapper lives here via OpenPersistentStore
	}
	var unexpected []string
	for _, p := range pkgs {
		// Skip test / generated or external packages.
		if p.Types == nil || p.Types.Scope() == nil {
			continue
		}
		for _, name := range p.Types.Scope().Names() {
			obj := p.Types.Scope().Lookup(name)
			// Only consider concrete types (structs) that could implement the interface.
			named, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}
			st, ok := named.Underlying().(*types.Struct)
			if !ok || st.NumFields() == 0 && named.NumMethods() == 0 { // still allow empty; method set matters
				// Not a struct or no methods; skip.
				continue
			}
			if types.Implements(types.NewPointer(named), persistentStore) {
				if _, ok := allowed[p.PkgPath]; !ok {
					unexpected = append(unexpected, p.PkgPath+"."+name)
				}
			}
		}
	}
	if len(unexpected) > 0 {
		_, file, line, _ := runtime.Caller(0)
		t.Fatalf("unexpected PersistentStore implementations (update allowed list intentionally if adding a new backend):\nfile=%s:%d\n%s", filepath.Base(file), line, unexpected)
	}
}
