//go:build js && wasm

package handler

import (
	"net/http"

	"github.com/robalyx/roscoe/internal/service/d1"
)

// AuthHeaderName is the header name for the API key.
const AuthHeaderName = "X-Auth-Token"

// AuthMiddleware checks the auth token against valid API keys in D1.
func AuthMiddleware(apiKeyService *d1.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedToken := r.Header.Get(AuthHeaderName)

			valid, err := apiKeyService.ValidateKey(r.Context(), providedToken)
			if err != nil {
				SendJSONError(w, ErrInternal, http.StatusInternalServerError)
				return
			}

			if !valid {
				SendJSONError(w, ErrUnauthorized, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
