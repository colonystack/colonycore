# Dataset Template Fixtures

These fixtures exercise dataset template validation rules implemented in `pkg/datasetapi`
and exposed through `colony lint dataset`.

Structure:

- `valid/`: expected to pass validation.
- `invalid/`: expected to fail validation.
- `edge/`: unusual but valid templates that guard boundary cases.

Quick checks:

```bash
go run ./cmd/colony lint dataset testutil/fixtures/dataset-templates/valid testutil/fixtures/dataset-templates/edge
go run ./cmd/colony lint dataset testutil/fixtures/dataset-templates/invalid
```

Notes:

- Fixtures use the transport shape (`datasetapi.TemplateDescriptor`) because runtime
  binders are code, not JSON.
- The CLI validates required fields, parameter/column/output invariants, and rendering
  semantics for SQL/DSL queries.
