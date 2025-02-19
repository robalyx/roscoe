//go:build js && wasm

package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/robalyx/roscoe/internal/http/handler"
	d1Flag "github.com/robalyx/roscoe/internal/service/d1"
	"github.com/syumai/workers"
	_ "github.com/syumai/workers/cloudflare/d1" // register driver
)

// newRouter creates a new HTTP router with middleware and routes.
func newRouter() (http.Handler, error) {
	// Initialize D1 database
	db, err := sql.Open("d1", "DB")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize D1 database: %w", err)
	}

	flagService := d1Flag.NewFlagService(db)
	apiKeyService := d1Flag.NewAPIKeyService(db)
	mux := http.NewServeMux()

	// Wrap handlers with auth middleware
	withAuth := func(h http.HandlerFunc) http.HandlerFunc {
		return handler.AuthMiddleware(apiKeyService)(h).ServeHTTP
	}

	// Routes
	mux.HandleFunc("/lookup", withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.BatchLookup(flagService)(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))

	mux.HandleFunc("/lookup/", withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract ID from path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 3 || parts[2] == "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		handler.SingleLookup(flagService)(w, r)
	}))

	return mux, nil
}

func main() {
	router, err := newRouter()
	if err != nil {
		panic(err)
	}
	workers.Serve(router)
}
