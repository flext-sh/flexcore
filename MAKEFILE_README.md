# FlexCore Makefile Guide

## Overview

The FlexCore Makefile is a comprehensive build system that supports:

- **Native Windmill builds** (Rust backend + Go client)
- **Go application development**
- **Docker integration**
- **Quality assurance pipelines**
- **Production deployments**

## Quick Start

```bash
# Install dependencies
make deps

# Build everything
make windmill
make build

# Check status
make status

# Start development
make dev-windmill
```

## Core Commands

### Development

- `make build` - Build FlexCore application
- `make run` - Run application locally
- `make dev` - Start development environment
- `make test` - Run all tests
- `make format` - Format code
- `make lint` - Run linting

### Windmill (Native)

- `make windmill` - Build complete Windmill stack
- `make windmill-backend` - Build Rust backend only
- `make windmill-backend-fast` - Quick incremental build
- `make windmill-backend-release` - Optimized production build
- `make windmill-go` - Build Go client only
- `make windmill-test` - Test binary functionality
- `make dev-windmill` - Development with native Windmill

### Quality & CI

- `make check` - Run all quality checks
- `make security` - Security analysis
- `make audit` - Dependency audit
- `make ci` - Full CI pipeline
- `make windmill-validate-native` - Native system validation

### Production

- `make build-prod` - Production build
- `make build-all` - Multi-platform builds
- `make docker` - Build Docker image
- `make release` - Create production release

### Utilities

- `make status` - Show project status
- `make info` - Show build information
- `make version` - Show version only
- `make clean` - Clean build artifacts
- `make clean-all` - Clean everything

## Build Modes

Set `BUILD_MODE` environment variable:

- `BUILD_MODE=dev` (default) - Fast development builds
- `BUILD_MODE=release` - Optimized production builds

Examples:

```bash
# Development build (fast)
make windmill BUILD_MODE=dev

# Production build (optimized)
make windmill BUILD_MODE=release
```

## Performance Features

- **Parallel builds**: Uses all CPU cores
- **Incremental compilation**: Rust incremental builds enabled
- **Smart caching**: sccache for Rust, Go module cache
- **Fast targets**: `windmill-backend-fast` for quick iterations

## Native Builds

The Makefile builds Windmill components natively without Docker:

### Rust Backend

- Compiles Windmill backend from source
- SQLx offline mode for faster builds
- Debug/release profiles
- Typical build time: ~1.5 minutes (clean), ~0.6s (incremental)

### Go Client

- Generates OpenAPI client from Windmill spec
- ~75K lines of generated Go code
- Full type safety and API coverage

## Validation & Testing

- `make windmill-validate-native` - Comprehensive system validation
- `make windmill-test` - Binary functionality tests
- `make windmill-benchmark` - Performance benchmarks

## File Structure

```
Makefile                 # Main build system
.makerc                  # Configuration defaults
scripts/
├── validate-native.sh   # Native system validation
└── benchmark-native.sh  # Performance benchmarks
```

## Configuration

Create `.makerc` for project-specific settings:

```bash
BUILD_MODE=dev
CARGO_INCREMENTAL=1
RUST_LOG=info
```

## Troubleshooting

### Common Issues

1. **Rust build fails**: Ensure Rust is installed (`rustup.rs`)
2. **Go build fails**: Ensure Go 1.20+ is installed
3. **Long build times**: Use `make windmill-backend-fast` for development
4. **Missing dependencies**: Run `make deps`

### Performance Tips

- Use `windmill-backend-fast` for development
- Enable sccache: `cargo install sccache`
- Keep `BUILD_MODE=dev` for fast iterations
- Use `make clean` only when necessary

## Architecture

The Makefile follows professional software development practices:

- **Modular design** with clear separation of concerns
- **DRY principles** with no code duplication
- **Performance optimization** with parallel builds
- **Quality gates** integrated into CI pipeline
- **Native compilation** without Docker overhead

## Examples

### Full Development Workflow

```bash
# Setup
make deps

# Build and test
make windmill
make build
make test

# Quality checks
make check

# Start development
make dev-windmill
```

### Quick Iteration Cycle

```bash
# Fast development builds
make windmill-backend-fast
make build-dev
make run-dev
```

### Production Deployment

```bash
# Full production pipeline
make ci
make build-all
make docker-prod
make release
```

This Makefile provides a complete, professional build system optimized for both development speed and production quality.
