# FlexCore Professional Makefile
# Production-ready build automation

.DEFAULT_GOAL := help
.PHONY: help clean deps build test lint format security audit docker run dev prod

# Configuration
APP_NAME := flexcore
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}"
BUILD_FLAGS := -trimpath ${LDFLAGS}

# Directories
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

# Go configuration
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

## help: Show this help message
help:
	@echo "FlexCore Professional Build System"
	@echo "=================================="
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Version: ${VERSION}"
	@echo "Build: ${BUILD_TIME}"

## deps: Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2

## format: Format Go code
format:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy

## lint: Run linting
lint:
	@echo "Running linters..."
	$(GOLINT) run ./...

## security: Run security analysis
security:
	@echo "Running security analysis..."
	@which gosec > /dev/null || $(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	gosec ./...

## test: Run tests
test:
	@echo "Running tests..."
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -tags=integration -race ./tests/integration/...

## test-e2e: Run end-to-end tests  
test-e2e:
	@echo "Running end-to-end tests..."
	docker-compose -f deployments/docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f deployments/docker-compose.test.yml down

## build: Build application
build: clean
	@echo "Building application..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/flexcore
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-node ./cmd/flexcore-node

## build-all: Build for all platforms
build-all: clean
	@echo "Building for all platforms..."
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd/flexcore
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/flexcore
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/flexcore

## docker: Build Docker images
docker:
	@echo "Building Docker images..."
	docker build -t $(APP_NAME):$(VERSION) -f deployments/Dockerfile .
	docker build -t $(APP_NAME):latest -f deployments/Dockerfile .

## audit: Run dependency audit
audit:
	@echo "Running dependency audit..."
	$(GOMOD) tidy
	@which nancy > /dev/null || $(GOGET) github.com/sonatypecommunity/nancy@latest
	$(GOMOD) list -json -m all | nancy sleuth

## dev: Start development environment
dev:
	@echo "Starting development environment..."
	docker-compose -f deployments/docker-compose.dev.yml up --build

## prod: Start production environment
prod:
	@echo "Starting production environment..."
	docker-compose -f deployments/docker-compose.prod.yml up -d

## run: Run application locally
run: build
	@echo "Running application..."
	./$(BUILD_DIR)/$(APP_NAME)

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	docker system prune -f

## install: Install application
install: build
	@echo "Installing application..."
	cp $(BUILD_DIR)/$(APP_NAME) $(shell go env GOPATH)/bin/

## release: Create release
release: test lint security build-all
	@echo "Creating release $(VERSION)..."
	@echo "Built binaries in $(DIST_DIR)/"

## check: Run all checks (test, lint, security)
check: format lint security test
	@echo "All checks passed!"

## ci: Continuous integration pipeline
ci: deps check audit
	@echo "CI pipeline completed successfully!"

## info: Display build information
info:
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Go Version: $(shell go version)"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...