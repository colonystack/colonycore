# Annex-0002: Contextual Accessor Pattern Operational Guide

## Overview
This annex provides operational guidance for implementing, maintaining, and troubleshooting the Contextual Accessor Pattern introduced in ADR-0010. It complements the architectural decision with practical procedures for development teams.

## Development Procedures

### Adding New Domain Concepts

When adding new domain concepts that plugins will access:

1. **Create Context Interface** in appropriate package (`pluginapi` or `datasetapi`):
```go
// pkg/pluginapi/new_concept_context.go
type NewConceptContext interface {
    Value1() NewConceptRef
    Value2() NewConceptRef
}

func NewNewConceptContext() NewConceptContext {
    return DefaultNewConceptContext{}
}
```

2. **Define Reference Interface** with semantic methods:
```go
type NewConceptRef interface {
    String() string
    IsSpecialProperty() bool
    Equals(other NewConceptRef) bool
    isNewConceptRef() // internal marker
}
```

3. **Implement Internal Reference Type**:
```go
type newConceptRef struct {
    value string
}

func (n newConceptRef) String() string { return n.value }
func (n newConceptRef) IsSpecialProperty() bool { return n.value == "special" }
func (n newConceptRef) Equals(other NewConceptRef) bool { /* ... */ }
func (n newConceptRef) isNewConceptRef() {}
```

4. **Add Contextual Methods to View Interfaces**:
```go
type RelevantView interface {
    // existing methods...
    
    // Contextual accessors
    GetNewConcept() NewConceptRef
    IsSpecialCase() bool
}
```

5. **Update Integration Tests** in:
   - `internal/integration/contextual_pattern_enforcement_test.go`
   - Package-specific enforcement tests
   - Anti-pattern detection rules

### Plugin Development Guidelines

#### Required Pattern Usage
```go
// ✅ CORRECT: Use context providers
housingCtx := pluginapi.NewHousingContext()
aquaticEnv := housingCtx.Aquatic()

formatProvider := datasetapi.GetFormatProvider()
jsonFormat := formatProvider.JSON()

// ❌ FORBIDDEN: Raw constant access (will fail CI)
// if housing.Environment() == "aquatic" { ... }
// template.Format = "json"
```

#### Semantic Method Usage
```go
// ✅ CORRECT: Use semantic methods
if organism.IsActive() && !organism.IsRetired() {
    // Process active organism
}

if housing.GetEnvironmentType().IsAquatic() {
    // Handle aquatic environment
}

// ❌ FORBIDDEN: String comparisons
// if organism.Stage() == "adult" { ... }
// if housing.Environment() == "aquatic" { ... }
```

#### Violation Creation
```go
// ✅ CORRECT: Use contextual references
entityCtx := pluginapi.NewEntityContext()
severityCtx := pluginapi.NewSeverityContext()

violation := pluginapi.NewViolationBuilder().
    WithRule("habitat_check").
    WithMessage("Aquatic species requires aquatic housing").
    WithEntityRef(entityCtx.Organism()).
    BuildWithSeverity(severityCtx.Warn())

// ❌ FORBIDDEN: Raw string/constant usage
// violation := pluginapi.NewViolation("habitat_check", "warn", "organism", "message")
```

### Testing Procedures

#### Unit Testing Contextual Interfaces
```go
func TestHousingContextualMethods(t *testing.T) {
    ctx := pluginapi.NewHousingContext()
    
    aquatic := ctx.Aquatic()
    terrestrial := ctx.Terrestrial()
    
    // Test semantic methods
    if !aquatic.IsAquatic() {
        t.Error("Aquatic environment should return true for IsAquatic()")
    }
    
    // Test equality
    if aquatic.Equals(terrestrial) {
        t.Error("Different environment types should not be equal")
    }
}
```

#### Mock Implementation for Testing
```go
type mockHousingContext struct {
    aquaticReturns    EnvironmentTypeRef
    terrestrialReturns EnvironmentTypeRef
}

func (m mockHousingContext) Aquatic() EnvironmentTypeRef { 
    return m.aquaticReturns 
}
func (m mockHousingContext) Terrestrial() EnvironmentTypeRef { 
    return m.terrestrialReturns 
}
```

## Operational Monitoring

### CI/CD Validation

The following checks run automatically on all commits:

1. **API Surface Snapshot Validation**:
   - Location: `internal/ci/datasetapi.snapshot`, `internal/ci/pluginapi.snapshot`  
   - Trigger: Any change to public API surfaces
   - Action: Fails build if breaking changes detected without version bump

2. **Contextual Pattern Enforcement**:
   - Location: `internal/integration/contextual_pattern_enforcement_test.go`
   - Scope: All view interfaces, contextual methods, pattern consistency
   - Trigger: Every test run

3. **Anti-Pattern Detection**:
   - Location: `pkg/pluginapi/plugin_antipattern_test.go`
   - Scope: AST analysis of plugin code for forbidden raw constant usage
   - Coverage: All files in `plugins/` directory

4. **Provider Interface Validation**:
   - Validates all provider interfaces are accessible and functional
   - Tests factory methods return non-nil implementations
   - Verifies method signatures match expected patterns

### Breaking Change Detection

When CI detects breaking changes:

1. **Review Required Elements**:
   - Semantic version bump justification
   - ADR-0009 compliance verification  
   - CHANGELOG entry documenting impact
   - Migration guide for affected plugins

2. **Snapshot Update Procedure**:
   ```bash
   # Update snapshots (local development only)
   go test ./pkg/datasetapi -run TestGenerateDatasetAPISnapshot -update
   go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
   git add internal/ci/*.snapshot
   git commit -m "api: accept surface changes for [reason]"
   ```

3. **Never Update Snapshots in CI**: Updates must happen locally in reviewable PRs

## Troubleshooting

### Common Plugin Development Issues

#### Issue: "Cannot access FormatJSON constant"
**Cause**: Raw constant access is deprecated/removed  
**Solution**: Use provider interface
```go
// Before
template.OutputFormats = []Format{FormatJSON}

// After  
formatProvider := datasetapi.GetFormatProvider()
template.OutputFormats = []Format{formatProvider.JSON()}
```

#### Issue: "Method IsActive() not found on string type"  
**Cause**: Using raw field access instead of contextual methods  
**Solution**: Use contextual accessor
```go
// Before
if organism.Stage() == "adult" { ... }

// After
if organism.IsActive() { ... }
```

#### Issue: "Cannot compare EnvironmentTypeRef to LifecycleStageRef"
**Cause**: Attempting to compare different reference types  
**Solution**: Use correct contextual comparison
```go
// Wrong
if housing.GetEnvironmentType().Equals(organism.GetCurrentStage()) { ... }

// Correct  
housingCtx := pluginapi.NewHousingContext()
if housing.GetEnvironmentType().Equals(housingCtx.Aquatic()) { ... }
```

### CI/CD Troubleshooting

#### Issue: "API surface drift detected" 
**Investigation Steps**:
1. Review what symbols changed in the error output
2. Determine if change is intentional (new feature) or accidental (refactoring)
3. If intentional: update snapshots locally and document in CHANGELOG
4. If accidental: revert the breaking changes

#### Issue: "Contextual pattern enforcement failed"
**Common Causes**:
- Missing contextual accessor methods on view interfaces
- Raw constant usage in plugin code  
- Inconsistent reference type implementations

**Debugging**:
1. Check integration test output for specific failures
2. Validate all view interfaces have required contextual methods
3. Run anti-pattern detection to identify forbidden raw constant usage

#### Issue: "Provider interface validation failed"  
**Common Causes**:
- Factory method returns nil
- Missing provider implementation
- Incorrect method signatures

**Resolution**:
1. Verify GetXxxProvider() functions return valid implementations
2. Check all provider interface methods are implemented
3. Validate method return types match expected patterns

## Migration Support

### Plugin Migration Checklist

For plugins transitioning to contextual accessor pattern:

- [ ] Replace all raw constant access with provider interfaces
- [ ] Update string comparisons to use semantic methods  
- [ ] Use contextual accessors (GetCurrentStage(), GetEnvironmentType())
- [ ] Update violation creation to use contextual references
- [ ] Add unit tests for contextual method usage
- [ ] Verify no anti-pattern warnings in CI

### Internal Code Migration

For core codebase updates:

- [ ] Add contextual methods to new view interfaces
- [ ] Implement semantic methods on reference types  
- [ ] Update integration tests for pattern enforcement
- [ ] Add anti-pattern detection rules if needed
- [ ] Document new patterns in examples and README

## Performance Considerations

### Runtime Performance
- **Contextual method calls**: Negligible overhead (~1-2ns per call)
- **Reference type creation**: Lightweight struct allocation
- **Provider interface access**: Static factory pattern, no reflection

### Memory Impact
- **Additional interfaces**: ~100 bytes per interface definition
- **Reference instances**: 8-16 bytes per reference (contains string value)
- **Context instances**: Stateless, can be singleton pattern

### Build Time Impact  
- **Integration tests**: Add ~2-5 seconds to test suite
- **AST analysis**: Add ~1-3 seconds for anti-pattern detection
- **API surface validation**: Add ~0.5-1 seconds per package

## Compliance Verification

### Manual Verification Checklist

**For New Domain Concepts**:
- [ ] Context interface follows naming convention (`NewXxxContext()`)
- [ ] Reference interface has semantic methods and internal marker
- [ ] Implementation is internal with opaque struct
- [ ] Integration tests validate pattern compliance
- [ ] Anti-pattern rules updated if applicable

**For View Interface Updates**:
- [ ] All contextual accessors return reference types  
- [ ] Semantic query methods return boolean
- [ ] Legacy methods marked deprecated with migration path
- [ ] Test coverage includes all new methods
- [ ] Documentation examples updated

**For Plugin Development**:
- [ ] No raw constant imports or usage
- [ ] All domain access via contextual interfaces
- [ ] Violation creation uses contextual references
- [ ] Unit tests demonstrate correct pattern usage
- [ ] CI passes all anti-pattern detection

This operational guide ensures consistent implementation and maintenance of the Contextual Accessor Pattern across the ColonyCore ecosystem.