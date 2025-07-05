# FlexCore

> **Professional Distributed Event-Driven Architecture System**

FlexCore is a production-ready, event-driven distributed system built in Go, designed for enterprise-scale data processing and workflow orchestration.

[![Go Version](https://img.shields.io/github/go-mod/go-version/flext/flexcore)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## ✨ Features

- **🔄 Event Sourcing** - Complete audit trail and state reconstruction
- **⚡ CQRS Pattern** - Separate read/write models for optimal performance  
- **🔌 Plugin System** - HashiCorp-style plugin architecture with dynamic loading
- **🌐 Distributed Cluster** - Multi-node coordination with Redis/etcd
- **📊 Observability** - Prometheus metrics, Grafana dashboards, and distributed tracing
- **🏢 Enterprise Ready** - Production-grade reliability and monitoring

## 🏗️ Architecture

```
flexcore/
├── cmd/                    # Application entrypoints
│   └── server/            # Main server application
├── internal/              # Private application code
│   ├── adapters/          # External integrations
│   ├── application/       # Business logic layer
│   │   ├── commands/      # Command handlers (CQRS)
│   │   ├── queries/       # Query handlers (CQRS)
│   │   └── services/      # Application services
│   ├── domain/            # Core business logic
│   │   ├── entities/      # Domain entities
│   │   ├── events/        # Domain events
│   │   └── repositories/  # Repository interfaces
│   └── infrastructure/    # Infrastructure implementations
│       ├── database/      # Database adapters
│       ├── messaging/     # Event bus implementation
│       ├── monitoring/    # Observability stack
│       └── plugins/       # Plugin system
├── pkg/                   # Public API
├── deployments/           # Deployment configurations
│   └── docker/           # Docker environments
├── configs/              # Configuration files
├── scripts/              # Build and utility scripts
└── docs/                 # Documentation
```

## 🚀 Quick Start

### Development

```bash
# Clone the repository
git clone https://github.com/flext/flexcore.git
cd flexcore

# Start development environment
docker-compose up -d

# Build the application
make build

# Run tests
make test

# Start the server
make run
```

### Production Deployment

```bash
# Deploy full cluster
docker-compose -f deployments/docker/production/docker-compose.production.yml up -d

# Check cluster status
./scripts/check-cluster-status.sh
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