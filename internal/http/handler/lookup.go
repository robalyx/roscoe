package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/robalyx/roscoe/internal/service/d1"
)

// lookupRequest represents the request body for batch flag lookups.
type lookupRequest struct {
	IDs []uint64 `json:"ids"`
}

// Reason represents a structured reason for flagging.
type Reason struct {
	Message    string   `json:"message"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

// UserFlagResponse represents the response data for a user flag lookup.
type UserFlagResponse struct {
	ID         uint64            `json:"id"`
	FlagType   uint8             `json:"flagType"`
	Confidence *float32          `json:"confidence,omitempty"`
	Reasons    map[string]Reason `json:"reasons,omitempty"`
}

// APIResponse represents the standard API response structure.
type APIResponse struct {
	Success bool    `json:"success"`
	Data    any     `json:"data,omitempty"`
	Error   *string `json:"error,omitempty"`
}

// BatchLookup handles batch flag lookup requests.
func BatchLookup(flagService *d1.FlagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req lookupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorMsg := "Invalid request body"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Check if the batch size is too large
		if len(req.IDs) > 100 {
			errorMsg := "Batch size too large (max 100)"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Validate IDs
		if slices.Contains(req.IDs, uint64(0)) {
			errorMsg := "Invalid ID in batch: must be greater than 0"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Get the flags for the IDs
		flags, err := flagService.GetUserFlags(r.Context(), req.IDs)
		if err != nil {
			errorMsg := "Internal server error"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusInternalServerError)
			return
		}

		// Convert to response format
		data := make([]UserFlagResponse, 0, len(req.IDs))
		for _, id := range req.IDs {
			flagData, exists := flags[id]
			if !exists {
				data = append(data, UserFlagResponse{
					ID:       id,
					FlagType: 0,
				})
				continue
			}

			// Parse reasons if they exist
			var parsedReasons map[string]Reason
			if flagData.Reasons.Valid && flagData.Reasons.String != "" {
				if err := json.Unmarshal([]byte(flagData.Reasons.String), &parsedReasons); err != nil {
					log.Printf("Failed to unmarshal reasons for user %d: %v", id, err)
				}
			}

			// Include flagged user
			data = append(data, UserFlagResponse{
				ID:         id,
				FlagType:   flagData.Flag,
				Confidence: flagData.Confidence,
				Reasons:    parsedReasons,
			})
		}

		SendJSONResponse(w, APIResponse{
			Success: true,
			Data:    data,
		}, http.StatusOK)
	}
}

// SingleLookup handles single flag lookup requests.
func SingleLookup(flagService *d1.FlagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 5 {
			errorMsg := "Invalid path"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		// Parse and validate ID
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			errorMsg := "Invalid ID format: " + parts[4]
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}
		if id == 0 {
			errorMsg := "Invalid ID: must be greater than 0"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusBadRequest)
			return
		}

		flags, err := flagService.GetUserFlags(r.Context(), []uint64{id})
		if err != nil {
			errorMsg := "Internal server error"
			SendJSONResponse(w, APIResponse{
				Success: false,
				Error:   &errorMsg,
			}, http.StatusInternalServerError)
			return
		}

		// Create response based on whether user is flagged
		var response UserFlagResponse
		if flagData, exists := flags[id]; exists {
			// Parse reasons if they exist
			var parsedReasons map[string]Reason
			if flagData.Reasons.Valid && flagData.Reasons.String != "" {
				if err := json.Unmarshal([]byte(flagData.Reasons.String), &parsedReasons); err != nil {
					log.Printf("Failed to unmarshal reasons for user %d: %v", id, err)
				}
			}

			response = UserFlagResponse{
				ID:         id,
				FlagType:   flagData.Flag,
				Confidence: flagData.Confidence,
				Reasons:    parsedReasons,
			}
		} else {
			response = UserFlagResponse{
				ID:       id,
				FlagType: 0,
			}
		}

		SendJSONResponse(w, APIResponse{
			Success: true,
			Data:    response,
		}, http.StatusOK)
	}
}

// SendJSONResponse sends a JSON response with the given status code.
func SendJSONResponse(w http.ResponseWriter, resp APIResponse, statusCode int) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
