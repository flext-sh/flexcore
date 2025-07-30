# FLEXCORE Makefile
PROJECT_NAME := flexcore
GO_VERSION := 1.24
BINARY_NAME := flexcore
BUILD_DIR := bin

# Quality standards
MIN_COVERAGE := 80

# Service Configuration
SERVICE_PORT := 8080
SERVICE_HOST := localhost

# Help
help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Setup
setup: ## Complete project setup
	go mod download
	go mod tidy

mod-tidy: ## Tidy Go modules
	go mod tidy

mod-verify: ## Verify Go modules
	go mod verify

# Quality gates
validate: lint vet security test mod-verify ## Run all quality gates

check: lint vet ## Quick health check

lint: ## Run Go linting
	golangci-lint run

format: ## Format Go code
	go fmt ./...
	goimports -w .

vet: ## Run Go vet
	go vet ./...

security: ## Run security scanning
	gosec ./...

fix: format ## Auto-fix issues
	golangci-lint run --fix

# Testing
test: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

test-unit: ## Run unit tests
	go test -v -short ./...

test-integration: ## Run integration tests
	go test -v -run Integration ./...

test-race: ## Run tests with race detection
	go test -v -race ./...

test-bench: ## Run benchmark tests
	go test -v -bench=. -benchmem ./...

coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Build
build: ## Build the application
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

build-release: ## Build release version
	mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .

install: ## Install the application
	go install .

# Service operations
run: ## Run the service
	go run .

service-start: ## Start service
	./$(BUILD_DIR)/$(BINARY_NAME) -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

service-dev: ## Start service in development mode
	go run . -dev -port=$(SERVICE_PORT) -host=$(SERVICE_HOST)

service-health: ## Check service health
	curl -f http://$(SERVICE_HOST):$(SERVICE_PORT)/health || echo "Service not responding"

# FlexCore specific
plugin-build: ## Build FlexCore plugins
	mkdir -p plugins
	go build -buildmode=plugin -o plugins/source.so ./plugins/source
	go build -buildmode=plugin -o plugins/target.so ./plugins/target
	go build -buildmode=plugin -o plugins/transformer.so ./plugins/transformer

plugin-test: ## Test FlexCore plugins
	go test -v ./plugins/...

# Code generation
generate: ## Generate all code
	go generate ./...

proto-gen: ## Generate protobuf code
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

# Dependencies
deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

deps-check: ## Check for dependency updates
	go list -u -m all

# Maintenance
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf plugins/*.so
	go clean

clean-all: clean ## Deep clean including cache
	go clean -cache -modcache -testcache

reset: clean-all setup ## Reset project

# Diagnostics
diagnose: ## Project diagnostics
	@echo "Go: $$(go version)"
	@echo "Project: $$(go list -m)"
	@go env

doctor: diagnose check ## Health check

# Aliases
t: test
l: lint
f: format
b: build
c: clean
r: run
v: validate
s: service-start
h: service-health

.DEFAULT_GOAL := help
.PHONY: help setup mod-tidy mod-verify validate check lint format vet security fix test test-unit test-integration test-race test-bench coverage-html build build-release install run service-start service-dev service-health plugin-build plugin-test generate proto-gen deps-update deps-check clean clean-all reset diagnose doctor t l f b c r v s h