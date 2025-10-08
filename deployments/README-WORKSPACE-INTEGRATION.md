# FlexCore Docker Integration with Workspace Infrastructure

**Date**: 2025-08-23  
**Status**: INTEGRATED · 1.0.0 Release Preparation
**Migration**: COMPLETED

## Overview

FlexCore's Docker infrastructure has been integrated with the FLEXT workspace unified Docker structure. The local Docker files in this directory are maintained for reference and project-specific configurations, but **production deployments should use the workspace centralized Docker infrastructure**.

## Workspace Integration

### Centralized Docker Files (Preferred)

The following workspace Docker files should be used for FlexCore:

- **Development**: `../../docker/Dockerfile.flexcore-dev`
- **Production**: `../../docker/Dockerfile.flexcore-node`
- **Compose**: `../../docker/docker-compose.flexcore.yml`

### Updated Makefile Commands

The FlexCore Makefile now includes Docker integration commands that use the workspace infrastructure:

```bash
# Build using workspace Docker infrastructure
make docker-build                    # Development image
make docker-build-prod              # Production image

# Run with workspace Docker stack
make docker-run                     # Start complete stack
make docker-stop                    # Stop stack
make docker-logs                    # View logs
make docker-health                  # Health check
make docker-clean                   # Clean artifacts

# Single-letter shortcut
make d                              # Same as docker-run
```

## Architecture Benefits

### Unified Infrastructure

1. **Consistent Configuration**: All FLEXT projects use the same Docker patterns
2. **Centralized Management**: Single source of truth for Docker configurations
3. **Cross-Service Integration**: Seamless integration between FlexCore and FLEXT services
4. **Standardized Observability**: Unified monitoring stack (Prometheus, Grafana, Jaeger)

### Service Coordination

```yaml
# Workspace Docker Compose Structure
services:
  flexcore-server: # FlexCore runtime (port 8080)
    - PostgreSQL 15 # Event sourcing & persistence
    - Redis 7 # CQRS buses & caching
    - Jaeger # Distributed tracing
    - Prometheus # Metrics collection
    - Grafana # Monitoring dashboards
```

## Local Docker Files (Maintained for Reference)

The files in `flexcore/deployments/docker/` are maintained for:

- **Development Testing**: Local isolated testing
- **CI/CD Pipelines**: Project-specific build automation
- **Documentation**: Reference implementation patterns
- **Compatibility**: Legacy support during migration period

## Best Practices

### Development Workflow

```bash
# Recommended development workflow
cd ..

# 1. Build and test locally
make build test validate

# 2. Build Docker image using workspace infrastructure
make docker-build

# 3. Run with complete workspace stack
make docker-run

# 4. Monitor and develop
make docker-logs          # Watch logs
make docker-health        # Check health
curl http://localhost:8080/health  # Direct health check
```

### Production Deployment

```bash
# Production deployment using workspace infrastructure
cd ../..

# Build production image
make docker-build-flexcore-prod

# Deploy with full observability stack
docker-compose -f docker/docker-compose.flexcore.yml up -d

# Monitor deployment
docker-compose -f docker/docker-compose.flexcore.yml logs -f
```

## Configuration Management

### Environment Variables

FlexCore now inherits environment configuration from the workspace Docker stack:

```yaml
# Workspace configuration (docker-compose.flexcore.yml)
environment:
  - FLEXCORE_ENV=development
  - FLEXCORE_DEBUG=true
  - POSTGRES_HOST=postgres
  - REDIS_HOST=redis
  - OTEL_SERVICE_NAME=flexcore
  - FLEXT_SERVICE_URL=http://host.docker.internal:8000
```

### Service Discovery

FlexCore automatically integrates with:

- **FLEXT Service**: `http://host.docker.internal:8000` (Python bridge)
- **PostgreSQL**: `postgres:5432` (Event store)
- **Redis**: `redis:6379` (CQRS buses)
- **Jaeger**: `jaeger:14268` (Tracing)
- **Prometheus**: `prometheus:9090` (Metrics)

## Migration Status

✅ **COMPLETED**: FlexCore Makefile updated with workspace Docker integration  
✅ **COMPLETED**: Docker commands reference workspace infrastructure  
✅ **COMPLETED**: Help documentation updated with Docker commands  
✅ **COMPLETED**: Single-letter shortcuts added (`make d` for docker-run)  
✅ **COMPLETED**: Build system maintains workspace bin/ output

## Next Steps

1. **Team Migration**: Update team documentation to use workspace Docker commands
2. **CI/CD Updates**: Migrate build pipelines to use workspace infrastructure
3. **Monitoring Setup**: Configure team dashboards using workspace Grafana
4. **Documentation**: Update FlexCore deployment documentation

---

**Integration Complete**: FlexCore now uses unified workspace Docker infrastructure while maintaining project-specific flexibility.
