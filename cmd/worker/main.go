//go:build js && wasm

package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/robalyx/roscoe/internal/http/handler"
	d1Flag "github.com/robalyx/roscoe/internal/service/d1"
	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare"
	_ "github.com/syumai/workers/cloudflare/d1" // register driver
)

// newRouter creates a new HTTP router with middleware and routes.
func newRouter() (http.Handler, error) {
	// Initialize D1 database
	db, err := sql.Open("d1", "DB")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize D1 database: %w", err)
	}

	// Initialize services
	flagService := d1Flag.NewFlagService(db)
	apiKeyService := d1Flag.NewAPIKeyService(db)
	queueService := d1Flag.NewQueueService(db, flagService)

	// Initialize queue table
	if err := queueService.InitQueueTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize queue table: %w", err)
	}

	mux := http.NewServeMux()

	// Get auth requirement from environment
	requireAuth := cloudflare.Getenv("REQUIRE_AUTH") != "false"

	// Wrap handlers with auth middleware if required
	withAuth := func(h http.HandlerFunc) http.HandlerFunc {
		if !requireAuth {
			return h
		}
		return handler.AuthMiddleware(apiKeyService)(h).ServeHTTP
	}

	// Routes
	mux.HandleFunc("/lookup/roblox/user", withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.BatchLookup(flagService)(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))

	mux.HandleFunc("/lookup/roblox/user/", withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract ID from path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 5 || parts[4] == "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		handler.SingleLookup(flagService)(w, r)
	}))

	mux.HandleFunc("/queue/roblox/user", withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.QueueUser(queueService)(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
