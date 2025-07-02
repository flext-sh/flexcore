# FlexCore - Clean Architecture Kernel for Go

**FlexCore** é uma biblioteca Go moderna que implementa Clean Architecture com DDD (Domain-Driven Design), sistema de eventos (windmill), workflows (luno/workflow) e dependency injection para substituir o core do projeto Flext.

## 🎯 Características Principais

- **Clean Architecture**: Estrutura hexagonal que força implementação correta
- **Domain-Driven Design**: Agregados, entidades, value objects e domain events
- **Sistema de Eventos**: Integração com Windmill para eventos distribuídos
- **Workflows**: Integração com luno/workflow para orquestração de processos
- **Dependency Injection**: Sistema de DI similar ao lato (Python)
- **Type Safety**: Tipos seguros com generics Go
- **Observabilidade**: Métricas e tracing built-in

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