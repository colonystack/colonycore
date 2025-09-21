# Annex 0001: Operational Risk & Compliance

- Status: Draft
- Linked RFCs: 0001-colonycore-base-module
- Owners: Tobias Harnickell
- Last Updated: 2025-09-21

## 1. Risk Register
| Risk | Impact | Probability | Mitigations | Owner |
| --- | --- | --- | --- | --- |
| Plugin logic corrupts core data states | High: cross-species outages, compliance violations | Medium | Transactional validation hooks, contract tests, staged rollout with canary tenants | Platform Engineering |
| Compliance rules misconfigured for jurisdiction | High: regulatory penalties, halted research | Medium | Jurisdiction templates, policy DSL linting, dual-approval workflow for rule changes | Compliance PMO |
| Hardware ingestion outage (sensors/scanners) | Medium: delayed husbandry evidence, manual workarounds | Medium | MQTT/HTTP queue buffering, offline capture UI, SLA monitoring & alerting | Integrations Team |
| Unauthorized data access via plugin API | Critical: breach of privacy & ethics commitments | Low | Least-privilege RBAC, capability sandboxing, automated security tests, third-party pen-test | Security Engineering |
| Performance regressions at scale | Medium: degraded UX, staff rework | Medium | Performance budgets in CI, synthetic load tests, async job prioritization policies | Reliability Engineering |

## 2. Threat Model
- **Threat surfaces:** authenticated APIs, plugin extensions, hardware ingestion endpoints, administrative interfaces, database access, and support tooling.
- **Primary threat categories:** privilege escalation, data exfiltration, replay or injection on device ingest, plugin supply-chain compromise, and inference attacks on pseudonymized data.
- **Controls:** field-level encryption for Class A data, per-tenant secret rotation, signed webhook payloads, mutual TLS for hardware bridges, continuous vulnerability scanning, plugin signature verification, and runtime policy enforcement (OPA) for high-risk actions.
- **Residual risks:** offline exports handled outside the system boundary require manual attestations and scheduled deletion audits.

## 3. Data Classification & Retention
### 3.1 Classification
- Class A (Regulated) covers protocol approvals, humane endpoint decisions, and compliance signatures.
- Class B (Sensitive Operational) covers organism health, lineage, environmental sensor feeds, and audit logs.
- Class C (Support) covers supply inventory, scheduling metadata, and anonymized telemetry.

### 3.2 Retention Matrix
| Jurisdiction | Entity Classes | Retention Baseline | Deletion & Archival Process | Notes |
| --- | --- | --- | --- | --- |
| EU (Directive 2010/63) | Organism, Procedure, Protocol, Audit logs (Class A/B) | 10 years post-study completion | Automated archival to a WORM store. Deletion requires approval from Compliance and Legal, and hashes remain in the audit ledger. | Aligns with Article 37 record-keeping guidance |
| US (IACUC/NIH) | Organism, Procedure, Training, Protocol (Class A/B) | 3 years after study closure | Move data to cold storage and purge it from the primary database. Maintain redacted metadata for reporting. | NIH OLAW guidance 2009-01 |
| UK (Home Office) | Breeding records, Humane endpoint actions (Class A/B) | 5 years after animal death | Schedule a purge job with supervisor attestation, and preserve the audit log indefinitely. | Incorporates ASPA code of practice |
| Global default | Class C datasets (inventory, telemetry) | 2 years rolling | Run a continuous deletion job while retaining aggregate summaries. | Can be overridden per tenant policy |

### 3.3 Open Items
- Determine an automated detection strategy for offline export retention breaches.
- Validate jurisdictional baselines with legal counsel and update this annex when localized policies are finalized.
