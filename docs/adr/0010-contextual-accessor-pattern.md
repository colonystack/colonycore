# ADR-0010: Contextual Accessor Pattern Implementation

## Status
Accepted (implemented in v0.2.0)

## Deciders
Core maintainers (see `MAINTAINERS.md`)

## Date
2025-10-09

## Context
The original plugin API exposed raw constants for domain values (dialects, formats, entity types, lifecycle stages, etc.). This tight coupling violated hexagonal architecture principles and created several problems:

1. **Tight Coupling**: Plugins depended directly on internal constant definitions
2. **Evolution Resistance**: Changing internal representations required coordinated plugin updates
3. **Limited Expressiveness**: String comparisons and raw constant checks made plugin code verbose and error-prone
4. **Type Safety Issues**: Raw constants allowed incorrect assignments and comparisons
5. **Testing Complexity**: Mocking domain values required extensive setup

With external plugin development anticipated and ADR-0009 establishing stability guarantees, we needed a pattern that:
- Decouples plugins from internal constant definitions
- Provides expressive, type-safe domain value access
- Enables internal evolution without breaking plugins
- Simplifies plugin testing and development

## Decision
We implement a **Contextual Accessor Pattern** that replaces all raw constant access with provider interfaces and contextual reference types.

### Pattern Components

#### 1. Provider Interfaces
Replace raw constants with provider interfaces that return domain-appropriate values:

```go
// Before (raw constants)
const FormatJSON Format = "json"
template.OutputFormats = []Format{FormatJSON}

// After (provider interface)
type FormatProvider interface {
    JSON() Format
    CSV() Format
    // ...
}
formatProvider := GetFormatProvider()
template.OutputFormats = []Format{formatProvider.JSON()}
```

#### 2. Contextual Reference Types
Domain concepts are accessed via context providers returning opaque reference types:

```go
// Before (raw comparisons)
if organism.Stage() == "adult" { ... }

// After (contextual reference)
stageRef := organism.GetCurrentStage()
if stageRef.IsActive() { ... }
```

#### 3. Semantic Methods
Reference types provide semantic methods instead of requiring string comparisons:

```go
// Before (string logic)
func isAquaticEnvironment(env string) bool {
    return env == "aquatic" || env == "semi-aquatic"
}

// After (semantic method)
envRef := housing.GetEnvironmentType()
if envRef.IsAquatic() { ... }
```

#### 4. Facade Contextual Methods
Domain facades provide contextual accessor methods for common queries:

```go
// Added to OrganismView interface
GetCurrentStage() LifecycleStageRef
IsActive() bool
IsRetired() bool
IsDeceased() bool

// Added to HousingUnitView interface  
GetEnvironmentType() EnvironmentTypeRef
IsAquaticEnvironment() bool
IsHumidEnvironment() bool
SupportsSpecies(species string) bool

// Added to ProtocolView interface
GetCurrentStatus() ProtocolStatusRef
IsActiveProtocol() bool
IsTerminalStatus() bool
CanAcceptNewSubjects() bool
```

### Implementation Scope

#### Provider Interfaces Implemented
- `DialectProvider`: SQL(), DSL()
- `FormatProvider`: JSON(), CSV(), Parquet(), PNG(), HTML()
- `VersionProvider`: APIVersion()

#### Contextual Contexts Implemented
- `LifecycleStageContext`: Planned(), Larva(), Juvenile(), Adult(), Retired(), Deceased()
- `HousingContext`: Aquatic(), Terrestrial(), Arboreal(), Humid()
- `ProtocolContext`: Draft(), Active(), Suspended(), Completed(), Cancelled()
- `ProcedureContext`: Scheduled(), InProgress(), Completed(), Cancelled(), Failed()
- `EntityContext`: Organism(), Housing(), Protocol()
- `ActionContext`: Create(), Update(), Delete()
- `SeverityContext`: Log(), Warn(), Block()

#### Reference Types
All context methods return opaque reference types implementing:
- `String()` - string representation
- `Equals(other)` - type-safe comparison  
- Semantic methods (e.g., `IsActive()`, `IsTerminal()`, `IsAquatic()`)
- Internal marker methods to prevent external implementations

### Enforcement Mechanisms

#### 1. Compile-Time Enforcement
- Raw constants removed from exported APIs
- Provider interfaces are the only way to access values
- Reference types are opaque (no public fields)

#### 2. Test-Time Enforcement
- Integration tests validate all view interfaces have contextual accessors
- AST-based scanning detects forbidden raw constant usage in plugins
- Comprehensive test coverage for all contextual methods

#### 3. Documentation Enforcement
- Updated examples in README and plugin documentation
- Migration guide for existing plugins
- ADR-0009 updated with contextual pattern requirements

### Migration Strategy

#### Phase 1: Additive (v0.2.0) - ✅ Completed
- Add provider interfaces alongside existing constants
- Add contextual methods to view interfaces
- Add contextual reference types and contexts
- Maintain backward compatibility with deprecated constants

#### Phase 2: Deprecation (v0.3.0) - Planned
- Mark raw constants as deprecated
- Update all internal usage to contextual pattern
- Provide migration tooling for plugins

#### Phase 3: Removal (v1.0.0) - Planned  
- Remove deprecated raw constants
- Enforce contextual pattern for all new APIs
- Full compliance required for stable plugin API

## Benefits Realized

### 1. Decoupling
Plugins are isolated from internal constant definitions. The core can change internal representations (e.g., using integer IDs instead of strings) without breaking plugins.

### 2. Expressiveness
Plugin code becomes more readable and self-documenting:
```go
// Before
if housing.Environment() == "aquatic" || housing.Environment() == "semi-aquatic" {
    return organism.Species() == "fish" || strings.Contains(organism.Species(), "frog")
}

// After  
if housing.GetEnvironmentType().IsAquatic() {
    return housing.SupportsSpecies(organism.Species())
}
```

### 3. Type Safety
Opaque references prevent incorrect operations:
```go
// Before (error-prone)
env1 := "aquatic" 
env2 := organism.Stage() // Wrong! Comparing environment to lifecycle
if env1 == env2 { ... }

// After (type-safe)
envRef := housing.GetEnvironmentType()
stageRef := organism.GetCurrentStage()
// envRef.Equals(stageRef) // Compile error - different types
```

### 4. Testability
Mock implementations are simpler with contextual interfaces:
```go
type mockHousingContext struct{}
func (m mockHousingContext) Aquatic() EnvironmentTypeRef {
    return mockEnvironmentRef{aquatic: true}
}
```

### 5. Evolution
New semantic methods can be added to reference types without breaking existing code:
```go
// Can add new methods to EnvironmentTypeRef
IsTropical() bool
RequiresHeating() bool  
GetOptimalTemperature() TemperatureRange
```

## Consequences

### Positive
- **Plugin Isolation**: Plugins immune to internal constant changes
- **Improved Developer Experience**: More expressive, readable plugin code
- **Type Safety**: Compile-time prevention of category errors
- **Easier Testing**: Simplified mock and stub creation
- **Future-Proof**: Pattern supports rich domain modeling evolution

### Negative  
- **Initial Learning Curve**: Developers must learn the contextual pattern
- **More Verbose**: Contextual access requires additional method calls
- **Implementation Complexity**: Pattern requires significant infrastructure

### Neutral
- **Breaking Change Timeline**: Full enforcement delayed until v1.0.0
- **Migration Required**: Existing plugins will need updates
- **Documentation Overhead**: Pattern requires comprehensive examples

## Compliance
This ADR implements the architectural requirements from:
- ADR-0009: Plugin Interface Stability (contextual access requirements)
- RFC-0001: ColonyCore Base Module (hexagonal architecture principles)

The implementation is validated by comprehensive integration tests in:
- `internal/integration/contextual_pattern_enforcement_test.go`
- `pkg/datasetapi/contextual_accessor_enforcement_test.go`  
- `pkg/pluginapi/contextual_accessor_enforcement_test.go`
- `pkg/pluginapi/plugin_antipattern_test.go`