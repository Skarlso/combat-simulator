.PHONY: help build build-wasm build-example run serve clean test static-pages

# Variables
BINARY_NAME=wasm-server
EXAMPLE_BINARY=example
WASM_OUTPUT=web/app.wasm
STATIC_DIR=docs

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: build-wasm build-example ## Build all binaries

build-wasm: ## Build WASM binary
	@echo "Building WASM binary..."
	GOARCH=wasm GOOS=js go build -o $(WASM_OUTPUT) ./cmd/wasm/
	@echo "WASM binary built: $(WASM_OUTPUT)"

build-server: ## Build server binary
	@echo "Building server binary..."
	go build -o bin/$(BINARY_NAME) ./cmd/wasm/
	@echo "Server binary built: bin/$(BINARY_NAME)"

build-example: ## Build example CLI binary
	@echo "Building example binary..."
	go build -o bin/$(EXAMPLE_BINARY) ./cmd/example/
	@echo "Example binary built: bin/$(EXAMPLE_BINARY)"

run-example: build-example ## Run the CLI example
	@echo "Running example simulation..."
	./bin/$(EXAMPLE_BINARY)

serve: build-wasm build-server ## Build and run the web server
	@echo "Starting web server on http://localhost:8080"
	./bin/$(BINARY_NAME)

run: serve ## Alias for serve

static-pages: build-wasm ## Generate static website for GitHub Pages
	@echo "Generating static pages..."
	go run ./cmd/static-gen/
	@echo "Static pages generated in $(STATIC_DIR)/"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f $(WASM_OUTPUT)
	rm -rf $(STATIC_DIR)/
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run || go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

dev: ## Run in development mode (rebuilds on change, requires entr)
	@echo "Starting development mode..."
	@echo "Watching for changes... (requires 'entr' to be installed)"
	find . -name '*.go' | entr -r make serve

all: clean deps build test ## Clean, download deps, build, and test

.DEFAULT_GOAL := help
