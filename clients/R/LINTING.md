# R Client Linting

The dataset client under `clients/R` is linted with [lintr](https://lintr.r-lib.org) to keep style consistent across contributors.

## Toolchain and pinning
- `lintr` **3.1.2** and its dependency `xml2` **1.3.6** are required.
- `scripts/run_lintr.py` bootstraps those exact versions via `remotes::install_version()` into `.cache/R-lintr` unless you export `LINTR_SKIP_AUTO_INSTALL=1`.
- Override the CRAN mirror with `LINTR_REPO=https://cloud.r-project.org` (default) when mirroring locally.

## Configuration
- Rules live in `clients/R/.lintr` and currently enforce:
  - Defaults from `lintr::linters_with_defaults()`.
  - An explicit `line_length_linter(120)` to match the Python client.
- Update this file when changing lint behaviour; document the change in this README for reviewers.

## Running the linter
- Use `make r-lint` to lint just the R client while iterating.
- `make lint` (or `pre-commit run --all-files`) runs the full suite, matching CI.

## Troubleshooting
- `Rscript` missing: install R ≥ 4.0 (`sudo apt install r-base` on Debian/Ubuntu) and ensure it's on `PATH`.
- R fails to start with `libblas.so.3` (or other shared library) errors: install a BLAS runtime (`libblas3` on Debian/Ubuntu) and rerun. Set `REQUIRE_R_LINT=1` if you want missing runtime deps to fail the lint step.
- Package install failures: install system deps (`libcurl4-openssl-dev libxml2-dev libxslt1-dev`) and rerun. As a fallback, run `make r-lint-setup` to preinstall the pinned packages.
- `shared object 'rlang.so' not found`: the rlang native library failed to build. Run `make r-lint-reset` then `make r-lint-setup`, and rerun `make r-lint`.
- Version mismatch after install (for example `xml2: 1.3.3 (expected 1.3.6)`): run `make r-lint-reset`, then `make r-lint-setup`, then rerun `make r-lint`. Opening a fresh shell can also help pick up updated packages.
- Respect `LINTR_SKIP_AUTO_INSTALL=1` if you manage packages yourself; the script will then fail fast with a list of mismatched versions.

## Auto-fix guidance
`lintr` surfaces issues but does not fix code automatically. For stylistic corrections, run `Rscript -e "styler::style_dir('clients/R')"` once you have `styler` installed, or update the code manually until `make r-lint` passes.
