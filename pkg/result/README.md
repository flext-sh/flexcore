# Result Package

**Package**: `github.com/flext-sh/flexcore/pkg/result`  
**Version**: 0.9.0  
**Status**: Production Ready

## Overview

The Result package provides railway-oriented programming patterns for FlexCore and the broader FLEXT ecosystem, enabling explicit error handling and functional composition patterns without exceptions.

## Key Features

- **Type-safe error handling** with explicit success/failure states
- **Monadic operations** for functional composition (Map, FlatMap, Filter)
- **Railway-oriented programming** with automatic error propagation
- **Async operation support** with channels and goroutines
- **Panic recovery** with Try/TryAsync functions
- **Result combination** and tuple operations

## Core Types

### Result[T]

The main Result type representing either success with a value or failure with an error.

```go
// Create results
success := result.Success("Hello, World!")
failure := result.Failure[string](errors.New("operation failed"))

// Check state
if success.IsSuccess() {
    value := success.Value()
    // Process successful value
}
```

### Async[T]

Asynchronous Result delivery using channels for non-blocking operations.

```go
async := result.NewAsync[string]()

go func() {
    // Background work
    time.Sleep(1 * time.Second)
    async.Complete("Background work completed")
}()

result := async.Await() // Blocks until completion
```

### Tuple[T, U]

Pair of values for combining multiple Results.

```go
combined := result.Combine(userResult, profileResult)
combined.ForEach(func(tuple result.Tuple[User, Profile]) {
    user := tuple.First
    profile := tuple.Second
    // Process both values
})
```

## Functional Operations

### Map - Transform Values

Transform successful values while preserving error propagation.

```go
result := result.Success(42)
    .Map(func(n int) string {
        return fmt.Sprintf("Number: %d", n)
    })
// Result contains "Number: 42"
```

### FlatMap - Chain Operations

Chain operations that can fail, flattening nested Results.

```go
result := result.Success("42")
    .FlatMap(func(s string) result.Result[int] {
        if num, err := strconv.Atoi(s); err != nil {
            return result.Failure[int](err)
        } else {
            return result.Success(num)
        }
    })
```

### Filter - Validate Values

Apply validation predicates to successful values.

```go
result := result.Success(42)
    .Filter(func(n int) bool { return n > 0 })
    .Map(func(n int) string { return fmt.Sprintf("Positive: %d", n) })
```

## Error Handling Patterns

### Safe Value Access

```go
// With default fallback
value := result.ValueOr("default")

// With zero value fallback
value := result.ValueOrZero()

// With explicit checking
if result.IsSuccess() {
    value := result.Value()
    // Safe to use value
}
```

### Go-style Error Handling

```go
value, err := result.Unwrap()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
return processValue(value)
```

### Panic-based Patterns (Use with Caution)

```go
// Will panic if Result is failed
value := result.Get()

// Explicit panic behavior
value := result.UnwrapOrPanic()
```

## Async Operations

### Background Processing

```go
async := result.NewAsync[ProcessedData]()

go func() {
    data, err := expensiveOperation()
    if err != nil {
        async.Fail(err)
    } else {
        async.Complete(data)
    }
}()

// Later, when result is needed
result := async.Await()
```

### Safe Async Execution

```go
// Execute function that might panic in background
async := result.TryAsync(func() Data {
    return riskyOperation() // Might panic
})

result := async.Await() // Returns Success or Failure
```

## Integration with FLEXT Ecosystem

The Result package serves as the foundation for error handling across FlexCore:

- **Domain Layer**: All business operations return Results
- **Application Layer**: Command and query handlers use Results
- **Infrastructure Layer**: Repository and external service calls
- **Plugin System**: Plugin execution results and error handling

## Best Practices

1. **Always handle errors explicitly** - avoid ignoring Result state
2. **Use method chaining** for functional composition
3. **Prefer ValueOr() over Get()** for safer value access
4. **Chain operations** to avoid nested error checking
5. **Use ForEach/IfFailure** for side effects without breaking chains

## Performance Considerations

- **Zero allocation** for success cases in most operations
- **Minimal allocation** for error cases and transformations
- **Efficient composition** without intermediate allocations
- **Thread-safe** immutable Result instances

## Thread Safety

Result instances are immutable and safe for concurrent read access. Async operations use proper goroutine synchronization with channels.

## Architecture Integration

- **Clean Architecture**: Results flow through all layers without exceptions
- **Railway-Oriented Programming**: Automatic error propagation in operation chains
- **Functional Programming**: Monadic operations for composable transformations
- **Event Sourcing**: Results used in event handler return values
- **CQRS**: Command and query results use Result pattern consistently

---

**See Also**: [FlexCore Architecture](../../docs/architecture/overview.md) | [Domain Layer](../../internal/domain/README.md) | [Error Handling Guide](../../docs/development/error-handling.md)
