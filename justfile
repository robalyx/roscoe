set dotenv-load
set dotenv-required
set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-c"]

# Default recipe to display help information
default:
    @just --list

# Generate wrangler.toml file
generate-config:
    envsubst < wrangler.template.toml > wrangler.toml

# Development server
dev: generate-config build
    wrangler dev

# Build the worker
build: generate-config
    cd cmd/worker && go mod tidy
    cd cmd/worker && go run github.com/syumai/workers/cmd/workers-assets-gen@v0.28.1
    cd cmd/worker && tinygo build -o build/app.wasm -target wasm -no-debug .
    @mkdir -p build
    @mv cmd/worker/build/* build/

# Deploy the worker
deploy: generate-config build
    wrangler deploy

# Setup D1 database
setup-d1: generate-config
    wrangler d1 create roscoe

# Update D1 with latest database state
update-d1:
    cd cmd/cli && go mod tidy && go run . sync

# Clean build artifacts
clean:
    rm -rf .wrangler/
    rm -rf build/
    rm -rf cmd/worker/build/
    rm -f wrangler.toml

# Add API key
add-key description: generate-config
    cd cmd/cli && go run . add-key "{{description}}"

# Remove API key
remove-key key: generate-config
    cd cmd/cli && go run . remove-key "{{key}}"

# List API keys
list-keys: generate-config
    cd cmd/cli && go run . list-keys
