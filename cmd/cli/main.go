package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/robalyx/roscoe/internal/cli"
)

func main() {
	log.SetFlags(0) // Remove timestamp prefix from log messages

	// Load .env file from parent directory
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("⚠️ Warning: .env file not found in parent directory")
	}

	// Load required environment variables
	dbURL := getEnvOrFatal("DATABASE_URL")
	accountID := getEnvOrFatal("ROSCOE_CF_ACCOUNT_ID")
	d1ID := getEnvOrFatal("ROSCOE_CF_D1_ID")
	token := getEnvOrFatal("ROSCOE_CF_API_TOKEN")

	// Parse command line arguments
	if len(os.Args) < 2 {
		log.Fatal("Command required: sync, add-key, remove-key, or list-keys")
	}

	command := os.Args[1]
	switch command {
	case "sync":
		if err := cli.RunSync(dbURL, accountID, d1ID, token); err != nil {
			log.Fatalf("❌ Sync failed: %v", err)
		}
	case "add-key":
		if len(os.Args) < 3 {
			log.Fatal("Usage: add-key <description>")
		}
		if err := cli.AddAPIKey(accountID, d1ID, token, os.Args[2]); err != nil {
			log.Fatalf("❌ Failed to add API key: %v", err)
		}
	case "remove-key":
		if len(os.Args) < 3 {
			log.Fatal("Usage: remove-key <key>")
		}
		if err := cli.RemoveAPIKey(accountID, d1ID, token, os.Args[2]); err != nil {
			log.Fatalf("❌ Failed to remove API key: %v", err)
		}
	case "list-keys":
		if err := cli.ListAPIKeys(accountID, d1ID, token); err != nil {
			log.Fatalf("❌ Failed to list API keys: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

// getEnvOrFatal returns the value of an environment variable or a fatal error if it's not set.
func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}
