# ADR-0009: Plugin Interface Stability & Semantic Versioning Policy

## Status
Accepted (initial policy established prior to first external plugin release)

## Deciders
Core maintainers (see `MAINTAINERS.md`)

## Date
2025-10-04

## Context
The `pkg/pluginapi` package defines the public surface consumed by runtime plugins: 

- Discovery & registration (`Plugin`, `Registry`, constant `Version`)
- Rule evaluation contracts (`Rule`, `RuleView`, domain view interfaces)
- Immutable change / result model (`Change`, `Violation`, `Result`)
- Canonical identifiers & enums (`Severity`, `LifecycleStage`, `EntityType`, `Action`)

External authors will compile against these Go interfaces to build modules distributed independently from the host. Without a clearly documented stability contract and versioning policy, authors face upgrade risk and the core team risks being locked into accidental APIs.

We must:
1. Enumerate which symbols are considered Stable, Experimental, or Internal.
2. Define what kinds of changes constitute MAJOR / MINOR / PATCH version increments for the host/plugin boundary.
3. Provide a deprecation pathway and minimum support window.
4. Clarify expectations for forward / backward compatibility across the host and plugin versions.

## Decision
We adopt a semantic versioning policy for the plugin host API surface with an explicit, host-defined API version constant (`pluginapi.Version`). The policy applies to all exported identifiers inside `pkg/pluginapi` and any transitive exported types re-exported there (e.g., selected dataset template types coming from `pkg/datasetapi` when exposed through `Registry`).

### Scope Classification
Each exported symbol in `pkg/pluginapi` is assigned one of:

- Stable: Covered by backwards compatibility guarantees until removed via documented deprecation cycle.
- Experimental: May change or be removed in a MINOR release; clearly annotated in code comments with the prefix `Experimental:`.
- Reserved/Internal: Not covered by compatibility promises (e.g., files beginning with `.` like `.import-restrictions` or unexported helper functions).

Initial classification:

Stable
- Interfaces: `Plugin`, `Registry`, `Rule`, `RuleView`, `OrganismView`, `HousingUnitView`, `ProtocolView`, `BaseView`
- Contextual Access Interfaces: `EntityContext`, `ActionContext`, `SeverityContext`, `LifecycleStageContext`, `HousingContext`, `ProtocolContext`
- Reference Interfaces: `EntityTypeRef`, `ActionRef`, `SeverityRef`, `LifecycleStageRef`, `EnvironmentTypeRef`, `ProtocolStatusRef`
- Data/Value Types: `Change`, `Violation`, `Result`
- Builder Types: `ChangeBuilder`, `ViolationBuilder`, `ResultBuilder`
- Functions / Constructors: `NewChange`, `NewViolation`, `NewResult`, `NewEntityContext`, `NewActionContext`, `NewSeverityContext`, `NewLifecycleStageContext`, `NewHousingContext`, `NewProtocolContext`
- Provider Interfaces: `VersionProvider`, `GetVersionProvider()`
- Contextual Facade Methods: All `GetCurrent*()`, `Is*()`, `Can*()`, and `Supports*()` methods on view interfaces

Experimental
- None at inception (future additions MUST be explicitly labeled if not yet stable).

### Semantic Versioning Rules (Host Perspective)
Let host version be H = MAJOR.MINOR.PATCH. (This is the overall module version; the plugin API version constant may advance at different cadence but SHOULD remain in lockstep for simplicity during 0.x).

A plugin compiled against host version H1 is considered compatible with a host H2 when:
- MAJOR(H1) == MAJOR(H2) AND MAJOR != 0
- MINOR(H2) >= MINOR(H1)
- (PATCH differences are always compatible)

During 0.x (pre-1.0) phase:
- Backwards-incompatible changes to Stable symbols MAY occur only at MINOR bumps but SHOULD be batched & rare.
- We treat removal or signature alteration of Stable APIs as a “pseudo major” and require: (a) deprecation warning for >=1 prior MINOR release whenever feasible, (b) entry in CHANGELOG, (c) ADR update if conceptual model changes.

Post 1.0:
- Removing or changing the type of a Stable interface method or exported struct field: MAJOR.
- Adding a required method to a Stable interface: MAJOR.
- Adding an optional method via interface split + adapter pattern: MINOR.
- Adding new Stable interfaces, enum values, or constructor helpers that do not break existing code: MINOR.
- Expanding accepted input value ranges or relaxing validation: MINOR.
- Tightening validation or introducing new failure modes in existing method contracts: MAJOR (unless guarded behind opt-in Experimental feature flags).
- Bug fixes, performance improvements without signature/behavioral contract change: PATCH.

### Enum Value Additions
Adding new values to enumerations (`Severity`, `EntityType`, etc.) is a MINOR change. Plugins MUST defensively handle unknown values (default switch case). We will document this requirement in code comments.

### Deprecation Process
1. Mark symbol with `// Deprecated: <reason>. Removal scheduled no earlier than <version>.` 
2. Provide a replacement or migration note.
3. Keep deprecated symbol for at least one MINOR release (or one MAJOR if post-1.0) before removal.
4. Track deprecations in CHANGELOG + dedicated section in `docs/` summarizing active deprecations.

### Plugin Compatibility Negotiation
`pluginapi.Version` encodes the API contract version (string form `v<major>` for now: `v1`). The host will:
- Reject plugin initialization if `plugin.Version()` reports a different major than `pluginapi.Version` (once we reach 1.x and if we extend plugin self-reporting to include target host major).
- Allow patch/minor skew within the same major.

Until we reach host 1.0, `pluginapi.Version` remains `v1` to indicate initial contract; changes within 0.x host versions rely on release notes.

### Forbidden Changes Without Major Bump (Post-1.0)
- Renaming or removing any Stable exported identifier.
- Changing method parameter order or types for Stable interfaces.
- Changing return types or error semantics in a way that breaks existing callers.
- Making previously synchronous calls asynchronous or vice versa (behaviorally observable timing contracts) without compatibility shims.

### Testing & Enforcement
We implement automated API surface checks and pattern enforcement:
- **API Surface Snapshots**: Generated symbol lists committed under `internal/ci/datasetapi.snapshot` and `internal/ci/pluginapi.snapshot`
- **CI Surface Drift Detection**: Jobs compare snapshots against current build; breaking changes without version bump rationale fail the build
- **Contextual Pattern Enforcement**: Integration tests validate that all view interfaces implement contextual accessors and that raw constants are not used in plugin implementations
- **Anti-Pattern Detection**: AST-based scanning detects forbidden raw constant usage in plugin code
- **Provider Interface Validation**: Tests ensure provider interfaces (DialectProvider, FormatProvider, VersionProvider) are properly implemented and accessible

### Contextual Accessor Pattern
The plugin API enforces a contextual accessor pattern that promotes hexagonal architecture:

**Principles:**
- **No Raw Constants**: Plugins must not access raw domain constants directly
- **Contextual Interfaces**: All domain concepts are accessed via context providers (e.g., `NewHousingContext()`, `NewProtocolContext()`)
- **Opaque References**: Context methods return opaque reference types (e.g., `EnvironmentTypeRef`, `ProtocolStatusRef`) with semantic methods
- **Semantic Queries**: View interfaces provide contextual methods like `IsActive()`, `CanAcceptNewSubjects()` instead of raw field access

**Implementation:**
- Context providers replace raw constants: `NewHousingContext().Aquatic()` instead of `EnvironmentAquatic`
- Reference types provide semantic methods: `envRef.IsAquatic()` instead of string comparisons
- View interfaces offer contextual accessors: `organism.GetCurrentStage()` returns `LifecycleStageRef`
- Builder patterns support fluent violation creation: `NewViolationBuilder().WithEntityRef(entityContext.Organism())`

**Benefits:**
- **Decoupling**: Plugins are isolated from internal constant definitions
- **Evolution**: Internal representations can change without breaking plugins
- **Expressiveness**: Semantic methods make plugin code more readable and maintainable
- **Type Safety**: Opaque references prevent incorrect comparisons and assignments

### Documentation
`README.md` (or dedicated `docs/plugins.md`) will link to this ADR. Each exported type in `pkg/pluginapi` must have a top-level comment clarifying stability expectations if non-default (Experimental/Deprecated).

## Rationale
- Predictability: Third-party authors can safely upgrade within a MAJOR line.
- Velocity: `Experimental:` label allows incubation without prematurely freezing design.
- Automation: Snapshot testing reduces human error in reviewing public API diffs.
- Simplicity: Single version constant sufficient until multiple concurrent majors are supported.

## Alternatives Considered
1. Separate module for plugin API. Rejected initially for repo simplicity; can be split later if churn requires independent versioning.
2. Go build tags for experimental features. Rejected: increases matrix complexity; commentary-based labeling + docs is lighter weight.
3. No explicit stability tiers; rely on SemVer only. Rejected: lacks nuance for early incubation.

## Consequences
Positive:
- Clear upgrade guidance reduces ecosystem friction.
- Internal refactors remain possible behind adapters (e.g., domain model changes) without breaking plugin contracts.

Negative / Costs:
- Overhead maintaining snapshot + deprecation cadence.
- Slight friction adding new methods (may require interface splits or wrapper types).

## Security & Governance
- Stability guarantees reduce incentive for plugins to use unsafe reflection into internal packages.
- Deprecation timeline transparency aids compliance for regulated environments tracking software SBOM updates.

## Open Questions / Future Work
- Define a minimal supported Go toolchain matrix for plugin authors.
- Decide when to freeze `v1` and bump to `v2` (likely only after multiple structural evolutions accumulate).
- Consider generating an SDK (wrappers, helpers) distinct from raw interfaces.
- Formalize error typing (sentinel errors or typed errors) for richer machine handling.

## Acceptance Criteria
- ADR merged in `docs/adr` as 0009.
- All current `pkg/pluginapi` exported symbols documented (already present; follow-up to add any missing comments or `Experimental:` prefixes if needed).
- CI follow-up issue filed to implement snapshot enforcement (reference this ADR).

## References
- Prior ADR 0002 (Versioning & Deprecation Policy) — foundational global policy this ADR refines for plugin boundary.
- Prior ADR 0005 (Packaging & Distribution) — relates to how stable contracts affect packaging decisions.
- SemVer 2.0.0: https://semver.org/
- Go wiki on Module Versioning: https://go.dev/doc/modules/versioning
