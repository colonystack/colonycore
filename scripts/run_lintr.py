#!/usr/bin/env python3
"""Run lintr against the R client helpers."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys


def _normalize_bool(value: str | None) -> bool:
    if value is None:
        return False
    return value.strip().lower() in {"1", "true", "yes", "on"}


def _escape_for_r(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


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
    required_pkgs = ["lintr", "xml2"]
    pkgs_r = ", ".join(f'"{_escape_for_r(pkg)}"' for pkg in required_pkgs)

    r_lines = [
        f'repos <- Sys.getenv("LINTR_REPO", unset="{repo_r}")',
        f'if (!nzchar(repos)) repos <- "{repo_r}"',
        f'pkgs <- c({pkgs_r})',
        'missing <- pkgs[!vapply(pkgs, requireNamespace, logical(1), quietly = TRUE)]',
    ]
    if skip_install:
        r_lines.extend(
            [
                'if (length(missing)) {',
                '  stop(sprintf(paste0(',
                '    "Missing R packages: %s. Install them with install.packages() after ensuring ",',
                '    "system libraries (libcurl, libxml2) exist, or unset LINTR_SKIP_AUTO_INSTALL."),',
                '    paste(missing, collapse = ", ")))',
                '}',
            ]
        )
    else:
        r_lines.extend(
            [
                'if (length(missing)) {',
                '  message("Installing required R packages: ", paste(missing, collapse = ", "))',
                '  install_ok <- tryCatch({',
                '    install.packages(missing, repos = repos, dependencies = TRUE)',
                '    TRUE',
                '  }, error = function(e) {',
                '    message("install.packages error: ", conditionMessage(e))',
                '    FALSE',
                '  })',
                '  if (!install_ok) {',
                '    message("Auto-install could not complete; falling back to manual resolution.")',
                '  }',
                '  missing <- pkgs[!vapply(pkgs, requireNamespace, logical(1), quietly = TRUE)]',
                '  if (length(missing)) {',
                '    stop(sprintf(paste0(',
                '      "Missing R packages even after install attempt: %s. ",',
                '      "Install system dependencies (e.g. libcurl4-openssl-dev libxml2-dev libxslt1-dev) ",',
                '      "and retry, or pre-install the R packages and export LINTR_SKIP_AUTO_INSTALL=1."),',
                '      paste(missing, collapse = ", ")))',
                '  }',
                '}',
            ]
        )

    r_lines.append(
        'invisible(lapply(pkgs, function(pkg) requireNamespace(pkg, quietly = TRUE)))'
    )
    r_lines.append(
        'results <- lintr::lint_dir("clients/R", relative_path = TRUE, show_progress = FALSE)'
    )
    r_lines.extend(
        [
            'if (length(results)) {',
            '  lintr::print.lints(results)',
            '  quit(save = "no", status = 1)',
            '}',
        ]
    )

    command = [rscript, "--vanilla", "-e", "\n".join(r_lines)]

    result = subprocess.run(command, check=False)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
