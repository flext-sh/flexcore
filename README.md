# FlexCore - Enterprise Runtime Container

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20Architecture%20%2B%20DDD-green.svg)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
[![CQRS](https://img.shields.io/badge/Pattern-CQRS%20%2B%20Event%20Sourcing-orange.svg)](https://martinfowler.com/bliki/CQRS.html)

FlexCore is a high-performance distributed runtime container service built with Go 1.24+, implementing Clean Architecture, Domain-Driven Design (DDD), CQRS, and Event Sourcing patterns. It serves as the orchestration layer for the FLEXT data integration ecosystem.

## Overview

FlexCore provides enterprise-grade container orchestration capabilities:

- **Plugin Management**: Dynamic loading and execution of data processing plugins
- **Event Sourcing**: Immutable event store with PostgreSQL backend
- **CQRS Implementation**: Separate command and query processing paths
- **Distributed Coordination**: Redis-based cluster coordination
- **Service Integration**: Native integration with FLEXT data processing services

## Architecture

### Core Components

- **Command Bus**: Routes and executes state-changing commands
- **Query Bus**: Processes read-only data queries  
- **Event Store**: PostgreSQL-based immutable event log
- **Plugin System**: Dynamic plugin loading and execution
- **Service Integration**: Communication with FLEXT services

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────┐
│               Presentation Layer                │
│            (HTTP API, gRPC)                     │
├─────────────────────────────────────────────────┤
│              Application Layer                  │
│         (Use Cases, Command/Query Handlers)     │
├─────────────────────────────────────────────────┤
│                Domain Layer                     │
│         (Entities, Aggregates, Events)          │
├─────────────────────────────────────────────────┤
│             Infrastructure Layer                │
│    (Database, Message Bus, External Services)   │
└─────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites
- Go 1.24+
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+

### Development Setup

```bash
# Install dependencies and setup environment
make setup

# Start infrastructure services
make docker-up

# Build and run FlexCore
make build
make run

# Verify service health
curl http://localhost:8080/health
```

### Production Deployment

```bash
# Build production image
make docker-build

# Deploy with full stack
docker-compose -f docker-compose.prod.yml up -d

# Health check
make health-check
```

## API Endpoints

### Core Endpoints
- `GET /health` - Service health check
- `GET /metrics` - Prometheus metrics
- `GET /api/v1/flexcore/plugins` - List registered plugins
- `POST /api/v1/flexcore/plugins/{id}/execute` - Execute plugin

### Plugin Management
- `POST /api/v1/flexcore/plugins/register` - Register new plugin
- `DELETE /api/v1/flexcore/plugins/{id}` - Unregister plugin
- `GET /api/v1/flexcore/plugins/{id}/status` - Plugin status

## Development Commands

### Essential Commands
```bash
# Development workflow
make build                  # Build Go binary
make test                   # Run all tests with coverage
make run                    # Start FlexCore server
make check                  # Quick lint and vet check
make validate               # Complete validation pipeline

# Quality gates
make lint                   # Go linting with golangci-lint
make vet                    # Go vet analysis
make security               # Security scanning with gosec
make format                 # Format code with gofmt
```

### Testing Commands
```bash
# Test execution
make test-unit              # Unit tests only
make test-integration       # Integration tests
make test-race              # Race condition detection
make coverage-html          # Generate HTML coverage report

# Benchmarking
make benchmark              # Run benchmark tests
make performance-test       # Performance testing
```

### Docker Operations
```bash
# Local development
make docker-up              # Start PostgreSQL + Redis + FlexCore
make docker-down            # Stop all services
make docker-logs            # View service logs

# Service management
make service-start          # Start built binary
make service-health         # Check service endpoints
```

## Configuration

### Environment Variables
```bash
# Service configuration
export FLEXCORE_PORT=8080
export FLEXCORE_DATABASE_URL="postgresql://localhost:5433/flexcore"
export FLEXCORE_REDIS_URL="redis://localhost:6380"

# Event sourcing
export FLEXCORE_EVENT_STORE_ENABLED=true
export FLEXCORE_CQRS_ENABLED=true

# Observability
export FLEXCORE_METRICS_ENABLED=true
export FLEXCORE_TRACING_ENABLED=true
export OTEL_SERVICE_NAME="flexcore"
```

### Service Dependencies
- **PostgreSQL**: Database for event store and application data (port 5433)
- **Redis**: Caching and distributed coordination (port 6380)
- **Prometheus**: Metrics collection (port 9090)
- **Jaeger**: Distributed tracing (port 16686)

## Plugin System

### Plugin Interface
```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config Config) error
    Process(ctx context.Context, data interface{}) (interface{}, error)
    Shutdown() error
}
```

### Plugin Registration
```go
// Register FLEXT service as plugin
flextPlugin := infrastructure.NewFlextServicePlugin(
    "flext-service",
    "http://localhost:8081",
    logger,
)

err := pluginManager.Register(flextPlugin)
```

## Event Sourcing & CQRS

### Event Store
FlexCore implements event sourcing with PostgreSQL:
- Immutable event log
- Event replay capabilities
- Aggregate reconstruction
- Snapshot support

### Command/Query Separation
- **Commands**: State-changing operations routed through command bus
- **Queries**: Read-only operations processed by query handlers
- **Events**: Domain events published for all state changes

## Monitoring and Observability

### Health Checks
```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health with dependencies
curl http://localhost:8080/health?detail=true
```

### Metrics
FlexCore exposes Prometheus metrics:
- Request latency and throughput
- Plugin execution metrics
- Event store performance
- Resource utilization

### Distributed Tracing
OpenTelemetry integration provides:
- Request tracing across services
- Plugin execution spans
- Database operation tracing
- Error correlation

## Performance

### Benchmarks
- **Request Processing**: ~1000 requests/second
- **Plugin Execution**: Sub-millisecond overhead
- **Event Store**: 10,000+ events/second write throughput
- **Memory Usage**: <100MB baseline

### Optimization Features
- Connection pooling for database operations
- Redis-based caching for frequently accessed data
- Async plugin execution with goroutines
- Efficient serialization with Protocol Buffers

## Testing

### Test Coverage
- Unit tests: >90% coverage
- Integration tests: Full database and Redis integration
- End-to-end tests: Complete plugin execution workflows
- Performance tests: Load testing and benchmarks

### Testing Patterns
```bash
# Run specific test types
go test -short ./...           # Unit tests only
go test -run Integration ./... # Integration tests
go test -race ./...            # Race condition detection
go test -bench=. ./...         # Benchmark tests
```

## Troubleshooting

### Common Issues
- **Port conflicts**: Ensure ports 8080, 5433, 6380 are available
- **Database connection**: Verify PostgreSQL is running with correct credentials
- **Plugin loading**: Check plugin files exist and have correct permissions
- **Memory issues**: Monitor Go garbage collection and heap usage

### Debug Commands
```bash
# Service diagnostics
make diagnose               # Complete system diagnostics
make service-logs          # View detailed service logs

# Database debugging
make db-connect            # Connect to PostgreSQL
make event-store-status    # Check event store health

# Performance debugging
make profile-cpu           # CPU profiling
make profile-memory        # Memory profiling
```

## Integration with FLEXT Ecosystem

FlexCore integrates seamlessly with the broader FLEXT platform:

- **FLEXT Service**: Manages Python-based data processing services
- **Singer Ecosystem**: Orchestrates data taps, targets, and transformations
- **Monitoring Stack**: Integrates with flext-observability for comprehensive monitoring
- **Configuration**: Shares configuration patterns with other FLEXT services

## Contributing

1. Follow Go coding standards and project conventions
2. Implement comprehensive tests for all new features
3. Ensure all quality gates pass: `make validate`
4. Update documentation for API changes
5. Submit pull requests with detailed descriptions

## License

This project is part of the FLEXT ecosystem and follows the same licensing terms.

## 🎯 **Comandos Essenciais**

```bash
# Ambiente local
make build              # Build binário
make test               # Executar testes
make run                # Executar servidor
make dev                # Ambiente completo

# Docker
make docker-build       # Build imagem
make docker-up          # Stack completa
make docker-down        # Parar stack

# Qualidade
make check              # Verificações
make lint               # Linting
make security-scan      # Segurança
```

## 🔗 **Integração com Workspace**

FlexCore está integrado ao workspace FLEXT via master Makefile:

```bash
# Do workspace root (/home/marlonsc/flext)
make build-go           # Build FlexCore
make test-go            # Test FlexCore
make docker-go          # Docker FlexCore
make check-all          # Quality check todos os projetos
```

---

**FlexCore** = Runtime container distribuído para executar e orquestrar serviços FLEXT com arquitetura enterprise.