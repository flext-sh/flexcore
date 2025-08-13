# =============================================================================
# FLEXCORE - Distributed Runtime Container Makefile
# =============================================================================
# Go 1.24+ Runtime Service - Clean Architecture + DDD + CQRS + Event Sourcing
# =============================================================================

# Project Configuration
PROJECT_NAME := flexcore
GO_VERSION := 1.24
BINARY_NAME := flexcore
BUILD_DIR := ../bin
LOCAL_BUILD_DIR := bin

# Quality Standards
MIN_COVERAGE := 80

# Service Configuration
SERVICE_PORT := 8080
SERVICE_HOST := localhost

# Export Configuration
export PROJECT_NAME GO_VERSION MIN_COVERAGE BINARY_NAME SERVICE_PORT SERVICE_HOST

# =============================================================================
# HELP & INFORMATION
# =============================================================================

.PHONY: help
help: ## Show available commands
	@echo "FLEXCORE - Distributed Runtime Container"
	@echo "======================================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: info
info: ## Show project information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Go Version: $(GO_VERSION)+"
	@echo "Coverage: $(MIN_COVERAGE)% minimum"
	@echo "Service: http://$(SERVICE_HOST):$(SERVICE_PORT)"
	@echo "Architecture: Clean Architecture + DDD + CQRS + Event Sourcing"

# =============================================================================
# SETUP & DEPENDENCIES
# =============================================================================

.PHONY: setup
setup: ## Complete project setup
	@echo "🚀 Setting up project..."
	@go mod download
	@go mod tidy
	@echo "✅ Setup complete"

.PHONY: mod-tidy
mod-tidy: ## Tidy Go modules
	@echo "🔧 Tidying modules..."
	@go mod tidy

.PHONY: mod-verify
mod-verify: ## Verify Go modules
	@echo "🔍 Verifying modules..."
	@go mod verify

# =============================================================================
# QUALITY GATES (MANDATORY)
# =============================================================================

.PHONY: validate
validate: lint vet security test mod-verify ## Run all quality gates

.PHONY: check
check: lint vet ## Quick health check

.PHONY: lint
lint: ## Run Go linting
	@echo "🧹 Linting..."
	@golangci-lint run

.PHONY: format
format: ## Format Go code
	@echo "🎨 Formatting..."
	@go fmt ./...
	@goimports -w .

.PHONY: vet
vet: ## Run Go vet
	@echo "🔍 Running vet..."
	@go vet ./...

.PHONY: security
security: ## Run security scanning
	@echo "🔒 Security scan..."
	@gosec ./...

.PHONY: fix
fix: format ## Auto-fix issues
	@golangci-lint run --fix

# =============================================================================
# TESTING
# =============================================================================

.PHONY: test
test: ## Run tests with coverage
	@echo "🧪 Running tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

.PHONY: test-unit
test-unit: ## Run unit tests
	@go test -v -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@go test -v -run Integration ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@go test -v -race ./...

.PHONY: test-bench
test-bench: ## Run benchmark tests
	@go test -v -bench=. -benchmem ./...

.PHONY: coverage-html
coverage-html: ## Generate HTML coverage report
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# =============================================================================
# BUILD & DISTRIBUTION
# =============================================================================

.PHONY: build
build: ## Build the application (workspace bin/)
	@echo "🏗️ Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/flexcore
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: build-local
build-local: ## Build locally (flexcore/bin/)
	@echo "🏗️ Building $(BINARY_NAME) locally..."
	@mkdir -p $(LOCAL_BUILD_DIR)
	@go build -o $(LOCAL_BUILD_DIR)/$(BINARY_NAME) ./cmd/flexcore

.PHONY: build-release
build-release: ## Build release version
	@echo "🏗️ Building release..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/flexcore

.PHONY: install
install: ## Install the application
	@echo "📦 Installing $(BINARY_NAME)..."
	@go install ./cmd/flexcore

# =============================================================================
# SERVICE OPERATIONS
# =============================================================================

.PHONY: run
run: ## Run the service
	@echo "🚀 Running $(BINARY_NAME)..."
	@go run ./cmd/flexcore

.PHONY: service-start
service-start: ## Start service
	@echo "🚀 Starting service..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

.PHONY: service-dev
service-dev: ## Start service in development mode
	@echo "🛠️ Starting in dev mode..."
	@go run . -dev -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

.PHONY: service-health
service-health: ## Check service health
	@echo "🏥 Checking service health..."
	@curl -f http://$(SERVICE_HOST):$(SERVICE_PORT)/health || echo "Service not responding"

# =============================================================================
# FLEXCORE SPECIFIC OPERATIONS
# =============================================================================

.PHONY: plugin-build
plugin-build: ## Build FlexCore plugins
	@echo "🔧 Building FlexCore plugins..."
	@mkdir -p plugins
	@go build -buildmode=plugin -o plugins/source.so ./plugins/source
	@go build -buildmode=plugin -o plugins/target.so ./plugins/target
	@go build -buildmode=plugin -o plugins/transformer.so ./plugins/transformer

.PHONY: plugin-test
plugin-test: ## Test FlexCore plugins
	@echo "🧪 Testing plugins..."
	@go test -v ./plugins/...

.PHONY: generate
generate: ## Generate all code
	@echo "⚡ Generating code..."
	@go generate ./...

.PHONY: proto-gen
proto-gen: ## Generate protobuf code
	@echo "⚡ Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

# =============================================================================
# DEPENDENCIES
# =============================================================================

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@go get -u ./...
	@go mod tidy

.PHONY: deps-check
deps-check: ## Check for dependency updates
	@echo "🔍 Checking for updates..."
	@go list -u -m all

# =============================================================================
# MAINTENANCE
# =============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf $(LOCAL_BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf plugins/*.so
	@go clean

.PHONY: clean-all
clean-all: clean ## Deep clean including cache
	@echo "🧹 Deep cleaning..."
	@go clean -cache -modcache -testcache

.PHONY: reset
reset: clean-all setup ## Reset project

# =============================================================================
# DIAGNOSTICS
# =============================================================================

.PHONY: diagnose
diagnose: ## Project diagnostics
	@echo "🔬 Diagnostics..."
	@echo "Go: $$(go version)"
	@echo "Project: $$(go list -m)"
	@go env

.PHONY: doctor
doctor: diagnose check ## Health check

# =============================================================================
# ALIASES (SINGLE LETTER SHORTCUTS)
# =============================================================================

.PHONY: t l f b c r v s h
t: test
l: lint
f: format
b: build
c: clean
r: run
v: validate
s: service-start
h: service-health

# =============================================================================
# CONFIGURATION
# =============================================================================

.DEFAULT_GOAL := help