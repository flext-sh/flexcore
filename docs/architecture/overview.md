# FlexCore Architecture Overview

**Version**: 0.9.0 | **Status**: Under Refactoring | **Last Updated**: 2025-08-01

This document provides a comprehensive overview of FlexCore's architecture, including current implementation, identified issues, and target architecture for the ongoing refactoring effort.

> ⚠️ **Critical Notice**: FlexCore is currently undergoing major architectural refactoring due to significant violations of Clean Architecture, DDD, CQRS, and Event Sourcing principles. See [TODO.md](../TODO.md) for detailed issues and remediation plan.

## 🎯 System Overview

### Purpose and Scope

FlexCore serves as the **enterprise runtime container service** and **primary orchestration engine** for the entire FLEXT data integration ecosystem. It bridges high-performance Go services with Python business logic while maintaining strict architectural boundaries.

### Key Responsibilities

- **Plugin Orchestration**: Secure, isolated execution of data processing plugins
- **Event Sourcing**: Immutable event streams with complete audit trails
- **CQRS Implementation**: Separate command and query processing paths
- **Distributed Coordination**: Multi-node coordination via Redis and PostgreSQL
- **Service Integration**: Bridge between Go performance layer and Python business logic

### FLEXT Ecosystem Position

```
┌─────────────────────────────────────────────────────┐
│                FLEXT Ecosystem                      │
├─────────────────────────────────────────────────────┤
│  Singer Ecosystem (15+ projects)                   │
│  ├─ Taps (5): Oracle, LDAP, LDIF, OIC, WMS        │
│  ├─ Targets (5): Oracle, LDAP, LDIF, OIC, WMS     │
│  └─ DBT (4): Transformation projects               │
├─────────────────────────────────────────────────────┤
│  Application Services                               │
│  ├─ flext-api (FastAPI)                           │
│  ├─ flext-auth (Authentication)                   │
│  ├─ flext-web (Web Interface)                     │
│  └─ flext-cli (Command Line Tools)                │
├─────────────────────────────────────────────────────┤
│  🎯 FLEXCORE (THIS PROJECT)                        │
│     Runtime Container & Orchestration Engine       │
├─────────────────────────────────────────────────────┤
│  Infrastructure Services                            │
│  ├─ flext-db-oracle (Database Connectivity)       │
│  ├─ flext-ldap (Directory Services)               │
│  ├─ flext-grpc (Communication Protocols)          │
│  └─ flext-observability (Monitoring)              │
├─────────────────────────────────────────────────────┤
│  Foundation Libraries                               │
│  ├─ flext-core (Python Base Patterns)             │
│  └─ flext-observability (Monitoring Foundation)   │
└─────────────────────────────────────────────────────┘
```

## 🏗️ Current Architecture (Problems Identified)

### Layer Structure (Current Implementation)

```
┌─────────────────────────────────────────────────────┐
│            HTTP Layer (Port 8080)                  │
│         Gin Framework - RESTful API                │
├─────────────────────────────────────────────────────┤
│          Application Layer (VIOLATED)              │
│   ⚠️ HTTP Server directly embedded here             │
│   ⚠️ Direct config dependencies                     │
│   ✅ Basic command/query separation                 │
├─────────────────────────────────────────────────────┤
│            Domain Layer (ANEMIC)                   │
│   ✅ Entities and Aggregates defined               │
│   ⚠️ Lacks rich domain behavior                     │
│   ⚠️ Event sourcing poorly implemented              │
├─────────────────────────────────────────────────────┤
│         Infrastructure Layer (CHAOTIC)             │
│   ❌ 3 different CQRS implementations               │
│   ❌ In-memory event store for production           │
│   ✅ PostgreSQL and Redis integration               │
│   ⚠️ Plugin system lacks security isolation         │
└─────────────────────────────────────────────────────┘
```

### Critical Architecture Violations

#### 1. Clean Architecture Boundary Violations

**Location**: `internal/app/application.go:15-20`

```go
type Application struct {
    config *config.Config     // ❌ Infrastructure dependency
    server *http.Server       // ❌ HTTP in Application layer
    mux    *http.ServeMux     // ❌ Web framework in Application
}
```

**Impact**:

- Impossible to test application logic without HTTP server
- Coupling between business logic and web infrastructure
- Violation of Dependency Inversion Principle

#### 2. Multiple CQRS Implementations

**Implementations Found**:

- `internal/app/commands/command_bus.go` - Generic implementation
- `internal/infrastructure/cqrs/cqrs_bus.go` - SQLite-based implementation
- `internal/infrastructure/command_bus.go` - Function-based implementation

**Impact**:

- Architectural inconsistency and confusion
- Maintenance burden with multiple implementations
- No clear separation of concerns

#### 3. Inadequate Event Sourcing

**Location**: `internal/infrastructure/event_store.go:24-36`

```go
type MemoryEventStore struct {
    events map[string][]EventEntry  // ❌ In-memory for production
    mu     sync.RWMutex              // ❌ Single-node only
}

func (ar *AggregateRoot[T]) ClearEvents() {
    ar.domainEvents = make([]DomainEvent, 0)  // ❌ Mutable events
}
```

**Impact**:

- Data loss on service restart
- No replay capability
- Events are mutable (violates Event Sourcing principles)

#### 4. Plugin System Security Gaps

**Issues**:

- No process isolation between plugins
- No resource limits or sandboxing
- Shared memory space allows cross-plugin interference
- No capability-based security model

## 🎯 Target Architecture (Post-Refactoring)

### Clean Architecture Implementation

```
┌─────────────────────────────────────────────────────┐
│           Presentation Layer                        │
│    HTTP Adapters + gRPC Adapters                   │
│    ├─ REST API (Port 8080)                         │
│    ├─ gRPC API (Port 50051)                        │
│    └─ Health/Metrics Endpoints                     │
├─────────────────────────────────────────────────────┤
│          Application Layer                          │
│    Use Cases + Command/Query Handlers              │
│    ├─ Pipeline Management Use Cases                 │
│    ├─ Plugin Execution Use Cases                   │
│    ├─ Event Processing Use Cases                   │
│    └─ System Monitoring Use Cases                  │
├─────────────────────────────────────────────────────┤
│            Domain Layer                             │
│    Rich Domain Model + Domain Services             │
│    ├─ Pipeline Aggregate (Rich Behavior)           │
│    ├─ Plugin Aggregate (Lifecycle Management)      │
│    ├─ Event Sourcing (Immutable Events)           │
│    └─ Domain Services (Complex Orchestration)     │
├─────────────────────────────────────────────────────┤
│         Infrastructure Layer                        │
│    External Integrations + Technical Concerns      │
│    ├─ PostgreSQL Event Store                       │
│    ├─ Redis Distributed Coordination               │
│    ├─ Secure Plugin Runtime                        │
│    └─ FLEXT Service Integration                     │
└─────────────────────────────────────────────────────┘
```

### Domain-Driven Design Implementation

#### Core Aggregates

```
Pipeline Aggregate
├─ PipelineId (Value Object)
├─ Pipeline Entity (Rich Behavior)
├─ PipelineStep Entities
├─ Domain Events
│  ├─ PipelineCreated
│  ├─ PipelineActivated
│  ├─ PipelineStarted
│  └─ PipelineCompleted
└─ Domain Services
   ├─ PipelineOrchestrationService
   └─ PipelineValidationService

Plugin Aggregate
├─ PluginId (Value Object)
├─ Plugin Entity (Lifecycle Management)
├─ PluginExecution Entities
├─ Domain Events
│  ├─ PluginRegistered
│  ├─ PluginExecutionStarted
│  └─ PluginExecutionCompleted
└─ Domain Services
   ├─ PluginSecurityService
   └─ PluginResourceManager
```

#### Domain Services

- **Pipeline Orchestration Service**: Complex multi-pipeline coordination
- **Plugin Security Service**: Isolation and capability management
- **Event Coordination Service**: Cross-aggregate event handling
- **Resource Management Service**: CPU, memory, and I/O optimization

### CQRS + Event Sourcing Implementation

#### Command Side (Write Model)

```
Command Bus
├─ CreatePipelineCommand
├─ StartPipelineCommand
├─ ExecutePluginCommand
└─ RegisterPluginCommand

Command Handlers
├─ CreatePipelineHandler
├─ StartPipelineHandler
├─ ExecutePluginHandler
└─ RegisterPluginHandler

Event Store (PostgreSQL)
├─ Immutable Event Streams
├─ Event Replay Capability
├─ Snapshot Storage
└─ Event Versioning
```

#### Query Side (Read Model)

```
Query Bus
├─ GetPipelineStatusQuery
├─ ListActivePluginsQuery
├─ GetExecutionHistoryQuery
└─ GetSystemMetricsQuery

Query Handlers
├─ PipelineStatusHandler
├─ ActivePluginsHandler
├─ ExecutionHistoryHandler
└─ SystemMetricsHandler

Read Models (Optimized Views)
├─ PipelineStatusView
├─ PluginInventoryView
├─ ExecutionHistoryView
└─ SystemMetricsView
```

### Plugin Architecture (Secure Runtime)

#### Security Model

```
Plugin Sandbox
├─ Process Isolation
│  ├─ Separate OS processes
│  ├─ Controlled system calls
│  └─ Resource limits (CPU, Memory, I/O)
├─ Capability-Based Security
│  ├─ Explicit permissions
│  ├─ API access control
│  └─ Data access restrictions
└─ Communication Channel
   ├─ Secure IPC mechanisms
   ├─ Serialized data exchange
   └─ Audit logging
```

#### Plugin Lifecycle

1. **Registration**: Validate plugin and establish security boundaries
2. **Initialization**: Controlled startup with resource allocation
3. **Execution**: Sandboxed execution with monitoring
4. **Cleanup**: Resource deallocation and security cleanup

## 🔧 Technology Stack

### Core Technologies

- **Go 1.24+**: High-performance runtime with generics support
- **PostgreSQL 15+**: Event store and application database
- **Redis 7+**: Distributed coordination and caching
- **Docker 24+**: Containerization and deployment

### Framework Integration

- **Gin Framework**: HTTP API layer (to be moved to presentation)
- **GORM**: Database ORM for read models
- **go-redis**: Redis client for distributed coordination
- **zap**: Structured logging framework

### Observability Stack

- **Prometheus**: Metrics collection and monitoring
- **Grafana**: Visualization and alerting dashboards
- **Jaeger**: Distributed tracing and performance analysis
- **OpenTelemetry**: Observability instrumentation

## 🔄 Integration Patterns

### FLEXT Ecosystem Integration

#### Service Communication

- **HTTP/REST**: Synchronous API calls for immediate operations
- **Event Streams**: Asynchronous coordination via PostgreSQL event store
- **Redis Pub/Sub**: Real-time state synchronization
- **gRPC**: High-performance service-to-service communication

#### flext-core Integration

```go
// Pattern integration with flext-core (Python)
type FlextCoreIntegration struct {
    ServiceResult[T]  // Use flext-core Result pattern
    DIContainer       // Dependency injection from flext-core
    LoggingContext    // Structured logging integration
    EventBus         // Cross-language event communication
}
```

### Data Flow Architecture

```
External Request
    ↓
HTTP/gRPC Adapter (Presentation)
    ↓
Use Case Handler (Application)
    ↓
Domain Service (Domain)
    ↓
Repository/Event Store (Infrastructure)
    ↓
PostgreSQL/Redis
```

## 📊 Quality Attributes

### Performance Requirements

- **API Response Time**: < 100ms for 95th percentile
- **Plugin Execution**: < 1s startup time per plugin
- **Event Processing**: 10,000+ events/second throughput
- **Memory Usage**: < 1GB baseline, < 4GB under load

### Scalability Requirements

- **Horizontal Scaling**: Support for multi-node clusters
- **Plugin Concurrency**: 100+ concurrent plugin executions
- **Event Store**: Handle millions of events with sub-second queries
- **Database Connections**: Efficient connection pooling

### Reliability Requirements

- **Availability**: 99.9% uptime SLA
- **Error Recovery**: Automatic retry with exponential backoff
- **Data Consistency**: ACID transactions for critical operations
- **Plugin Isolation**: Plugin failures cannot affect system stability

### Security Requirements

- **Plugin Sandboxing**: Complete process isolation
- **Event Integrity**: Tamper-proof event streams
- **Access Control**: Role-based API access
- **Audit Logging**: Complete audit trail for compliance

## 🚧 Migration Strategy

### Phase 1: Foundation (2-3 weeks)

1. **Extract HTTP Layer**: Move HTTP server to presentation layer
2. **Unify CQRS**: Choose single CQRS implementation
3. **Implement PostgreSQL Event Store**: Replace in-memory store
4. **Add Integration Tests**: Comprehensive test coverage

### Phase 2: Domain Enhancement (3-4 weeks)

1. **Rich Domain Model**: Implement proper aggregates with behavior
2. **Domain Services**: Add complex business logic orchestration
3. **Plugin Security**: Implement process isolation and sandboxing
4. **Event Sourcing**: Complete immutable event streams with replay

### Phase 3: Production Readiness (4-6 weeks)

1. **Performance Optimization**: Meet performance SLA requirements
2. **Security Hardening**: Complete security audit and fixes
3. **Observability**: Comprehensive monitoring and alerting
4. **Documentation**: Complete technical documentation

## 📚 Architecture Decision Records

### ADR-001: Clean Architecture Adoption

- **Status**: Approved
- **Decision**: Implement Clean Architecture with strict layer boundaries
- **Rationale**: Ensure testability, maintainability, and technology independence

### ADR-002: Event Sourcing with PostgreSQL

- **Status**: Approved
- **Decision**: Use PostgreSQL for event store instead of specialized event databases
- **Rationale**: Leverage existing PostgreSQL expertise and infrastructure

### ADR-003: Plugin Process Isolation

- **Status**: Approved
- **Decision**: Implement plugin sandboxing with separate OS processes
- **Rationale**: Security isolation and fault tolerance requirements

### ADR-004: CQRS Implementation Strategy

- **Status**: Approved
- **Decision**: Single CQRS implementation with PostgreSQL for both read and write models
- **Rationale**: Operational simplicity while maintaining CQRS benefits

## ⚠️ Known Limitations and Risks

### Current Limitations

- **Not Production Ready**: Critical architectural violations prevent production use
- **Single Node**: Current implementation doesn't support horizontal scaling
- **Plugin Security**: No isolation between plugins creates security risks
- **Event Sourcing**: In-memory implementation loses data on restart

### Technical Debt

- **Multiple CQRS Implementations**: Creates maintenance burden and confusion
- **Anemic Domain Model**: Business logic scattered across layers
- **Missing Integration Tests**: Limited confidence in system behavior
- **Inconsistent Error Handling**: Mix of Result pattern and Go errors

### Migration Risks

- **Breaking Changes**: API contracts will change during refactoring
- **Data Migration**: Event store migration requires careful planning
- **Performance Impact**: Temporary performance degradation during migration
- **Testing Gaps**: Need comprehensive testing during architectural changes

---

## 📖 Related Documentation

- [TODO.md](../TODO.md) - **Critical issues and refactoring roadmap**
- [API Reference](../api-reference.md) - Current API documentation
- [Plugin Development](../development/plugins.md) - Plugin development guide
- [FLEXT Integration](../integration/flext-ecosystem.md) - Ecosystem integration

For the most current architectural status and critical issues, always refer to [TODO.md](../TODO.md).
