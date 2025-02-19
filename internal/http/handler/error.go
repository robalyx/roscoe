package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	ContentTypeJSON  = "application/json"
	ContentTypePlain = "text/plain"
)

var (
	ErrUnauthorized = &ErrorResponse{Message: "Unauthorized"}
	ErrInternal     = &ErrorResponse{Message: "Internal Server Error"}
	ErrBadGateway   = &ErrorResponse{Message: "Bad Gateway"}
)

// ErrorResponse represents an error message structure.
type ErrorResponse struct {
	Message string `json:"message"`
}

// SendJSONError sends a JSON error response.
func SendJSONError(w http.ResponseWriter, err *ErrorResponse, status int) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(err); encErr != nil {
		fmt.Printf("Error encoding JSON response: %v\n", encErr)
		w.Header().Set("Content-Type", ContentTypePlain)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error: Failed to encode JSON response"))
	}
}
