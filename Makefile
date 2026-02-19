# =============================================================================
# FLEXCORE - Distributed Runtime Container Makefile
# =============================================================================
# Go 1.24+ Runtime Service - Clean Architecture + DDD + CQRS + Event Sourcing
# =============================================================================

# Project Configuration
PROJECT_NAME := flexcore
CORE_STACK := go
GO_VERSION := 1.24
BINARY_NAME := flexcore
BUILD_DIR := ../bin
LOCAL_BUILD_DIR := bin

ifneq ("$(wildcard ../base.mk)", "")
include ../base.mk
else
include base.mk
endif

# Quality Standards
# Service Configuration
SERVICE_PORT := 8080
SERVICE_HOST := localhost

# Export Configuration
export PROJECT_NAME GO_VERSION MIN_COVERAGE BINARY_NAME SERVICE_PORT SERVICE_HOST

# =============================================================================
# HELP & INFORMATION
# =============================================================================

.PHONY: help-local
help-local: ## Show flexcore-specific commands
	@echo "FLEXCORE - Distributed Runtime Container"
	@echo "======================================="
	@echo ""
	@echo "🏗️  BUILD COMMANDS:"
	@echo "  build            Build application (workspace bin/)"
	@echo "  build-local      Build locally (flexcore/bin/)"
	@echo "  build-release    Build optimized release version"
	@echo ""
	@echo "🧪 QUALITY COMMANDS:"
	@echo "  validate         Run validate gates only (use FIX=1 for auto-fix first)"
	@echo "  check            Quick health check (lint+vet)"
	@echo "  security         Run all security checks"
	@echo "  format           Run all formatting"
	@echo "  docs             Build docs"
	@echo "  test             Run tests with coverage"
	@echo "  test-flext-integration  Run FLEXT service integration tests"
	@echo "  test-python      Run Python tests for FLEXT bridge"
	@echo "  test-cross-runtime Test Go-Python runtime coordination"
	@echo "  lint             Run Go linting"
	@echo "  format           Format Go code"
	@echo ""
	@echo "🐳 DOCKER COMMANDS (Workspace Integration):"
	@echo "  docker-build     Build Docker image using workspace infrastructure"
	@echo "  docker-run       Run with workspace Docker stack"
	@echo "  docker-test      Run tests using FLEXT Docker infrastructure"
	@echo "  docker-test-integration  Run integration tests with full FLEXT stack"
	@echo "  docker-test-flext Test FlexCore with FLEXT service integration"
	@echo "  docker-stop      Stop Docker stack"
	@echo "  docker-logs      Show container logs"
	@echo "  docker-health    Check container health"
	@echo "  docker-clean     Clean Docker artifacts"
	@echo ""
	@echo "🚀 SERVICE COMMANDS:"
	@echo "  run              Run service locally"
	@echo "  service-start    Start built service"
	@echo "  service-health   Check service health"
	@echo ""
	@echo "⚡ WINDMILL WORKFLOW ORCHESTRATION:"
	@echo "  windmill-setup   Setup Windmill workflow engine"
	@echo "  windmill-engine  Build and start Windmill engine"
	@echo "  windmill-dev     Start engine in development mode"
	@echo "  windmill-test    Test Windmill integration"
	@echo "  windmill-status  Check engine health"
	@echo "  windmill-clean   Clean Windmill artifacts"
	@echo ""
	@echo "📋 All commands: make [command] or use single-letter shortcuts (b=build, t=test, r=run)"

.PHONY: info
info: ## Show project information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Go Version: $(GO_VERSION)+"
	@echo "Coverage: $(MIN_COVERAGE)% minimum"
	@echo "Service: http://$(SERVICE_HOST):$(SERVICE_PORT)"
	@echo "Architecture: Clean Architecture + DDD + CQRS + Event Sourcing"
	@echo "Build Output: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Docker Integration: Workspace unified infrastructure"
	@echo "Docker Stack: ../docker/docker-compose.flexcore.yml"

# =============================================================================
# SETUP & DEPENDENCIES
# =============================================================================

.PHONY: setup-local
setup-local: ## Complete project setup (local legacy)
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

.PHONY: validate-local
validate-local: ## Run validate gates only (local legacy)
	@echo "WARNING: optional mode available - run 'make validate FIX=1' to auto-run fix before validate gates"
	@if [ "$(FIX)" = "1" ]; then $(MAKE) fix; fi
	@$(MAKE) mod-verify

.PHONY: check-local
check-local: lint type-check markdown-lint ## Run lint gates (local legacy)

.PHONY: lint
lint: ## Run Go linting (ZERO TOLERANCE)
	@echo "🧹 Linting..."
	@golangci-lint run

.PHONY: type-check
type-check: vet ## Run type checking (ZERO TOLERANCE)
	@echo "🔍 Running type checking..."
	@go vet ./...

.PHONY: format-local
format-local: ## Format Go code (local legacy)
	@echo "🎨 Formatting..."
	@go fmt ./...
	@goimports -w .
	@md_files=$$(find . -type f -name '*.md' ! -path './.git/*' ! -path './.reports/*' ! -path './reports/*' ! -path './bin/*' ! -path './build/*'); \
	md_config=""; \
	if [ -f "../.markdownlint.json" ]; then md_config="--config ../.markdownlint.json"; fi; \
	if [ -n "$$md_files" ]; then printf '%s\n' "$$md_files" | xargs -r mdformat; markdownlint --fix $$md_config $$md_files || true; fi

.PHONY: markdown-lint
markdown-lint: ## Run markdown linting
	@md_files=$$(find . -type f -name '*.md' ! -path './.git/*' ! -path './.reports/*' ! -path './reports/*' ! -path './bin/*' ! -path './build/*'); \
	md_config=""; \
	if [ -f "../.markdownlint.json" ]; then md_config="--config ../.markdownlint.json"; fi; \
	if [ -n "$$md_files" ]; then markdownlint $$md_config $$md_files; fi

.PHONY: vet
vet: ## Run Go vet
	@echo "🔍 Running vet..."
	@go vet ./...

.PHONY: security-local
security-local: ## Run security scanning (local legacy)
	@echo "🔒 Security scan..."
	@gosec ./...

.PHONY: fix
fix: format ## Auto-fix issues
	@golangci-lint run --fix

# =============================================================================
# TESTING (MANDATORY - 80% COVERAGE)
# =============================================================================

.PHONY: test-local
test-local: ## Run tests with coverage (local legacy)
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

.PHONY: test-flext-integration
test-flext-integration: ## Run tests with FLEXT service integration
	@echo "🔗 Running FLEXT service integration tests..."
	@go test -v -tags=flext ./test/integration/...
	@cd .. && python -m pytest flexcore/tests/ -v -k flext
	@echo "✅ FLEXT integration tests completed"

.PHONY: test-python
test-python: ## Run Python tests for FLEXT bridge
	@echo "🐍 Running Python tests..."
	@python -m pytest tests/ -v --cov=src --cov-report=term-missing
	@echo "✅ Python tests completed"

.PHONY: test-cross-runtime
test-cross-runtime: ## Test Go-Python runtime coordination
	@echo "🔄 Testing cross-runtime coordination..."
	@go test -v -tags=integration ./pkg/runtimes/...
	@python -m pytest tests/ -v -k "runtime or bridge"
	@echo "✅ Cross-runtime tests completed"

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
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY_NAME)"

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

# =============================================================================
# WINDMILL WORKFLOW ORCHESTRATION
# =============================================================================

.PHONY: windmill-setup
windmill-setup: ## Setup Windmill workflow engine
	@echo "⚡ Setting up Windmill workflow engine..."
	@mkdir -p $(BUILD_DIR)/workflows
	@mkdir -p $(BUILD_DIR)/windmill
	@go build -o $(BUILD_DIR)/windmill/engine ./pkg/windmill/engine
	@echo "✅ Windmill engine built: $(BUILD_DIR)/windmill/engine"

.PHONY: windmill-workflows
windmill-workflows: ## Deploy FlexCore workflows to Windmill
	@echo "🔄 Deploying FlexCore workflows..."
	@mkdir -p $(BUILD_DIR)/workflows
	@cp -r workflows/* $(BUILD_DIR)/workflows/ 2>/dev/null || echo "No workflows directory found"
	@echo "✅ Workflows deployed to $(BUILD_DIR)/workflows"

.PHONY: windmill-test
windmill-test: ## Test Windmill integration with FlexCore
	@echo "🧪 Testing Windmill integration..."
	@go test -v ./pkg/windmill/... -tags=integration
	@go test -v ./internal/adapters/primary/windmill/... -tags=integration

.PHONY: windmill-engine
windmill-engine: windmill-setup ## Build and start Windmill engine
	@echo "🚀 Starting Windmill workflow engine..."
	@$(BUILD_DIR)/windmill/engine -config configs/windmill.yaml

.PHONY: windmill-dev
windmill-dev: ## Start Windmill engine in development mode
	@echo "🛠️ Starting Windmill engine in dev mode..."
	@go run ./pkg/windmill/engine -dev -config configs/dev.yaml

.PHONY: windmill-status
windmill-status: ## Check Windmill engine status
	@echo "📊 Checking Windmill engine status..."
	@curl -f http://$(SERVICE_HOST):8081/windmill/health || echo "Windmill engine not responding"

.PHONY: windmill-clean
windmill-clean: ## Clean Windmill artifacts
	@echo "🧹 Cleaning Windmill artifacts..."
	@rm -rf $(BUILD_DIR)/windmill
	@rm -rf $(BUILD_DIR)/workflows

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
# DOCKER INTEGRATION (WORKSPACE UNIFIED INFRASTRUCTURE)
# =============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image using workspace infrastructure
	@echo "🐳 Building FlexCore Docker image..."
	@cd .. && docker build -f docker/Dockerfile.flexcore-dev -t flexcore:dev .
	@echo "✅ Docker image built: flexcore:dev"

.PHONY: docker-build-prod
docker-build-prod: ## Build production Docker image
	@echo "🐳 Building FlexCore production image..."
	@cd .. && docker build -f docker/Dockerfile.flexcore-node -t flexcore:latest .
	@echo "✅ Production image built: flexcore:latest"

.PHONY: docker-run
docker-run: ## Run FlexCore container using workspace Docker stack
	@echo "🐳 Starting FlexCore with workspace Docker stack..."
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml up -d
	@echo "✅ FlexCore stack running at http://localhost:8080"

.PHONY: docker-stop
docker-stop: ## Stop FlexCore Docker stack
	@echo "🐳 Stopping FlexCore Docker stack..."
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml down
	@echo "✅ FlexCore stack stopped"

.PHONY: docker-logs
docker-logs: ## Show FlexCore container logs
	@echo "📋 FlexCore container logs..."
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml logs -f flexcore-server

.PHONY: docker-health
docker-health: ## Check FlexCore container health
	@echo "🏥 Checking FlexCore health..."
	@curl -f http://localhost:8080/health || echo "FlexCore not responding"

.PHONY: docker-test
docker-test: ## Run tests using FLEXT Docker infrastructure
	@echo "🧪 Running FlexCore tests with FLEXT Docker infrastructure..."
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml up -d postgres redis
	@sleep 5
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml exec -T flexcore-server make test
	@echo "✅ Docker tests completed"

.PHONY: docker-test-integration
docker-test-integration: ## Run integration tests with full FLEXT stack
	@echo "🔗 Running integration tests with full FLEXT stack..."
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml up -d
	@sleep 10
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml exec -T flexcore-server make test-integration
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml exec -T flexcore-server make windmill-test
	@echo "✅ Integration tests completed"

.PHONY: docker-test-flext
docker-test-flext: ## Test FlexCore with FLEXT service integration
	@echo "🔄 Testing FlexCore with FLEXT service integration..."
	@cd .. && docker-compose -f docker/docker-compose.yml up -d flext-service
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml up -d flexcore-server
	@sleep 15
	@curl -f http://localhost:8081/health || echo "FLEXT service not responding"
	@curl -f http://localhost:8080/health || echo "FlexCore not responding"
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml exec -T flexcore-server make test-integration
	@echo "✅ FLEXT integration tests completed"

.PHONY: docker-clean
docker-clean: ## Clean FlexCore Docker artifacts
	@echo "🧹 Cleaning Docker artifacts..."
	@docker rmi flexcore:dev flexcore:latest 2>/dev/null || true
	@cd .. && docker-compose -f docker/docker-compose.flexcore.yml down -v
	@echo "✅ Docker artifacts cleaned"

# =============================================================================
# MAINTENANCE
# =============================================================================

.PHONY: clean-local
clean-local: ## Clean build artifacts (local legacy)
	@echo "🧹 Cleaning..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf $(LOCAL_BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf plugins/*.so
	@rm -rf $(BUILD_DIR)/windmill
	@rm -rf $(BUILD_DIR)/workflows
	@go clean

.PHONY: clean-all
clean-all: clean ## Deep clean including cache
	@echo "🧹 Deep cleaning..."
	@go clean -cache -modcache -testcache

.PHONY: docs-local
docs-local: ## Build docs (local legacy)
	@if [ -f mkdocs.yml ]; then mkdocs build; else echo "SKIP: docs (mkdocs.yml not found)"; fi

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
# CONFIGURATION
# =============================================================================

.DEFAULT_GOAL := help
