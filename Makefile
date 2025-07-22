# FLEXT flexcore Makefile
# Go application with standardized FLEXT patterns

.PHONY: help install build test lint format clean check dev-install
.DEFAULT_GOAL := help

# Variables
APP_NAME := flexcore
CMD_PATH := ./cmd/flexcore
BUILD_DIR := ./build
GO_VERSION := 1.24
BINARY_NAME := $(APP_NAME)

# Build metadata
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH) -w -s"

help: ## Show this help message
	@echo "FLEXT FlexCore - Go Application"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

install: ## Install dependencies
	@echo "Installing Go dependencies..."
	go mod download
	go mod verify
	@echo "Dependencies installed successfully"

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(APP_NAME) to $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Tests completed - coverage report: coverage.html"

lint: ## Run linting
	@echo "Running Go linting..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...
	@echo "Linting completed"

format: ## Format code
	@echo "Formatting Go code..."
	go fmt ./...
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .
	@echo "Code formatted"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache -testcache -modcache
	@echo "Clean completed"

check: lint test ## Run all quality checks

dev-install: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/sast-scan@latest
	@echo "Development dependencies installed"

run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

docker-run: docker-build ## Build and run Docker container
	@echo "Running $(APP_NAME) in Docker..."
	docker run --rm -p 8080:8080 $(APP_NAME):latest

tidy: ## Tidy go modules
	@echo "Tidying Go modules..."
	go mod tidy
	go mod verify

# Integration with workspace
workspace-check: check ## Alias for workspace integration
workspace-lint: lint ## Alias for workspace integration
workspace-test: test ## Alias for workspace integration

.PHONY: workspace-check workspace-lint workspace-test