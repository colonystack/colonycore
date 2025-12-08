# Plugin Contract (Entity Model v0)

_Source: `docs/schema/entity-model.json` v0.1.0 (status: seed)._

This document enumerates the canonical fields, relationships, extension hooks, and invariants each plugin must respect. Generate it via `make entity-model-generate`.

## ID Semantics

- Type: `uuidv7`
- Scope: global
- Required: true
- Description: Opaque identifiers required for every entity instance.

## Enums

| Name | Values | Initial | Terminal | Description |
| --- | --- | --- | --- | --- |
| HousingEnvironment | `aquatic`<br>`terrestrial`<br>`arboreal`<br>`humid` | - | - | Canonical housing environments (ADR-0010 contextual helpers). |
| HousingState | `quarantine`<br>`active`<br>`cleaning`<br>`decommissioned` | `quarantine` | `decommissioned` | Housing lifecycle states (RFC-0001 §5.2). |
| LifecycleStage | `planned`<br>`embryo_larva`<br>`juvenile`<br>`adult`<br>`retired`<br>`deceased` | `planned` | `retired`<br>`deceased` | Organism lifecycle states (RFC-0001 §5.1). |
| PermitStatus | `draft`<br>`submitted`<br>`approved`<br>`on_hold`<br>`expired`<br>`archived` | - | - | Compliance lifecycle states for permits. |
| ProcedureStatus | `scheduled`<br>`in_progress`<br>`completed`<br>`cancelled`<br>`failed` | - | - | Procedure workflow states (RFC-0001 §5.4). |
| ProtocolStatus | `draft`<br>`submitted`<br>`approved`<br>`on_hold`<br>`expired`<br>`archived` | - | - | Compliance lifecycle states (RFC-0001 §5.3) used by contextual accessors. |
| SampleStatus | `stored`<br>`in_transit`<br>`consumed`<br>`disposed` | - | - | Sample custody states. |
| TreatmentStatus | `planned`<br>`in_progress`<br>`completed`<br>`flagged` | - | - | Treatment lifecycle states. |

## Entities

### BreedingUnit

Configured breeding group with lineage targets.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `strategy`

**Natural keys:**

- `name`, `line_id` (scope: line)

**States:** _none declared._

**Invariants:** `lineage_integrity`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `female_ids` | Organism | 0..n | fk |
| `housing_id` | HousingUnit | 0..1 | fk |
| `line_id` | Line | 0..1 | fk |
| `male_ids` | Organism | 0..n | fk |
| `protocol_id` | Protocol | 0..1 | fk |
| `strain_id` | Strain | 0..1 | fk |
| `target_line_id` | Line | 0..1 | fk |
| `target_strain_id` | Strain | 0..1 | fk |

**Extension hooks:** `pairing_attributes`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `created_at` | `timestamp` | Yes | - |
| `female_ids` | `array<uuid>` | No | - |
| `housing_id` | `uuid` | No | FK to HousingUnit |
| `id` | `uuid` | Yes | - |
| `line_id` | `uuid` | No | FK to Line |
| `male_ids` | `array<uuid>` | No | - |
| `name` | `string` | Yes | - |
| `pairing_attributes` | `ExtensionAttributes` | No | Pairing attribute extension slot |
| `pairing_intent` | `string` | No | - |
| `pairing_notes` | `string` | No | - |
| `protocol_id` | `uuid` | No | FK to Protocol |
| `strain_id` | `uuid` | No | FK to Strain |
| `strategy` | `string` | Yes | - |
| `target_line_id` | `uuid` | No | Target FK to Line |
| `target_strain_id` | `uuid` | No | Target FK to Strain |
| `updated_at` | `timestamp` | Yes | - |

### Cohort

Managed group of organisms bound to housing and project context.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `purpose`

**Natural keys:**

- `project_id`, `name` (scope: project)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `housing_id` | HousingUnit | 0..1 | fk |
| `project_id` | Project | 0..1 | fk |
| `protocol_id` | Protocol | 0..1 | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `created_at` | `timestamp` | Yes | - |
| `housing_id` | `uuid` | No | FK to HousingUnit |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `project_id` | `uuid` | No | FK to Project |
| `protocol_id` | `uuid` | No | FK to Protocol |
| `purpose` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Facility

Facility with zone and access policy constraints.

**Required fields:** `id`, `created_at`, `updated_at`, `code`, `name`, `zone`, `access_policy`

**Natural keys:**

- `code` (scope: global) — Facility code must be unique.

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `housing_unit_ids` | HousingUnit | 0..n | derived |
| `project_ids` | Project | 0..n | fk |

**Extension hooks:** `environment_baselines`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `access_policy` | `string` | Yes | - |
| `code` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `environment_baselines` | `ExtensionAttributes` | No | Facility environment baselines extension slot |
| `housing_unit_ids` | `array<uuid>` | No | - |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `project_ids` | `array<uuid>` | No | - |
| `updated_at` | `timestamp` | Yes | - |
| `zone` | `string` | Yes | - |

### GenotypeMarker

Genotype marker metadata with assay details.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `locus`, `alleles`, `assay_method`, `interpretation`, `version`

**Natural keys:**

- `name`, `version` (scope: global)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

_none_

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `alleles` | `array<string>` | Yes | - |
| `assay_method` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `id` | `uuid` | Yes | - |
| `interpretation` | `string` | Yes | - |
| `locus` | `string` | Yes | - |
| `name` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |
| `version` | `string` | Yes | - |

### HousingUnit

Physical housing with capacity and environmental baseline.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `facility_id`, `capacity`, `environment`, `state`

**Natural keys:**

- `facility_id`, `name` (scope: facility)

**States:** Enum `HousingState` (initial `quarantine`; terminal: `decommissioned`).

**Invariants:** `housing_capacity`, `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `facility_id` | Facility | 1..1 | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `capacity` | `integer` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `environment` | `enum HousingEnvironment` | Yes | - |
| `facility_id` | `uuid` | Yes | FK to Facility |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `state` | `enum HousingState` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Line

Genetic lineage definition.

**Required fields:** `id`, `created_at`, `updated_at`, `code`, `name`, `origin`, `genotype_marker_ids`

**Natural keys:**

- `code` (scope: global)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `genotype_marker_ids` | GenotypeMarker | 1..n | fk |

**Extension hooks:** `default_attributes`, `extension_overrides`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `code` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `default_attributes` | `ExtensionAttributes` | No | Default attributes extension slot |
| `deprecated_at` | `timestamp` | No | - |
| `deprecation_reason` | `string` | No | - |
| `description` | `string` | No | - |
| `extension_overrides` | `ExtensionAttributes` | No | Override attributes extension slot |
| `genotype_marker_ids` | `array<uuid>` | Yes | - |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `origin` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Observation

Observation or measurement captured during workflows.

**Required fields:** `id`, `created_at`, `updated_at`, `recorded_at`, `observer`

**Natural keys:**

- `procedure_id`, `recorded_at`, `observer` (scope: procedure)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `cohort_id` | Cohort | 0..1 | fk |
| `organism_id` | Organism | 0..1 | fk |
| `procedure_id` | Procedure | 0..1 | fk |

**Extension hooks:** `data`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `cohort_id` | `uuid` | No | FK to Cohort |
| `created_at` | `timestamp` | Yes | - |
| `data` | `ExtensionAttributes` | No | Schema-less observation payload |
| `id` | `uuid` | Yes | - |
| `notes` | `string` | No | - |
| `observer` | `string` | Yes | - |
| `organism_id` | `uuid` | No | FK to Organism |
| `procedure_id` | `uuid` | No | FK to Procedure |
| `recorded_at` | `timestamp` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Organism

Individual organism with lifecycle and housing context.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `species`, `line`, `stage`

**Natural keys:**

- `species`, `line`, `name` (scope: species)

**States:** Enum `LifecycleStage` (initial `planned`; terminal: `retired`, `deceased`).

**Invariants:** `housing_capacity`, `protocol_subject_cap`, `lineage_integrity`, `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `cohort_id` | Cohort | 0..1 | fk |
| `housing_id` | HousingUnit | 0..1 | fk |
| `line_id` | Line | 0..1 | fk |
| `parent_ids` | Organism | 0..n | fk |
| `project_id` | Project | 0..1 | fk |
| `protocol_id` | Protocol | 0..1 | fk |
| `strain_id` | Strain | 0..1 | fk |

**Extension hooks:** `attributes`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `attributes` | `ExtensionAttributes` | No | Species-agnostic extension slot |
| `cohort_id` | `uuid` | No | FK to Cohort |
| `created_at` | `timestamp` | Yes | - |
| `housing_id` | `uuid` | No | FK to HousingUnit |
| `id` | `uuid` | Yes | - |
| `line` | `string` | Yes | Human-readable line code or name. |
| `line_id` | `uuid` | No | FK to Line |
| `name` | `string` | Yes | - |
| `parent_ids` | `array<uuid>` | No | - |
| `project_id` | `uuid` | No | FK to Project |
| `protocol_id` | `uuid` | No | FK to Protocol |
| `species` | `string` | Yes | - |
| `stage` | `enum LifecycleStage` | Yes | - |
| `strain_id` | `uuid` | No | FK to Strain |
| `updated_at` | `timestamp` | Yes | - |

### Permit

External authorization for protocols and facilities.

**Required fields:** `id`, `created_at`, `updated_at`, `permit_number`, `authority`, `status`, `valid_from`, `valid_until`, `allowed_activities`, `facility_ids`, `protocol_ids`

**Natural keys:**

- `authority`, `permit_number` (scope: authority)

**States:** Enum `PermitStatus` (initial `draft`; terminal: `expired`, `archived`).

**Invariants:** `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `facility_ids` | Facility | 1..n | fk |
| `protocol_ids` | Protocol | 1..n | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `allowed_activities` | `array<string>` | Yes | - |
| `authority` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `facility_ids` | `array<uuid>` | Yes | - |
| `id` | `uuid` | Yes | - |
| `notes` | `string` | No | - |
| `permit_number` | `string` | Yes | - |
| `protocol_ids` | `array<uuid>` | Yes | - |
| `status` | `enum PermitStatus` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |
| `valid_from` | `timestamp` | Yes | - |
| `valid_until` | `timestamp` | Yes | - |

### Procedure

Scheduled or executed procedure with protocol coverage.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `status`, `scheduled_at`, `protocol_id`

**Natural keys:**

- `protocol_id`, `name`, `scheduled_at` (scope: protocol)

**States:** Enum `ProcedureStatus` (initial `scheduled`; terminal: `completed`, `cancelled`, `failed`).

**Invariants:** `protocol_coverage`, `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `cohort_id` | Cohort | 0..1 | fk |
| `observation_ids` | Observation | 0..n | fk |
| `organism_ids` | Organism | 0..n | fk |
| `project_id` | Project | 0..1 | fk |
| `protocol_id` | Protocol | 1..1 | fk |
| `treatment_ids` | Treatment | 0..n | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `cohort_id` | `uuid` | No | FK to Cohort |
| `created_at` | `timestamp` | Yes | - |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `observation_ids` | `array<uuid>` | No | - |
| `organism_ids` | `array<uuid>` | No | - |
| `project_id` | `uuid` | No | FK to Project |
| `protocol_id` | `uuid` | Yes | FK to Protocol |
| `scheduled_at` | `timestamp` | Yes | - |
| `status` | `enum ProcedureStatus` | Yes | - |
| `treatment_ids` | `array<uuid>` | No | - |
| `updated_at` | `timestamp` | Yes | - |

### Project

Project with facility and protocol affiliations.

**Required fields:** `id`, `created_at`, `updated_at`, `code`, `title`, `facility_ids`

**Natural keys:**

- `code` (scope: global)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `facility_ids` | Facility | 1..n | fk |
| `organism_ids` | Organism | 0..n | fk |
| `procedure_ids` | Procedure | 0..n | fk |
| `protocol_ids` | Protocol | 0..n | fk |
| `supply_item_ids` | SupplyItem | 0..n | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `code` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `description` | `string` | No | - |
| `facility_ids` | `array<uuid>` | Yes | - |
| `id` | `uuid` | Yes | - |
| `organism_ids` | `array<uuid>` | No | - |
| `procedure_ids` | `array<uuid>` | No | - |
| `protocol_ids` | `array<uuid>` | No | - |
| `supply_item_ids` | `array<uuid>` | No | - |
| `title` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Protocol

Compliance protocol with subject cap and status.

**Required fields:** `id`, `created_at`, `updated_at`, `code`, `title`, `max_subjects`, `status`

**Natural keys:**

- `code` (scope: global)

**States:** Enum `ProtocolStatus` (initial `draft`; terminal: `expired`, `archived`).

**Invariants:** `protocol_subject_cap`, `lifecycle_transition`

**Relationships**

_none_

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `code` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `description` | `string` | No | - |
| `id` | `uuid` | Yes | - |
| `max_subjects` | `integer` | Yes | - |
| `status` | `enum ProtocolStatus` | Yes | - |
| `title` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Sample

Sample with chain-of-custody and facility linkage.

**Required fields:** `id`, `created_at`, `updated_at`, `identifier`, `source_type`, `facility_id`, `collected_at`, `status`, `storage_location`, `assay_type`, `chain_of_custody`

**Natural keys:**

- `facility_id`, `identifier` (scope: facility)

**States:** Enum `SampleStatus` (initial `stored`; terminal: `consumed`, `disposed`).

**Invariants:** `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `cohort_id` | Cohort | 0..1 | fk |
| `facility_id` | Facility | 1..1 | fk |
| `organism_id` | Organism | 0..1 | fk |

**Extension hooks:** `attributes`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `assay_type` | `string` | Yes | - |
| `attributes` | `ExtensionAttributes` | No | Sample attribute extension slot |
| `chain_of_custody` | `array<SampleCustodyEvent>` | Yes | - |
| `cohort_id` | `uuid` | No | FK to Cohort |
| `collected_at` | `timestamp` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `facility_id` | `uuid` | Yes | FK to Facility |
| `id` | `uuid` | Yes | - |
| `identifier` | `string` | Yes | - |
| `organism_id` | `uuid` | No | FK to Organism |
| `source_type` | `string` | Yes | - |
| `status` | `enum SampleStatus` | Yes | - |
| `storage_location` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Strain

Managed strain derived from a Line.

**Required fields:** `id`, `created_at`, `updated_at`, `code`, `name`, `line_id`

**Natural keys:**

- `line_id`, `code` (scope: line)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `genotype_marker_ids` | GenotypeMarker | 0..n | fk |
| `line_id` | Line | 1..1 | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `code` | `string` | Yes | - |
| `created_at` | `timestamp` | Yes | - |
| `description` | `string` | No | - |
| `generation` | `string` | No | - |
| `genotype_marker_ids` | `array<uuid>` | No | - |
| `id` | `uuid` | Yes | - |
| `line_id` | `uuid` | Yes | FK to Line |
| `name` | `string` | Yes | - |
| `retired_at` | `timestamp` | No | - |
| `retirement_reason` | `string` | No | - |
| `updated_at` | `timestamp` | Yes | - |

### SupplyItem

Inventory item linked to facilities and projects.

**Required fields:** `id`, `created_at`, `updated_at`, `sku`, `name`, `quantity_on_hand`, `unit`, `facility_ids`, `project_ids`, `reorder_level`

**Natural keys:**

- `sku` (scope: global)

**States:** _none declared._

**Invariants:** _none declared._

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `facility_ids` | Facility | 1..n | fk |
| `project_ids` | Project | 1..n | fk |

**Extension hooks:** `attributes`

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `attributes` | `ExtensionAttributes` | No | Supply attribute extension slot |
| `created_at` | `timestamp` | Yes | - |
| `description` | `string` | No | - |
| `expires_at` | `timestamp` | No | - |
| `facility_ids` | `array<uuid>` | Yes | - |
| `id` | `uuid` | Yes | - |
| `lot_number` | `string` | No | - |
| `name` | `string` | Yes | - |
| `project_ids` | `array<uuid>` | Yes | - |
| `quantity_on_hand` | `integer` | Yes | - |
| `reorder_level` | `integer` | Yes | - |
| `sku` | `string` | Yes | - |
| `unit` | `string` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

### Treatment

Therapeutic intervention bound to procedure subjects.

**Required fields:** `id`, `created_at`, `updated_at`, `name`, `status`, `procedure_id`, `dosage_plan`

**Natural keys:**

- `procedure_id`, `name` (scope: procedure)

**States:** Enum `TreatmentStatus` (initial `planned`; terminal: `completed`, `flagged`).

**Invariants:** `protocol_coverage`, `lifecycle_transition`

**Relationships**

| Field | Target | Cardinality | Storage |
| --- | --- | --- | --- |
| `cohort_ids` | Cohort | 0..n | fk |
| `organism_ids` | Organism | 0..n | fk |
| `procedure_id` | Procedure | 1..1 | fk |

**Extension hooks:** _none_.

**Fields**

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `administration_log` | `array<string>` | No | - |
| `adverse_events` | `array<string>` | No | - |
| `cohort_ids` | `array<uuid>` | No | - |
| `created_at` | `timestamp` | Yes | - |
| `dosage_plan` | `string` | Yes | - |
| `id` | `uuid` | Yes | - |
| `name` | `string` | Yes | - |
| `organism_ids` | `array<uuid>` | No | - |
| `procedure_id` | `uuid` | Yes | FK to Procedure |
| `status` | `enum TreatmentStatus` | Yes | - |
| `updated_at` | `timestamp` | Yes | - |

<!--
CONTRACT-METADATA
{
  "version": "0.1.0",
  "entities": {
    "BreedingUnit": {
      "required": [
        "created_at",
        "id",
        "name",
        "strategy",
        "updated_at"
      ],
      "extension_hooks": [
        "pairing_attributes"
      ]
    },
    "Cohort": {
      "required": [
        "created_at",
        "id",
        "name",
        "purpose",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "Facility": {
      "required": [
        "access_policy",
        "code",
        "created_at",
        "id",
        "name",
        "updated_at",
        "zone"
      ],
      "extension_hooks": [
        "environment_baselines"
      ]
    },
    "GenotypeMarker": {
      "required": [
        "alleles",
        "assay_method",
        "created_at",
        "id",
        "interpretation",
        "locus",
        "name",
        "updated_at",
        "version"
      ],
      "extension_hooks": []
    },
    "HousingUnit": {
      "required": [
        "capacity",
        "created_at",
        "environment",
        "facility_id",
        "id",
        "name",
        "state",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "Line": {
      "required": [
        "code",
        "created_at",
        "genotype_marker_ids",
        "id",
        "name",
        "origin",
        "updated_at"
      ],
      "extension_hooks": [
        "default_attributes",
        "extension_overrides"
      ]
    },
    "Observation": {
      "required": [
        "created_at",
        "id",
        "observer",
        "recorded_at",
        "updated_at"
      ],
      "extension_hooks": [
        "data"
      ]
    },
    "Organism": {
      "required": [
        "created_at",
        "id",
        "line",
        "name",
        "species",
        "stage",
        "updated_at"
      ],
      "extension_hooks": [
        "attributes"
      ]
    },
    "Permit": {
      "required": [
        "allowed_activities",
        "authority",
        "created_at",
        "facility_ids",
        "id",
        "permit_number",
        "protocol_ids",
        "status",
        "updated_at",
        "valid_from",
        "valid_until"
      ],
      "extension_hooks": []
    },
    "Procedure": {
      "required": [
        "created_at",
        "id",
        "name",
        "protocol_id",
        "scheduled_at",
        "status",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "Project": {
      "required": [
        "code",
        "created_at",
        "facility_ids",
        "id",
        "title",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "Protocol": {
      "required": [
        "code",
        "created_at",
        "id",
        "max_subjects",
        "status",
        "title",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "Sample": {
      "required": [
        "assay_type",
        "chain_of_custody",
        "collected_at",
        "created_at",
        "facility_id",
        "id",
        "identifier",
        "source_type",
        "status",
        "storage_location",
        "updated_at"
      ],
      "extension_hooks": [
        "attributes"
      ]
    },
    "Strain": {
      "required": [
        "code",
        "created_at",
        "id",
        "line_id",
        "name",
        "updated_at"
      ],
      "extension_hooks": []
    },
    "SupplyItem": {
      "required": [
        "created_at",
        "facility_ids",
        "id",
        "name",
        "project_ids",
        "quantity_on_hand",
        "reorder_level",
        "sku",
        "unit",
        "updated_at"
      ],
      "extension_hooks": [
        "attributes"
      ]
    },
    "Treatment": {
      "required": [
        "created_at",
        "dosage_plan",
        "id",
        "name",
        "procedure_id",
        "status",
        "updated_at"
      ],
      "extension_hooks": []
    }
  }
}
-->
