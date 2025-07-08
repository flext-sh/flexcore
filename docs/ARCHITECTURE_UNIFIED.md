# FlexCore Architecture - Unified Documentation

## 🎯 Overview

FlexCore é uma biblioteca Go moderna que implementa Clean Architecture com Domain-Driven Design (DDD), fornecendo um kernel robusto para aplicações empresariais distribuídas. O sistema é projetado para processamento de dados em escala empresarial e orquestração de workflows, seguindo os princípios de Clean Architecture com clara separação de responsabilidades e inversão de dependência.

Inspirada nas melhores práticas do Python (lato, dependency-injector) e padrões avançados Go, FlexCore torna o desenvolvimento de adapters e aplicações extremamente simples, sem expor a complexidade interna.

## 🏗️ Princípios Arquiteturais

### 1. Clean Architecture (Hexagonal)

```
┌─────────────────────────────────────────────────────────┐
│                    External Systems                     │
│  HTTP/gRPC • Database • Message Queue • External APIs   │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│                  Adapters Layer                        │
│              (HTTP, gRPC, CLI, WebSocket)               │
├─────────────────────────────────────────────────────────┤
│                 Application Layer                       │
│         (Commands, Queries, Application Services)        │
├─────────────────────────────────────────────────────────┤
│                   Domain Layer                          │
│      (Entities, Value Objects, Domain Services)         │
├─────────────────────────────────────────────────────────┤
│                Infrastructure Layer                     │
│          (Database, File System, External APIs)         │
└─────────────────────────────────────────────────────────┘
```

### 2. Domain-Driven Design (DDD)

- **Bounded Contexts**: Clear boundaries between different business domains
- **Aggregates**: Consistency boundaries for domain operations  
- **Domain Events**: Communication mechanism between aggregates
- **Value Objects**: Immutable domain concepts
- **Entities**: Objects with identity and lifecycle
- **Domain Services**: Operations that don't belong to entities

### 3. Event Sourcing & CQRS

- **Event Store**: Immutable log of all domain events
- **Event Streams**: Ordered sequences of events per aggregate
- **Projections**: Read-optimized views derived from events
- **Snapshots**: Performance optimization for aggregate reconstruction
- **Command/Query Separation**: Clear separation between write and read operations

## 🔧 Core Components

### Domain Layer

#### Bounded Contexts

```go
// Pipeline Context - Workflow orchestration
type Pipeline struct {
    ID          PipelineID
    Name        string
    Version     Version
    Status      PipelineStatus
    Steps       []Step
    Events      []DomainEvent
}

// Plugin Context - Extension management  
type Plugin struct {
    ID          PluginID
    Name        string
    Version     Version
    Type        PluginType
    Config      PluginConfig
    Status      PluginStatus
}

// Data Context - Data processing
type DataStream struct {
    ID          StreamID
    Schema      DataSchema
    Source      SourceConfig
    Target      TargetConfig
    Transform   TransformConfig
}
```

#### Domain Events

```go
type DomainEvent interface {
    AggregateID() string
    EventType() string
    OccurredAt() time.Time
    Version() int
}

// Pipeline Events
type PipelineCreated struct {
    PipelineID string
    Name       string
    CreatedBy  string
    CreatedAt  time.Time
}

type PipelineStepCompleted struct {
    PipelineID string
    StepID     string
    Output     interface{}
    Duration   time.Duration
}
```

### Application Layer

#### Commands & Queries

```go
// Commands (Write Operations)
type CreatePipelineCommand struct {
    Name        string
    Description string
    Steps       []StepDefinition
    CreatedBy   string
}

type ExecutePipelineCommand struct {
    PipelineID string
    Input      interface{}
    Options    ExecutionOptions
}

// Queries (Read Operations)
type GetPipelineQuery struct {
    PipelineID string
}

type ListPipelinesQuery struct {
    Filter PipelineFilter
    Limit  int
    Offset int
}
```

#### Application Services

```go
type PipelineService interface {
    CreatePipeline(ctx context.Context, cmd CreatePipelineCommand) (*Pipeline, error)
    ExecutePipeline(ctx context.Context, cmd ExecutePipelineCommand) (*ExecutionResult, error)
    GetPipeline(ctx context.Context, query GetPipelineQuery) (*Pipeline, error)
    ListPipelines(ctx context.Context, query ListPipelinesQuery) (*PipelineList, error)
}
```

### Infrastructure Layer

#### Repositories

```go
type PipelineRepository interface {
    Save(ctx context.Context, pipeline *Pipeline) error
    FindByID(ctx context.Context, id PipelineID) (*Pipeline, error)
    FindAll(ctx context.Context, filter PipelineFilter) ([]*Pipeline, error)
    Delete(ctx context.Context, id PipelineID) error
}

type EventStore interface {
    SaveEvents(ctx context.Context, aggregateID string, events []DomainEvent) error
    GetEvents(ctx context.Context, aggregateID string, fromVersion int) ([]DomainEvent, error)
    GetAllEvents(ctx context.Context, fromTimestamp time.Time) ([]DomainEvent, error)
}
```

#### External Integrations

```go
// Windmill Integration
type WindmillAdapter struct {
    client windmill.Client
    config WindmillConfig
}

func (w *WindmillAdapter) ExecuteScript(ctx context.Context, script Script) (*Result, error)
func (w *WindmillAdapter) CreateJob(ctx context.Context, job Job) (*JobID, error)
func (w *WindmillAdapter) GetJobStatus(ctx context.Context, jobID JobID) (*JobStatus, error)
```

### Presentation Layer

#### HTTP API

```go
// REST API Handlers
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request)
func (h *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request)
func (h *PipelineHandler) ExecutePipeline(w http.ResponseWriter, r *http.Request)

// gRPC Service
type PipelineGRPCService struct {
    pipelineService application.PipelineService
}

func (s *PipelineGRPCService) CreatePipeline(ctx context.Context, req *pb.CreatePipelineRequest) (*pb.Pipeline, error)
```

## 🔄 Event Flow Architecture

### Event-Driven Communication

```
┌─────────────┐    Events    ┌─────────────┐    Events    ┌─────────────┐
│   Pipeline  │──────────────▶│ Event Store │──────────────▶│ Projections │
│  Aggregate  │              │             │              │             │
└─────────────┘              └─────────────┘              └─────────────┘
       │                           │                           │
       │                           ▼                           ▼
       │                    ┌─────────────┐              ┌─────────────┐
       └────────────────────▶│ Event Bus   │──────────────▶│ Read Models │
                            └─────────────┘              └─────────────┘
                                   │
                                   ▼
                            ┌─────────────┐
                            │ Event       │
                            │ Handlers    │
                            └─────────────┘
```

### Data Flow

1. **Command Reception**: HTTP/gRPC requests received by presentation layer
2. **Command Processing**: Application services process commands and create domain events
3. **Event Persistence**: Events stored in event store for durability
4. **Event Publishing**: Events published to event bus for async processing
5. **Projection Updates**: Read models updated based on events
6. **Query Processing**: Queries served from optimized read models

## 📊 Data Management

### Event Sourcing Implementation

```go
type EventSourcingPipelineRepository struct {
    eventStore EventStore
    snapshots  SnapshotStore
}

func (r *EventSourcingPipelineRepository) Save(ctx context.Context, pipeline *Pipeline) error {
    events := pipeline.GetUncommittedEvents()
    return r.eventStore.SaveEvents(ctx, pipeline.ID.String(), events)
}

func (r *EventSourcingPipelineRepository) FindByID(ctx context.Context, id PipelineID) (*Pipeline, error) {
    events, err := r.eventStore.GetEvents(ctx, id.String(), 0)
    if err != nil {
        return nil, err
    }
    
    pipeline := &Pipeline{}
    return pipeline.LoadFromHistory(events), nil
}
```

### CQRS Implementation

```go
// Write Side (Commands)
type CommandBus interface {
    Dispatch(ctx context.Context, command interface{}) error
}

// Read Side (Queries)
type QueryBus interface {
    Execute(ctx context.Context, query interface{}) (interface{}, error)
}

type PipelineProjection struct {
    ID          string
    Name        string
    Description string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 🔌 Plugin Architecture

### Plugin Interface

```go
type Plugin interface {
    ID() PluginID
    Name() string
    Version() Version
    Execute(ctx context.Context, input interface{}) (interface{}, error)
    Validate(config interface{}) error
    GetSchema() PluginSchema
}

type PluginRegistry interface {
    Register(plugin Plugin) error
    Get(id PluginID) (Plugin, error)
    List() []Plugin
    Unregister(id PluginID) error
}
```

### Plugin Lifecycle

1. **Registration**: Plugins register with the plugin registry
2. **Discovery**: System discovers available plugins
3. **Validation**: Plugin configurations validated
4. **Execution**: Plugins executed within pipeline steps
5. **Monitoring**: Plugin performance and health monitored

## 🌐 Distributed Architecture

### Microservices Deployment

```yaml
services:
  flexcore-api:
    image: flexcore/api:latest
    environment:
      - DATABASE_URL=postgres://...
      - EVENT_STORE_URL=postgres://...
      - MESSAGE_QUEUE_URL=redis://...
    
  flexcore-worker:
    image: flexcore/worker:latest
    environment:
      - MESSAGE_QUEUE_URL=redis://...
      - PLUGIN_REGISTRY_URL=http://plugin-registry:8080
    
  flexcore-projection:
    image: flexcore/projection:latest
    environment:
      - EVENT_STORE_URL=postgres://...
      - READ_DB_URL=postgres://...
```

### Service Communication

- **Synchronous**: HTTP/gRPC for immediate responses
- **Asynchronous**: Message queues for event-driven communication
- **Event Streaming**: Kafka/NATS for real-time event processing

## 🔧 Integration with Windmill

### Native Integration

FlexCore integrates natively with Windmill for workflow execution:

```go
type WindmillExecutor struct {
    client     windmill.Client
    converter  TypeConverter
    monitor    ExecutionMonitor
}

func (e *WindmillExecutor) ExecutePipeline(ctx context.Context, pipeline *Pipeline) (*ExecutionResult, error) {
    // Convert FlexCore pipeline to Windmill workflow
    workflow := e.converter.ToWindmillWorkflow(pipeline)
    
    // Execute in Windmill
    job, err := e.client.CreateJob(ctx, workflow)
    if err != nil {
        return nil, err
    }
    
    // Monitor execution
    return e.monitor.WaitForCompletion(ctx, job.ID)
}
```

### Benefits of Integration

1. **Native Performance**: Direct binary execution without Docker overhead
2. **Type Safety**: Full Go type system integration
3. **Unified Monitoring**: Single observability stack
4. **Simplified Deployment**: No container orchestration complexity

## 📈 Observability & Monitoring

### Metrics Collection

```go
type Metrics interface {
    IncrementCounter(name string, tags map[string]string)
    RecordGauge(name string, value float64, tags map[string]string)
    RecordTimer(name string, duration time.Duration, tags map[string]string)
}

// Pipeline execution metrics
metrics.IncrementCounter("pipeline.executions.started", map[string]string{
    "pipeline_id": pipeline.ID.String(),
    "version":     pipeline.Version.String(),
})

metrics.RecordTimer("pipeline.execution.duration", executionTime, map[string]string{
    "pipeline_id": pipeline.ID.String(),
    "status":      "success",
})
```

### Distributed Tracing

```go
func (s *PipelineService) ExecutePipeline(ctx context.Context, cmd ExecutePipelineCommand) (*ExecutionResult, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "pipeline.execute")
    defer span.Finish()
    
    span.SetTag("pipeline.id", cmd.PipelineID)
    span.SetTag("pipeline.version", pipeline.Version)
    
    // Execution logic with trace propagation
    result, err := s.executor.Execute(ctx, pipeline)
    if err != nil {
        span.SetTag("error", true)
        span.LogFields(log.Error(err))
    }
    
    return result, err
}
```

## 🚀 Performance Characteristics

### Scalability

- **Horizontal Scaling**: Stateless services scale independently
- **Event Sourcing**: Natural partitioning by aggregate ID
- **CQRS**: Read and write loads scaled separately
- **Plugin Isolation**: Plugins executed in separate processes/containers

### Performance Optimizations

1. **Connection Pooling**: Database connections pooled across requests
2. **Caching**: Redis caching for frequently accessed data
3. **Async Processing**: Non-critical operations processed asynchronously
4. **Batch Operations**: Multiple operations batched for efficiency

## 🔒 Security Architecture

### Authentication & Authorization

```go
type SecurityContext struct {
    UserID      string
    Roles       []string
    Permissions []string
    TenantID    string
}

func (s *PipelineService) ExecutePipeline(ctx context.Context, cmd ExecutePipelineCommand) (*ExecutionResult, error) {
    secCtx := security.FromContext(ctx)
    if !secCtx.HasPermission("pipeline.execute") {
        return nil, ErrUnauthorized
    }
    
    // Execution logic
}
```

### Data Security

- **Encryption at Rest**: All sensitive data encrypted in database
- **Encryption in Transit**: TLS for all network communication
- **Secret Management**: Integration with HashiCorp Vault
- **Audit Logging**: All operations logged for compliance

## 🧪 Testing Strategy

### Testing Pyramid

```go
// Unit Tests - Domain Logic
func TestPipeline_Execute(t *testing.T) {
    pipeline := NewPipeline("test-pipeline")
    result, err := pipeline.Execute(context.Background(), testInput)
    assert.NoError(t, err)
    assert.Equal(t, expectedOutput, result)
}

// Integration Tests - Repository Layer  
func TestPipelineRepository_Save(t *testing.T) {
    repo := NewPostgresPipelineRepository(db)
    pipeline := testPipeline()
    err := repo.Save(context.Background(), pipeline)
    assert.NoError(t, err)
}

// End-to-End Tests - Full System
func TestPipelineAPI_CreateAndExecute(t *testing.T) {
    client := NewAPIClient(testServer.URL)
    pipeline := client.CreatePipeline(createRequest)
    result := client.ExecutePipeline(pipeline.ID, executeRequest)
    assert.Equal(t, expectedResult, result)
}
```

### Test Doubles

```go
type MockPipelineRepository struct {
    pipelines map[string]*Pipeline
}

func (m *MockPipelineRepository) Save(ctx context.Context, pipeline *Pipeline) error {
    m.pipelines[pipeline.ID.String()] = pipeline
    return nil
}

func (m *MockPipelineRepository) FindByID(ctx context.Context, id PipelineID) (*Pipeline, error) {
    pipeline, exists := m.pipelines[id.String()]
    if !exists {
        return nil, ErrPipelineNotFound
    }
    return pipeline, nil
}
```

## 📚 Development Guidelines

### Code Organization

```
cmd/
├── api/           # HTTP API server
├── worker/        # Background worker
└── cli/           # Command line interface

internal/
├── domain/        # Domain layer
│   ├── pipeline/  # Pipeline bounded context
│   ├── plugin/    # Plugin bounded context
│   └── shared/    # Shared kernel
├── application/   # Application services
├── infrastructure/ # Infrastructure implementations
└── interfaces/    # External interfaces

pkg/
├── client/        # Go client library
├── types/         # Shared types
└── utils/         # Utility functions
```

### Coding Standards

1. **Dependency Injection**: Constructor injection with interfaces
2. **Error Handling**: Explicit error handling with context
3. **Context Propagation**: Context passed through all layers
4. **Resource Management**: Proper cleanup with defer statements
5. **Concurrency**: Safe concurrent access with proper synchronization

## 🔄 Migration & Evolution

### Schema Evolution

- **Event Versioning**: Events versioned for backward compatibility
- **Projection Updates**: Read models updated via event replay
- **API Versioning**: REST API versioned for client compatibility

### Deployment Strategy

1. **Blue-Green Deployments**: Zero-downtime deployments
2. **Feature Flags**: Gradual feature rollouts
3. **Database Migrations**: Automated schema migrations
4. **Rollback Capability**: Quick rollback to previous versions

This unified architecture documentation consolidates all architectural decisions and patterns used in FlexCore, providing a single source of truth for the system design and implementation guidelines.