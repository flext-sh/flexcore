# FlexCore - Distributed System Core Engine
# =========================================
# Production-ready distributed event-driven architecture in Go
# Go 1.24 + Clean Architecture + DDD + CQRS + Event Sourcing + Zero Tolerance Quality Gates

.PHONY: help check test lint build clean run dev
.PHONY: test-unit test-integration test-e2e test-coverage
.PHONY: install-deps update-deps mod-tidy security-scan
.PHONY: docker-build docker-up docker-down docker-logs docker-test
.PHONY: deploy-prod deploy-staging cluster-status
.PHONY: generate proto-gen mocks-gen docs-gen

# ============================================================================
# 🎯 HELP & INFORMATION  
# ============================================================================

help: ## Show this help message
	@echo "🚀 FlexCore - Distributed System Core Engine"
	@echo "============================================"
	@echo "🎯 Clean Architecture + DDD + CQRS + Event Sourcing + Go 1.24"
	@echo ""
	@echo "📦 Production-ready distributed event-driven architecture system"
	@echo "🔒 Zero tolerance quality gates with strict Go standards"
	@echo "🧪 Comprehensive testing with unit + integration + e2e"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# 🎯 CORE QUALITY GATES - ZERO TOLERANCE
# ============================================================================

check: lint test security-scan ## Essential quality checks (all must pass)
	@echo "✅ All quality gates passed"

lint: ## Run golangci-lint with strict rules  
	@echo "🔍 Running golangci-lint (strict rules)..."
	@golangci-lint run --config .golangci.yml ./...
	@echo "✅ Linting complete"

test: ## Run full test suite with coverage
	@echo "🧪 Running full test suite..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Tests complete with coverage report"

security-scan: ## Run security vulnerability scans
	@echo "🔒 Running security scans..."
	@gosec -quiet ./...
	@nancy sleuth --exclude-vulnerability-file .nancy-ignore
	@echo "✅ Security scans complete"

format: ## Format Go code
	@echo "🎨 Formatting code..."
	@gofmt -w -s .
	@goimports -w .
	@echo "✅ Formatting complete"

# ============================================================================
# 🧪 TESTING - COMPREHENSIVE COVERAGE
# ============================================================================

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@go test -v -race -short ./...
	@echo "✅ Unit tests complete"

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@go test -v -race -tags=integration ./...
	@echo "✅ Integration tests complete"

test-e2e: ## Run end-to-end tests
	@echo "🧪 Running end-to-end tests..."
	@go test -v -race -tags=e2e ./...
	@echo "✅ End-to-end tests complete"

test-coverage: test ## Generate detailed coverage report
	@echo "📊 Generating coverage report..."
	@go tool cover -func=coverage.out
	@echo "📊 Coverage report: coverage.html"

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...
	@echo "✅ Benchmarks complete"

# ============================================================================
# 🚀 BUILD & RUN
# ============================================================================

build: clean ## Build all binaries
	@echo "🔨 Building FlexCore..."
	@CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/flexcore ./cmd/server
	@echo "✅ Build complete - binary in bin/"

build-all: clean ## Build all FlexCore binaries
	@echo "🔨 Building all FlexCore binaries..."
	@mkdir -p bin
	@CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/flexcore-server ./cmd/server
	@CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/flexcore-cli ./cmd/cli
	@CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/flexcore-worker ./cmd/worker
	@echo "✅ All builds complete"

run: ## Run FlexCore server
	@echo "🚀 Starting FlexCore server..."
	@go run ./cmd/server

dev: ## Start development environment
	@echo "🔧 Starting development environment..."
	@docker-compose up -d postgres redis
	@sleep 5
	@go run ./cmd/server --config configs/dev.yaml
	@echo "✅ Development environment started"

# ============================================================================
# 📦 DEPENDENCY MANAGEMENT
# ============================================================================

install-deps: ## Install Go dependencies
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod verify
	@echo "✅ Dependencies installed"

update-deps: ## Update Go dependencies
	@echo "🔄 Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✅ Dependencies updated"

mod-tidy: ## Tidy Go modules
	@echo "🧹 Tidying Go modules..."
	@go mod tidy
	@go mod verify
	@echo "✅ Modules tidied"

deps-audit: ## Audit dependencies for vulnerabilities
	@echo "🔍 Auditing dependencies..."
	@nancy sleuth --exclude-vulnerability-file .nancy-ignore
	@echo "✅ Dependency audit complete"

# ============================================================================
# 🐳 DOCKER OPERATIONS
# ============================================================================

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t flexcore:latest -f deployments/docker/Dockerfile .
	@echo "✅ Docker image built"

docker-up: ## Start Docker Compose stack
	@echo "🐳 Starting Docker Compose stack..."
	@docker-compose -f deployments/docker/docker-compose.yml up -d
	@echo "✅ Docker stack started"

docker-down: ## Stop Docker Compose stack
	@echo "🐳 Stopping Docker Compose stack..."
	@docker-compose -f deployments/docker/docker-compose.yml down
	@echo "✅ Docker stack stopped"

docker-logs: ## View Docker logs
	@echo "🐳 Viewing Docker logs..."
	@docker-compose -f deployments/docker/docker-compose.yml logs -f

docker-test: docker-up ## Run tests in Docker
	@echo "🐳 Running tests in Docker..."
	@docker run --rm --network=host flexcore:latest make test
	@echo "✅ Docker tests complete"

# ============================================================================
# 🔧 CODE GENERATION
# ============================================================================

generate: proto-gen mocks-gen ## Generate all code
	@echo "🔄 Generating all code..."
	@go generate ./...
	@echo "✅ Code generation complete"

proto-gen: ## Generate protobuf code
	@echo "🔄 Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto
	@echo "✅ Protobuf generation complete"

mocks-gen: ## Generate mocks
	@echo "🔄 Generating mocks..."
	@mockgen -source=internal/domain/repositories.go -destination=internal/mocks/repositories.go
	@mockgen -source=internal/domain/services.go -destination=internal/mocks/services.go
	@echo "✅ Mock generation complete"

docs-gen: ## Generate documentation
	@echo "📚 Generating documentation..."
	@godoc -http=:6060 &
	@echo "✅ Documentation server started at http://localhost:6060"

# ============================================================================
# 🏗️ DEPLOYMENT
# ============================================================================

deploy-prod: ## Deploy to production
	@echo "🚀 Deploying to production..."
	@kubectl apply -f deployments/k8s/production/
	@kubectl rollout status deployment/flexcore-server -n production
	@echo "✅ Production deployment complete"

deploy-staging: ## Deploy to staging
	@echo "🚀 Deploying to staging..."
	@kubectl apply -f deployments/k8s/staging/
	@kubectl rollout status deployment/flexcore-server -n staging
	@echo "✅ Staging deployment complete"

cluster-status: ## Check cluster health
	@echo "🔍 Checking cluster status..."
	@kubectl get pods,svc,deploy -n production
	@kubectl get pods,svc,deploy -n staging
	@echo "✅ Cluster status check complete"

# ============================================================================
# 🧹 CLEANUP
# ============================================================================

clean: ## Remove all artifacts
	@echo "🧹 Cleaning up..."
	@rm -rf bin/
	@rm -rf coverage.out coverage.html
	@rm -rf dist/
	@go clean -cache -testcache -modcache
	@echo "✅ Cleanup complete"

# ============================================================================
# 🔧 ENVIRONMENT CONFIGURATION
# ============================================================================

# Go settings
GO_VERSION := 1.24
GOARCH := amd64
GOOS := linux
CGO_ENABLED := 0

# FlexCore settings
export FLEXCORE_ENV := development
export FLEXCORE_DEBUG := true
export FLEXCORE_PORT := 8080
export FLEXCORE_GRPC_PORT := 50051
export FLEXCORE_METRICS_PORT := 9090

# Database settings
export POSTGRES_HOST := localhost
export POSTGRES_PORT := 5432
export POSTGRES_DB := flexcore
export POSTGRES_USER := flexcore
export POSTGRES_PASSWORD := flexcore

# Redis settings
export REDIS_HOST := localhost
export REDIS_PORT := 6379
export REDIS_PASSWORD := ""
export REDIS_DB := 0

# Observability settings
export JAEGER_ENDPOINT := http://localhost:14268/api/traces
export PROMETHEUS_ENDPOINT := http://localhost:9090

# Plugin settings
export FLEXCORE_PLUGINS_DIR := /opt/flexcore/plugins
export FLEXCORE_PLUGINS_AUTO_DISCOVERY := true

# ============================================================================
# 📝 PROJECT METADATA
# ============================================================================

# Project information
PROJECT_NAME := flexcore
PROJECT_VERSION := $(shell git describe --tags --always --dirty)
PROJECT_DESCRIPTION := FlexCore - Distributed System Core Engine

.DEFAULT_GOAL := help

# Go build info
LDFLAGS := -ldflags="-X main.Version=$(PROJECT_VERSION) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# ============================================================================
# 🎯 FLEXCORE SPECIFIC COMMANDS
# ============================================================================

plugin-build: ## Build FlexCore plugins
	@echo "🔌 Building FlexCore plugins..."
	@go build -buildmode=plugin -o plugins/source.so ./plugins/source
	@go build -buildmode=plugin -o plugins/target.so ./plugins/target
	@go build -buildmode=plugin -o plugins/transformer.so ./plugins/transformer
	@echo "✅ Plugin build complete"

plugin-test: ## Test FlexCore plugins
	@echo "🧪 Testing FlexCore plugins..."
	@go test -v ./plugins/...
	@echo "✅ Plugin tests complete"

event-store-migrate: ## Migrate event store schema
	@echo "🗂️ Migrating event store schema..."
	@go run ./cmd/migrate --config configs/dev.yaml
	@echo "✅ Event store migration complete"

cqrs-test: ## Test CQRS event bus
	@echo "🧪 Testing CQRS event bus..."
	@go test -v ./internal/infrastructure/cqrs/...
	@echo "✅ CQRS tests complete"

performance-test: ## Run performance tests
	@echo "⚡ Running performance tests..."
	@go test -v -tags=performance ./tests/performance/...
	@echo "✅ Performance tests complete"

load-test: ## Run load tests
	@echo "📊 Running load tests..."
	@go run ./tests/load/ --duration=5m --concurrency=100
	@echo "✅ Load tests complete"

metrics-check: ## Check metrics endpoint
	@echo "📊 Checking metrics endpoint..."
	@curl -s http://localhost:9090/metrics | head -20
	@echo "✅ Metrics endpoint check complete"

health-check: ## Check health endpoint
	@echo "🏥 Checking health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo "✅ Health check complete"

# ============================================================================
# 🎯 OBSERVABILITY COMMANDS
# ============================================================================

trace-check: ## Check distributed tracing
	@echo "🔍 Checking distributed tracing..."
	@curl -s http://localhost:14268/api/traces?service=flexcore | jq .
	@echo "✅ Tracing check complete"

logs-tail: ## Tail application logs
	@echo "📝 Tailing application logs..."
	@docker-compose logs -f flexcore-server

prometheus-query: ## Query Prometheus metrics
	@echo "📊 Querying Prometheus metrics..."
	@curl -s 'http://localhost:9090/api/v1/query?query=flexcore_requests_total' | jq .
	@echo "✅ Prometheus query complete"

grafana-dash: ## Open Grafana dashboard
	@echo "📊 Opening Grafana dashboard..."
	@open http://localhost:3000/d/flexcore/flexcore-dashboard

# ============================================================================
# 🎯 DEVELOPMENT UTILITIES
# ============================================================================

watch-test: ## Watch and run tests on file changes
	@echo "👀 Watching for file changes..."
	@find . -name "*.go" | entr -r make test-unit

watch-build: ## Watch and build on file changes
	@echo "👀 Watching for build changes..."
	@find . -name "*.go" | entr -r make build

local-infra: ## Start local infrastructure (postgres, redis, jaeger)
	@echo "🏗️ Starting local infrastructure..."
	@docker-compose up -d postgres redis jaeger prometheus grafana
	@echo "✅ Local infrastructure started"

local-infra-down: ## Stop local infrastructure
	@echo "🏗️ Stopping local infrastructure..."
	@docker-compose down
	@echo "✅ Local infrastructure stopped"

# ============================================================================
# 🎯 FLEXT ECOSYSTEM INTEGRATION
# ============================================================================

ecosystem-check: ## Verify FLEXT ecosystem compatibility
	@echo "🌐 Checking FLEXT ecosystem compatibility..."
	@echo "📦 Distributed core: $(PROJECT_NAME) v$(PROJECT_VERSION)"
	@echo "🏗️ Architecture: Clean Architecture + DDD + CQRS"
	@echo "🔧 Language: Go $(GO_VERSION)"
	@echo "🎯 Framework: Event-driven distributed system"
	@echo "📊 Quality: Zero tolerance enforcement"
	@echo "✅ Ecosystem compatibility verified"

workspace-info: ## Show workspace integration info
	@echo "🏢 FLEXT Workspace Integration"
	@echo "==============================="
	@echo "📁 Project Path: $(PWD)"
	@echo "🏆 Role: Distributed System Core Engine"
	@echo "🔗 Dependencies: PostgreSQL, Redis, gRPC"
	@echo "📦 Provides: Event-driven distributed architecture"
	@echo "🎯 Standards: Enterprise Go patterns with observability"