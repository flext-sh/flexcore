# FlexCore


<!-- TOC START -->
- [🚀 Key Features](#-key-features)
- [📦 Installation](#-installation)
  - [Prerequisites](#prerequisites)
  - [Development Setup](#development-setup)
- [🛠️ Usage](#-usage)
  - [API Endpoints](#api-endpoints)
  - [Docker Deployment](#docker-deployment)
- [🏗️ Architecture](#-architecture)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)
<!-- TOC END -->

[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Clean Architecture](https://img.shields.io/badge/Architecture-Clean_Architecture-green)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

**FlexCore** is the high-performance distributed runtime container service that serves as the primary orchestration engine for the FLEXT data integration ecosystem. Built with Go 1.24+, it implements Clean Architecture, Domain-Driven Design (DDD), CQRS, and Event Sourcing patterns to provide a robust, scalable foundation for enterprise data operations.

**Reviewed**: 2026-02-17 | **Version**: 0.10.0-dev

Part of the [FLEXT](https://github.com/flext-sh/flext) ecosystem.

## 🚀 Key Features

- **Distributed Orchestration**: Coordinates microservices and data pipelines across the FLEXT ecosystem.
- **Plugin System**: Secure, isolated execution environment for dynamic data processing plugins.
- **Event Sourcing**: Immutable event streams for auditability, replayability, and state reconstruction.
- **CQRS Architecture**: Separation of Command and Query responsibilities for optimized performance and scalability.
- **High Performance**: Built on Go's concurrency model for efficient parallel processing and low latency.
- **Observability**: Built-in metrics, logging, and tracing via OpenTelemetry standardization.

## 📦 Installation

### Prerequisites

- **Go 1.24+**
- **Docker 24+**
- **PostgreSQL 15+**
- **Redis 7+**

### Development Setup

```bash
# Clone and setup
git clone https://github.com/flext-sh/flexcore.git
cd flexcore
make setup

# Start infrastructure services
docker-compose up -d postgres redis

# Build and run
make build
make run
```

## 🛠️ Usage

### API Endpoints

FlexCore exposes a RESTful API for management and orchestration.

```bash
# Check service health
curl http://localhost:8080/health

# List registered plugins
curl http://localhost:8080/api/v1/flexcore/plugins

# Execute a plugin operation
curl -X POST http://localhost:8080/api/v1/flexcore/plugins/flext-service/execute \
  -H "Content-Type: application/json" \
  -d '{"operation": "health"}'
```

### Docker Deployment

Deploy the full stack using Docker Compose.

```bash
# Build production image
make docker-build

# Deploy stack
docker-compose -f deployments/docker-compose.prod.yml up -d
```

## 🏗️ Architecture

FlexCore follows strict Clean Architecture principles to ensure modularity and testability:

- **Presentation Layer**: Handles HTTP and gRPC entry points (`api`, `grpc`).
- **Application Layer**: Contains use cases, command handlers, and query handlers (`internal/app`).
- **Domain Layer**: Defines core business entities, aggregates, and domain events (`internal/domain`).
- **Infrastructure Layer**: Implements persistence, external adapters, and cross-cutting concerns (`internal/infra`).

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) for details on the development workflow, architectural guidelines, and submission process.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
