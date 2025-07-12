# Advanced Go Patterns for FlexCore

## 1. HashiCorp go-plugin Pattern Implementation

### Overview

The HashiCorp go-plugin pattern provides a robust plugin system using RPC over local sockets. This ensures plugin isolation and allows plugins to be written in any language.

### Implementation for FlexCore

```go
// pkg/plugin/plugin.go
package plugin

import (
    "github.com/hashicorp/go-plugin"
    "net/rpc"
)

// PluginMap is the map of plugins we support
var PluginMap = map[string]plugin.Plugin{
    "adapter": &AdapterPlugin{},
}

// Handshake is the shared handshake config for all plugins
var Handshake = plugin.HandshakeConfig{
    ProtocolVersion:  1,
    MagicCookieKey:   "FLEXCORE_PLUGIN",
    MagicCookieValue: "flexcore-adapter-v1",
}

// Adapter is the interface that all adapters must implement
type Adapter interface {
    Name() string
    Version() string
    Configure(config map[string]interface{}) error
    Extract(ctx context.Context, request ExtractRequest) (*ExtractResponse, error)
    Load(ctx context.Context, request LoadRequest) (*LoadResponse, error)
    Transform(ctx context.Context, request TransformRequest) (*TransformResponse, error)
}

// AdapterPlugin implements plugin.Plugin
type AdapterPlugin struct {
    Impl Adapter
}

func (p *AdapterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
    return &AdapterRPCServer{Impl: p.Impl}, nil
}

func (p *AdapterPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
    return &AdapterRPCClient{client: c}, nil
}
```

### Plugin Discovery and Loading

```go
// pkg/plugin/manager.go
package plugin

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/hashicorp/go-plugin"
)

type PluginManager struct {
    pluginDir string
    plugins   map[string]*LoadedPlugin
}

type LoadedPlugin struct {
    Path     string
    Client   *plugin.Client
    Adapter  Adapter
    Metadata PluginMetadata
}

type PluginMetadata struct {
    Name        string
    Version     string
    Author      string
    Description string
    Capabilities []string
}

func NewPluginManager(pluginDir string) *PluginManager {
    return &PluginManager{
        pluginDir: pluginDir,
        plugins:   make(map[string]*LoadedPlugin),
    }
}

func (pm *PluginManager) DiscoverPlugins() error {
    pattern := filepath.Join(pm.pluginDir, "flexcore-adapter-*")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return fmt.Errorf("failed to glob plugins: %w", err)
    }

    for _, path := range matches {
        if err := pm.loadPlugin(path); err != nil {
            // Log error but continue loading other plugins
            fmt.Printf("Failed to load plugin %s: %v\n", path, err)
        }
    }

    return nil
}

func (pm *PluginManager) loadPlugin(path string) error {
    client := plugin.NewClient(&plugin.ClientConfig{
        HandshakeConfig: Handshake,
        Plugins:         PluginMap,
        Cmd:             exec.Command(path),
        AllowedProtocols: []plugin.Protocol{
            plugin.ProtocolNetRPC,
            plugin.ProtocolGRPC,
        },
    })

    rpcClient, err := client.Client()
    if err != nil {
        return fmt.Errorf("failed to create RPC client: %w", err)
    }

    raw, err := rpcClient.Dispense("adapter")
    if err != nil {
        return fmt.Errorf("failed to dispense adapter: %w", err)
    }

    adapter := raw.(Adapter)

    // Get plugin metadata
    name := adapter.Name()
    version := adapter.Version()

    pm.plugins[name] = &LoadedPlugin{
        Path:    path,
        Client:  client,
        Adapter: adapter,
        Metadata: PluginMetadata{
            Name:    name,
            Version: version,
        },
    }

    return nil
}

// Version Negotiation
func (pm *PluginManager) CheckCompatibility(pluginName string, requiredVersion string) error {
    plugin, exists := pm.plugins[pluginName]
    if !exists {
        return fmt.Errorf("plugin %s not found", pluginName)
    }

    if !isVersionCompatible(plugin.Metadata.Version, requiredVersion) {
        return fmt.Errorf("plugin %s version %s is not compatible with required version %s",
            pluginName, plugin.Metadata.Version, requiredVersion)
    }

    return nil
}
```

## 2. Viper Advanced Configuration

### Implementation with Hot Reloading and Validation

```go
// pkg/config/config.go
package config

import (
    "fmt"
    "strings"
    "sync"

    "github.com/fsnotify/fsnotify"
    "github.com/go-playground/validator/v10"
    "github.com/spf13/viper"
)

type ConfigManager struct {
    viper      *viper.Viper
    validator  *validator.Validate
    mu         sync.RWMutex
    onChange   []func(*Config)
    config     *Config
}

// Config represents the complete application configuration
type Config struct {
    App      AppConfig      `mapstructure:"app" validate:"required"`
    Database DatabaseConfig `mapstructure:"database" validate:"required"`
    Adapters AdaptersConfig `mapstructure:"adapters"`
    Features FeaturesConfig `mapstructure:"features"`
}

type AppConfig struct {
    Name        string `mapstructure:"name" validate:"required"`
    Version     string `mapstructure:"version" validate:"required,semver"`
    Environment string `mapstructure:"environment" validate:"required,oneof=dev staging prod"`
    LogLevel    string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

func NewConfigManager() *ConfigManager {
    cm := &ConfigManager{
        viper:     viper.New(),
        validator: validator.New(),
        onChange:  make([]func(*Config), 0),
    }

    // Set up configuration sources
    cm.setupViper()

    return cm
}

func (cm *ConfigManager) setupViper() {
    v := cm.viper

    // Configuration file settings
    v.SetConfigName("flexcore")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.AddConfigPath("./config")
    v.AddConfigPath("/etc/flexcore")

    // Environment variables
    v.SetEnvPrefix("FLEXCORE")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // Defaults
    v.SetDefault("app.environment", "dev")
    v.SetDefault("app.log_level", "info")
    v.SetDefault("database.max_idle_conns", 10)
    v.SetDefault("database.max_open_conns", 100)
}

// Load loads configuration from all sources
func (cm *ConfigManager) Load() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    // Try to read config file
    if err := cm.viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("failed to read config file: %w", err)
        }
        // Config file not found is OK, we'll use defaults and env vars
    }

    // Unmarshal into struct
    var config Config
    if err := cm.viper.Unmarshal(&config); err != nil {
        return fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // Validate configuration
    if err := cm.validator.Struct(&config); err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }

    cm.config = &config
    return nil
}

// EnableHotReload enables configuration hot reloading
func (cm *ConfigManager) EnableHotReload() error {
    cm.viper.WatchConfig()
    cm.viper.OnConfigChange(func(e fsnotify.Event) {
        fmt.Printf("Config file changed: %s\n", e.Name)

        if err := cm.Load(); err != nil {
            fmt.Printf("Failed to reload config: %v\n", err)
            return
        }

        // Notify observers
        cm.mu.RLock()
        config := cm.config
        observers := cm.onChange
        cm.mu.RUnlock()

        for _, observer := range observers {
            observer(config)
        }
    })

    return nil
}

// OnChange registers a callback for configuration changes
func (cm *ConfigManager) OnChange(fn func(*Config)) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    cm.onChange = append(cm.onChange, fn)
}

// Get returns the current configuration
func (cm *ConfigManager) Get() *Config {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    return cm.config
}

// Type-safe accessors
func (cm *ConfigManager) GetString(key string) string {
    return cm.viper.GetString(key)
}

func (cm *ConfigManager) GetInt(key string) int {
    return cm.viper.GetInt(key)
}

func (cm *ConfigManager) GetBool(key string) bool {
    return cm.viper.GetBool(key)
}

// Remote configuration support
func (cm *ConfigManager) EnableRemoteConfig(provider, endpoint, path string) error {
    switch provider {
    case "consul":
        return cm.viper.AddRemoteProvider("consul", endpoint, path)
    case "etcd":
        return cm.viper.AddRemoteProvider("etcd", endpoint, path)
    default:
        return fmt.Errorf("unsupported remote provider: %s", provider)
    }
}
```

## 3. Advanced Handler Patterns

### Middleware Chain Implementation

```go
// pkg/http/middleware/chain.go
package middleware

import (
    "context"
    "net/http"
    "time"
)

type Middleware func(http.Handler) http.Handler

// Chain creates a middleware chain
type Chain struct {
    middlewares []Middleware
}

func NewChain(middlewares ...Middleware) *Chain {
    return &Chain{middlewares: middlewares}
}

func (c *Chain) Then(h http.Handler) http.Handler {
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        h = c.middlewares[i](h)
    }
    return h
}

func (c *Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
    return c.Then(fn)
}

// Common middleware implementations

// RequestID adds a unique request ID to the context
func RequestID() Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            id := r.Header.Get("X-Request-ID")
            if id == "" {
                id = generateRequestID()
            }

            ctx := context.WithValue(r.Context(), "request-id", id)
            w.Header().Set("X-Request-ID", id)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Timeout adds a timeout to the request
func Timeout(duration time.Duration) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), duration)
            defer cancel()

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Recovery recovers from panics
func Recovery() Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    // Log the error
                    requestID := r.Context().Value("request-id")
                    fmt.Printf("Panic recovered in request %v: %v\n", requestID, err)

                    // Return 500
                    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                }
            }()

            next.ServeHTTP(w, r)
        })
    }
}
```

### Request/Response Transformers

```go
// pkg/http/transform/transformer.go
package transform

import (
    "encoding/json"
    "net/http"
    "github.com/flext/flexcore/shared/result"
)

type RequestTransformer[T any] interface {
    Transform(r *http.Request) result.Result[T]
}

type ResponseTransformer[T any] interface {
    Transform(w http.ResponseWriter, data T) error
}

// JSON request transformer
type JSONRequestTransformer[T any] struct {
    validator func(T) error
}

func NewJSONRequestTransformer[T any](validator func(T) error) *JSONRequestTransformer[T] {
    return &JSONRequestTransformer[T]{validator: validator}
}

func (t *JSONRequestTransformer[T]) Transform(r *http.Request) result.Result[T] {
    var data T

    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        return result.Failure[T](fmt.Errorf("invalid JSON: %w", err))
    }

    if t.validator != nil {
        if err := t.validator(data); err != nil {
            return result.Failure[T](fmt.Errorf("validation failed: %w", err))
        }
    }

    return result.Success(data)
}

// JSON response transformer
type JSONResponseTransformer[T any] struct {
    pretty bool
}

func NewJSONResponseTransformer[T any](pretty bool) *JSONResponseTransformer[T] {
    return &JSONResponseTransformer[T]{pretty: pretty}
}

func (t *JSONResponseTransformer[T]) Transform(w http.ResponseWriter, data T) error {
    w.Header().Set("Content-Type", "application/json")

    encoder := json.NewEncoder(w)
    if t.pretty {
        encoder.SetIndent("", "  ")
    }

    return encoder.Encode(data)
}

// Handler with transformers
type TransformHandler[TReq, TRes any] struct {
    reqTransformer RequestTransformer[TReq]
    resTransformer ResponseTransformer[TRes]
    handler        func(context.Context, TReq) result.Result[TRes]
}

func (h *TransformHandler[TReq, TRes]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Transform request
    reqResult := h.reqTransformer.Transform(r)
    if reqResult.IsFailure() {
        http.Error(w, reqResult.Error().Error(), http.StatusBadRequest)
        return
    }

    // Execute handler
    resResult := h.handler(r.Context(), reqResult.Value())
    if resResult.IsFailure() {
        http.Error(w, resResult.Error().Error(), http.StatusInternalServerError)
        return
    }

    // Transform response
    if err := h.resTransformer.Transform(w, resResult.Value()); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

### OpenTelemetry Integration

```go
// pkg/observability/tracing.go
package observability

import (
    "context"
    "net/http"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TracingMiddleware adds OpenTelemetry tracing to HTTP handlers
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return otelhttp.NewHandler(next, serviceName,
            otelhttp.WithTracerProvider(otel.GetTracerProvider()),
            otelhttp.WithPropagators(otel.GetTextMapPropagator()),
        )
    }
}

// TraceMethod traces a method execution
func TraceMethod(ctx context.Context, name string, fn func(context.Context) error) error {
    tracer := otel.Tracer("flexcore")
    ctx, span := tracer.Start(ctx, name)
    defer span.End()

    err := fn(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.Status{Code: trace.StatusCodeError})
    }

    return err
}

// TraceAsync traces an async operation
func TraceAsync[T any](ctx context.Context, name string, fn func(context.Context) result.Result[T]) result.Result[T] {
    tracer := otel.Tracer("flexcore")
    ctx, span := tracer.Start(ctx, name)
    defer span.End()

    result := fn(ctx)
    if result.IsFailure() {
        span.RecordError(result.Error())
        span.SetStatus(trace.Status{Code: trace.StatusCodeError})
    }

    return result
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attrs...)
}
```

## 4. Modern Go Patterns

### Functional Options Pattern

```go
// pkg/patterns/options.go
package patterns

// Example: Building a flexible HTTP client
type HTTPClient struct {
    baseURL     string
    timeout     time.Duration
    retries     int
    headers     map[string]string
    interceptor func(*http.Request) error
    transport   http.RoundTripper
}

type HTTPClientOption func(*HTTPClient)

func NewHTTPClient(baseURL string, opts ...HTTPClientOption) *HTTPClient {
    client := &HTTPClient{
        baseURL: baseURL,
        timeout: 30 * time.Second,
        retries: 3,
        headers: make(map[string]string),
        transport: http.DefaultTransport,
    }

    for _, opt := range opts {
        opt(client)
    }

    return client
}

func WithTimeout(timeout time.Duration) HTTPClientOption {
    return func(c *HTTPClient) {
        c.timeout = timeout
    }
}

func WithRetries(retries int) HTTPClientOption {
    return func(c *HTTPClient) {
        c.retries = retries
    }
}

func WithHeader(key, value string) HTTPClientOption {
    return func(c *HTTPClient) {
        c.headers[key] = value
    }
}

func WithInterceptor(interceptor func(*http.Request) error) HTTPClientOption {
    return func(c *HTTPClient) {
        c.interceptor = interceptor
    }
}

func WithTransport(transport http.RoundTripper) HTTPClientOption {
    return func(c *HTTPClient) {
        c.transport = transport
    }
}
```

### Builder Pattern with Method Chaining

```go
// pkg/patterns/builder.go
package patterns

// PipelineBuilder builds complex pipelines with method chaining
type PipelineBuilder struct {
    name        string
    description string
    steps       []Step
    hooks       PipelineHooks
    config      map[string]interface{}
    err         error
}

type PipelineHooks struct {
    OnStart    func(context.Context) error
    OnComplete func(context.Context, error)
    OnStep     func(context.Context, string, error)
}

func NewPipelineBuilder(name string) *PipelineBuilder {
    return &PipelineBuilder{
        name:   name,
        steps:  make([]Step, 0),
        config: make(map[string]interface{}),
    }
}

func (b *PipelineBuilder) WithDescription(desc string) *PipelineBuilder {
    if b.err != nil {
        return b
    }
    b.description = desc
    return b
}

func (b *PipelineBuilder) AddStep(step Step) *PipelineBuilder {
    if b.err != nil {
        return b
    }
    if step == nil {
        b.err = fmt.Errorf("step cannot be nil")
        return b
    }
    b.steps = append(b.steps, step)
    return b
}

func (b *PipelineBuilder) WithConfig(key string, value interface{}) *PipelineBuilder {
    if b.err != nil {
        return b
    }
    b.config[key] = value
    return b
}

func (b *PipelineBuilder) OnStart(fn func(context.Context) error) *PipelineBuilder {
    if b.err != nil {
        return b
    }
    b.hooks.OnStart = fn
    return b
}

func (b *PipelineBuilder) OnComplete(fn func(context.Context, error)) *PipelineBuilder {
    if b.err != nil {
        return b
    }
    b.hooks.OnComplete = fn
    return b
}

func (b *PipelineBuilder) Build() (Pipeline, error) {
    if b.err != nil {
        return nil, b.err
    }

    if len(b.steps) == 0 {
        return nil, fmt.Errorf("pipeline must have at least one step")
    }

    return &pipeline{
        name:        b.name,
        description: b.description,
        steps:       b.steps,
        hooks:       b.hooks,
        config:      b.config,
    }, nil
}

// Validation can be done at build time
func (b *PipelineBuilder) Validate() *PipelineBuilder {
    if b.err != nil {
        return b
    }

    // Validate pipeline configuration
    if b.name == "" {
        b.err = fmt.Errorf("pipeline name is required")
        return b
    }

    // Validate steps compatibility
    for i := 0; i < len(b.steps)-1; i++ {
        if !b.steps[i].OutputSchema().CompatibleWith(b.steps[i+1].InputSchema()) {
            b.err = fmt.Errorf("step %d output incompatible with step %d input", i, i+1)
            return b
        }
    }

    return b
}
```

### Option/Maybe Types

```go
// pkg/patterns/option.go
package patterns

// Option represents an optional value
type Option[T any] struct {
    value   T
    present bool
}

// Some creates an Option with a value
func Some[T any](value T) Option[T] {
    return Option[T]{value: value, present: true}
}

// None creates an empty Option
func None[T any]() Option[T] {
    var zero T
    return Option[T]{value: zero, present: false}
}

// IsSome returns true if the Option contains a value
func (o Option[T]) IsSome() bool {
    return o.present
}

// IsNone returns true if the Option is empty
func (o Option[T]) IsNone() bool {
    return !o.present
}

// Unwrap returns the value, panics if None
func (o Option[T]) Unwrap() T {
    if !o.present {
        panic("called Unwrap on None value")
    }
    return o.value
}

// UnwrapOr returns the value or a default
func (o Option[T]) UnwrapOr(defaultValue T) T {
    if o.present {
        return o.value
    }
    return defaultValue
}

// UnwrapOrElse returns the value or calls a function
func (o Option[T]) UnwrapOrElse(fn func() T) T {
    if o.present {
        return o.value
    }
    return fn()
}

// Map transforms the value if present
func Map[T, U any](o Option[T], fn func(T) U) Option[U] {
    if o.IsNone() {
        return None[U]()
    }
    return Some(fn(o.value))
}

// FlatMap transforms the value if present
func FlatMap[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
    if o.IsNone() {
        return None[U]()
    }
    return fn(o.value)
}

// Filter returns None if the predicate fails
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
    if o.IsNone() || !predicate(o.value) {
        return None[T]()
    }
    return o
}

// Example usage with database operations
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) FindByID(id int64) Option[User] {
    var user User
    err := r.db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user)
    if err == sql.ErrNoRows {
        return None[User]()
    }
    if err != nil {
        // Log error
        return None[User]()
    }
    return Some(user)
}

// Usage
func GetUserDisplayName(repo *UserRepository, userID int64) string {
    return repo.FindByID(userID).
        Map(func(u User) string { return u.DisplayName }).
        UnwrapOr("Anonymous")
}
```

### Railway-Oriented Programming

```go
// pkg/patterns/railway.go
package patterns

// Railway represents a computation that can succeed or fail
type Railway[T any] struct {
    result result.Result[T]
}

// Track creates a new Railway from a Result
func Track[T any](r result.Result[T]) Railway[T] {
    return Railway[T]{result: r}
}

// Success creates a successful Railway
func Success[T any](value T) Railway[T] {
    return Railway[T]{result: result.Success(value)}
}

// Failure creates a failed Railway
func Failure[T any](err error) Railway[T] {
    return Railway[T]{result: result.Failure[T](err)}
}

// Then chains operations on the success track
func (r Railway[T]) Then(fn func(T) error) Railway[T] {
    if r.result.IsFailure() {
        return r
    }

    if err := fn(r.result.Value()); err != nil {
        return Failure[T](err)
    }

    return r
}

// Map transforms the value on the success track
func (r Railway[T]) Map(fn func(T) T) Railway[T] {
    if r.result.IsFailure() {
        return r
    }

    return Success(fn(r.result.Value()))
}

// FlatMap chains Railway operations
func (r Railway[T]) FlatMap(fn func(T) Railway[T]) Railway[T] {
    if r.result.IsFailure() {
        return r
    }

    return fn(r.result.Value())
}

// Recover handles errors and returns to success track
func (r Railway[T]) Recover(fn func(error) T) Railway[T] {
    if r.result.IsSuccess() {
        return r
    }

    return Success(fn(r.result.Error()))
}

// Result returns the underlying Result
func (r Railway[T]) Result() result.Result[T] {
    return r.result
}

// Example: Complex business operation using Railway
type OrderService struct {
    repo      OrderRepository
    inventory InventoryService
    payment   PaymentService
    shipping  ShippingService
}

func (s *OrderService) ProcessOrder(orderID string) Railway[Order] {
    return Track(s.repo.FindByID(orderID)).
        Then(func(order Order) error {
            return s.validateOrder(order)
        }).
        FlatMap(func(order Order) Railway[Order] {
            return s.reserveInventory(order)
        }).
        FlatMap(func(order Order) Railway[Order] {
            return s.processPayment(order)
        }).
        FlatMap(func(order Order) Railway[Order] {
            return s.scheduleShipping(order)
        }).
        Map(func(order Order) Order {
            order.Status = "completed"
            return order
        }).
        Then(func(order Order) error {
            return s.repo.Update(order)
        }).
        Recover(func(err error) Order {
            // Handle error, maybe compensate
            s.logError(err)
            return Order{Status: "failed", Error: err.Error()}
        })
}
```

## 5. Practical FlexCore Adapter Implementation

Using all these patterns, here's how a simple adapter would look:

```go
// adapters/oracle/adapter.go
package oracle

import (
    "context"
    "github.com/flext/flexcore/pkg/plugin"
    "github.com/flext/flexcore/pkg/patterns"
    "github.com/flext/flexcore/shared/result"
)

type OracleAdapter struct {
    config  *Config
    client  *OracleClient
    metrics *MetricsCollector
}

type Config struct {
    ConnectionString string `validate:"required"`
    MaxConnections   int    `validate:"min=1,max=100"`
    BatchSize        int    `validate:"min=1,max=10000"`
}

// Plugin implementation
func (a *OracleAdapter) Name() string {
    return "oracle-adapter"
}

func (a *OracleAdapter) Version() string {
    return "1.0.0"
}

func (a *OracleAdapter) Configure(config map[string]interface{}) error {
    // Use Viper for configuration
    v := viper.New()
    v.SetConfigType("yaml")

    for k, val := range config {
        v.Set(k, val)
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return err
    }

    // Validate configuration
    if err := validator.New().Struct(&cfg); err != nil {
        return err
    }

    a.config = &cfg

    // Initialize client with functional options
    a.client = NewOracleClient(
        cfg.ConnectionString,
        WithMaxConnections(cfg.MaxConnections),
        WithBatchSize(cfg.BatchSize),
        WithRetryPolicy(ExponentialBackoff(3, 1*time.Second)),
    )

    return nil
}

// Extract implements data extraction using Railway pattern
func (a *OracleAdapter) Extract(ctx context.Context, req plugin.ExtractRequest) (*plugin.ExtractResponse, error) {
    // Start tracing
    ctx, span := otel.Tracer("oracle-adapter").Start(ctx, "Extract")
    defer span.End()

    // Use Railway pattern for the operation
    result := patterns.Track(a.validateRequest(req)).
        FlatMap(func(req plugin.ExtractRequest) patterns.Railway[*QueryResult] {
            return a.executeQuery(ctx, req)
        }).
        Map(func(qr *QueryResult) *plugin.ExtractResponse {
            return a.transformToResponse(qr)
        }).
        Then(func(resp *plugin.ExtractResponse) error {
            return a.recordMetrics(resp)
        })

    if result.Result().IsFailure() {
        span.RecordError(result.Result().Error())
        return nil, result.Result().Error()
    }

    return result.Result().Value(), nil
}

// Builder pattern for complex queries
func (a *OracleAdapter) buildQuery(table string) *QueryBuilder {
    return NewQueryBuilder(table).
        WithFields("id", "name", "created_at").
        WithFilter("status", "=", "active").
        WithOrderBy("created_at", "DESC").
        WithLimit(a.config.BatchSize)
}

// Main function to register as plugin
func main() {
    adapter := &OracleAdapter{}

    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: plugin.Handshake,
        Plugins: map[string]plugin.Plugin{
            "adapter": &plugin.AdapterPlugin{Impl: adapter},
        },
    })
}
```

## Summary

These patterns provide:

1. **Plugin Isolation**: HashiCorp go-plugin ensures adapters run in separate processes
2. **Configuration Management**: Viper provides flexible, type-safe configuration with hot reload
3. **Clean HTTP Handling**: Middleware chains, transformers, and OpenTelemetry integration
4. **Functional Programming**: Options, Results, and Railway patterns for better error handling
5. **Builder Pattern**: Fluent interfaces for complex object construction

FlexCore can leverage these patterns to make adapter development straightforward while maintaining robustness and scalability.
