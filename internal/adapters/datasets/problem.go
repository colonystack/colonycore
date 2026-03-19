package datasets

import (
	"encoding/json"
	"net/http"

	"colonycore/pkg/datasetapi"
)

const (
	problemContentType = "application/problem+json"
	problemTypeBlank   = "about:blank"
)

type problemDetail struct {
	Type   string                      `json:"type"`
	Title  string                      `json:"title"`
	Status int                         `json:"status"`
	Detail string                      `json:"detail"`
	Errors []datasetapi.ParameterError `json:"errors,omitempty"`
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	writeProblemWithErrors(w, status, detail, nil)
}

func writeProblemWithErrors(w http.ResponseWriter, status int, detail string, errs []datasetapi.ParameterError) {
	title := http.StatusText(status)
	if title == "" {
		title = "Error"
	}
	if detail == "" {
		detail = title
	}

	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	problem := problemDetail{
		Type:   problemTypeBlank,
		Title:  title,
		Status: status,
		Detail: detail,
	}
	if len(errs) > 0 {
		problem.Errors = append([]datasetapi.ParameterError(nil), errs...)
	}
	_ = json.NewEncoder(w).Encode(problem)
}
