# Installation Guide

This comprehensive guide will help you install FlexCore and set up your development environment for both Go and Python components.

## 📋 System Requirements

### Hardware Requirements
- **CPU**: 2+ cores (4+ recommended for development)
- **Memory**: 4GB RAM minimum (8GB+ recommended)
- **Storage**: 2GB free disk space
- **Network**: Internet connection for downloading dependencies

### Software Requirements

#### Core Runtime
- **Go**: 1.24 or later ([Download Go](https://golang.org/dl/))
- **Python**: 3.13 or later ([Download Python](https://python.org/downloads/))
- **Git**: For version control ([Download Git](https://git-scm.com/downloads))

#### Infrastructure Dependencies
- **PostgreSQL**: 13+ for event store ([Download PostgreSQL](https://postgresql.org/download/))
- **Redis**: 6+ for caching and message queue ([Download Redis](https://redis.io/download))
- **Docker**: For containerized deployment ([Download Docker](https://docker.com/get-started))

#### Optional Tools
- **Docker Compose**: For multi-container orchestration
- **kubectl**: For Kubernetes deployment
- **Make**: For build automation (usually pre-installed on Unix systems)

## 🚀 Installation Methods

### Method 1: Source Installation (Recommended for Development)

#### 1. Clone the Repository

```bash
# Clone the FlexCore repository
git clone https://github.com/flext-sh/flext.git
cd flext/flexcore

# Verify you're in the correct directory
ls -la
# Should show: go.mod, pyproject.toml, Makefile, etc.
```

#### 2. Install Go Dependencies

```bash
# Download and verify Go modules
go mod download
go mod verify

# Optional: Update to latest versions
go get -u ./...
go mod tidy
```

#### 3. Install Python Dependencies

```bash
# Using pip (ensure Python 3.13+ is active)
pip install -e .

# Or using Poetry (recommended)
curl -sSL https://install.python-poetry.org | python3 -
poetry install

# Verify installation
python -c "import flexcore; print(flexcore.__version__)"
```

#### 4. Setup Infrastructure

```bash
# Start required services with Docker Compose
make local-infra

# Or manually start services
docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=flexcore postgres:15
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Wait for services to be ready
sleep 10
```

#### 5. Verify Installation

```bash
# Run quality checks
make check

# Run basic tests
make test-unit

# Start the server
make run
```

If successful, you should see:
```
✅ All quality gates passed
✅ Unit tests complete
🚀 FlexCore server started on :8080
```

### Method 2: Docker Installation

#### 1. Quick Start with Docker

```bash
# Clone repository
git clone https://github.com/flext-sh/flext.git
cd flext/flexcore

# Build and start everything
make docker-up

# Check status
make cluster-status

# View logs
make docker-logs
```

#### 2. Docker Compose Configuration

Create `docker-compose.override.yml` for local customization:

```yaml
version: '3.8'

services:
  flexcore-server:
    environment:
      - FLEXCORE_DEBUG=true
      - FLEXCORE_LOG_LEVEL=debug
    ports:
      - "8080:8080"
      - "50051:50051"  # gRPC
    volumes:
      - ./configs:/app/configs
      - ./plugins:/app/plugins

  postgres:
    environment:
      - POSTGRES_DB=flexcore_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### Method 3: Binary Installation

#### 1. Download Pre-built Binaries

```bash
# Download latest release
curl -L https://github.com/flext-sh/flext/releases/latest/download/flexcore-linux-amd64.tar.gz | tar xz

# Move to PATH
sudo mv flexcore-linux-amd64/flexcore /usr/local/bin/
sudo chmod +x /usr/local/bin/flexcore

# Verify installation
flexcore --version
```

#### 2. Install Python Package

```bash
# Install from PyPI
pip install flexcore

# Verify installation
python -c "import flexcore; print('FlexCore installed successfully')"
```

## 🔧 Development Environment Setup

### IDE Configuration

#### Visual Studio Code

1. Install recommended extensions:
```bash
# Go extension
code --install-extension golang.go

# Python extension  
code --install-extension ms-python.python

# Docker extension
code --install-extension ms-azuretools.vscode-docker
```

2. Configure workspace settings (`.vscode/settings.json`):
```json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "python.defaultInterpreterPath": "./venv/bin/python",
    "python.linting.enabled": true,
    "python.linting.ruffEnabled": true,
    "python.formatting.provider": "black"
}
```

#### GoLand/PyCharm

1. Configure Go SDK to point to Go 1.24+
2. Set Python interpreter to 3.13+
3. Enable Go modules support
4. Configure code style to match project standards

### Git Hooks Setup

```bash
# Install pre-commit hooks
pip install pre-commit
pre-commit install

# Run hooks on all files
pre-commit run --all-files
```

### Environment Variables

Create `.env` file for local development:

```bash
# FlexCore Configuration
FLEXCORE_ENV=development
FLEXCORE_DEBUG=true
FLEXCORE_PORT=8080
FLEXCORE_GRPC_PORT=50051

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=flexcore_dev
POSTGRES_USER=flexcore
POSTGRES_PASSWORD=flexcore

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Observability
JAEGER_ENDPOINT=http://localhost:14268/api/traces
PROMETHEUS_ENDPOINT=http://localhost:9090
```

## 🔍 Verification & Testing

### Health Checks

```bash
# Check service health
make health-check

# Expected output:
# {
#   "status": "healthy",
#   "version": "0.7.0",
#   "uptime": "5m30s",
#   "dependencies": {
#     "database": "connected",
#     "redis": "connected"
#   }
# }
```

### API Testing

```bash
# Test HTTP API
curl http://localhost:8080/health

# Test gRPC API (requires grpcurl)
grpcurl -plaintext localhost:50051 health.Health/Check
```

### Integration Tests

```bash
# Run full integration test suite
make test-integration

# Run end-to-end tests
make test-e2e
```

## 🐛 Troubleshooting

### Common Issues

#### Go Module Issues
```bash
# Error: "module not found"
# Solution: Ensure Go 1.24+ and proper GOPATH
go version
go env GOPATH
go clean -modcache
go mod download
```

#### Python Import Errors
```bash
# Error: "No module named 'flexcore'"
# Solution: Verify Python version and virtual environment
python --version  # Should be 3.13+
pip list | grep flexcore
pip install -e . --force-reinstall
```

#### Database Connection Issues
```bash
# Error: "connection refused"
# Solution: Ensure PostgreSQL is running
docker ps | grep postgres
make local-infra
psql -h localhost -U flexcore -d flexcore_dev -c "SELECT 1;"
```

#### Port Conflicts
```bash
# Error: "port already in use"
# Solution: Check what's using the port
lsof -i :8080
# Kill the process or change FlexCore port
export FLEXCORE_PORT=8081
```

### Getting Help

If you encounter issues not covered here:

1. **Check Logs**: Look at service logs for detailed error messages
   ```bash
   make docker-logs
   journalctl -f  # For systemd services
   ```

2. **Search Issues**: Check existing GitHub issues
   ```bash
   # Search for similar problems
   https://github.com/flext-sh/flext/issues
   ```

3. **Create Issue**: Report new problems with:
   - Operating system and version
   - Go and Python versions
   - Complete error message
   - Steps to reproduce

## ✅ Next Steps

Once installation is complete:

1. **Follow Tutorial**: Complete the [Quick Start Guide](quickstart.md)
2. **Explore Examples**: Check out [example projects](https://github.com/flext-sh/flext-examples)
3. **Read Architecture**: Understand the [system design](../architecture/overview.md)
4. **Join Community**: Participate in [GitHub Discussions](https://github.com/flext-sh/flext/discussions)

## 📚 Additional Resources

- **Configuration Guide**: [Detailed configuration options](configuration.md)
- **Development Setup**: [Advanced development environment](development.md)
- **Docker Guide**: [Complete Docker deployment](../deployment/docker.md)
- **Kubernetes Guide**: [Production Kubernetes setup](../deployment/kubernetes.md)

---

**Installation complete!** 🎉 You're ready to build high-performance distributed systems with FlexCore.