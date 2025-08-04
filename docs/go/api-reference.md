# Go API Reference

This document provides comprehensive API documentation for FlexCore's Go components. FlexCore's Go layer serves as the high-performance core engine, handling command processing, event sourcing, and infrastructure concerns.

## 📦 Package Structure

```
github.com/flext-sh/flexcore/
├── internal/
│   ├── domain/           # Domain layer - business logic
│   ├── app/             # Application layer - use cases
│   ├── adapters/        # Interface adapters
│   └── infrastructure/  # Infrastructure layer
├── pkg/                 # Public APIs
│   ├── config/         # Configuration management
│   ├── logging/        # Structured logging
│   ├── patterns/       # Common patterns
│   └── result/         # Result types
└── cmd/                # CLI applications
```

## 🏗️ Core Interfaces

### Command Bus

The Command Bus routes and executes commands that modify system state.

```go
package app

import (
    "context"
)

// CommandBus handles command routing and execution
type CommandBus interface {
    // Send executes a command and returns any error
    Send(ctx context.Context, cmd Command) error

    // RegisterHandler registers a command handler for a specific command type
    RegisterHandler(cmdType string, handler CommandHandler)

    // RegisterMiddleware adds middleware to the command processing pipeline
    RegisterMiddleware(middleware CommandMiddleware)
}

// Command represents a command to be executed
type Command interface {
    // CommandType returns the unique identifier for this command type
    CommandType() string

    // Validate performs command validation
    Validate() error
}

// CommandHandler processes commands
type CommandHandler interface {
    // Handle processes the command and returns any error
    Handle(ctx context.Context, cmd Command) error
}

// CommandMiddleware wraps command execution with cross-cutting concerns
type CommandMiddleware func(next CommandHandler) CommandHandler
```

#### Usage Example

```go
// Define a command
type CreateUserCommand struct {
    Email   string `json:"email" validate:"required,email"`
    Name    string `json:"name" validate:"required,min=2,max=100"`
    Profile UserProfile `json:"profile"`
}

func (c CreateUserCommand) CommandType() string {
    return "create_user"
}

func (c CreateUserCommand) Validate() error {
    return validator.New().Struct(c)
}

// Implement a command handler
type CreateUserHandler struct {
    userRepo UserRepository
    eventBus EventBus
    logger   Logger
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd Command) error {
    createCmd := cmd.(CreateUserCommand)

    // Check if user already exists
    exists, err := h.userRepo.ExistsByEmail(ctx, createCmd.Email)
    if err != nil {
        return fmt.Errorf("checking user existence: %w", err)
    }
    if exists {
        return ErrUserAlreadyExists
    }

    // Create new user aggregate
    user := NewUser(
        UserID(uuid.New().String()),
        Email(createCmd.Email),
        createCmd.Profile,
    )

    // Save to repository
    if err := h.userRepo.Save(ctx, user); err != nil {
        return fmt.Errorf("saving user: %w", err)
    }

    // Publish domain events
    for _, event := range user.UncommittedEvents() {
        if err := h.eventBus.Publish(ctx, event); err != nil {
            h.logger.Error("failed to publish event",
                "error", err,
                "event_type", event.EventType(),
                "aggregate_id", event.AggregateID(),
            )
        }
    }

    user.MarkEventsAsCommitted()
    return nil
}

// Register the handler
func SetupCommandBus() CommandBus {
    bus := NewCommandBus()
    bus.RegisterHandler("create_user", &CreateUserHandler{
        userRepo: userRepository,
        eventBus: eventBus,
        logger:   logger,
    })
    return bus
}
```

### Query Bus

The Query Bus handles read operations with optimized data access patterns.

```go
package app

// QueryBus handles query routing and execution
type QueryBus interface {
    // Ask executes a query and returns the result
    Ask(ctx context.Context, query Query) (interface{}, error)

    // RegisterHandler registers a query handler for a specific query type
    RegisterHandler(queryType string, handler QueryHandler)

    // RegisterMiddleware adds middleware to the query processing pipeline
    RegisterMiddleware(middleware QueryMiddleware)
}

// Query represents a query for reading data
type Query interface {
    // QueryType returns the unique identifier for this query type
    QueryType() string

    // Validate performs query validation
    Validate() error
}

// QueryHandler processes queries
type QueryHandler interface {
    // Handle processes the query and returns the result
    Handle(ctx context.Context, query Query) (interface{}, error)
}

// QueryMiddleware wraps query execution
type QueryMiddleware func(next QueryHandler) QueryHandler

// PagedResult represents paginated query results
type PagedResult struct {
    Data       interface{} `json:"data"`
    TotalCount int64       `json:"total_count"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    HasNext    bool        `json:"has_next"`
}
```

#### Usage Example

```go
// Define a query
type GetUserQuery struct {
    UserID string `json:"user_id" validate:"required,uuid"`
}

func (q GetUserQuery) QueryType() string {
    return "get_user"
}

func (q GetUserQuery) Validate() error {
    return validator.New().Struct(q)
}

// Query response DTO
type UserResponse struct {
    ID       string      `json:"id"`
    Email    string      `json:"email"`
    Name     string      `json:"name"`
    Profile  UserProfile `json:"profile"`
    Created  time.Time   `json:"created_at"`
    Updated  time.Time   `json:"updated_at"`
}

// Implement query handler
type GetUserHandler struct {
    userRepo UserRepository
    cache    Cache
    logger   Logger
}

func (h *GetUserHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
    getUserQuery := query.(GetUserQuery)

    // Check cache first
    cacheKey := fmt.Sprintf("user:%s", getUserQuery.UserID)
    if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
        var user UserResponse
        if err := json.Unmarshal(cached, &user); err == nil {
            return user, nil
        }
    }

    // Fetch from repository
    user, err := h.userRepo.FindByID(ctx, UserID(getUserQuery.UserID))
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("fetching user: %w", err)
    }

    // Convert to response DTO
    response := UserResponse{
        ID:      user.ID().String(),
        Email:   user.Email().String(),
        Name:    user.Name(),
        Profile: user.Profile(),
        Created: user.CreatedAt(),
        Updated: user.UpdatedAt(),
    }

    // Cache the result
    if data, err := json.Marshal(response); err == nil {
        h.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    }

    return response, nil
}
```

### Event Store

The Event Store provides persistent storage for domain events with full audit trail.

```go
package infrastructure

// EventStore provides persistent event storage
type EventStore interface {
    // AppendEvents adds events to a stream
    AppendEvents(ctx context.Context, streamID string, expectedVersion int, events []Event) error

    // ReadEvents reads events from a stream
    ReadEvents(ctx context.Context, streamID string, fromVersion int) ([]Event, error)

    // ReadAllEvents reads all events from all streams
    ReadAllEvents(ctx context.Context, fromPosition int64) ([]Event, error)

    // CreateSnapshot stores an aggregate snapshot
    CreateSnapshot(ctx context.Context, snapshot Snapshot) error

    // GetSnapshot retrieves the latest snapshot for an aggregate
    GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error)
}

// Event represents a domain event
type Event interface {
    // EventID returns the unique event identifier
    EventID() string

    // EventType returns the event type name
    EventType() string

    // AggregateID returns the ID of the aggregate that generated this event
    AggregateID() string

    // AggregateVersion returns the version of the aggregate when this event was generated
    AggregateVersion() int

    // EventData returns the event data as bytes
    EventData() []byte

    // Metadata returns event metadata
    Metadata() map[string]interface{}

    // OccurredAt returns when the event occurred
    OccurredAt() time.Time
}

// Snapshot represents an aggregate snapshot
type Snapshot struct {
    AggregateID      string                 `json:"aggregate_id"`
    AggregateType    string                 `json:"aggregate_type"`
    AggregateVersion int                    `json:"aggregate_version"`
    SnapshotData     []byte                 `json:"snapshot_data"`
    Metadata         map[string]interface{} `json:"metadata"`
    CreatedAt        time.Time              `json:"created_at"`
}
```

#### Usage Example

```go
// Domain event implementation
type UserCreatedEvent struct {
    eventID          string    `json:"event_id"`
    aggregateID      string    `json:"aggregate_id"`
    aggregateVersion int       `json:"aggregate_version"`
    occurredAt       time.Time `json:"occurred_at"`

    // Event-specific data
    Email     string      `json:"email"`
    Name      string      `json:"name"`
    Profile   UserProfile `json:"profile"`
}

func (e UserCreatedEvent) EventID() string          { return e.eventID }
func (e UserCreatedEvent) EventType() string        { return "user.created" }
func (e UserCreatedEvent) AggregateID() string      { return e.aggregateID }
func (e UserCreatedEvent) AggregateVersion() int    { return e.aggregateVersion }
func (e UserCreatedEvent) OccurredAt() time.Time    { return e.occurredAt }

func (e UserCreatedEvent) EventData() []byte {
    data, _ := json.Marshal(struct {
        Email   string      `json:"email"`
        Name    string      `json:"name"`
        Profile UserProfile `json:"profile"`
    }{
        Email:   e.Email,
        Name:    e.Name,
        Profile: e.Profile,
    })
    return data
}

func (e UserCreatedEvent) Metadata() map[string]interface{} {
    return map[string]interface{}{
        "event_version": "1.0",
        "content_type": "application/json",
    }
}

// Using the event store
func SaveUserAggregate(ctx context.Context, eventStore EventStore, user *User) error {
    events := user.UncommittedEvents()
    if len(events) == 0 {
        return nil // No changes to save
    }

    streamID := fmt.Sprintf("user-%s", user.ID())
    expectedVersion := user.Version() - len(events)

    return eventStore.AppendEvents(ctx, streamID, expectedVersion, events)
}

func LoadUserAggregate(ctx context.Context, eventStore EventStore, userID string) (*User, error) {
    streamID := fmt.Sprintf("user-%s", userID)

    // Try to load from snapshot first
    snapshot, err := eventStore.GetSnapshot(ctx, userID)
    var user *User
    var fromVersion int

    if err == nil && snapshot != nil {
        // Deserialize from snapshot
        user = &User{}
        if err := json.Unmarshal(snapshot.SnapshotData, user); err != nil {
            return nil, fmt.Errorf("deserializing snapshot: %w", err)
        }
        fromVersion = snapshot.AggregateVersion + 1
    } else {
        // Create new aggregate
        user = NewEmptyUser(UserID(userID))
        fromVersion = 0
    }

    // Load events since snapshot
    events, err := eventStore.ReadEvents(ctx, streamID, fromVersion)
    if err != nil {
        return nil, fmt.Errorf("reading events: %w", err)
    }

    // Apply events to rebuild state
    for _, event := range events {
        if err := user.ApplyEvent(event); err != nil {
            return nil, fmt.Errorf("applying event %s: %w", event.EventID(), err)
        }
    }

    return user, nil
}
```

### Plugin System

The Plugin System enables hot-swappable functionality with dynamic loading.

```go
package infrastructure

// Plugin represents a loadable plugin
type Plugin interface {
    // Name returns the plugin name
    Name() string

    // Version returns the plugin version
    Version() string

    // Initialize initializes the plugin with configuration
    Initialize(config Config) error

    // Process processes data through the plugin
    Process(ctx context.Context, data interface{}) (interface{}, error)

    // Shutdown gracefully shuts down the plugin
    Shutdown() error

    // HealthCheck returns the plugin health status
    HealthCheck() error
}

// PluginManager manages plugin lifecycle
type PluginManager interface {
    // LoadPlugin loads a plugin from a file
    LoadPlugin(pluginPath string) (Plugin, error)

    // RegisterPlugin registers a plugin instance
    RegisterPlugin(name string, plugin Plugin) error

    // GetPlugin retrieves a registered plugin
    GetPlugin(name string) (Plugin, bool)

    // ListPlugins returns all registered plugins
    ListPlugins() []PluginInfo

    // UnloadPlugin unloads a plugin
    UnloadPlugin(name string) error

    // ReloadPlugin reloads a plugin
    ReloadPlugin(name string) error
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Status      PluginStatus          `json:"status"`
    LoadedAt    time.Time             `json:"loaded_at"`
    Config      map[string]interface{} `json:"config"`
    HealthCheck HealthStatus          `json:"health_check"`
}

type PluginStatus string

const (
    PluginStatusLoaded    PluginStatus = "loaded"
    PluginStatusUnloaded  PluginStatus = "unloaded"
    PluginStatusError     PluginStatus = "error"
)

type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
    HealthStatusUnknown   HealthStatus = "unknown"
)
```

#### Usage Example

```go
// Example plugin implementation
type DataProcessorPlugin struct {
    name    string
    version string
    config  Config
    logger  Logger
}

func (p *DataProcessorPlugin) Name() string {
    return p.name
}

func (p *DataProcessorPlugin) Version() string {
    return p.version
}

func (p *DataProcessorPlugin) Initialize(config Config) error {
    p.config = config
    p.logger = config.Logger

    // Initialize plugin resources
    p.logger.Info("Data processor plugin initialized",
        "plugin", p.name,
        "version", p.version,
    )

    return nil
}

func (p *DataProcessorPlugin) Process(ctx context.Context, data interface{}) (interface{}, error) {
    // Type assertion to expected input type
    input, ok := data.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("expected map[string]interface{}, got %T", data)
    }

    // Process the data
    result := make(map[string]interface{})
    for key, value := range input {
        // Transform the data based on plugin logic
        processed := p.transformValue(value)
        result[fmt.Sprintf("processed_%s", key)] = processed
    }

    return result, nil
}

func (p *DataProcessorPlugin) transformValue(value interface{}) interface{} {
    // Plugin-specific transformation logic
    switch v := value.(type) {
    case string:
        return strings.ToUpper(v)
    case int:
        return v * 2
    case float64:
        return v * 1.5
    default:
        return value
    }
}

func (p *DataProcessorPlugin) Shutdown() error {
    p.logger.Info("Data processor plugin shutting down", "plugin", p.name)
    // Cleanup resources
    return nil
}

func (p *DataProcessorPlugin) HealthCheck() error {
    // Perform health checks
    return nil
}

// Plugin factory function (required for dynamic loading)
func NewDataProcessorPlugin() Plugin {
    return &DataProcessorPlugin{
        name:    "data-processor",
        version: "1.0.0",
    }
}

// Using the plugin manager
func SetupPlugins(manager PluginManager) error {
    // Load plugin from file
    plugin, err := manager.LoadPlugin("./plugins/data-processor.so")
    if err != nil {
        return fmt.Errorf("loading plugin: %w", err)
    }

    // Initialize plugin
    config := Config{
        Logger: logger,
        Settings: map[string]interface{}{
            "max_processing_time": "30s",
            "batch_size":         100,
        },
    }

    if err := plugin.Initialize(config); err != nil {
        return fmt.Errorf("initializing plugin: %w", err)
    }

    // Register for use
    return manager.RegisterPlugin(plugin.Name(), plugin)
}
```

## 🔧 Configuration Management

### Config Interface

```go
package config

// Config provides application configuration
type Config interface {
    // GetString returns a string configuration value
    GetString(key string) string

    // GetInt returns an integer configuration value
    GetInt(key string) int

    // GetBool returns a boolean configuration value
    GetBool(key string) bool

    // GetDuration returns a duration configuration value
    GetDuration(key string) time.Duration

    // GetStringSlice returns a string slice configuration value
    GetStringSlice(key string) []string

    // IsSet checks if a configuration key is set
    IsSet(key string) bool

    // Sub returns a sub-configuration for a key
    Sub(key string) Config

    // UnmarshalKey unmarshals a configuration key into a struct
    UnmarshalKey(key string, dest interface{}) error

    // WatchConfig watches for configuration changes
    WatchConfig(callback func(Config))
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    Database     string        `mapstructure:"database"`
    Username     string        `mapstructure:"username"`
    Password     string        `mapstructure:"password"`
    SSLMode      string        `mapstructure:"ssl_mode"`
    MaxOpenConns int           `mapstructure:"max_open_conns"`
    MaxIdleConns int           `mapstructure:"max_idle_conns"`
    ConnTimeout  time.Duration `mapstructure:"conn_timeout"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    Password     string        `mapstructure:"password"`
    Database     int           `mapstructure:"database"`
    PoolSize     int           `mapstructure:"pool_size"`
    DialTimeout  time.Duration `mapstructure:"dial_timeout"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
}
```

## 📊 Result Types

### Result Pattern Implementation

```go
package result

// Result represents the outcome of an operation
type Result[T any] struct {
    value T
    err   error
}

// Ok creates a successful result
func Ok[T any](value T) Result[T] {
    return Result[T]{value: value}
}

// Err creates an error result
func Err[T any](err error) Result[T] {
    return Result[T]{err: err}
}

// IsOk returns true if the result is successful
func (r Result[T]) IsOk() bool {
    return r.err == nil
}

// IsErr returns true if the result is an error
func (r Result[T]) IsErr() bool {
    return r.err != nil
}

// Unwrap returns the value or panics if error
func (r Result[T]) Unwrap() T {
    if r.err != nil {
        panic(fmt.Sprintf("called Unwrap on error result: %v", r.err))
    }
    return r.value
}

// UnwrapOr returns the value or the provided default
func (r Result[T]) UnwrapOr(defaultValue T) T {
    if r.err != nil {
        return defaultValue
    }
    return r.value
}

// Map transforms the value if result is Ok
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
    if r.err != nil {
        return Err[U](r.err)
    }
    return Ok(f(r.value))
}

// FlatMap transforms the value and flattens nested results
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
    if r.err != nil {
        return Err[U](r.err)
    }
    return f(r.value)
}
```

## 🔍 Testing Utilities

### Test Helpers

```go
package testing

// TestEventStore provides an in-memory event store for testing
type TestEventStore struct {
    events    map[string][]Event
    snapshots map[string]*Snapshot
    mutex     sync.RWMutex
}

func NewTestEventStore() *TestEventStore {
    return &TestEventStore{
        events:    make(map[string][]Event),
        snapshots: make(map[string]*Snapshot),
    }
}

func (s *TestEventStore) AppendEvents(ctx context.Context, streamID string, expectedVersion int, events []Event) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    currentEvents := s.events[streamID]
    if len(currentEvents) != expectedVersion {
        return fmt.Errorf("concurrency conflict: expected version %d, got %d",
            expectedVersion, len(currentEvents))
    }

    s.events[streamID] = append(currentEvents, events...)
    return nil
}

// MockRepository provides a mock repository for testing
type MockRepository[T any] struct {
    entities map[string]T
    mutex    sync.RWMutex
}

func NewMockRepository[T any]() *MockRepository[T] {
    return &MockRepository[T]{
        entities: make(map[string]T),
    }
}

func (r *MockRepository[T]) Save(ctx context.Context, id string, entity T) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    r.entities[id] = entity
    return nil
}

func (r *MockRepository[T]) FindByID(ctx context.Context, id string) (T, error) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()

    entity, exists := r.entities[id]
    if !exists {
        var zero T
        return zero, fmt.Errorf("entity not found: %s", id)
    }
    return entity, nil
}
```

---

This API reference provides the core interfaces and patterns for building applications with FlexCore's Go layer. For Python integration, see the [Python API Reference](../python/api-reference.md). For more architectural details, refer to the [Architecture Overview](../architecture/overview.md).
