# Hexagonal Architecture Enforcement

This document describes the architectural guards implemented to ensure hexagonal architecture principles are maintained in the ColonyCore plugin system.

## Architecture Guard Layers

### 1. Import-Boss Restrictions (Compile Time)
- **Location**: `.import-restrictions` files
- **Purpose**: Prevents plugins from importing internal domain packages
- **Enforcement**: `make lint` runs import-boss validation

### 2. Static Analysis Guards (Test Time)  
- **Location**: `architecture_guard_test.go`
- **Purpose**: Validates contextual interface design patterns
- **Checks**:
  - Context interfaces return only Ref types
  - Ref interfaces have required behavioral methods
  - Internal marker methods prevent external implementation

### 3. Anti-Pattern Detection (Test Time)
- **Location**: `plugin_antipattern_test.go`
- **Purpose**: Scans plugin code for hexagonal architecture violations
- **Detects**:
  - Direct constant usage instead of contextual interfaces
  - Raw entity type comparisons
  - Legacy violation creation patterns

### 4. CI Architecture Tests (Test Time)
- **Location**: `architecture_ci_test.go`
- **Purpose**: Ensures API stability and behavioral correctness
- **Validates**:
  - Backwards compatibility maintained
  - Performance characteristics acceptable
  - Documentation compliance

## Architecture Invariants

### Contextual Interface Design
```go
// ✅ CORRECT: Contextual access via opaque references
entities := pluginapi.NewEntityContext()
organism := entities.Organism()
if organism.IsCore() { ... }

// ❌ INCORRECT: Direct constant usage
if entity == pluginapi.EntityOrganism { ... }
```

### Violation Creation Patterns
```go
// ✅ PREFERRED: Interface-based violation creation
violation := pluginapi.NewViolationWithEntityRef(
    "rule", severities.Warn(), "message", entities.Organism(), "id")

// ✅ SUPPORTED: Legacy pattern for backwards compatibility  
violation := pluginapi.NewViolation(
    "rule", pluginapi.SeverityWarn, "message", pluginapi.EntityOrganism, "id")
```

### Import Restrictions
```go
// ✅ ALLOWED: Plugin API usage
import "colonycore/pkg/pluginapi"
import "colonycore/pkg/datasetapi"

// ❌ FORBIDDEN: Internal domain access
import "colonycore/pkg/domain"        // Blocked by import-boss
import "colonycore/internal/core"     // Blocked by import-boss
```

## Running Architecture Guards

```bash
# Run all architecture enforcement checks
make lint && make test

# Run specific architecture tests
go test ./pkg/pluginapi -run "Architecture"
go test ./pkg/pluginapi -run "AntiPattern" 

# Validate import restrictions
make import-boss
```

## Adding New Contextual Interfaces

When adding new contextual interfaces, ensure:

1. **Interface Design**:
   - Context interface returns only Ref types
   - Ref interface has behavioral methods + internal marker
   - Implementation struct is unexported

2. **Guard Updates**:
   - Add to `architecture_guard_test.go` checks
   - Update anti-pattern detection rules
   - Add CI tests for new behavioral methods

3. **Documentation**:
   - Update this guard documentation
   - Add examples showing correct usage patterns
   - Document behavioral method semantics

## Hexagonal Architecture Benefits

- **Domain Independence**: Plugins decoupled from internal representations
- **Testability**: Contextual interfaces easily mockable
- **Evolution**: Internal changes don't break plugin compatibility  
- **Type Safety**: Opaque references prevent invalid operations