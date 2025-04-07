package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/robalyx/roscoe/internal/service/d1"
)

// queueRequest represents the request body for queueing a user.
type queueRequest struct {
	ID uint64 `json:"id"`
}

// QueueUser handles requests to queue a user for processing.
func QueueUser(queueService *d1.QueueService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req queueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorMsg := "Invalid request body"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Validate ID
		if req.ID == 0 {
			errorMsg := "Invalid ID: must be greater than 0"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Attempt to queue the user
		err := queueService.QueueUser(r.Context(), req.ID)
		if err != nil {
			statusCode := http.StatusInternalServerError
			errorMsg := "Failed to queue user"

			if errors.Is(err, d1.ErrUserAlreadyFlagged) {
				statusCode = http.StatusConflict
				errorMsg = "User is already flagged or confirmed"
			} else if errors.Is(err, d1.ErrUserRecentlyQueued) {
				statusCode = http.StatusConflict
				errorMsg = "User was queued within the past 7 days"
			}

			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, statusCode)
			return
		}

		// Success response
		SendJSONResponse(w, APIResponse{
			Success: true,
			Data:    map[string]uint64{"queued": req.ID},
		}, http.StatusOK)
	}
}
