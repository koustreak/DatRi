.PHONY: help build test lint fmt clean install run docker-build

# Variables
BINARY_NAME=datri
MAIN_PATH=./cmd/datri
BUILD_DIR=./bin
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✓ Multi-platform build complete"

# Run
run: ## Run the application
	$(GO) run $(MAIN_PATH)

# Testing
test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -race -tags=integration ./test/integration/...

test-e2e: ## Run end-to-end tests
	@echo "Running E2E tests..."
	$(GO) test -v -race -tags=e2e ./test/e2e/...

# Code Quality
lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@which gofumpt > /dev/null || (echo "Installing gofumpt..." && go install mvdan.cc/gofumpt@latest)
	gofumpt -l -w .
	$(GO) mod tidy

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/golang/mock/mockgen@latest
	@echo "✓ Tools installed"

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest -f deployments/docker/Dockerfile .
	@echo "✓ Docker image built: $(BINARY_NAME):latest"

docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 $(BINARY_NAME):latest

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Install
install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(MAIN_PATH)
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Development
dev: ## Run with hot reload (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Benchmarks
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Security
security: ## Run security checks
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec ./...

# Generate
generate: ## Run go generate
	@echo "Running go generate..."
	$(GO) generate ./...

# CI targets
ci-test: lint test ## Run CI tests (lint + test)

ci-build: deps build ## Run CI build (deps + build)

# Release
release: clean build-all ## Prepare release builds
	@echo "✓ Release builds ready in $(BUILD_DIR)/"
