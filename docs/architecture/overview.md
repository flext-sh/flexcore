# FlexCore Architecture Overview

**Version**: 0.9.9 RC  
**Go Version**: 1.24+  
**Architecture**: Clean Architecture + DDD + CQRS + Event Sourcing

## System Overview

FlexCore implements a hexagonal architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                          FlexCore Runtime                       │
├─────────────────────────────────────────────────────────────────┤
│  Adapters (Interface Layer)                                    │
│  ┌─────────────────────┐  ┌─────────────────────────────────┐  │
│  │   Primary Adapters  │  │    Secondary Adapters           │  │
│  │   • HTTP/gRPC API   │  │    • PostgreSQL Repository     │  │
│  │   • CLI Interface   │  │    • Redis Message Bus         │  │
│  │   • Web Interface   │  │    • External Service Clients  │  │
│  └─────────────────────┘  └─────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  Application Layer (Use Cases)                                 │
│  ┌─────────────────────┐  ┌─────────────────────────────────┐  │
│  │   Command Handlers  │  │    Query Handlers               │  │
│  │   • Start Processor │  │    • Get Processor Status       │  │
│  │   • Stop Processor  │  │    • List Active Processors     │  │
│  │   • Load Plugin     │  │    • Plugin Health Check        │  │
│  └─────────────────────┘  └─────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  Domain Layer (Business Logic)                                 │
│  ┌─────────────────────┐  ┌─────────────────────────────────┐  │
│  │    Aggregates       │  │    Domain Services              │  │
│  │   • ProcessorRoot   │  │    • Plugin Orchestrator       │  │
│  │   • PipelineEntity  │  │    • Health Monitor             │  │
│  │   • PluginEntity    │  │    • Data Transformer          │  │
│  └─────────────────────┘  └─────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure Layer (Technical Details)                      │
│  • Event Store (PostgreSQL) • Message Bus (Redis)             │
│  • Plugin System • Observability • Configuration               │
└─────────────────────────────────────────────────────────────────┘
```

## Key Architectural Patterns

### 1. Clean Architecture

- **Dependency Inversion**: Core business logic independent of external concerns
- **Layer Separation**: Strict boundaries between domain, application, and infrastructure
- **Interface Segregation**: Small, focused interfaces for better testability

### 2. Domain-Driven Design (DDD)

- **Aggregates**: ProcessorAggregate, PipelineAggregate, PluginAggregate
- **Value Objects**: ProcessorID, DataFormat, FlextPluginConfig
- **Domain Events**: ProcessorStarted, PluginLoaded, PipelineCompleted

### 3. CQRS (Command Query Responsibility Segregation)

- **Command Side**: Write operations with business logic validation
- **Query Side**: Optimized read models for different use cases
- **Separate Models**: Different data structures for reads and writes

### 4. Event Sourcing

- **Immutable Events**: All state changes captured as events
- **Event Store**: PostgreSQL-based persistence with snapshots
- **Event Replay**: Ability to rebuild state from event history

## Directory Structure

```
flexcore/
├── internal/                   # Private application code
│   ├── adapters/              # Interface adapters (controllers, gateways)
│   │   ├── primary/           # Inbound adapters (HTTP, CLI)
│   │   └── secondary/         # Outbound adapters (DB, external APIs)
│   ├── app/                   # Application layer (use cases)
│   │   ├── commands/          # Command handlers
│   │   └── queries/           # Query handlers
│   ├── domain/                # Domain layer (business logic)
│   │   ├── entities/          # Domain entities and aggregates
│   │   ├── events/            # Domain events
│   │   └── services/          # Domain services
│   └── infrastructure/        # Infrastructure layer
│       ├── eventsourcing/     # Event store implementation
│       ├── persistence/       # Database repositories
│       └── observability/     # Monitoring and metrics
├── pkg/                       # Public packages
│   ├── api/                   # Public API definitions
│   ├── plugin/                # Plugin interfaces
│   └── result/                # Result type for error handling
├── cmd/                       # Application entry points
│   └── flexcore/              # Main service
├── api/                       # API definitions (gRPC, OpenAPI)
├── configs/                   # Configuration files
├── deployments/               # Deployment configurations
└── docs/                      # Architecture documentation
```

## Integration Points

### FLEXT Ecosystem Integration

- **flext-core**: Python foundation library integration
- **Singer SDK**: Data pipeline orchestration
- **Enterprise Services**: Oracle, LDAP, WMS integrations

### External Dependencies

- **PostgreSQL 15+**: Event store and persistence
- **Redis 7+**: Message bus and caching
- **Observability Stack**: Prometheus, Grafana, Jaeger

## Development Status

**Current Phase**: Architecture Foundation  
**Next Phase**: Core Systems Implementation  
**Target**: Production-ready v1.0

See [TODO.md](../TODO.md) for detailed implementation roadmap.
