# FlexCore Legacy Docker Files

**Status**: DEPRECATED · 1.0.0 Release Preparation
**Migration Date**: 2025-08-23  
**Replacement**: Workspace unified Docker infrastructure

## Overview

These Docker files have been moved to legacy status following FlexCore's integration with the FLEXT workspace unified Docker infrastructure. They are maintained for reference and historical purposes.

## Workspace Migration

FlexCore now uses the centralized workspace Docker infrastructure:

- **Development**: `../../../../docker/Dockerfile.flexcore-dev`
- **Production**: `../../../../docker/Dockerfile.flexcore-node`
- **Compose**: `../../../../docker/docker-compose.flexcore.yml`

## Updated Commands

```bash
# NEW: Use workspace infrastructure
make docker-build          # Uses workspace Dockerfile
make docker-run            # Uses workspace docker-compose
make docker-stop           # Workspace stack management

# OLD: Local docker files (now in legacy/)
docker-compose -f deployments/docker-compose.yml up -d
```

## Legacy Files

- `docker-compose.yml` - Original main configuration
- `docker-compose.prod.yml` - Production configuration
- `docker-compose.test.yml` - Testing configuration
- `docker-compose.override.yml` - Development overrides
- `docker-compose-windmill-*.yml` - Windmill integration variants
- `docker-compose.observability.yml` - Monitoring stack

## Removal Timeline

These files will be removed in FlexCore v1.0 once workspace integration is fully validated in production.

---

**Migration Complete**: Use workspace Docker infrastructure for all new deployments.
