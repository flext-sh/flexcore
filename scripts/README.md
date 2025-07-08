# FlexCore Scripts Organization

This directory contains organized scripts following DRY principles and professional categorization.

## Structure

```
scripts/
├── build/           # Build and compilation scripts
│   ├── build-all.sh         # Unified build master (RECOMMENDED)
│   ├── build.sh             # Legacy build script
│   ├── build-real-distributed.sh  # Distributed build
│   ├── mkall.sh             # Make all utility
│   └── mkerrors.sh          # Error handling utility
│
├── test/            # Testing and validation scripts
│   ├── run-tests.sh         # Unified test master (RECOMMENDED)
│   ├── test-*.sh            # Individual test scripts
│   ├── run_*test*.sh        # E2E test runners
│   └── go.test.sh           # Go testing utility
│
├── validation/      # System validation and benchmarks
│   ├── validate-native.sh   # Native system validation
│   ├── validate-flexcore-100-percent.sh  # Complete validation
│   ├── benchmark-native.sh  # Performance benchmarks
│   └── check-cluster-status.sh  # Cluster health checks
│
├── deployment/      # Deployment and cluster management
│   ├── deploy.sh            # Unified deployment master (RECOMMENDED)
│   ├── start-*.sh           # Service startup scripts
│   └── stop-cluster.sh      # Cluster shutdown
│
└── utilities/       # General utilities and tools
    ├── fix-imports.sh       # Import fixing utility
    └── init-postgres.sql    # Database initialization
```

## Master Scripts (RECOMMENDED)

### 🏗️ Build Master: `build/build-all.sh`

Unified build system that eliminates duplication:

```bash
# Build everything in development mode
./scripts/build/build-all.sh

# Build for production
BUILD_MODE=release ./scripts/build/build-all.sh

# Build only Windmill components
./scripts/build/build-all.sh windmill

# Build only Go application
./scripts/build/build-all.sh go

# Clean and rebuild everything
./scripts/build/build-all.sh clean all
```

### 🧪 Test Master: `test/run-tests.sh`

Unified test suite that consolidates all testing:

```bash
# Quick essential tests (default)
./scripts/test/run-tests.sh

# Comprehensive testing including builds
./scripts/test/run-tests.sh full

# Help information
./scripts/test/run-tests.sh --help
```

### 🚀 Deployment Master: `deployment/deploy.sh`

Unified deployment system for all environments:

```bash
# Local development
./scripts/deployment/deploy.sh local

# Docker deployment
./scripts/deployment/deploy.sh docker

# Production Docker
ENVIRONMENT=production ./scripts/deployment/deploy.sh docker

# Distributed cluster with 3 workers
REPLICAS=3 ./scripts/deployment/deploy.sh cluster

# Native deployment (no Docker)
./scripts/deployment/deploy.sh native

# Stop all deployments
./scripts/deployment/deploy.sh stop
```

## Integration with Makefile

The master scripts are integrated with the Makefile for convenience:

```bash
# Uses test master
make windmill-validate-native

# Uses deployment master
make dev
make dev-windmill
make dev-full
```

## Benefits

1. **DRY Principle**: Eliminates code duplication across 25+ scripts
2. **Unified Interface**: Consistent command-line interface
3. **Professional Output**: Color-coded, structured output with progress tracking
4. **Error Handling**: Comprehensive error tracking and reporting
5. **Flexibility**: Support for different modes and environments
6. **Maintainability**: Centralized logic reduces maintenance overhead

## Migration Strategy

### From Legacy Scripts

Instead of running individual scripts:

```bash
# OLD: Multiple individual scripts
./scripts/test-native-quick.sh
./scripts/test-native-system.sh
./scripts/test-100-percent-complete.sh

# NEW: Single unified command
./scripts/test/run-tests.sh full
```

### Makefile Integration

The Makefile has been updated to use master scripts where appropriate, maintaining backward compatibility while providing the benefits of the unified system.

## Environment Variables

All master scripts support environment variables for configuration:

- `BUILD_MODE`: dev/debug, release/prod, all
- `ENVIRONMENT`: development, production, windmill, full
- `REPLICAS`: Number of worker replicas
- `JOBS`: Number of parallel jobs

## Logging and Output

All master scripts provide:
- Color-coded output for easy reading
- Progress tracking with counters
- Detailed error reporting
- Summary reports at completion
- Structured logging with timestamps

This organization follows the same professional principles applied to the Makefile reorganization, ensuring consistency across the entire FlexCore project.