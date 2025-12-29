#!/usr/bin/env Rscript

read_env <- function(name, default = NA_character_) {
  value <- Sys.getenv(name, unset = "")
  if (!nzchar(value) && !is.na(default)) {
    value <- default
  }
  if (!nzchar(value)) {
    stop(sprintf("%s must be set", name), call. = FALSE)
  }
  value
}

repos <- read_env("LINTR_REPO", "https://cloud.r-project.org")
lintr_version <- read_env("LINTR_VERSION")
xml2_version <- read_env("XML2_VERSION")

lib_dir <- Sys.getenv("R_LIBS_USER", unset = "")
if (!nzchar(lib_dir)) {
  stop("R_LIBS_USER must be set for lintr setup", call. = FALSE)
}
dir.create(lib_dir, recursive = TRUE, showWarnings = FALSE)

lock_dirs <- list.files(lib_dir, pattern = "^00LOCK", full.names = TRUE)
if (length(lock_dirs) > 0) {
  unlink(lock_dirs, recursive = TRUE, force = TRUE)
}

ensure_remotes <- function(repos) {
  if (!requireNamespace("remotes", quietly = TRUE)) {
    install.packages("remotes", repos = repos, dependencies = TRUE)
  }
  if (!requireNamespace("remotes", quietly = TRUE)) {
    stop("Failed to install the 'remotes' package", call. = FALSE)
  }
}

needs_install <- function(pkg, version) {
  if (!requireNamespace(pkg, quietly = TRUE)) {
    return(TRUE)
  }
  installed <- as.character(utils::packageVersion(pkg))
  installed != version
}

install_version <- function(pkg, version, repos, lib_dir) {
  if (!needs_install(pkg, version)) {
    return(invisible(FALSE))
  }
  pkg_path <- file.path(lib_dir, pkg)
  if (dir.exists(pkg_path)) {
    unlink(pkg_path, recursive = TRUE, force = TRUE)
  }
  message(sprintf("Installing %s (%s)", pkg, version))
  tryCatch(
    remotes::install_version(
      pkg,
      version = version,
      repos = repos,
      dependencies = TRUE,
      upgrade = "never"
    ),
    error = function(err) {
      stop(sprintf("Failed to install %s %s: %s", pkg, version, err$message), call. = FALSE)
    }
  )
  if (!requireNamespace(pkg, quietly = TRUE)) {
    stop(sprintf("Package %s failed to load after installation", pkg), call. = FALSE)
  }
  installed <- as.character(utils::packageVersion(pkg))
  if (installed != version) {
    stop(sprintf("Installed %s %s, expected %s", pkg, installed, version), call. = FALSE)
  }
  invisible(TRUE)
}

ensure_remotes(repos)
install_version("lintr", lintr_version, repos, lib_dir)
install_version("xml2", xml2_version, repos, lib_dir)
