.PHONY: help build build-wasm build-server run serve static-pages clean test fmt lint deps dev all

# Variables
BINARY_NAME=wasm-server
WASM_OUTPUT=web/app.wasm
STATIC_DIR=docs
# Repository name when the static site is served from a GitHub Pages project URL (user.github.io/<name>/)
PAGES_REPO?=

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: build-wasm build-server ## Build all binaries

build-wasm: ## Build WASM binary
	@echo "Building WASM binary..."
	GOARCH=wasm GOOS=js go build -o $(WASM_OUTPUT) ./cmd/wasm/
	@echo "WASM binary built: $(WASM_OUTPUT)"

build-server: ## Build server binary
	@echo "Building server binary..."
	go build -o bin/$(BINARY_NAME) ./cmd/wasm/
	@echo "Server binary built: bin/$(BINARY_NAME)"

serve: build-wasm build-server ## Build and run the web server
	@echo "Starting web server on http://localhost:8080"
	./bin/$(BINARY_NAME)

run: serve ## Alias for serve

static-pages: build-wasm ## Generate a self-contained static site in docs/ (PAGES_REPO=<name> for a GitHub Pages project site)
	@echo "Generating static pages..."
	go run ./cmd/static-gen/ -repo "$(PAGES_REPO)"
	mkdir -p $(STATIC_DIR)/web
	cp $(WASM_OUTPUT) $(STATIC_DIR)/web/
	cp -R web/static $(STATIC_DIR)/web/
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
