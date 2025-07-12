# FlexCore Deployment Guide

This directory contains organized Docker deployment configurations following DRY principles and professional DevOps practices.

## 🐳 Docker Compose Files

### Main Configurations

```
docker-compose.yml          # Base production-ready configuration
docker-compose.override.yml # Development overrides (auto-included)
docker-compose.prod.yml     # Production optimizations and monitoring
docker-compose.test.yml     # Testing environment with temporary data
```

### Usage Patterns

```bash
# Development (default)
docker-compose up

# Production deployment
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Testing environment
docker-compose -f docker-compose.yml -f docker-compose.test.yml up

# Production with monitoring
docker-compose -f docker-compose.yml -f docker-compose.prod.yml --profile monitoring up -d

# Development with tools
docker-compose --profile dev-tools up
```

## 🏗️ Architecture

### Base Services (docker-compose.yml)

- **PostgreSQL 16**: Primary database with health checks
- **Redis 7**: Caching and session storage
- **Windmill Backend**: Native binary mounted from local build
- **FlexCore API**: Optional API service (profile: `api`)

### Development Overrides (docker-compose.override.yml)

- Different ports to avoid conflicts
- Debug logging enabled
- Development database with different credentials
- Admin tools (Adminer, Redis Commander)
- Volume mounts for configuration files

### Production Configuration (docker-compose.prod.yml)

- Resource limits and reservations
- Multi-replica deployments
- Secret management
- Monitoring stack (Prometheus, Grafana)
- Load balancer (Nginx)
- Performance optimizations

### Testing Configuration (docker-compose.test.yml)

- Temporary databases using tmpfs
- Test-specific environment variables
- Test runner service
- E2E testing with Selenium
- Mock services for external dependencies

## 🔧 Service Profiles

### Available Profiles

```bash
api          # FlexCore API service
dev-tools    # Development tools (Adminer, Redis Commander)
monitoring   # Prometheus + Grafana
loadbalancer # Nginx load balancer
e2e          # Selenium for E2E testing
mocks        # Mock external services
test-runner  # Automated test execution
test-data    # Test data initialization
```

### Profile Usage Examples

```bash
# Start with API and development tools
docker-compose --profile api --profile dev-tools up

# Production with full monitoring
docker-compose -f docker-compose.yml -f docker-compose.prod.yml \
  --profile monitoring --profile loadbalancer up -d

# Complete testing environment
docker-compose -f docker-compose.yml -f docker-compose.test.yml \
  --profile test-runner --profile e2e --profile mocks up
```

## 🌐 Port Mapping

### Development (docker-compose + override)

```
PostgreSQL:     5433  (dev database)
Redis:          6380  (dev cache)
Windmill:       8001  (dev API)
FlexCore:       3002  (dev API, optional)
Adminer:        8080  (database REDACTED_LDAP_BIND_PASSWORD)
Redis Commander: 8081  (Redis REDACTED_LDAP_BIND_PASSWORD)
```

### Production

```
PostgreSQL:     5432  (internal only)
Redis:          6379  (internal only)
Windmill:       8000  (internal only)
FlexCore:       3001  (internal only)
Nginx:          80/443 (public)
Prometheus:     9090  (monitoring)
Grafana:        3000  (dashboards)
```

## 🔒 Security Configuration

### Secrets Management

```
deployments/secrets/
├── postgres_password.txt    # Production database password
└── grafana_password.txt     # Grafana REDACTED_LDAP_BIND_PASSWORD password
```

**Note**: Actual secret files are not in version control. Create them manually:

```bash
# Create production secrets
echo "your_secure_postgres_password" > deployments/secrets/postgres_password.txt
echo "your_secure_grafana_password" > deployments/secrets/grafana_password.txt
chmod 600 deployments/secrets/*.txt
```

### Environment Security

- Development: Simple passwords for convenience
- Production: File-based secrets with restricted permissions
- Testing: Temporary passwords, no persistence

## 📊 Configuration Files

### Directory Structure

```
deployments/
├── config/
│   ├── nginx/           # Load balancer configuration
│   ├── prometheus/      # Metrics collection
│   ├── grafana/         # Dashboard configuration
│   └── postgres/        # Database tuning
├── secrets/             # Production secrets (not in VCS)
└── docker/              # Legacy configurations (for reference)
```

### Configuration Management

Each service configuration is externalized and mounted as volumes:

- **Nginx**: Reverse proxy and SSL termination
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Dashboards and visualization
- **PostgreSQL**: Performance tuning for production

## 🚀 Deployment Strategies

### Local Development

```bash
# Quick start
make dev

# With native Windmill
make dev-windmill

# Full stack with tools
docker-compose --profile api --profile dev-tools up
```

### Production Deployment

```bash
# Build all components
make build-all

# Deploy production stack
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Deploy with monitoring
docker-compose -f docker-compose.yml -f docker-compose.prod.yml \
  --profile monitoring up -d
```

### Testing Environment

```bash
# Run unified test suite
./scripts/test/run-tests.sh full

# Start test environment
docker-compose -f docker-compose.yml -f docker-compose.test.yml \
  --profile test-runner up

# E2E testing
docker-compose -f docker-compose.yml -f docker-compose.test.yml \
  --profile e2e --profile test-data up
```

## 📈 Monitoring & Observability

### Production Monitoring Stack

When using `--profile monitoring`:

- **Prometheus**: Metrics collection from all services
- **Grafana**: Pre-configured dashboards for FlexCore metrics
- **Health Checks**: All services include health check endpoints
- **Logs**: Structured logging with proper levels

### Available Dashboards

- FlexCore Application Metrics
- Windmill Performance
- Database Performance
- Redis Metrics
- System Resources

### Metrics Endpoints

```
Windmill:    http://localhost:8000/metrics
FlexCore:    http://localhost:3001/metrics
Prometheus:  http://localhost:9090
Grafana:     http://localhost:3000
```

## 🔄 Migration from Legacy

### Legacy Docker Files

The old Docker configurations have been preserved in `docker/` subdirectories for reference but are superseded by the new unified system.

### Migration Path

1. **Replace multiple docker-compose files** with profile-based configuration
2. **Consolidate environment variables** using consistent naming
3. **Standardize service definitions** with health checks and resource limits
4. **Implement proper secret management** for production
5. **Add monitoring and observability** by default

### Benefits of New Structure

- **DRY Principle**: No configuration duplication
- **Environment Parity**: Consistent service definitions across environments
- **Profile-Based**: Selective service activation
- **Production-Ready**: Resource limits, health checks, monitoring
- **Developer-Friendly**: Auto-loaded development overrides

## 🧪 Testing Integration

### Automated Testing

```bash
# Test with Docker environment
docker-compose -f docker-compose.yml -f docker-compose.test.yml run test-runner

# Manual testing with services
docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d
./scripts/test/run-tests.sh full
```

### Test Data Management

- Test databases use tmpfs for speed
- Automatic test data initialization
- Mock services for external dependencies
- Selenium for E2E browser testing

## 📚 Best Practices

### Development Workflow

1. Use `docker-compose up` for daily development
2. Enable profiles as needed (`--profile api`)
3. Use override file for environment-specific settings
4. Keep secrets out of version control

### Production Deployment

1. Use production compose file for optimization
2. Enable monitoring profile for observability
3. Configure proper secrets management
4. Set up log aggregation and alerting

### Maintenance

1. Regular health check monitoring
2. Volume backup procedures for persistent data
3. Secret rotation schedules
4. Performance monitoring and optimization

This unified Docker deployment system provides a professional, scalable foundation for FlexCore across all environments while maintaining simplicity and following DevOps best practices.
