# Sample R helper for ColonyCore Dataset Service API
# Requires the httr and jsonlite packages.

library(httr)
library(jsonlite)

cc_dataset_headers <- function(token = NULL) {
  headers <- add_headers(
    `User-Agent` = "colonycore-dataset-client/0.1",
    Accept = "application/json"
  )
  if (!is.null(token) && nzchar(token)) {
    headers <- add_headers(headers, Authorization = paste("Bearer", token))
  }
  headers
}

cc_list_templates <- function(base_url, token = NULL, timeout = 30) {
  url <- paste0(rtrim(base_url), "/api/v1/datasets/templates")
  resp <- GET(url, cc_dataset_headers(token), timeout(timeout))
  stop_for_status(resp)
  content(resp, as = "parsed", simplifyVector = TRUE)$templates
}

cc_get_template <- function(
  base_url,
  plugin,
  key,
  version,
  token = NULL,
  timeout = 30
) {
  url <- sprintf(
    "%s/api/v1/datasets/templates/%s/%s/%s",
    rtrim(base_url),
    plugin,
    key,
    version
  )
  resp <- GET(url, cc_dataset_headers(token), timeout(timeout))
  stop_for_status(resp)
  content(resp, as = "parsed", simplifyVector = TRUE)$template
}

cc_validate_template <- function(
  base_url,
  plugin,
  key,
  version,
  parameters = list(),
  token = NULL,
  timeout = 30
) {
  url <- sprintf(
    "%s/api/v1/datasets/templates/%s/%s/%s/validate",
    rtrim(base_url),
    plugin,
    key,
    version
  )
  resp <- POST(
    url,
    cc_dataset_headers(token),
    timeout(timeout),
    body = list(parameters = parameters),
    encode = "json"
  )
  stop_for_status(resp)
  content(resp, as = "parsed", simplifyVector = TRUE)
}

cc_run_template <- function(
  base_url,
  plugin,
  key,
  version,
  parameters = list(),
  scope = list(),
  format = "json",
  token = NULL,
  timeout = 60
) {
  url <- sprintf(
    "%s/api/v1/datasets/templates/%s/%s/%s/run",
    rtrim(base_url),
    plugin,
    key,
    version
  )
  query <- list(format = tolower(format))
  headers <- cc_dataset_headers(token)
  if (tolower(format) == "csv") {
    headers <- add_headers(headers, Accept = "text/csv")
  }
  resp <- POST(
    url,
    headers,
    timeout(timeout),
    query = query,
    body = list(parameters = parameters, scope = scope),
    encode = "json"
  )
  stop_for_status(resp)
  if (tolower(format) == "csv") {
    return(content(resp, as = "text", encoding = "UTF-8"))
  }
  content(resp, as = "parsed", simplifyVector = TRUE)
}

cc_queue_export <- function(
  base_url,
  template_slug = NULL,
  plugin = NULL,
  key = NULL,
  version = NULL,
  parameters = list(),
  scope = list(),
  formats = character(),
  requested_by = NULL,
  reason = NULL,
  project_id = NULL,
  protocol_id = NULL,
  token = NULL,
  timeout = 30
) {
  template <- if (!is.null(template_slug) && nzchar(template_slug)) {
    list(slug = template_slug)
  } else {
    if (any(!nzchar(c(plugin, key, version)))) {
      stop(
        "plugin, key, and version must be supplied when slug is omitted",
        call. = FALSE
      )
    }
    list(plugin = plugin, key = key, version = version)
  }
  body <- list(
    template = template,
    parameters = parameters,
    scope = scope,
    formats = as.list(formats),
    requested_by = requested_by,
    reason = reason,
    project_id = project_id,
    protocol_id = protocol_id
  )
  url <- paste0(rtrim(base_url), "/api/v1/datasets/exports")
  resp <- POST(
    url,
    cc_dataset_headers(token),
    timeout(timeout),
    body = body,
    encode = "json"
  )
  stop_for_status(resp)
  content(resp, as = "parsed", simplifyVector = TRUE)$export
}

cc_get_export <- function(base_url, export_id, token = NULL, timeout = 30) {
  url <- sprintf("%s/api/v1/datasets/exports/%s", rtrim(base_url), export_id)
  resp <- GET(url, cc_dataset_headers(token), timeout(timeout))
  stop_for_status(resp)
  content(resp, as = "parsed", simplifyVector = TRUE)$export
}

cc_wait_for_export <- function(
  base_url,
  export_id,
  token = NULL,
  poll_seconds = 2,
  timeout = 300
) {
  deadline <- Sys.time() + timeout
  repeat {
    export <- cc_get_export(base_url, export_id, token = token)
    if (export$status %in% c("succeeded", "failed")) {
      return(export)
    }
    if (Sys.time() > deadline) {
      stop(sprintf("export %s timed out", export_id), call. = FALSE)
    }
    Sys.sleep(poll_seconds)
  }
}

cc_download_artifact <- function(url, path = NULL, token = NULL, timeout = 60) {
  resp <- GET(url, cc_dataset_headers(token), timeout(timeout))
  stop_for_status(resp)
  if (is.null(path)) {
    return(content(resp, as = "raw"))
  }
  dir.create(dirname(path), showWarnings = FALSE, recursive = TRUE)
  writeBin(content(resp, as = "raw"), path)
  invisible(path)
}

rtrim <- function(x) {
  sub("/*$", "", x)
}
