package server

import (
	"fmt"
	"net/http"
)

var (
	ErrorSlug   = fmt.Errorf("no slug provided or no data found")
	ErrNotFound = fmt.Errorf("not found")
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if r.Method != http.MethodGet {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}

	http.ServeFile(w, r, "templates/index.html")
}

// HealthHandler handles the API healthz params.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	SendJSONResponse(w, http.StatusOK, "hurray. api is working fine.")
}

// SearchQuestionHandler
func SearchQuestionHandler(w http.ResponseWriter, r *http.Request) {
	question_slug := r.URL.Query().Get("query")
	// When slug are not provided nicely
	if len(question_slug) == 0 {
		SendErrorResponse(w, http.StatusBadRequest, ErrorSlug)
		return
	}
	SendJSONResponse(w, http.StatusOK, question_slug)
}
