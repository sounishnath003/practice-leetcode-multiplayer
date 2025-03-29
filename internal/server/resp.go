package server

import (
	"encoding/json"
	"net/http"
)

// SendJSONResponse sends a JSON response with the provided status and data.
func SendJSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("X-API-KEY", "some-api-secret-keys")
	w.Header().Add("X-Tracker-ID", "tracker-123")
	w.WriteHeader(status)

	// Encode the data into a JSON response
	json.NewEncoder(w).Encode(map[string]any{
		"data": data,
	})

}

// SendErrorResponse sends a JSON error response with the provided status and error message.
func SendErrorResponse(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("X-API-KEY", "some-api-secret-keys")
	w.Header().Add("X-Tracker-ID", "tracker-123")
	w.WriteHeader(status)

	// Encode the error message into a JSON response
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}
