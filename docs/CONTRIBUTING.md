# Contributing to FlexCore

Thank you for your interest in contributing to FlexCore! This document provides guidelines and information for contributors.

## Development Environment

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- golangci-lint
- GNU Make

### Setup

```bash
# Clone the repository
git clone https://github.com/flext/flexcore.git
cd flexcore

# Install dependencies
make deps

# Run tests
make test

# Run linting
make lint
```

## Code Standards

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Pass all `golangci-lint` checks
- Maintain test coverage above 80%
- Document all public APIs

### Architecture Principles

- Follow Clean Architecture patterns
- Implement proper separation of concerns
- Use dependency injection for loose coupling
- Apply SOLID principles
- Prefer composition over inheritance

### Code Style

```go
// Good: Clear, descriptive naming
type PipelineService struct {
    repository PipelineRepository
    eventBus   EventBus
}

func (s *PipelineService) CreatePipeline(ctx context.Context, cmd CreatePipelineCommand) (*Pipeline, error) {
    // Implementation
}

// Bad: Unclear naming and responsibilities
type PS struct {
    r Repo
    e EB
}

func (s *PS) Create(c interface{}) interface{} {
    // Implementation
}
```

## Testing

### Test Categories

1. **Unit Tests** - Test individual functions/methods
2. **Integration Tests** - Test component interactions
3. **End-to-End Tests** - Test complete workflows
4. **Performance Tests** - Validate performance requirements

### Test Structure

```go
func TestPipelineService_CreatePipeline(t *testing.T) {
    // Arrange
    ctx := context.Background()
    service := setupPipelineService(t)
    
    // Act
    pipeline, err := service.CreatePipeline(ctx, CreatePipelineCommand{
        Name: "test-pipeline",
    })
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, pipeline)
    assert.Equal(t, "test-pipeline", pipeline.Name)
}
```

## Pull Request Process

### Before Submitting

1. Ensure all tests pass: `make test`
2. Run linting: `make lint`
3. Update documentation if needed
4. Add/update tests for new functionality

### PR Requirements

- **Clear Title**: Describe what the PR does
- **Description**: Explain the problem and solution
- **Tests**: Include appropriate test coverage
- **Documentation**: Update relevant docs
- **No Breaking Changes**: Unless specifically approved

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes
```

## Issue Guidelines

### Bug Reports

Include:
- Go version
- Operating system
- Minimal reproduction steps
- Expected vs actual behavior
- Relevant logs/errors

### Feature Requests

Include:
- Use case description
- Proposed solution
- Alternative solutions considered
- Impact assessment

## Architecture Decisions

### Adding New Features

1. Discuss architecture in GitHub issue
2. Create design document if complex
3. Get maintainer approval
4. Implement with proper tests
5. Update documentation

### Breaking Changes

- Require explicit approval
- Must include migration guide
- Need deprecation period when possible
- Require major version bump

## Documentation

### Code Documentation

- Document all public APIs
- Include usage examples
- Explain complex algorithms
- Document error conditions

### Architecture Documentation

- Update architecture diagrams
- Explain design decisions
- Include sequence diagrams for complex flows
- Maintain ADR (Architecture Decision Records)

## Release Process

1. Feature freeze
2. Release candidate testing
3. Documentation review
4. Performance validation
5. Security review
6. Final release

## Getting Help

- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: Architecture and design questions
- **Discord**: Real-time development chat

## Code of Conduct

We follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/0/code_of_conduct/) code of conduct.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.