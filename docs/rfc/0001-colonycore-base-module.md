# RFC: ColonyCore Base Module

- Status: Draft
- Created: 2025-09-21
- Authors: Tobias Harnickell
- Stakeholders: Tobias Harnickell
- Target Release: v0.1.0

## 1. Purpose & Scope
- Track organisms and cohorts from planning through end-of-life with auditable housing, lineage, health, procedural, and inventory histories.
- Provide a stable, species-agnostic domain model plus plugin extension points for species packages (frog, rat, ant, etc.) without hard-coding species logic.
- Enable compliant colony operations aligned with EU Directive 2010/63, IACUC analogs, and local permit regimes while remaining policy-neutral.

### 1.1 Non-Goals
- We will not implement facility-specific SOPs or husbandry policies beyond configurable rules.
- We will not replace downstream ELN/LIMS systems. Instead, we provide integration points.
- We will not deliver a production-grade UI. This RFC focuses on the backend domain, APIs, and plugin contracts.

## 2. Success Metrics
- Onboard a new species module (from template) in under 10 engineer-days with zero core code changes.
- 99.5% of core workflows (breeding, housing moves, procedure logging) executable via API with auditable history checks.
- Detect and block 95% of rule violations (housing capacity, protocol limits, restricted pairings) at transaction time with explainable feedback.
- Maintain sub-1s median response time on read-heavy endpoints for installations with 25k active organisms and 1M historical events.
- Achieve 100% trace coverage for compliance-sensitive actions (create/update/remove) linked to user identity and protocol context.

## 3. Domain Glossary (Species-Agnostic)
| Term | Definition |
| --- | --- |
| Organism | Individual animal tracked throughout its lifecycle with immutable ID and versioned demographic/health data. |
| Cohort | Time-bounded group of organisms managed jointly (e.g., clutch, litter, experimental batch). |
| BreedingUnit | Pair, trio, or colony configured for reproduction with defined fertility history. |
| HousingUnit | Physical housing resource (cage, tank, vivarium) with capacity, zone, and sanitation metadata. |
| Facility | Named location grouping housing units with shared biosecurity rules. |
| Procedure | Planned or executed action requiring scheduling, approvals, and outcomes (e.g., surgery). |
| Treatment | Therapeutic intervention with dosage schedule and compliance linkage. |
| Observation | Timestamped qualitative or quantitative data point recorded during husbandry or research. |
| Sample | Physical material derived from an organism or cohort with chain-of-custody metadata. |
| GenotypeMarker | Defined genetic marker with allele data, assay method, and interpretation rules. |
| Line | Breeding lineage representing defined genetic background with inheritance constraints. |
| Strain | Managed sub-population derived from a line with versioned attributes. |
| Protocol | Regulatory or internal approval covering procedures, species limits, and humane endpoints. |
| Project | Cost-center-bound initiative consuming organisms, procedures, and supplies. |
| Permit | External authorization (e.g., governmental) required for certain activities or counts. |
| SupplyItem | Consumable or reusable inventory tracked by lot, expiry, and storage. |

## 4. Core Entities & Responsibilities
| Entity | Primary Responsibilities | Key Relationships |
| --- | --- | --- |
| Organism | Lifecycle state management, health metrics, identifiers, compliance status | Parentage, Cohort membership, Housing assignments, Protocol links |
| Cohort | Group operations (feeding, procedures), lifecycle rollups, cohort metrics | Organisms, HousingUnits, Project |
| BreedingUnit | Mating schedules, fertility analytics, planned matings | Organisms (adults), HousingUnit, Line/Strain |
| HousingUnit | Capacity enforcement, sanitation cycles, environmental readings | Facility, Organisms, Cohorts |
| Facility | Biosecurity zoning, access control, environmental baselines | HousingUnits, Projects |
| Procedure | Scheduling, execution state, staff assignments | Organisms/Cohorts, Protocol, Observations, Treatments |
| Treatment | Dosage plan, administration log, adverse events | Organisms/Cohorts, Procedure |
| Observation | Structured/unstructured data capture | Organisms/Cohorts, Procedure |
| Sample | Chain-of-custody, storage, assay linkage | Organisms/Cohorts, Facility |
| Protocol | Compliance scope, permitted counts/severity, audit linkage | Procedures, Projects, Permits |
| Project | Budgeting, cost allocations, scope definitions | Organisms, Procedures, SupplyItems, Protocol |
| Permit | Validity period, regulatory metadata, allowed activities | Protocols, Facilities |
| SupplyItem | Inventory management, lot tracking, replenishment | Facilities, Projects |

## 5. Lifecycle & State Machines
### 5.1 Organism Lifecycle
States: `planned → embryo/larva → juvenile → adult → retired → deceased`. Transitions enforce:
- Species module may refine stage mapping while preserving canonical states.
- Automatic stage updates triggered by age thresholds, procedures, or manual overrides with justification.
- Terminal `deceased` state requires cause-of-death, disposal method, and audit trail.

### 5.2 Housing Lifecycle
States: `quarantine`, `active`, `cleaning`, `decommissioned`. Rules:
- Entry to `quarantine` requires assigned protocol and biosecurity clearance.
- `active` enforces capacity and compatibility via species module hooks.
- Transition to `cleaning` requires occupants relocated or tagged for sanitation.
- `decommissioned` prevents future assignments.

### 5.3 Compliance Lifecycle
States: `draft → submitted → approved → on-hold → expired → archived`.
- Approval gating integrates with electronic signatures and timestamping.
- `on-hold` suspends dependent actions. Resumption requires recorded justification.

### 5.4 Procedure Lifecycle
States: `scheduled → in-progress → completed` with optional `cancelled`.
- Scheduling validates resource availability, protocol coverage, and species-specific prerequisites.
- Completion writes immutable outcome records and triggers follow-up tasks (treatments, observations).

## 6. Plugin Architecture & Extension Contracts
### 6.1 Capability Interfaces
| Interface | Responsibility |
| --- | --- |
| `LifecycleRules` | Map age, time, and metrics to the canonical lifecycle states and expose transition validation. |
| `BreedingPlanner` | Recommend pairings, detect kinship conflicts, and project timelines. |
| `PhenotypeSchema` | Provide JSON Schema fragments for species-specific observation fields. |
| `GenotypingRules` | Validate marker sets, allele plausibility, and inheritance warnings. |
| `HusbandrySchedule` | Emit task templates (feeding, tank changes) with recurrence rules. |
| `BodyMetrics` | Define measurement units, normal ranges, and conversion logic. |
| `EnvironmentalNeeds` | Specify acceptable ranges for temperature, humidity, pH, etc. |
| `EuthanasiaMethods` | Enumerate allowable methods with compliance references. |
| `AgeStageMapper` | Convert chronological age and morphometrics to stage labels. |
| `ComplianceChecks` | Add species-specific hard/soft validation hooks for workflows. |

### 6.2 Data Extension Points
- Species modules register JSON Schema fragments to extend Organism, Procedure, Observation, and Sample payloads. Each fragment is versioned and traceable.
- Derived attribute calculators run post-transaction to compute metrics (e.g., condition scores) and persist to materialized views.
- Validation hooks execute within database transactions to block invalid state changes and emit structured errors.

### 6.3 UI/API Composition
- Core API publishes an OpenAPI specification with extension slots, and species modules contribute additional paths and definitions through manifests.
- Declarative UI descriptors (JSON) describe form fields, ordering, validation messaging, and context-specific visibility.
- Reporting layer loads plugin-provided templates (SQL/DSL) with parameter metadata for dashboards and exports.

### 6.4 Rules Engine Hooks
- Policy DSL allows species modules to express constraints (age, kinship, housing compatibility) evaluated pre-commit.
- Rule evaluations produce explainable outcomes (pass/block/warn) with remediation hints and trace IDs.
- Conflict detection framework flags overlapping policies and surfaces to operators for resolution.

## 7. Data & Integration Standards
- Core schema uses relational tables with foreign keys and temporal tables for history, and species fields are stored in JSONB with schema references.
- Identifiers: UUID primary keys, optional local accessions, barcode/RFID/QR attachments with immutable binding history.
- External references: NCBI Taxonomy IDs, OBO phenotypic/assay terms, ISO 8601 timestamps, SI units with metadata.
- Integrations include hardware ingestion (barcode/RFID scanners, scales, environmental sensors) via driver abstraction or MQTT/HTTP bridges.
- Authentication relies on SSO (OIDC or SAML), and RBAC enforces least-privilege roles, per-project data scoping, and audit logs that capture actor, timestamp, and context.

## 8. Compliance & Governance Constraints
- Protocol and permit linkage required before executing regulated procedures or exceeding species/strain counts.
- Severity classifications are stored per organism and procedure, and humane endpoints are enforced through blocking rules with override logging.
- Tamper-evident audit trails use digital signatures for critical events, and retention policies remain configurable for each jurisdiction.
- Privacy controls support pseudonymization, minimal data capture, consent linkage, and access review workflows.
- Semantic versioning is enforced across the core platform and species plugins, with migration guides, feature flags, and deprecation policy documented for every release.

## 9. Observability, Quality, and Performance
- Maintain structured logs with correlation IDs for multi-step workflows, and track metrics for queue lag, rule evaluation latency, and occupancy utilization.
- Tracing spans cover external hardware integrations and long-running procedures.
- Monitor data quality to detect orphaned entities, inconsistent states, and overdue sanitation, and run nightly integrity checks that produce actionable reports.
- Testing strategy: deterministic fixtures for species plugins, contract tests for API extensions, migration tests with sample datasets, fuzzers for rules engine.
- Performance targets: asynchronous workers for heavy computations (pedigree analysis, report generation), sub-second reads for common queries.

## 10. Acceptance Criteria
- Core schema implements entities and relationships listed in Section 4 with migration scripts and temporal history.
- CRUD APIs and audit logging provided for Organism, Cohort, HousingUnit, BreedingUnit, Procedure, Protocol, and Project entities.
- Rules engine executes species and compliance hooks in-transaction with configurable severity levels (block/warn/log).
- Plugin SDK exposes capability interfaces, JSON Schema extension registration, and test harness with seed fixtures.
- Compliance workflows enforce protocol/permit checks before breeding, procedures, or inventory actions, with override logging.
- The observability stack emits the structured logs, metrics, and traces described above and runs scheduled integrity check jobs.
- Documentation delivered for plugin authors and integration partners.

## 11. Reference Species Module: Frog (Minimal)
### 11.1 Purpose
Provide a working module that exercises every plugin interface, validates template assumptions, and demonstrates species customization without core changes.

### 11.2 Capability Implementations
- `LifecycleRules`: Maps developmental stages using Gosner stages grouped to the canonical lifecycle states and enforces completion of metamorphosis before adult classification.
- `BreedingPlanner`: Supports pair and harem configurations, enforces water quality thresholds before spawning, and blocks pairings closer than second-degree kinship.
- `PhenotypeSchema`: Adds fields for skin coloration index, voice call frequency, and limb regeneration observations.
- `GenotypingRules`: Defines microsatellite markers with inheritance plausibility checks and cross contamination warnings.
- `HusbandrySchedule`: Generates feeding (live insects) and water change tasks with temperature dependencies.
- `BodyMetrics`: Declares default units (grams, millimeters), calculates body condition based on snout–vent length ratio.
- `EnvironmentalNeeds`: Specifies acceptable water temperature (18–24°C), pH (6.5–7.5), humidity (>70%), and light cycle.
- `EuthanasiaMethods`: Lists MS-222 immersion and double pithing with references to AVMA guidelines.
- `AgeStageMapper`: Converts days post-fertilization and morphological markers to stage labels for reporting.
- `ComplianceChecks`: Validates that metamorphs in quarantine cannot be released to common rooms without negative pathogen tests.

### 11.3 Data Extensions
- Organism schema adds fields: `developmental_stage`, `water_source`, `disease_screening_status`.
- Observation schema adds enumerations for `skin_score` and numeric `call_frequency_hz`.
- Sample schema adds `tissue_type` with frog-specific options (e.g., toe clip, skin swab).

### 11.4 Module Manifest (Illustrative)
```yaml
species: Lithobates catesbeianus
version: 0.1.0
capabilities:
  lifecycleRules: FrogLifecycleRules
  breedingPlanner: FrogBreedingPlanner
  phenotypeSchema: schemas/frog_phenotype.json
  genotypingRules: FrogGenotypingRules
  husbandrySchedule: FrogHusbandrySchedule
  bodyMetrics: FrogBodyMetrics
  environmentalNeeds: FrogEnvironmentalNeeds
  euthanasiaMethods: FrogEuthanasiaCatalog
  ageStageMapper: FrogAgeStageMapper
  complianceChecks: FrogComplianceHooks
assets:
  reports:
    - reports/frog_housing_capacity.sql
  ui:
    organism:
      - ui/forms/frog_organism_form.json
fixtures:
  - fixtures/frog_reference_cohort.json
```

### 11.5 Reference Workflows
- Create breeding unit template generating spawning tasks, hatching monitoring, and automatic cohort creation when eggs recorded.
- Trigger housing compatibility check ensuring tadpoles never co-house with adults unless partitioned.
- Demonstrate compliance override: manual release from quarantine requires supervisor approval logged with justification.

### 11.6 Test Coverage
- Contract tests validating lifecycle transitions across 5 sample organisms through metamorphosis.
- Fuzzed kinship matrix ensuring breeding planner rejects relatedness > 0.25.
- Integration tests for water quality sensor ingestion updating environmental alert thresholds.

## 12. Linked Annexes & ADRs
- Annex-0001: Operational Risk & Compliance (`docs/annex/0001-operational-risk-and-compliance.md`).
- ADR-0001: Migration, Backfill & Rollback Strategy (`docs/adr/0001-migration-and-rollback.md`).
- ADR-0002: Versioning & Deprecation Policy (`docs/adr/0002-versioning-and-deprecation.md`).
- ADR-0003: Core Domain Schema Normalization (`docs/adr/0003-core-domain-schema.md`).
- ADR-0004: Rules Engine Evaluation Model (`docs/adr/0004-rules-engine-evaluation.md`).
- ADR-0005: Plugin Packaging & Distribution (`docs/adr/0005-plugin-packaging-and-distribution.md`).
- ADR-0006: Observability Stack Architecture (`docs/adr/0006-observability-architecture.md`).

## 13. RFC Lifecycle Governance
- **Reviewers:** Tobias Harnickell
- **Quorum:** Tobias Harnickell must approve for status change to `Accepted`.
- **Decision recording:** Outcome, dissenting notes, and effective date captured in `docs/rfc/registry.yaml` and linked ADRs or annexes.
- **Revision cadence:** Conduct an annual review or respond to major compliance regulation changes, and track minor updates through patch PRs with changelog entries.
- **Sunset:** An RFC can be superseded by a replacement RFC. Once marked `Superseded`, move it to the archive after 12 months while retaining references in the registry.

## 14. Outstanding Questions
- Should plugin schemas support backward incompatible changes via version negotiation or require parallel versions?
- What minimum hardware certification is acceptable for sensor integrations to participate in compliance-critical workflows?
- Do we need automated detection of offline export retention breaches or rely on manual attestations (see Annex-0001 §3.3)?

## 15. Milestones
1. Core schema & API scaffolding with rules engine MVP.
2. Plugin SDK with frog reference module and conformance tests.
3. Compliance workflow integration and audit log hardening.
4. Observability, data quality automation, and migration toolkit.
5. External integrations (hardware, ELN/LIMS) beta adapters.
