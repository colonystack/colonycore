# Python Client Linting

The dataset client under `clients/python` is linted with [Ruff](https://docs.astral.sh/ruff/) for a fast, opinionated check of the public surface.

## Toolchain and pinning
- Ruff is pinned to **0.5.7** for this client. Install it with `python -m pip install --require-virtualenv -r clients/python/requirements-lint.txt` or via your environment manager of choice.
- Pre-commit runs `make lint`, which delegates to the pinned Ruff version above; update `requirements-lint.txt` when bumping the linter.

## Configuration
- Rules live in `clients/python/ruff.toml` (referenced by the root `pyproject.toml`).
- Current settings:
  - `line-length = 120` to match R and Go clients.
  - `target-version = "py311"` to reflect the supported runtime.
  - `select = ["E", "F"]` (Pyflakes and pycodestyle errors) with `ignore = ["E501"]` for the relaxed line length.

## Running the linter
- Use `make python-lint` during local development; it wraps the Ruff invocation below.
- `make lint` (or `pre-commit run --all-files`) runs both client linters alongside the Go toolchain, mirroring CI.

## Troubleshooting
- Missing Ruff: install the pinned version via the requirements file above or `pipx install ruff==0.5.7`.
- Platform-specific wheels: if you see compilation errors, upgrade `pip`/`setuptools` and reinstall Ruff; pre-built wheels exist for macOS, Linux, and Windows.
- Unexpected rules: inspect `clients/python/ruff.toml`; override locally only in throwaway branches and send a PR if a rule change benefits everyone.

## Auto-fix guidance
Ruff can fix many rule classes automatically. Run `python -m ruff check --config clients/python/ruff.toml --fix clients/python` (or append `--select <code>` for specific fixes). Always rerun `make python-lint` afterwards to confirm the working tree is clean.
