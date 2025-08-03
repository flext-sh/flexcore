# FlexCore Documentation

**Version**: 0.9.0 | **Status**: Development | **Last Updated**: 2025-08-01

This directory contains comprehensive documentation for FlexCore, the enterprise runtime container service that serves as the primary orchestration engine for the FLEXT data integration ecosystem.

> ⚠️ **Important**: FlexCore is currently under active development with critical architectural issues. Please review [TODO.md](TODO.md) before contributing or deploying.

## 📚 Documentation Structure

### 🏗️ Architecture & Design

- [**Architecture Overview**](architecture/overview.md) - System architecture, patterns, and design decisions
- [**TODO List**](TODO.md) - **Critical architectural issues and refactoring roadmap**
- [**Design Patterns**](architecture/patterns.md) - Clean Architecture, DDD, CQRS, Event Sourcing
- [**Plugin Architecture**](architecture/plugins.md) - Plugin system design and security model

### 🚀 Development

- [**Getting Started**](getting-started/installation.md) - Complete setup and installation guide
- [**Development Workflow**](development/workflow.md) - Development process and quality gates
- [**Plugin Development**](development/plugins.md) - Creating and testing plugins
- [**Testing Guide**](development/testing.md) - Testing strategies and coverage requirements

### 🔌 API & Integration

- [**API Reference**](api-reference.md) - Complete REST API documentation
- [**FLEXT Integration**](integration/flext-ecosystem.md) - Integration with FLEXT ecosystem
- [**Event Sourcing API**](integration/event-sourcing.md) - Event store and CQRS endpoints
- [**Plugin API**](integration/plugin-api.md) - Plugin development interfaces

### 🚀 Operations

- [**Deployment Guide**](operations/deployment.md) - Production deployment strategies
- [**Monitoring & Observability**](operations/monitoring.md) - Metrics, logging, and tracing
- [**Configuration**](operations/configuration.md) - Environment variables and settings
- [**Troubleshooting**](operations/troubleshooting.md) - Common issues and solutions

### 📊 Quality & Standards

- [**Code Quality Standards**](standards/code-quality.md) - Quality gates and requirements
- [**Architecture Standards**](standards/architecture.md) - Clean Architecture compliance
- [**Security Guidelines**](standards/security.md) - Security best practices
- [**Performance Requirements**](standards/performance.md) - Performance benchmarks and SLA

## 🎯 Quick Navigation

### For New Developers

1. **Start Here**: [Getting Started](getting-started/installation.md)
2. **Understand Issues**: [TODO.md](TODO.md) - **Critical architectural problems**
3. **Architecture**: [Architecture Overview](architecture/overview.md)
4. **Development**: [Development Workflow](development/workflow.md)

### For Contributors

1. **Current State**: [TODO.md](TODO.md) - **Must read before contributing**
2. **Architecture Compliance**: [Architecture Standards](standards/architecture.md)
3. **Quality Gates**: [Code Quality Standards](standards/code-quality.md)
4. **Testing**: [Testing Guide](development/testing.md)

### For Operators

1. **Deployment**: [Deployment Guide](operations/deployment.md)
2. **Configuration**: [Configuration](operations/configuration.md)
3. **Monitoring**: [Monitoring & Observability](operations/monitoring.md)
4. **Troubleshooting**: [Troubleshooting](operations/troubleshooting.md)

### For Integrators

1. **API Documentation**: [API Reference](api-reference.md)
2. **FLEXT Integration**: [FLEXT Ecosystem](integration/flext-ecosystem.md)
3. **Plugin Development**: [Plugin API](integration/plugin-api.md)
4. **Event Sourcing**: [Event Sourcing API](integration/event-sourcing.md)

## 📊 Current Project Status

### Architecture Compliance

- ❌ **Clean Architecture**: 30% - Critical boundary violations
- ❌ **DDD**: 40% - Anemic domain model
- ❌ **CQRS**: 25% - Multiple conflicting implementations
- ❌ **Event Sourcing**: 20% - Inadequate implementation

### Documentation Coverage

- ✅ **README**: Complete and up-to-date
- ✅ **TODO**: Comprehensive issue tracking
- 🟡 **Architecture**: Partially documented
- 🟡 **API Reference**: Basic coverage
- ❌ **Operations**: Minimal documentation
- ❌ **Development Guides**: Incomplete

## 🚨 Critical Issues

### Immediate Attention Required

1. **Clean Architecture Violations** - HTTP server in application layer
2. **CQRS Implementation Chaos** - 3 different implementations
3. **Inadequate Event Sourcing** - In-memory store, mutable events
4. **Plugin Security Gaps** - No isolation or resource management
5. **Missing Integration Tests** - Critical testing gaps

See [TODO.md](TODO.md) for detailed analysis and refactoring roadmap.

## 🛠️ Development Phases

### Phase 1: Critical Fixes (2-3 weeks)

- Fix Clean Architecture boundary violations
- Unify CQRS implementation
- Implement proper Event Sourcing with PostgreSQL
- Add comprehensive integration tests

### Phase 2: Domain Enhancement (3-4 weeks)

- Implement rich domain model
- Add domain services and proper aggregates
- Enhance plugin system security
- Improve error handling and observability

### Phase 3: Production Readiness (4-6 weeks)

- Performance optimizations
- Security hardening
- Complete documentation
- Comprehensive monitoring

## 🔗 External References

### FLEXT Ecosystem

- [flext-core](https://github.com/flext/flext-core) - Python foundation library
- [FLEXT Service](../cmd/flext/) - Python data processing service
- [Singer Ecosystem](https://github.com/flext/) - Data taps, targets, and transformations

### Architecture Resources

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) - Robert C. Martin
- [Domain-Driven Design](https://martinfowler.com/tags/domain%20driven%20design.html) - Martin Fowler
- [CQRS](https://martinfowler.com/bliki/CQRS.html) - Command Query Responsibility Segregation
- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) - Event-driven architecture

## 📝 Contributing to Documentation

### Documentation Standards

- **Language**: English only
- **Format**: Markdown with consistent structure
- **Code Examples**: Functional and tested
- **Updates**: Keep synchronized with code changes

### Update Process

1. Review current documentation state
2. Identify gaps or outdated content
3. Update following FLEXT documentation standards
4. Test all code examples
5. Submit PR with documentation updates

## ⚠️ Important Disclaimers

### Production Readiness

**FlexCore is NOT production-ready** due to critical architectural issues identified in [TODO.md](TODO.md). Use only for development and testing until architectural refactoring is complete.

### API Stability

API endpoints may change significantly during the refactoring process. Do not rely on current API contracts for production integrations.

### Architecture Evolution

The architecture is undergoing significant refactoring to properly implement Clean Architecture, DDD, CQRS, and Event Sourcing patterns. Expect breaking changes.

---

**For the most current information about FlexCore's status and critical issues, always refer to [TODO.md](TODO.md).**
