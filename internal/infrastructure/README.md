# Infrastructure Layer

**Package**: `github.com/flext/flexcore/internal/infrastructure`  
**Version**: 0.9.0  
**Status**: In Development - Major Refactoring Required

## Overview

The Infrastructure Layer provides technical implementations for external concerns including data persistence, event storage, plugin execution, observability, and integration with external systems. This layer implements the technical details that support the domain and application layers while maintaining proper architectural boundaries.

## Architecture

This layer implements Clean Architecture infrastructure patterns:

- **Data Persistence**: PostgreSQL event store and aggregate persistence
- **Distributed Coordination**: Redis-based coordination and caching
- **Plugin System**: Dynamic plugin loading and execution
- **Event Sourcing**: Immutable event streams with PostgreSQL storage
- **Observability**: Comprehensive monitoring, metrics, and tracing
- **External Integration**: HTTP adapters and proxy patterns

## ⚠️ Current Status: Major Issues

**Critical Architectural Problems** (See [TODO.md](../../docs/TODO.md)):

- **Clean Architecture Violations**: Infrastructure concerns mixed with application logic
- **Multiple CQRS Implementations**: Conflicting patterns causing confusion
- **Inadequate Event Sourcing**: In-memory stores and mutable events
- **Plugin Security Issues**: No isolation or resource management
- **Performance Bottlenecks**: Inefficient data access patterns

## Core Components

### Event Sourcing (`eventsourcing/`, `postgres_event_store.go`)

#### Event Store Implementation
PostgreSQL-based event storage with stream management.

```go
// Located in: postgres_event_store.go
type PostgreSQLEventStore struct {
    db     *sql.DB
    logger *slog.Logger
}

func (es *PostgreSQLEventStore) SaveEvents(
    ctx context.Context,
    streamID string,
    events []domain.DomainEvent,
    expectedVersion int,
) error {
    // Atomic event persistence with optimistic concurrency
    // Serialization and compression of event data
    // Stream versioning and consistency validation
}

func (es *PostgreSQLEventStore) LoadEvents(
    ctx context.Context,
    streamID string,
    fromVersion int,
) ([]domain.DomainEvent, error) {
    // Event stream reconstruction
    // Deserialization and validation
    // Snapshot optimization support
}
```

#### Event Types and Serialization
```go
// Located in: eventsourcing/event_types.go
type EventData struct {
    ID            string                 `json:"id"`
    Type          string                 `json:"type"`
    StreamID      string                 `json:"stream_id"`
    Version       int                    `json:"version"`
    Data          map[string]interface{} `json:"data"`
    Metadata      map[string]interface{} `json:"metadata"`
    Timestamp     time.Time              `json:"timestamp"`
}
```

### Plugin System (`plugin_loader.go`, `plugin_execution_services.go`)

#### Dynamic Plugin Loading
```go
// Located in: plugin_loader.go
type PluginLoader struct {
    pluginDir string
    plugins   map[string]*plugin.Plugin
    logger    *slog.Logger
}

func (pl *PluginLoader) LoadPlugin(name string) (*plugin.Plugin, error) {
    // Dynamic shared object (.so) loading
    // Symbol resolution and interface validation
    // Plugin lifecycle management
    // Security context initialization
}
```

#### Plugin Execution Services
```go
// Located in: plugin_execution_services.go
type PluginExecutionService struct {
    loader     *PluginLoader
    coordinator *RedisCoordinator
    metrics    *observability.Metrics
}

func (pes *PluginExecutionService) ExecutePlugin(
    ctx context.Context,
    pluginID string,
    data map[string]interface{},
) (map[string]interface{}, error) {
    // Resource allocation and limits
    // Execution monitoring and timeout
    // Error recovery and logging
    // Metrics collection and reporting
}
```

### Distributed Coordination (`redis_coordinator.go`)

#### Redis-Based Coordination
```go
// Located in: redis_coordinator.go
type RedisCoordinator struct {
    client     redis.Client
    lockPrefix string
    ttl        time.Duration
}

func (rc *RedisCoordinator) AcquireLock(
    ctx context.Context,
    resource string,
    ttl time.Duration,
) (*DistributedLock, error) {
    // Distributed locking with automatic cleanup
    // Lock renewal and heartbeat management
    // Deadlock prevention and recovery
}

func (rc *RedisCoordinator) Publish(
    ctx context.Context,
    channel string,
    message interface{},
) error {
    // Pub/sub for real-time coordination
    // Message serialization and compression
    // Error handling and retry logic
}
```

### Observability (`observability/`)

#### Comprehensive Monitoring
```go
// Located in: observability/metrics.go
type Metrics struct {
    registry   prometheus.Registry
    counters   map[string]prometheus.Counter
    histograms map[string]prometheus.Histogram
    gauges     map[string]prometheus.Gauge
}

// System metrics collection
func (m *Metrics) RecordPipelineExecution(duration time.Duration, status string) {
    m.histograms["pipeline_duration"].Observe(duration.Seconds())
    m.counters["pipeline_executions_total"].WithLabelValues(status).Inc()
}
```

#### Distributed Tracing
```go
// Located in: observability/tracing.go
type TracingService struct {
    tracer opentracing.Tracer
    closer io.Closer
}

func (ts *TracingService) StartSpan(
    ctx context.Context,
    operationName string,
) (opentracing.Span, context.Context) {
    // Jaeger integration for distributed tracing
    // Span context propagation
    // Performance and error tracking
}
```

### CQRS Infrastructure (`cqrs/`)

#### Command/Query Bus Implementation
```go
// Located in: cqrs/cqrs_bus.go
type CQRSBus struct {
    commandHandlers map[string]CommandHandler
    queryHandlers   map[string]QueryHandler
    eventBus        EventBus
    metrics         *Metrics
}

func (bus *CQRSBus) ExecuteCommand(
    ctx context.Context,
    command Command,
) error {
    // Command validation and routing
    // Transaction boundary management
    // Event publication after successful execution
    // Metrics and monitoring integration
}
```

### Data Processors (`data-processor/`, `postgres-processor/`, `json-processor/`)

Various specialized data processing components:

- **Data Processor**: Generic data transformation pipeline
- **PostgreSQL Processor**: Database-specific operations and optimizations  
- **JSON Processor**: JSON data parsing, validation, and transformation

### External Integration

#### HTTP Adapters (`flext_http_adapter.go`, `flext_proxy_adapter.go`)
```go
// Located in: flext_http_adapter.go
type FlextHTTPAdapter struct {
    baseURL    string
    httpClient *http.Client
    logger     *slog.Logger
}

func (adapter *FlextHTTPAdapter) ExecutePlugin(
    ctx context.Context,
    pluginID string,
    parameters map[string]interface{},
) (*PluginResult, error) {
    // HTTP integration with FLEXT service
    // Request/response serialization
    // Error handling and retry logic
    // Circuit breaker patterns
}
```

## Observability Stack

### Monitoring Components
Located in dedicated directories with configuration:

#### Prometheus (`prometheus/`)
- **Metrics Collection**: System and business metrics
- **Alert Rules**: Proactive monitoring and alerting
- **Configuration**: `prometheus.yml` with scraping targets

#### Grafana (`grafana/`)
- **Dashboards**: Visual monitoring and analytics
- **Data Sources**: Prometheus integration configuration
- **Alerts**: Visual alerting and notification management

#### Jaeger (`jaeger/`)
- **Distributed Tracing**: Request flow across services
- **Performance Analysis**: Latency and bottleneck identification
- **Service Map**: Visual service dependency mapping

#### Alertmanager (`alertmanager/`)
- **Alert Routing**: Notification management and escalation
- **Silence Management**: Temporary alert suppression
- **Integration**: Email, Slack, PagerDuty notifications

### Docker Compose Integration
```yaml
# Located in: docker-compose.observability.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus:/etc/prometheus
  
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - ./grafana:/etc/grafana/provisioning
  
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"
```

## Data Persistence

### PostgreSQL Integration (`database.go`)
- **Connection Management**: Connection pooling and lifecycle
- **Schema Management**: Automatic migrations and versioning
- **Transaction Support**: ACID compliance and isolation levels
- **Performance Optimization**: Prepared statements and query optimization

### Event Store Schema
```sql
-- Event storage table structure
CREATE TABLE events (
    id UUID PRIMARY KEY,
    stream_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB,
    version INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(stream_id, version)
);

-- Indexes for performance
CREATE INDEX idx_events_stream_id ON events(stream_id);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_created_at ON events(created_at);
```

## Middleware (`middleware/`)

### Logging Middleware
```go
// Located in: middleware/logging.go
type LoggingMiddleware struct {
    logger *slog.Logger
}

func (lm *LoggingMiddleware) Handle(
    next http.Handler,
) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Request logging with correlation IDs
        // Response time measurement
        // Error logging and stack traces
        // Structured logging with context
    })
}
```

## Performance Considerations

### Current Bottlenecks
- **Database Access**: N+1 query problems and missing indexes
- **Plugin Loading**: Synchronous loading blocks request processing
- **Event Processing**: Lack of async processing and batching
- **Memory Usage**: Plugin instances not properly cleaned up

### Planned Optimizations
- **Connection Pooling**: Optimized database connection management
- **Async Processing**: Background event processing and plugin execution
- **Caching Strategies**: Redis integration for frequently accessed data
- **Resource Management**: Proper plugin lifecycle and memory management

## Security Considerations

### Current Security Issues
- **Plugin Isolation**: No sandboxing or resource limits
- **Input Validation**: Insufficient validation of plugin parameters
- **Access Control**: Limited authentication and authorization
- **Secret Management**: Hardcoded credentials and configuration

### Planned Security Enhancements
- **Plugin Sandboxing**: Containerized plugin execution
- **Input Sanitization**: Comprehensive validation and sanitization
- **OAuth/JWT Integration**: Proper authentication and authorization
- **Secret Management**: HashiCorp Vault integration

## Deployment and Operations

### Docker Configuration
Multiple Dockerfile configurations for different components:
- **Metrics**: `Dockerfile.metrics` for monitoring stack
- **Processors**: Individual containers for data processing components
- **Observability**: Complete monitoring and alerting stack

### Configuration Management
- **Environment Variables**: Runtime configuration
- **Config Files**: YAML-based configuration for complex settings
- **Feature Flags**: Runtime behavior modification
- **Service Discovery**: Automatic service registration and discovery

## Integration Testing

### Test Infrastructure
- **PostgreSQL Test Database**: Isolated testing environment
- **Redis Test Instance**: Separate coordination layer for tests
- **Mock Services**: External service simulation
- **Load Testing**: Performance validation under realistic load

## Migration and Refactoring Plan

### Phase 1: Architecture Cleanup
- Separate infrastructure concerns from application logic
- Consolidate CQRS implementations into single pattern
- Implement proper Clean Architecture boundaries

### Phase 2: Event Sourcing Enhancement
- Replace in-memory stores with persistent implementations
- Add event replay and snapshot capabilities
- Implement proper event versioning and migration

### Phase 3: Plugin System Security
- Add proper plugin isolation and sandboxing
- Implement resource limits and monitoring
- Add comprehensive security validation

### Phase 4: Performance Optimization
- Optimize database access patterns
- Implement async processing where appropriate
- Add comprehensive caching strategies

---

**See Also**: [Application Layer](../app/README.md) | [Domain Layer](../domain/README.md) | [TODO List](../../docs/TODO.md)