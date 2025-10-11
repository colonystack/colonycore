# Dataset Model Mapping

The table below maps the legacy dataset types in `internal/core/dataset.go` to
the canonical equivalents in `pkg/datasetapi`.

| internal/core type / helper                | pkg/datasetapi counterpart                         |
|-------------------------------------------|----------------------------------------------------|
| `DatasetDialect`                          | `Dialect`                                          |
| `DatasetDialectSQL`, `DatasetDialectDSL`  | `DialectSQL`, `DialectDSL`                         |
| `DatasetFormat`                           | `Format`                                           |
| `FormatJSON`, `FormatCSV`, `FormatParquet`, `FormatPNG`, `FormatHTML` | `FormatJSON`, `FormatCSV`, `FormatParquet`, `FormatPNG`, `FormatHTML` |
| `DatasetScope`                            | `Scope`                                            |
| `DatasetParameter`                        | `Parameter`                                        |
| `DatasetColumn`                           | `Column`                                           |
| `DatasetTemplateMetadata`                 | `Metadata`                                         |
| `DatasetBinder`                           | `Binder`                                           |
| `DatasetRunner`                           | `Runner`                                           |
| `DatasetEnvironment`                      | `Environment`                                      |
| `DatasetRunRequest`                       | `RunRequest`                                       |
| `DatasetRunResult`                        | `RunResult`                                        |
| `DatasetParameterError`                   | `ParameterError`                                   |
| `DatasetTemplateDescriptor`               | `TemplateDescriptor`                               |
| `DatasetTemplate` (host-bound struct)     | `HostTemplate` (new)                               |
| `DatasetTemplateCollection`               | `SortTemplateDescriptors` helper (new)             |

Supporting helpers such as parameter validation, slug computation, and runtime
binding now live in `pkg/datasetapi/host_template.go` and operate directly on
the canonical dataset structures.
