# FlexCore - Enterprise Runtime Container Service

**Type**: Core Service | **Status**: Development | **Dependencies**: Go 1.24+, PostgreSQL, Redis

**FlexCore** is the high-performance distributed runtime container service that serves as the **primary orchestration engine** for the entire FLEXT data integration ecosystem. Built with Go 1.24+ and implementing Clean Architecture, Domain-Driven Design (DDD), CQRS, and Event Sourcing patterns.

> ⚠️ **Development Status**: This project is currently under active development. See [TODO.md](docs/TODO.md) for architectural issues and implementation roadmap.

## 🎯 Project Objectives

### Primary Goals

- **Orchestration Engine**: Serve as the central runtime container for all FLEXT ecosystem services
- **Plugin Management**: Provide secure, isolated execution environment for data processing plugins
- **Event-Driven Architecture**: Implement comprehensive Event Sourcing with immutable event streams
- **Distributed Coordination**: Enable multi-node coordination and scalability
- **Integration Hub**: Bridge between Go performance layer and Python business logic (flext-core)

### FLEXT Ecosystem Integration

FlexCore acts as the **coordination layer** between:

- **flext-core**: Python foundation library with base patterns and dependency injection
- **Singer Ecosystem**: 15+ data taps, targets, and DBT transformation projects
- **Infrastructure Services**: Oracle, LDAP, WMS, and other enterprise integrations
- **Application Services**: API, auth, web interface, and CLI tools

## 🏗️ Architecture Overview

### Current Architecture

```
┌─────────────────────────────────────────────────┐
│               HTTP Layer (Port 8080)           │
│                (Gin Framework)                 │
├─────────────────────────────────────────────────┤
│              Application Layer                  │
│    ⚠️ Currently mixed with Infrastructure       │
├─────────────────────────────────────────────────┤
│                Domain Layer                     │
│     Entities, Aggregates, Domain Events        │
├─────────────────────────────────────────────────┤
│             Infrastructure Layer                │
│   PostgreSQL, Redis, Plugin System, CQRS       │
└─────────────────────────────────────────────────┘
```

### Target Architecture (Post-Refactoring)

```
┌─────────────────────────────────────────────────┐
│            Presentation Layer                   │
│         HTTP API (8080) + gRPC (50051)         │
├─────────────────────────────────────────────────┤
│            Application Layer                    │
│      Use Cases + Command/Query Handlers        │
├─────────────────────────────────────────────────┤
│              Domain Layer                       │
│    Rich Domain Model + Domain Services         │
├─────────────────────────────────────────────────┤
│           Infrastructure Layer                  │
│  Event Store + Plugin Runtime + Coordination   │
└─────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- **Go 1.24+** with generics support
- **Docker 24+** and Docker Compose
- **PostgreSQL 15+** for event sourcing
- **Redis 7+** for distributed coordination

### Development Setup

```bash
# Clone and setup
git clone <repository>
cd flexcore
make setup

# Start infrastructure services
docker-compose up -d postgres redis

# Build and run FlexCore
make build
make run

# Verify health
curl http://localhost:8080/health
```

### Production Deployment

```bash
# Build production image
make docker-build

# Deploy full stack with observability
docker-compose -f deployments/docker-compose.prod.yml up -d

# Monitor health
make service-health
curl http://localhost:8080/metrics
```

## 🔧 Development Commands

### Essential Development Workflow

```bash
# Quality gates (MANDATORY before commits)
make validate              # Complete validation pipeline
make check                 # Quick lint + vet + test
make test                  # Run comprehensive test suite
make security              # Security scanning with gosec

# Build and run
make build                 # Build optimized Go binary
make run                   # Start FlexCore server
make service-start         # Start with production settings

# Plugin development
make plugin-build          # Build all plugins
make plugin-test           # Test plugin functionality
```

### Docker Operations

```bash
# Development stack
docker-compose up -d                    # Full development stack
docker-compose logs -f flexcore-server  # Follow FlexCore logs

# Service endpoints
# FlexCore API: http://localhost:8080
# PostgreSQL: localhost:5433 (flexcore:flexcore_dev_2025)
# Redis: localhost:6380
# Grafana: http://localhost:3000 (REDACTED_LDAP_BIND_PASSWORD:flexcore_REDACTED_LDAP_BIND_PASSWORD_2025)
# Adminer: http://localhost:8081
```

## 📊 Current Project Status

### Architecture Compliance

- 🟡 **Clean Architecture**: 30% - Boundary violations identified, refactoring roadmap established
- 🟡 **DDD**: 40% - Domain entities exist, rich behavior implementation in progress
- ❌ **CQRS**: 25% - Multiple implementations need consolidation to single pattern
- ❌ **Event Sourcing**: 20% - Foundation exists, immutable implementation needed

### Development Progress

- ✅ **Basic HTTP API**: Functional with Gin framework
- ✅ **Docker Integration**: Complete development stack
- ✅ **Plugin Framework**: Basic plugin loading capability
- 🟡 **Database Integration**: PostgreSQL + Redis connected
- ❌ **Event Sourcing**: Requires complete refactoring
- 🟡 **Production Ready**: Development environment stable, production deployment needs architecture completion

### Key Issues (See [TODO.md](docs/TODO.md))

1. **Clean Architecture Violations**: HTTP server in application layer
2. **Multiple CQRS Implementations**: 3 different implementations causing confusion
3. **Inadequate Event Sourcing**: In-memory store, mutable events
4. **Plugin Security**: No isolation or resource management
5. **Testing Coverage**: Insufficient integration and domain tests

## 🔌 API Reference

### Core Endpoints

```bash
# Health and metrics
GET  /health                           # Service health check
GET  /metrics                          # Prometheus metrics
GET  /api/v1/flexcore/status          # Detailed system status

# Plugin management
GET  /api/v1/flexcore/plugins         # List registered plugins
POST /api/v1/flexcore/plugins/{id}/execute  # Execute plugin
GET  /api/v1/flexcore/plugins/{id}/status   # Plugin status

# Event sourcing (planned)
GET  /api/v1/flexcore/events          # Retrieve events
POST /api/v1/flexcore/events          # Publish event

# CQRS endpoints (planned)
POST /api/v1/flexcore/commands        # Execute command
POST /api/v1/flexcore/queries         # Execute query
```

## 🌐 FLEXT Ecosystem Integration

### Service Communication

- **FLEXT Service** (Port 8081): Python-based data processing service
- **flext-core Integration**: Foundation patterns and dependency injection
- **Singer Ecosystem**: Orchestrates 15+ taps, targets, and DBT projects
- **Infrastructure Projects**: Oracle, LDAP, WMS connectivity

### Integration Patterns

```bash
# FlexCore to FLEXT Service communication
curl -X POST http://localhost:8080/api/v1/flexcore/plugins/flext-service/execute \
  -H "Content-Type: application/json" \
  -d '{"operation": "health"}'

# Event-driven integration with flext-core patterns
# (Implementation pending architecture refactoring)
```

## 🔧 Configuration

### Environment Variables

```bash
# Core service configuration
FLEXCORE_PORT=8080
FLEXCORE_ENV=development
FLEXCORE_DEBUG=true

# Database configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_DB=flexcore
POSTGRES_USER=flexcore
POSTGRES_PASSWORD=flexcore_dev_2025

# Redis configuration
REDIS_HOST=localhost
REDIS_PORT=6380

# Event sourcing configuration
EVENT_STORE_TYPE=postgres
EVENT_STORE_CONNECTION=postgres://flexcore:flexcore_dev_2025@localhost:5433/flexcore

# Observability
OTEL_SERVICE_NAME=flexcore
PROMETHEUS_ENDPOINT=http://localhost:9090
```

## 📚 Documentation

### Architecture and Design

- [Architecture Overview](docs/architecture/overview.md) - System architecture and patterns
- [TODO List](docs/TODO.md) - **Critical issues and refactoring roadmap**
- [API Reference](docs/api-reference.md) - Complete API documentation

### Development Guides

- [Getting Started](docs/getting-started/installation.md) - Detailed setup instructions
- [Plugin Development](docs/development/plugins.md) - Plugin development guide
- [FLEXT Integration](docs/integration/flext-ecosystem.md) - Ecosystem integration

### Operations

- [Deployment Guide](docs/operations/deployment.md) - Production deployment
- [Monitoring](docs/operations/monitoring.md) - Observability and monitoring
- [Troubleshooting](docs/operations/troubleshooting.md) - Common issues and solutions

## 🧪 Testing

### Test Execution

```bash
# Comprehensive testing
make test                  # All tests with coverage
make test-unit             # Unit tests only
make test-integration      # Integration tests with DB/Redis
make test-race             # Race condition detection

# Coverage reporting
make coverage-html         # Generate HTML coverage report
```

### Current Test Coverage

- **Overall**: ~60% (Target: 90%+)
- **Domain Layer**: ~40% (Critical gap)
- **Infrastructure**: ~70%
- **Integration Tests**: ~30% (Major gap)

## 🛠️ Quality Standards

### Code Quality Requirements

- **Test Coverage**: Minimum 90% for production readiness
- **Linting**: Zero warnings with golangci-lint
- **Security**: Zero vulnerabilities with gosec scanning
- **Race Detection**: All concurrent code must pass race tests
- **Performance**: Sub-100ms API response times

### Architecture Requirements

- **Clean Architecture**: Strict layer boundaries
- **DDD**: Rich domain model with proper aggregates
- **CQRS**: Single, consistent implementation
- **Event Sourcing**: Immutable event streams with replay capability

## 🚧 Development Roadmap

### Phase 1: Critical Architecture Fixes (2-3 weeks)

- [ ] Refactor Clean Architecture violations
- [ ] Implement proper CQRS with single implementation
- [ ] Create immutable event sourcing with PostgreSQL
- [ ] Add comprehensive integration tests

### Phase 2: Domain Enhancement (3-4 weeks)

- [ ] Implement rich domain model
- [ ] Add domain services and proper aggregates
- [ ] Enhance plugin system with security isolation
- [ ] Improve error handling and observability

### Phase 3: Production Readiness (4-6 weeks)

- [ ] Performance optimizations
- [ ] Comprehensive monitoring and alerting
- [ ] Security hardening
- [ ] Documentation completion

## 🤝 Contributing

### Development Process

1. **Read [TODO.md](docs/TODO.md)** to understand current architectural issues
2. **Follow FLEXT patterns** from flext-core foundation library
3. **Maintain quality gates**: All tests must pass with `make validate`
4. **Architecture compliance**: Follow Clean Architecture principles strictly
5. **Submit PRs** with comprehensive tests and documentation

### Code Standards

- **Go 1.24+** with generics and modern patterns
- **Clean Architecture** with proper layer boundaries
- **Domain-Driven Design** with rich domain models
- **Comprehensive testing** with 90%+ coverage
- **Security-first** approach with plugin isolation

## 📄 License

This project is part of the FLEXT ecosystem. See [LICENSE](LICENSE) for details.

---

## ⚠️ Important Notes

**This project is currently in active development with significant architectural issues that prevent production use. Please review [TODO.md](docs/TODO.md) for detailed information about current limitations and the refactoring roadmap.**

**For production FLEXT deployments, please use stable components until FlexCore architectural refactoring is complete.**
