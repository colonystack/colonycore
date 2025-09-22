#!/usr/bin/env python3
"""Run lintr against the R client helpers."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path


def _normalize_bool(value: str | None) -> bool:
    if value is None:
        return False
    return value.strip().lower() in {"1", "true", "yes", "on"}


def _escape_for_r(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


REPO_ROOT = Path(__file__).resolve().parent.parent
REQUIRED_R_PACKAGES: dict[str, str] = {
    "lintr": "3.1.2",
    "xml2": "1.3.6",
}


def main() -> int:
    rscript = shutil.which("Rscript")
    if not rscript:
        sys.stderr.write(
            "Rscript not found. Install R (>=4.0); the hook will install the lintr and xml2 packages automatically.\n"
        )
        return 1

    repo_hint = os.environ.get("LINTR_REPO", "https://cloud.r-project.org")
    skip_install = _normalize_bool(os.environ.get("LINTR_SKIP_AUTO_INSTALL"))

    repo_r = _escape_for_r(repo_hint)
    skip_literal = "TRUE" if skip_install else "FALSE"
    required_assignments = ", ".join(
        f"{pkg} = \"{_escape_for_r(version)}\"" for pkg, version in REQUIRED_R_PACKAGES.items()
    )

    r_lines = [
        f'repos <- Sys.getenv("LINTR_REPO", unset="{repo_r}")',
        f'if (!nzchar(repos)) repos <- "{repo_r}"',
        f'skip_install <- {skip_literal}',
        f'required <- c({required_assignments})',
        'check_required <- function(required) {',
        '  vapply(names(required), function(pkg) {',
        '    if (!requireNamespace(pkg, quietly = TRUE)) {',
        '      return("missing")',
        '    }',
        '    installed <- as.character(utils::packageVersion(pkg))',
        '    if (installed != required[[pkg]]) {',
        '      return(installed)',
        '    }',
        '    ""',
        '  }, character(1), USE.NAMES = TRUE)',
        '}',
        'report_status <- function(pkgs, status, required) {',
        '  vapply(pkgs, function(pkg) {',
        '    if (status[[pkg]] == "missing") {',
        '      sprintf("%s: missing (expected %s)", pkg, required[[pkg]])',
        '    } else {',
        '      sprintf("%s: %s (expected %s)", pkg, status[[pkg]], required[[pkg]])',
        '    }',
        '  }, character(1))',
        '}',
        'status <- check_required(required)',
        'needs_install <- status != ""',
        'if (any(needs_install)) {',
        '  details <- report_status(names(required)[needs_install], status, required)',
        '  if (skip_install) {',
        '    stop(paste0("Missing or mismatched R packages -> ", paste(details, collapse = ", "), ". ",',
        '      "Install them manually (remotes::install_version) or unset LINTR_SKIP_AUTO_INSTALL."))',
        '  }',
        '  if (!requireNamespace("remotes", quietly = TRUE)) {',
        '    install.packages("remotes", repos = repos, dependencies = TRUE)',
        '  }',
        '  for (pkg in names(required)[needs_install]) {',
        '    target_version <- required[[pkg]]',
        '    message(sprintf("Installing %s (version %s)", pkg, target_version))',
        '    remotes::install_version(pkg, version = target_version, repos = repos, dependencies = TRUE, upgrade = FALSE)',
        '  }',
        '  status <- check_required(required)',
        '  needs_install <- status != ""',
        '  if (any(needs_install)) {',
        '    details <- report_status(names(required)[needs_install], status, required)',
        '    stop(paste0("Unable to install required R packages -> ", paste(details, collapse = ", ")))',
        '  }',
        '}',
        'invisible(lapply(names(required), function(pkg) requireNamespace(pkg, quietly = TRUE)))',
        'results <- lintr::lint_dir("clients/R", relative_path = TRUE, show_progress = FALSE)',
        'if (length(results)) {',
        '  lintr::print.lints(results)',
        '  quit(save = "no", status = 1)',
        '}',
    ]

    command = [rscript, "--vanilla", "-e", "\n".join(r_lines)]

    env = os.environ.copy()
    env.setdefault("R_LIBS_USER", str(REPO_ROOT / ".cache" / "R-lintr"))
    Path(env["R_LIBS_USER"]).expanduser().mkdir(parents=True, exist_ok=True)

    result = subprocess.run(command, env=env, check=False)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
