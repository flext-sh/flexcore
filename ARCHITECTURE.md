# FlexCore Architecture Documentation

## 🎯 Overview

FlexCore é uma biblioteca Go moderna que implementa Clean Architecture com DDD, fornecendo um kernel robusto para aplicações empresariais. Inspirada nas melhores práticas do Python (lato, dependency-injector) e padrões avançados Go, FlexCore torna o desenvolvimento de adapters e aplicações extremamente simples, sem expor a complexidade interna.

## 🏗️ Princípios Arquiteturais

### 1. **Clean Architecture (Hexagonal)**

```
┌─────────────────────────────────────────────────────────┐
│                    Presentation Layer                    │
│              (HTTP, gRPC, CLI, WebSocket)               │
├─────────────────────────────────────────────────────────┤
│                    Application Layer                     │
│         (Commands, Queries, Application Services)        │
├─────────────────────────────────────────────────────────┤
│                      Domain Layer                        │
│      (Entities, Value Objects, Domain Services)         │
├─────────────────────────────────────────────────────────┤
│                  Infrastructure Layer                    │
│    (Repositories, Event Bus, External Services)         │
└─────────────────────────────────────────────────────────┘
```

### 2. **Domain-Driven Design (DDD)**

- **Entities**: Objetos com identidade (Pipeline, Plugin)
- **Value Objects**: Objetos imutáveis (PipelineID, Status)
- **Aggregates**: Raízes de agregação com consistência
- **Domain Events**: Comunicação assíncrona entre contextos
- **Domain Services**: Lógica de negócio complexa

### 3. **CQRS (Command Query Responsibility Segregation)**

```go
// Commands modificam estado
type CreatePipelineCommand struct {
    Name        string
    Description string
    Owner       string
}

// Queries leem estado
type GetPipelineQuery struct {
    PipelineID PipelineID
}
```

## 📦 Estrutura de Módulos

### Domain Layer (`/domain`)

**Responsabilidades:**
- Modelos de domínio puros
- Regras de negócio
- Interfaces de repositórios
- Eventos de domínio

**Componentes:**
- `base.go`: Tipos base (Entity, AggregateRoot, ValueObject)
- `entities/`: Entidades do domínio
- `events/`: Eventos de domínio
- `services/`: Serviços de domínio

### Application Layer (`/application`)

**Responsabilidades:**
- Orquestração de casos de uso
- Command/Query handlers
- Serviços de aplicação
- DTOs e transformações

**Componentes:**
- `commands/`: Command handlers e bus
- `queries/`: Query handlers
- `services/`: Serviços de aplicação

### Infrastructure Layer (`/infrastructure`)

**Responsabilidades:**
- Implementações concretas
- Integrações externas
- Persistência
- Configuração

**Componentes:**
- `config/`: Gerenciamento de configuração (Viper)
- `di/`: Container de injeção de dependência
- `events/`: Event bus (Windmill)
- `workflow/`: Engine de workflow (luno/workflow)
- `handlers/`: Cadeia de handlers HTTP/gRPC

### Shared Kernel (`/shared`)

**Responsabilidades:**
- Tipos compartilhados
- Utilitários
- Padrões funcionais

**Componentes:**
- `errors/`: Sistema de erros padronizado
- `result/`: Result type para error handling
- `patterns/`: Padrões funcionais (Maybe, Railway, Options)

## 🔌 Sistema de Adapters

### Base Adapter

Todos os adapters herdam funcionalidade comum:

```go
type BaseAdapter struct {
    // Lifecycle management
    Initialize(ctx) error
    Start(ctx) error
    Stop(ctx) error
    
    // Health checking
    HealthCheck(ctx) HealthStatus
    
    // Configuration
    Config() *config.Manager
    
    // Dependency injection
    Container() *di.AdvancedContainer
}
```

### Tipos de Adapters

1. **Source Adapters**: Produzem dados
2. **Target Adapters**: Consomem dados
3. **Transformer Adapters**: Transformam dados
4. **Connector Adapters**: Conectam sistemas

### Adapter Builder Pattern

```go
adapter := adapters.NewAdapterBuilder("name", "1.0.0", AdapterTypeSource).
    WithConfiguration(config).
    WithDependencyInjection(container).
    WithLogging(logger).
    WithMetrics(metrics).
    OnInitialize(initFunc).
    OnStart(startFunc).
    Build()
```

## 💉 Dependency Injection

### Inspirado em Python (lato/dependency-injector)

#### Automatic Resolution (lato-style)

```go
container.Provide("database", dbConnection)
container.Provide("logger", logger)

// Automatic injection by parameter name
result := container.Call(ctx, func(database DB, logger Logger) error {
    // database and logger are auto-injected
    return nil
})
```

#### Provider Pattern (dependency-injector style)

```go
container := di.NewContainerBuilder().
    WithSingleton("config", configFactory).
    WithFactory("service", serviceFactory).
    WithResource("database", dbFactory, dbCleanup).
    Build()
```

### Lifecycle Management

- **Singleton**: Uma instância por container
- **Transient**: Nova instância a cada resolução
- **Scoped**: Uma instância por escopo
- **Resource**: Com inicialização/cleanup

## 🎯 Padrões Funcionais

### Result Type

```go
func DoSomething() result.Result[string] {
    if err != nil {
        return result.Failure[string](err)
    }
    return result.Success("value")
}
```

### Maybe/Option Type

```go
func FindUser(id string) patterns.Maybe[User] {
    if user, exists := users[id]; exists {
        return patterns.Some(user)
    }
    return patterns.None[User]()
}
```

### Railway-Oriented Programming

```go
result := validateInput(input).
    Then(processData).
    ThenMap(transformResult).
    Recover(handleError)
```

## 📡 Event-Driven Architecture

### Event Bus (Windmill Integration)

```go
// Publish domain events
bus.Publish(ctx, PipelineCreatedEvent{
    PipelineID: pipeline.ID,
    CreatedAt:  time.Now(),
})

// Subscribe to events
bus.Subscribe("PipelineCreated", func(ctx context.Context, event DomainEvent) error {
    // Handle event
    return nil
})
```

### Event Sourcing Support

- Todos os agregados podem emitir eventos
- Event store para persistência
- Replay de eventos para reconstrução

## 🔄 Workflow Engine

### Luno/Workflow Integration

```go
workflow := NewPipelineWorkflow(pipelineID)
workflow.Configure(builder.
    AddStep("validate", validateStep).
    AddStep("execute", executeStep).
    AddStep("cleanup", cleanupStep))

engine.RegisterWorkflow(workflow)
engine.StartWorkflow(ctx, "pipeline-workflow", input)
```

## 🔧 Configuration Management

### Viper Integration

```go
config := config.NewManager(
    config.WithConfigFile("config.yaml"),
    config.WithEnvPrefix("FLEXCORE"),
    config.WithRemoteConfig(&RemoteConfig{
        Provider: "consul",
        Endpoint: "localhost:8500",
        Path:     "/config/flexcore",
    }),
)

// Hot-reloading
config.WatchConfig()
config.AddWatcher(func(old, new interface{}) {
    // Handle config change
})
```

### Typed Configuration

```go
dbConfig := config.GetDatabaseConfig()
serverConfig := config.GetServerConfig()
```

## 🛡️ Handler Chain Pattern

### Middleware Composition

```go
chain := handlers.NewChainBuilder().
    UseRecovery().                      // Panic recovery
    UseLogging(logger).                 // Request logging
    UseTracing(tracer).                 // Distributed tracing
    UseAuthentication(authenticator).   // Auth
    UseRateLimiting(limiter).          // Rate limiting
    UseTimeout(30*time.Second).        // Timeouts
    UseRetry(3, 100*time.Millisecond). // Retry logic
    Build()

handler := chain.Then(businessLogic)
```

## 🔌 Plugin System (go-plugin ready)

### Plugin Interface

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, input interface{}) (interface{}, error)
}
```

### Plugin Discovery & Loading

```go
registry := NewPluginRegistry()
registry.Discover("/path/to/plugins")
plugin := registry.Load("my-plugin")
```

## 🎨 Uso Prático

### Criando um Adapter Simples

```go
// 1. Defina seu adapter
type MyAdapter struct {
    *adapters.BaseAdapter
    client *MyClient
}

// 2. Use o builder
adapter := adapters.NewAdapterBuilder("my-adapter", "1.0.0", AdapterTypeSource).
    OnInitialize(func(ctx context.Context) error {
        // Inicialização
        return nil
    }).
    OnStart(func(ctx context.Context) error {
        // Lógica de start
        return nil
    }).
    Build()

// 3. Registre e use
registry.Register(adapter)
adapter.Initialize(ctx)
adapter.Start(ctx)
```

### Implementando um Command Handler

```go
type MyCommandHandler struct {
    repo Repository
    bus  EventBus
}

func (h *MyCommandHandler) Handle(ctx context.Context, cmd MyCommand) result.Result[interface{}] {
    // Validação
    if err := h.validate(cmd); err != nil {
        return result.Failure[interface{}](err)
    }
    
    // Execução
    entity := NewEntity(cmd.Data)
    
    // Persistência
    if err := h.repo.Save(ctx, entity); err != nil {
        return result.Failure[interface{}](err)
    }
    
    // Eventos
    h.bus.Publish(ctx, EntityCreatedEvent{ID: entity.ID})
    
    return result.Success[interface{}](entity)
}
```

## 🚀 Performance & Escalabilidade

### Concorrência

- Uso extensivo de channels e goroutines
- Thread-safe singleton initialization
- Concurrent-safe event bus

### Caching

- Provider-level caching
- Configuration caching
- Query result caching

### Observabilidade

- OpenTelemetry integration
- Prometheus metrics
- Structured logging
- Health checks

## 🧪 Testabilidade

### Mocking

```go
// Todos os componentes são baseados em interfaces
mockRepo := &MockRepository{}
mockBus := &MockEventBus{}

handler := NewHandler(mockRepo, mockBus)
```

### Test Helpers

```go
// Container de teste
testContainer := di.NewTestContainer()
testContainer.Override("service", mockService)

// Event bus de teste
testBus := events.NewTestEventBus()
testBus.ExpectEvent(PipelineCreatedEvent{})
```

## 📚 Conclusão

FlexCore fornece uma base sólida e extensível para aplicações Go empresariais, combinando:

- **Clean Architecture**: Separação clara de responsabilidades
- **DDD**: Modelagem rica do domínio
- **Padrões Funcionais**: Error handling robusto
- **DI Avançado**: Inspirado em Python mas idiomático Go
- **Simplicidade**: Complexidade escondida, API simples

O desenvolvedor pode focar na lógica de negócio enquanto FlexCore cuida da infraestrutura.