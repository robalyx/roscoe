package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/robalyx/roscoe/internal/service/d1"
)

type lookupRequest struct {
	IDs []uint64 `json:"ids"`
}

type userFlag struct {
	ID         uint64   `json:"id"`
	Flag       uint8    `json:"flag"`
	Confidence *float32 `json:"confidence,omitempty"`
}

type singleResponse struct {
	Data *userFlag `json:"data"`
}

type batchResponse struct {
	Data []userFlag `json:"data"`
}

// BatchLookup handles batch flag lookup requests.
func BatchLookup(flagService *d1.FlagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req lookupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendJSONError(w, &ErrorResponse{Message: "Invalid request body"}, http.StatusBadRequest)
			return
		}

		// Check if the batch size is too large
		if len(req.IDs) > 1000 {
			SendJSONError(w, &ErrorResponse{Message: "Batch size too large (max 1000)"}, http.StatusBadRequest)
			return
		}

		// Validate IDs
		for _, id := range req.IDs {
			if id == 0 {
				SendJSONError(w, &ErrorResponse{Message: "Invalid ID in batch: must be greater than 0"}, http.StatusBadRequest)
				return
			}
		}

		// Get the flags for the IDs
		flags, err := flagService.GetUserFlags(r.Context(), req.IDs)
		if err != nil {
			SendJSONError(w, ErrInternal, http.StatusInternalServerError)
			return
		}

		// Convert to response format
		data := make([]userFlag, 0, len(req.IDs))
		for _, id := range req.IDs {
			flag, exists := flags[id]
			if !exists {
				// Include unflagged user without confidence
				data = append(data, userFlag{
					ID:         id,
					Flag:       0,
					Confidence: nil,
				})
				continue
			}

			// Include flagged user
			data = append(data, userFlag{
				ID:         id,
				Flag:       flag.Flag,
				Confidence: &flag.Confidence,
			})
		}

		w.Header().Set("Content-Type", ContentTypeJSON)
		_ = json.NewEncoder(w).Encode(batchResponse{Data: data})
	}
}

// SingleLookup handles single flag lookup requests.
func SingleLookup(flagService *d1.FlagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 3 {
			SendJSONError(w, &ErrorResponse{Message: "Invalid path"}, http.StatusBadRequest)
			return
		}

		// Parse and validate ID
		id, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			SendJSONError(w, &ErrorResponse{Message: "Invalid ID format: " + parts[2]}, http.StatusBadRequest)
			return
		}
		if id == 0 {
			SendJSONError(w, &ErrorResponse{Message: "Invalid ID: must be greater than 0"}, http.StatusBadRequest)
			return
		}

		flags, err := flagService.GetUserFlags(r.Context(), []uint64{id})
		if err != nil {
			SendJSONError(w, ErrInternal, http.StatusInternalServerError)
			return
		}

		// Create response based on whether user is flagged
		var response singleResponse
		if flagData, exists := flags[id]; exists {
			response = singleResponse{
				Data: &userFlag{
					ID:         id,
					Flag:       flagData.Flag,
					Confidence: &flagData.Confidence,
				},
			}
		} else {
			response = singleResponse{
				Data: &userFlag{
					ID:   id,
					Flag: 0,
				},
			}
		}

		w.Header().Set("Content-Type", ContentTypeJSON)
		_ = json.NewEncoder(w).Encode(response)
	}
}
