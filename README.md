# FlexCore - Event-Driven Distributed Architecture

FlexCore is a production-ready, event-driven distributed system built in Go, designed for enterprise-scale data processing and workflow orchestration.

## 🎯 Core Features

- **Event Sourcing** - Complete audit trail and state reconstruction
- **CQRS** - Separate read/write models for optimal performance  
- **Plugin System** - HashiCorp-style plugin architecture
- **Service Mesh** - Microservices with service discovery
- **Real-time Processing** - Event-driven data pipelines
- **Enterprise Ready** - Production-grade reliability and monitoring

## 🏗️ Arquitetura

```
flexcore/
├── domain/          # Camada de domínio (mais interna)
│   ├── entities/    # Entidades do domínio
│   ├── valueobjects/# Value objects
│   ├── aggregates/  # Aggregate roots
│   └── events/      # Domain events
├── application/     # Casos de uso e comandos
│   ├── commands/    # Command handlers
│   ├── queries/     # Query handlers
│   └── services/    # Application services
├── infrastructure/ # Adapters externos
│   ├── events/      # Event bus (windmill)
│   ├── workflow/    # Workflow engine (luno)
│   ├── persistence/ # Repositories
│   └── di/          # Dependency injection
└── shared/          # Tipos compartilhados
    ├── errors/      # Error handling
    ├── result/      # Result pattern
    └── validation/  # Validation framework
```

## 🚀 Quick Start

```go
package main

import (
    "github.com/flext/flexcore"
    "github.com/flext/flexcore/infrastructure/di"
)

func main() {
    // Initialize FlexCore kernel
    kernel := flexcore.NewKernel()
    
    // Setup dependency injection
    container := di.NewContainer()
    
    // Register services
    container.RegisterSingleton(NewPipelineService)
    
    // Start application
    app := kernel.BuildApplication(container)
    app.Run()
}
```

## 📦 Módulos

### Domain Layer
- Entidades principais do negócio
- Value objects imutáveis
- Aggregate roots para consistência
- Domain events para comunicação

### Application Layer
- Command/Query handlers (CQRS)
- Application services
- Use cases orquestration
- Business workflows

### Infrastructure Layer
- Event bus com Windmill
- Workflow engine com luno/workflow
- Repositories e adapters
- Dependency injection container

## 🔧 Dependências

- **github.com/luno/workflow**: Workflow engine
- **github.com/samber/do**: Dependency injection
- **github.com/google/uuid**: UUID generation
- **github.com/stretchr/testify**: Testing framework

## 📋 Exemplo de Uso

```go
// Definir um aggregate
type Pipeline struct {
    *flexcore.AggregateRoot
    ID     PipelineID
    Name   string
    Status PipelineStatus
}

// Command handler
type CreatePipelineCommand struct {
    Name string
}

func (h *PipelineCommandHandler) Handle(cmd CreatePipelineCommand) *flexcore.Result[Pipeline] {
    pipeline := NewPipeline(cmd.Name)
    
    // Emitir domain event
    pipeline.Emit(PipelineCreatedEvent{ID: pipeline.ID})
    
    // Salvar via repository
    return h.repo.Save(pipeline)
}

// Workflow definition
func PipelineWorkflow(w *workflow.Workflow) {
    w.AddStep("create", CreatePipelineStep)
    w.AddStep("validate", ValidatePipelineStep)
    w.AddStep("execute", ExecutePipelineStep)
}
```

## 🧪 Testing

```bash
go test ./...
```

## 📄 Licença

MIT License