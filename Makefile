# FlexCore Makefile - Real Implementation Testing

.PHONY: all build test e2e clean deps docker-up docker-down

# Variables
BINARY_NAME=flexcore
PLUGIN_DIR=plugins
TEST_TIMEOUT=60s
COVERAGE_FILE=coverage.out

# Default target
all: deps build test

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy

# Build main binary
build:
	@echo "🔨 Building FlexCore..."
	go build -o $(BINARY_NAME) ./cmd/flexcore/

# Build all plugins
build-plugins:
	@echo "🔌 Building plugins..."
	@for plugin in $(PLUGIN_DIR)/*; do \
		if [ -d "$$plugin" ] && [ -f "$$plugin/main.go" ]; then \
			echo "Building $$plugin..."; \
			(cd $$plugin && go build -o $$(basename $$plugin)); \
		fi \
	done

# Run unit tests
test:
	@echo "🧪 Running unit tests..."
	go test -v -race -timeout $(TEST_TIMEOUT) ./...

# Run E2E tests
e2e: docker-up build-plugins
	@echo "🚀 Running E2E tests..."
	./run_real_e2e_tests.sh

# Run specific E2E test
test-coordination:
	@echo "🔄 Testing distributed coordination..."
	go test -v -run TestRealDistributedCoordination ./tests/e2e/...

test-plugins:
	@echo "🔌 Testing plugin system..."
	go test -v -run TestRealPluginExecution ./tests/e2e/...

test-cluster:
	@echo "🌐 Testing multi-node cluster..."
	go test -v -run TestRealMultiNodeCluster ./tests/e2e/...

test-di:
	@echo "💉 Testing dependency injection..."
	go test -v -run TestRealDependencyInjection ./tests/e2e/...

test-integration:
	@echo "🔗 Testing complete integration..."
	go test -v -run TestCompleteIntegration ./tests/e2e/...

# Run tests with coverage
coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmarks
bench:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

# Start Docker services
docker-up:
	@echo "🐳 Starting Docker services..."
	@docker-compose up -d redis postgres

# Stop Docker services
docker-down:
	@echo "🛑 Stopping Docker services..."
	@docker-compose down

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE) coverage.html
	rm -f $(PLUGIN_DIR)/*/$(basename $(PLUGIN_DIR)/*)
	find . -name "*.log" -delete
	find . -name "*.test" -delete

# Development mode with hot reload
dev:
	@echo "👨‍💻 Starting development mode..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Lint code
lint:
	@echo "🔍 Linting code..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Security scan
security:
	@echo "🔒 Running security scan..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -quiet ./...

# Generate mocks
mocks:
	@echo "🎭 Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./...

# Multi-node cluster test
cluster-test: build docker-up
	@echo "🌍 Running multi-node cluster test..."
	./test_multi_node_real.sh

# Plugin test
plugin-test: build-plugins
	@echo "🔌 Running plugin test..."
	go run test_plugin_real.go

# Full validation
validate: deps fmt lint security test e2e
	@echo "✅ Full validation complete!"

# Help
help:
	@echo "FlexCore Makefile Commands:"
	@echo "  make all          - Install deps, build, and test"
	@echo "  make build        - Build FlexCore binary"
	@echo "  make build-plugins - Build all plugins"
	@echo "  make test         - Run unit tests"
	@echo "  make e2e          - Run E2E tests"
	@echo "  make coverage     - Generate coverage report"
	@echo "  make bench        - Run benchmarks"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make dev          - Start development mode"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make security     - Run security scan"
	@echo "  make validate     - Full validation suite"
	@echo ""
	@echo "Specific E2E tests:"
	@echo "  make test-coordination - Test distributed coordination"
	@echo "  make test-plugins      - Test plugin system"
	@echo "  make test-cluster      - Test multi-node cluster"
	@echo "  make test-di          - Test dependency injection"
	@echo "  make test-integration  - Test complete integration"