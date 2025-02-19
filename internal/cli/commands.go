package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robalyx/roscoe/internal/service/d1"
	"github.com/robalyx/roscoe/internal/service/database"
)

// RunSync syncs the database with D1.
func RunSync(dbURL, accountID, d1ID, token string) error {
	start := time.Now()
	log.Printf("🚀 Starting flag update process...")

	ctx := context.Background()

	// Initialize database client
	log.Printf("🔌 Connecting to database...")
	db, err := database.NewClient(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close(ctx)
	log.Printf("✅ Database connection established")

	// Initialize sync service
	syncService := d1.NewSyncService(db.DB(), accountID, d1ID, token)

	// Update flags
	if err := syncService.UpdateFlags(ctx); err != nil {
		return fmt.Errorf("failed to update flags: %w", err)
	}

	duration := time.Since(start).Round(time.Millisecond)
	log.Printf("✨ Successfully updated flags (took %v)", duration)
	return nil
}

// AddAPIKey adds a new API key to D1.
func AddAPIKey(accountID, d1ID, token, description string) error {
	ctx := context.Background()

	key, err := d1.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	cfAPI := d1.NewCloudflareAPI(accountID, d1ID, token)

	sql := `INSERT INTO api_keys (key, description, created_at) VALUES (?, ?, ?)`
	params := []interface{}{key, description, time.Now().Unix()}

	if _, err := cfAPI.ExecuteSQL(ctx, sql, params); err != nil {
		return fmt.Errorf("failed to add API key: %w", err)
	}

	log.Printf("✅ Successfully added API key: %s", key)
	return nil
}

// RemoveAPIKey removes an API key from D1.
func RemoveAPIKey(accountID, d1ID, token, key string) error {
	ctx := context.Background()
	cfAPI := d1.NewCloudflareAPI(accountID, d1ID, token)

	sql := `DELETE FROM api_keys WHERE key = ?`
	params := []interface{}{key}

	if _, err := cfAPI.ExecuteSQL(ctx, sql, params); err != nil {
		return fmt.Errorf("failed to remove API key: %w", err)
	}

	log.Printf("✅ Successfully removed API key: %s", key)
	return nil
}

// ListAPIKeys lists all API keys in D1.
func ListAPIKeys(accountID, d1ID, token string) error {
	ctx := context.Background()
	cfAPI := d1.NewCloudflareAPI(accountID, d1ID, token)

	sql := `SELECT key, description, created_at FROM api_keys ORDER BY created_at DESC`

	results, err := cfAPI.ExecuteSQL(ctx, sql, nil)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	if len(results) == 0 {
		log.Printf("No API keys found")
		return nil
	}

	log.Printf("📝 API Keys:")
	for _, result := range results {
		key := result["key"].(string)
		description := result["description"].(string)
		createdAt := int64(result["created_at"].(float64))

		timestamp := time.Unix(createdAt, 0).Format("2006-01-02 15:04:05")
		log.Printf("• %s - %s (created: %s)", key, description, timestamp)
	}

	return nil
}
