package datasets

import (
	"encoding/json"
	"net/http"
)

const (
	problemTypeBlank = "about:blank"
)

type problemDetail struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	title := http.StatusText(status)
	if title == "" {
		title = "Error"
	}
	if detail == "" {
		detail = title
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemDetail{
		Type:   problemTypeBlank,
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
