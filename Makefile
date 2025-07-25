# =============================================================================
# FLEXCORE - PROJECT MAKEFILE
# =============================================================================
# Enterprise Go Core Service with Clean Architecture + DDD + Zero Tolerance Quality
# Go 1.21+ + Modern Build Tools + Professional Standards
# =============================================================================

# Project Configuration
PROJECT_NAME := flexcore
PROJECT_TYPE := go-service
GO_VERSION := 1.21
BINARY_NAME := flexcore
SRC_DIR := .
TESTS_DIR := .
DOCS_DIR := docs

# Quality Gates Configuration
MIN_COVERAGE := 80
GO_LINT_CONFIG := .golangci.yml
BUILD_DIR := bin

# Service Configuration
SERVICE_PORT := 8080
SERVICE_HOST := localhost
SERVICE_ENV := development

# Export environment variables
export GO_VERSION
export MIN_COVERAGE
export BINARY_NAME
export SERVICE_PORT
export SERVICE_HOST
export SERVICE_ENV

# =============================================================================
# HELP & INFORMATION
# =============================================================================

.PHONY: help
help: ## Show available commands
	@echo "$(PROJECT_NAME) - Go Core Service"
	@echo "==================================="
	@echo ""
	@echo "📋 AVAILABLE COMMANDS:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🔧 PROJECT INFO:"
	@echo "  Type: $(PROJECT_TYPE)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Coverage: $(MIN_COVERAGE)%"
	@echo "  Binary: $(BINARY_NAME)"
	@echo "  Service: $(SERVICE_HOST):$(SERVICE_PORT)"

.PHONY: info
info: ## Show project information
	@echo "Project Information"
	@echo "=================="
	@echo "Name: $(PROJECT_NAME)"
	@echo "Type: $(PROJECT_TYPE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Source Directory: $(SRC_DIR)"
	@echo "Tests Directory: $(TESTS_DIR)"
	@echo "Quality Standards: Zero Tolerance"
	@echo "Architecture: Clean Architecture + DDD + Go"
	@echo "Service Configuration:"
	@echo "  Host: $(SERVICE_HOST)"
	@echo "  Port: $(SERVICE_PORT)"
	@echo "  Environment: $(SERVICE_ENV)"

# =============================================================================
# INSTALLATION & SETUP
# =============================================================================

.PHONY: setup
setup: ## Complete project setup
	@echo "🚀 Setting up $(PROJECT_NAME)..."
	@make deps-download
	@make mod-tidy
	@echo "✅ Setup complete"

.PHONY: deps-download
deps-download: ## Download Go dependencies
	@echo "📦 Downloading dependencies..."
	@go mod download

.PHONY: mod-tidy
mod-tidy: ## Tidy Go modules
	@echo "🔧 Tidying Go modules..."
	@go mod tidy

.PHONY: mod-verify
mod-verify: ## Verify Go modules
	@echo "🔍 Verifying Go modules..."
	@go mod verify

# =============================================================================
# QUALITY GATES & VALIDATION
# =============================================================================

.PHONY: validate
validate: ## Run complete validation (quality gate)
	@echo "🔍 Running complete validation for $(PROJECT_NAME)..."
	@make lint
	@make vet
	@make security
	@make test
	@make mod-verify
	@echo "✅ Validation complete"

.PHONY: check
check: ## Quick health check
	@echo "🏥 Running health check..."
	@make lint
	@make vet
	@echo "✅ Health check complete"

.PHONY: lint
lint: ## Run Go linting
	@echo "🧹 Running linting..."
	@golangci-lint run

.PHONY: format
format: ## Format Go code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@goimports -w .

.PHONY: format-check
format-check: ## Check Go code formatting
	@echo "🎨 Checking code formatting..."
	@test -z "$$(gofmt -l .)"

.PHONY: vet
vet: ## Run Go vet
	@echo "🔍 Running go vet..."
	@go vet ./...

.PHONY: security
security: ## Run security scanning
	@echo "🔒 Running security scanning..."
	@gosec ./...

.PHONY: fix
fix: ## Auto-fix code issues
	@echo "🔧 Auto-fixing code issues..."
	@make format
	@golangci-lint run --fix

# =============================================================================
# TESTING
# =============================================================================

.PHONY: test
test: ## Run all tests with coverage
	@echo "🧪 Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@go test -v -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	@go test -v -run Integration ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	@go test -v -race ./...

.PHONY: test-bench
test-bench: ## Run benchmark tests
	@echo "🧪 Running benchmark tests..."
	@go test -v -bench=. -benchmem ./...

.PHONY: coverage
coverage: ## Generate coverage report
	@echo "📊 Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

.PHONY: coverage-html
coverage-html: ## Generate HTML coverage report
	@echo "📊 Generating HTML coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# =============================================================================
# BUILD & DISTRIBUTION
# =============================================================================

.PHONY: build
build: ## Build the application
	@echo "🏗️ Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: build-release
build-release: ## Build release version
	@echo "🏗️ Building release version..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "🏗️ Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

.PHONY: install
install: ## Install the application
	@echo "📦 Installing $(BINARY_NAME)..."
	@go install .

# =============================================================================
# SERVICE OPERATIONS
# =============================================================================

.PHONY: run
run: ## Run the service
	@echo "🚀 Running $(BINARY_NAME) service..."
	@go run .

.PHONY: service-start
service-start: ## Start service
	@echo "🌐 Starting $(BINARY_NAME) service..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

.PHONY: service-dev
service-dev: ## Start service in development mode
	@echo "🛠️ Starting $(BINARY_NAME) in development mode..."
	@go run . -dev -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

.PHONY: service-prod
service-prod: build-release ## Start service in production mode
	@echo "🏭 Starting $(BINARY_NAME) in production mode..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -prod -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

.PHONY: service-health
service-health: ## Check service health
	@echo "🏥 Checking service health..."
	@curl -f http://$(SERVICE_HOST):$(SERVICE_PORT)/health || echo "❌ Service not responding"

.PHONY: service-test
service-test: ## Test service functionality
	@echo "🧪 Testing service functionality..."
	@go test -v -run ServiceTest ./... -count=1

.PHONY: service-benchmark
service-benchmark: ## Run service benchmarks
	@echo "⚡ Running service benchmarks..."
	@go test -bench=BenchmarkService -benchmem ./...

# =============================================================================
# SERVICE MANAGEMENT
# =============================================================================

.PHONY: service-logs
service-logs: ## View service logs
	@echo "📝 Viewing service logs..."
	@tail -f /var/log/$(BINARY_NAME).log || echo "❌ Log file not found"

.PHONY: service-restart
service-restart: ## Restart service
	@echo "🔄 Restarting $(BINARY_NAME) service..."
	@pkill -f $(BINARY_NAME) || true
	@sleep 2
	@make service-start

.PHONY: service-stop
service-stop: ## Stop service
	@echo "🛑 Stopping $(BINARY_NAME) service..."
	@pkill -f $(BINARY_NAME) || echo "Service not running"

.PHONY: service-config
service-config: ## Show service configuration
	@echo "⚙️ Service Configuration:"
	@echo "  Host: $(SERVICE_HOST)"
	@echo "  Port: $(SERVICE_PORT)"
	@echo "  Environment: $(SERVICE_ENV)"
	@echo "  Binary: $(BINARY_NAME)"

.PHONY: service-metrics
service-metrics: ## Get service metrics
	@echo "📊 Getting service metrics..."
	@curl -s http://$(SERVICE_HOST):$(SERVICE_PORT)/metrics || echo "❌ Metrics not available"

.PHONY: service-status
service-status: ## Check service status
	@echo "📊 Checking service status..."
	@curl -s http://$(SERVICE_HOST):$(SERVICE_PORT)/status || echo "❌ Status not available"

# =============================================================================
# FLEXCORE SPECIFIC OPERATIONS
# =============================================================================

.PHONY: plugin-build
plugin-build: ## Build FlexCore plugins
	@echo "🔌 Building FlexCore plugins..."
	@mkdir -p plugins
	@go build -buildmode=plugin -o plugins/source.so ./plugins/source
	@go build -buildmode=plugin -o plugins/target.so ./plugins/target
	@go build -buildmode=plugin -o plugins/transformer.so ./plugins/transformer
	@echo "✅ Plugin build complete"

.PHONY: plugin-test
plugin-test: ## Test FlexCore plugins
	@echo "🧪 Testing FlexCore plugins..."
	@go test -v ./plugins/...
	@echo "✅ Plugin tests complete"

.PHONY: event-store-migrate
event-store-migrate: ## Migrate event store schema
	@echo "🗂️ Migrating event store schema..."
	@go run ./cmd/migrate --config configs/dev.yaml
	@echo "✅ Event store migration complete"

.PHONY: cqrs-test
cqrs-test: ## Test CQRS event bus
	@echo "🧪 Testing CQRS event bus..."
	@go test -v ./internal/infrastructure/cqrs/...
	@echo "✅ CQRS tests complete"

.PHONY: performance-test
performance-test: ## Run performance tests
	@echo "⚡ Running performance tests..."
	@go test -v -tags=performance ./tests/performance/...
	@echo "✅ Performance tests complete"

.PHONY: load-test
load-test: ## Run load tests
	@echo "📊 Running load tests..."
	@go run ./tests/load/ --duration=5m --concurrency=100
	@echo "✅ Load tests complete"

# =============================================================================
# DOCUMENTATION
# =============================================================================

.PHONY: docs
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	@godoc -http=:6060

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	@echo "📚 Serving documentation..."
	@godoc -http=:6060

.PHONY: api-docs
api-docs: ## Generate API documentation
	@echo "📚 Generating API documentation..."
	@mkdir -p $(DOCS_DIR)
	@go run . --generate-api-docs > $(DOCS_DIR)/api.md
	@echo "📚 API documentation: $(DOCS_DIR)/api.md"

# =============================================================================
# CODE GENERATION
# =============================================================================

.PHONY: generate
generate: proto-gen mocks-gen ## Generate all code
	@echo "🔄 Generating all code for $(PROJECT_NAME)..."
	@go generate ./...
	@echo "✅ Code generation complete"

.PHONY: proto-gen
proto-gen: ## Generate protobuf code
	@echo "🔄 Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto
	@echo "✅ Protobuf generation complete"

.PHONY: mocks-gen
mocks-gen: ## Generate mocks
	@echo "🔄 Generating mocks..."
	@mockgen -source=internal/domain/repositories.go -destination=internal/mocks/repositories.go
	@mockgen -source=internal/domain/services.go -destination=internal/mocks/services.go
	@echo "✅ Mock generation complete"

# =============================================================================
# DEPENDENCY MANAGEMENT
# =============================================================================

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@go get -u ./...
	@go mod tidy

.PHONY: deps-check
deps-check: ## Check for dependency updates
	@echo "🔍 Checking for dependency updates..."
	@go list -u -m all

.PHONY: deps-audit
deps-audit: ## Audit dependencies for security
	@echo "🔍 Auditing dependencies..."
	@go list -json -deps ./... | nancy sleuth

# =============================================================================
# MAINTENANCE & CLEANUP
# =============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf plugins/*.so
	@go clean

.PHONY: clean-all
clean-all: clean ## Deep clean including cache
	@echo "🧹 Deep cleaning..."
	@go clean -cache -modcache -testcache

.PHONY: reset
reset: clean-all ## Reset project to clean state
	@echo "🔄 Resetting project..."
	@make setup

# =============================================================================
# DIAGNOSTICS & TROUBLESHOOTING
# =============================================================================

.PHONY: diagnose
diagnose: ## Run project diagnostics
	@echo "🔬 Running project diagnostics..."
	@echo "Go version: $$(go version)"
	@echo "Project info:"
	@go list -m
	@echo "Environment:"
	@go env

.PHONY: doctor
doctor: ## Check project health
	@echo "👩‍⚕️ Checking project health..."
	@make diagnose
	@make check
	@echo "✅ Health check complete"

# =============================================================================
# CONVENIENCE ALIASES
# =============================================================================

.PHONY: t
t: test ## Alias for test

.PHONY: l
l: lint ## Alias for lint

.PHONY: f
f: format ## Alias for format

.PHONY: b
b: build ## Alias for build

.PHONY: c
c: clean ## Alias for clean

.PHONY: r
r: run ## Alias for run

.PHONY: v
v: validate ## Alias for validate

.PHONY: s
s: service-start ## Alias for service-start

.PHONY: h
h: service-health ## Alias for service-health

.PHONY: pb
pb: plugin-build ## Alias for plugin-build

.PHONY: pt
pt: plugin-test ## Alias for plugin-test

# =============================================================================
# Default target
# =============================================================================

.DEFAULT_GOAL := help