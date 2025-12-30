#!/usr/bin/env python3
"""Run lintr against the R client helpers (or install deps with --setup-only)."""

from __future__ import annotations

import argparse
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


def _parse_args() -> argparse.Namespace:
    """
    Parse command-line arguments for the lintr runner.
    
    The returned namespace contains the parsed CLI options:
    - `setup_only`: `True` when dependencies should be installed/verified without running linters.
    - `paths`: list of positional path arguments provided by the user.
    
    Returns:
        argparse.Namespace: Namespace with `setup_only` (bool) and `paths` (list[str]).
    """
    parser = argparse.ArgumentParser(description="Run lintr against the R client helpers.")
    parser.add_argument(
        "--setup-only",
        action="store_true",
        help="Install or verify lintr dependencies without running lint.",
    )
    parser.add_argument("paths", nargs="*", help=argparse.SUPPRESS)
    return parser.parse_args()


def main() -> int:
    """
    Run R dependency setup and optionally execute lintr for the repository's R clients.
    
    Performs these actions: locates Rscript, prepares a user R library, ensures required R packages and versions are present (installing them if allowed), and when not run in setup-only mode, runs lintr::lint_dir on clients/R. Behavior can be influenced via environment variables:
    - LINTR_REPO: R repository URL hint (default: https://cloud.r-project.org)
    - LINTR_SKIP_AUTO_INSTALL: when truthy, skip automatic installation of missing/mismatched packages
    - R_LIBS_USER: user R library directory (defaults to .cache/R-lintr under the repo root)
    - REQUIRE_R_LINT: when truthy, treat R runtime/shared-library detection errors as fatal
    
    Returns:
        int: Exit code suitable for use as a process return value (0 on success; non-zero on failure).
    """
    args = _parse_args()
    lint_enabled = not args.setup_only
    action_label = "R lint" if lint_enabled else "R lint setup"

    rscript = shutil.which("Rscript")
    if not rscript:
        sys.stderr.write(
            "Rscript not found. Install R (>=4.0); the hook will install the lintr and xml2 packages automatically.\n"
        )
        return 1

    env = os.environ.copy()
    env.setdefault("R_LIBS_USER", str(REPO_ROOT / ".cache" / "R-lintr"))
    env.setdefault("R_INSTALL_STAGED", "false")

    user_lib = Path(env["R_LIBS_USER"]).expanduser()
    user_lib.mkdir(parents=True, exist_ok=True)
    for lock_dir in user_lib.glob("00LOCK*"):
        shutil.rmtree(lock_dir, ignore_errors=True)

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
        'lib_dir <- Sys.getenv("R_LIBS_USER")',
        'if (nzchar(lib_dir)) { .libPaths(c(lib_dir, .Library)) }',
        f'required <- c({required_assignments})',
        'check_required <- function(required, lib_dir) {',
        '  vapply(names(required), function(pkg) {',
        '    if (pkg %in% loadedNamespaces()) {',
        '      pkg_path <- tryCatch(getNamespaceInfo(pkg, "path"), error = function(err) "")',
        '      if (nzchar(lib_dir) && nzchar(pkg_path) && !startsWith(pkg_path, lib_dir)) {',
        '        unload_error <- NULL',
        '        tryCatch(',
        '          unloadNamespace(pkg),',
        '          error = function(err) { unload_error <<- err }',
        '        )',
        '        if (!is.null(unload_error)) {',
        '          message(sprintf("Failed to unload namespace %s from %s: %s", pkg, pkg_path, conditionMessage(unload_error)))',
        '        }',
        '        if (pkg %in% loadedNamespaces()) {',
        '          reloaded_path <- tryCatch(getNamespaceInfo(pkg, "path"), error = function(err) "")',
        '          if (nzchar(reloaded_path) && !startsWith(reloaded_path, lib_dir)) {',
        '            warning(sprintf("Namespace %s remains loaded from %s; expected path under %s. Subsequent requireNamespace calls may use the wrong library.", pkg, reloaded_path, lib_dir), call. = FALSE)',
        '          }',
        '        }',
        '      }',
        '    }',
        '    if (!requireNamespace(pkg, quietly = TRUE, lib.loc = lib_dir)) {',
        '      return("missing")',
        '    }',
        '    installed <- as.character(utils::packageVersion(pkg, lib.loc = lib_dir))',
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
        'status <- check_required(required, lib_dir)',
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
        '  lib_dir <- Sys.getenv("R_LIBS_USER")',
        '  for (pkg in names(required)[needs_install]) {',
        '    target_version <- required[[pkg]]',
        '    pkg_path <- file.path(lib_dir, pkg)',
        '    if (dir.exists(pkg_path)) {',
        '      unlink(pkg_path, recursive = TRUE, force = TRUE)',
        '    }',
        '    message(sprintf("Installing %s (version %s)", pkg, target_version))',
        '    remotes::install_version(pkg, version = target_version, repos = repos, dependencies = TRUE, upgrade = FALSE)',
        '  }',
        '  status <- check_required(required, lib_dir)',
        '  needs_install <- status != ""',
        '  if (any(needs_install)) {',
        '    details <- report_status(names(required)[needs_install], status, required)',
        '    stop(paste0("Unable to install required R packages -> ", paste(details, collapse = ", ")))',
        '  }',
        '}',
        'invisible(lapply(names(required), function(pkg) requireNamespace(pkg, quietly = TRUE, lib.loc = lib_dir)))',
    ]
    if lint_enabled:
        r_lines.extend(
            [
                'results <- lintr::lint_dir("clients/R", relative_path = TRUE, show_progress = FALSE)',
                'if (length(results)) {',
                '  lintr::print.lints(results)',
                '  quit(save = "no", status = 1)',
                '}',
            ]
        )

    command = [rscript, "--vanilla", "-e", "\n".join(r_lines)]

    # Run R command; capture output to detect environment (shared library) issues distinctly
    result = subprocess.run(command, env=env, check=False, capture_output=True, text=True)

    if result.returncode == 0:
        return 0

    combined = (result.stdout or "") + (result.stderr or "")
    # Detect dynamic loader / shared library problems (e.g., missing BLAS) and optionally downgrade to skip
    if "error while loading shared libraries" in combined.lower() or "libblas.so" in combined.lower():
        require = _normalize_bool(os.environ.get("REQUIRE_R_LINT"))
        msg = (
            "R runtime dependency issue detected (likely missing BLAS/lib dependencies). "
            f"Skipping {action_label}. Set REQUIRE_R_LINT=1 to enforce failure."
        )
        if require:
            sys.stderr.write(combined)
            sys.stderr.write("\n" + msg + "\n")
            return result.returncode or 1
        else:
            sys.stderr.write(msg + "\n")
            return 0

    # For normal failures (actual lint errors etc.), replay output so caller sees details
    sys.stderr.write(combined)
    return result.returncode or 1


if __name__ == "__main__":
    raise SystemExit(main())