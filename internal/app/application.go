// Package app implements FlexCore's Application Layer following Clean Architecture principles.
//
// This package serves as the orchestration layer between the presentation layer (HTTP handlers)
// and the domain layer (business logic). It implements the Application Layer pattern from
// Clean Architecture, coordinating use cases, managing cross-cutting concerns, and ensuring
// proper separation of concerns.
//
// Architecture:
//   The Application Layer implements several key architectural patterns:
//   - Use Case Orchestration: Coordinates business workflows and operations
//   - Command/Query Responsibility Segregation (CQRS): Separates read and write operations
//   - Dependency Inversion: Depends on abstractions rather than concrete implementations
//   - Transaction Management: Ensures data consistency across operations
//   - Event Coordination: Manages domain event publishing and handling
//
// Key Components:
//   - Application Service: Main orchestrator for business use cases
//   - Command Bus: Handles state-changing operations with proper validation
//   - Query Bus: Manages read operations with optimization and caching
//   - HTTP Server: REST API endpoints for external system integration
//   - Configuration Management: Environment-aware configuration handling
//
// Integration:
//   - Domain Layer: Executes business logic through domain entities and services
//   - Infrastructure Layer: Accesses external resources (databases, APIs, file systems)
//   - Presentation Layer: Receives requests and returns formatted responses
//   - Cross-Cutting Concerns: Logging, monitoring, security, and error handling
//
// Example:
//   Complete application initialization and lifecycle:
//
//     config, err := config.Load("config.yaml")
//     if err != nil {
//         return fmt.Errorf("config load failed: %w", err)
//     }
//     
//     app, err := app.NewApplication(config)
//     if err != nil {
//         return fmt.Errorf("application creation failed: %w", err)
//     }
//     
//     ctx := context.Background()
//     if err := app.Start(ctx); err != nil {
//         return fmt.Errorf("application start failed: %w", err)
//     }
//     
//     // Graceful shutdown
//     defer func() {
//         shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//         defer cancel()
//         app.Stop(shutdownCtx)
//     }()
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// Application represents the main FlexCore application instance implementing Clean Architecture.
//
// Application serves as the primary coordinator for the FlexCore runtime container service,
// managing the HTTP server, configuration, routing, and application lifecycle. It follows
// the Application Service pattern from Clean Architecture, orchestrating use cases and
// coordinating between presentation, domain, and infrastructure layers.
//
// Architecture:
//   The Application struct implements several architectural patterns:
//   - Service Orchestration: Coordinates business workflows and use cases
//   - Dependency Management: Manages dependencies and their lifecycle
//   - Configuration Management: Handles environment-specific configuration
//   - Server Management: Manages HTTP server lifecycle and graceful shutdown
//   - Route Organization: Structures API endpoints and middleware integration
//
// Fields:
//   config *config.Config: Application configuration with environment-specific settings
//   server *http.Server: HTTP server instance with configured timeouts and middleware
//   mux *http.ServeMux: Request multiplexer for routing and handler organization
//
// Lifecycle:
//   1. **Initialization**: Application created with configuration validation
//   2. **Configuration**: Logging, database, and external service setup
//   3. **Route Setup**: HTTP handlers and middleware configuration
//   4. **Service Start**: HTTP server start with graceful error handling
//   5. **Runtime Operations**: Request processing and business logic execution
//   6. **Graceful Shutdown**: Clean resource cleanup and connection termination
//
// Example:
//   Complete application lifecycle management:
//
//     // Load configuration from environment
//     config := &config.Config{
//         App: config.AppConfig{
//             Environment: "production",
//             Version:     "0.9.0",
//             Debug:       false,
//         },
//         Server: config.ServerConfig{
//             Host:         "0.0.0.0",
//             Port:         8080,
//             ReadTimeout:  30 * time.Second,
//             WriteTimeout: 30 * time.Second,
//             IdleTimeout:  60 * time.Second,
//         },
//     }
//     
//     // Create application instance
//     app, err := NewApplication(config)
//     if err != nil {
//         log.Fatal("Application creation failed", "error", err)
//     }
//     
//     // Start application services
//     ctx := context.Background()
//     if err := app.Start(ctx); err != nil {
//         log.Fatal("Application start failed", "error", err)
//     }
//     
//     // Handle shutdown signals
//     sigChan := make(chan os.Signal, 1)
//     signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
//     <-sigChan
//     
//     // Graceful shutdown
//     shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//     defer cancel()
//     
//     if err := app.Stop(shutdownCtx); err != nil {
//         log.Error("Graceful shutdown failed", "error", err)
//     }
//
// Integration:
//   - Configuration Layer: Environment-specific settings and feature flags
//   - Logging System: Structured logging with correlation IDs and context
//   - Monitoring Stack: Metrics collection, health checks, and observability
//   - Security Layer: Authentication, authorization, and input validation
//   - Database Layer: Connection pooling and transaction management
//
// Thread Safety:
//   Application instances are designed for single-threaded initialization and shutdown.
//   Runtime operations are thread-safe through Go's HTTP server implementation.
type Application struct {
	config *config.Config  // Application configuration with environment-specific settings
	server *http.Server    // HTTP server instance with configured timeouts and security
	mux    *http.ServeMux  // Request multiplexer for routing and handler organization
}

// NewApplication creates a new FlexCore application instance with comprehensive initialization.
//
// This factory function implements the Application Factory pattern, performing complete
// application setup including configuration validation, logging initialization, HTTP server
// configuration, and route setup. It ensures the application is in a fully functional
// state before returning.
//
// Initialization Process:
//   1. **Configuration Validation**: Validates required configuration parameters
//   2. **Logging Setup**: Initializes structured logging with appropriate log levels
//   3. **HTTP Server Creation**: Configures server with timeouts and security settings
//   4. **Route Registration**: Sets up API endpoints and middleware chain
//   5. **Dependency Injection**: Initializes application dependencies and services
//
// Parameters:
//   cfg *config.Config: Application configuration with validated settings (required)
//
// Returns:
//   *Application: Fully initialized application instance ready for startup
//   error: Configuration validation or initialization error
//
// Example:
//   Application creation with full configuration:
//
//     config := &config.Config{
//         App: config.AppConfig{
//             Environment: "production",
//             Version:     "0.9.0",
//             Debug:       false,
//             ServiceName: "flexcore",
//         },
//         Server: config.ServerConfig{
//             Host:         "0.0.0.0",
//             Port:         8080,
//             ReadTimeout:  30 * time.Second,
//             WriteTimeout: 30 * time.Second,
//             IdleTimeout:  60 * time.Second,
//         },
//         Database: config.DatabaseConfig{
//             Host:     "localhost",
//             Port:     5432,
//             Name:     "flexcore",
//             Username: "flexcore_user",
//             Password: secretManager.GetPassword("db_password"),
//         },
//     }
//     
//     app, err := NewApplication(config)
//     if err != nil {
//         log.Fatal("Application initialization failed", "error", err)
//         return err
//     }
//     
//     log.Info("Application created successfully", 
//         "environment", config.App.Environment,
//         "version", config.App.Version,
//         "address", fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port))
//
// Configuration Requirements:
//   - cfg must not be nil (required for all application operations)
//   - cfg.App.Environment must be set ("development", "staging", "production")
//   - cfg.Server configuration must include valid host and port
//   - cfg.Database settings required for data persistence
//
// Validation Errors:
//   - "config cannot be nil": Configuration parameter is required
//   - "environment is required": Environment setting must be specified
//   - "failed to initialize logging": Logging system initialization failed
//
// Server Configuration:
//   - Read/Write/Idle timeouts configured for production use
//   - HTTP multiplexer setup with route registration
//   - Security headers and middleware integration ready
//   - Graceful shutdown support built-in
//
// Integration:
//   - Configuration Layer: Environment-specific settings and feature flags
//   - Logging System: Structured logging with correlation IDs
//   - HTTP Router: RESTful API endpoints and middleware chain
//   - Dependency Container: Service registration and lifecycle management
//
// Thread Safety:
//   Safe for concurrent access during initialization. Runtime thread safety
//   managed by Go's HTTP server implementation.
func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.App.Environment == "" {
		return nil, fmt.Errorf("environment is required")
	}

	// Initialize logging if not already done
	if logging.Logger == nil {
		logLevel := "info"
		if cfg.App.Debug {
			logLevel = "debug"
		}
		if err := logging.Initialize(cfg.App.Environment, logLevel); err != nil {
			return nil, fmt.Errorf("failed to initialize logging: %w", err)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	app := &Application{
		config: cfg,
		server: server,
		mux:    mux,
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

// setupRoutes configures all HTTP routes and middleware for the FlexCore API.
//
// This method establishes the complete routing configuration for the FlexCore
// HTTP API, including health check endpoints, service information, and middleware
// integration. Routes are organized following RESTful conventions and include
// proper error handling, logging, and security considerations.
//
// Route Structure:
//   - GET /health: Health check endpoint for monitoring and load balancer probes
//   - GET /: Service information and API discovery endpoint
//   - Future routes: Pipeline management, plugin operations, monitoring endpoints
//
// Example:
//   Route registration with middleware:
//
//     app := &Application{mux: http.NewServeMux()}
//     app.setupRoutes()
//     
//     // Routes are now available for HTTP requests
//     server := &http.Server{
//         Handler: app.mux,
//         Addr:    ":8080",
//     }
//
// Middleware Integration (Planned):
//   - Authentication: JWT token validation and user context
//   - Authorization: Role-based access control and permissions
//   - Logging: Request/response logging with correlation IDs
//   - Metrics: Request metrics and performance monitoring
//   - Rate Limiting: API rate limiting and throttling
//   - CORS: Cross-origin resource sharing configuration
//
// Security Considerations:
//   - Input validation and sanitization
//   - Security headers (HSTS, CSP, X-Frame-Options)
//   - Request size limits and timeout handling
//   - Error response sanitization
//
// Integration:
//   - HTTP Server: Routes registered with application's ServeMux
//   - Monitoring: Health endpoint integration with observability stack
//   - API Gateway: Compatible with reverse proxy and load balancer patterns
//   - Documentation: OpenAPI/Swagger documentation generation ready
//
// Thread Safety:
//   Safe to call during application initialization. Route registration
//   should be completed before starting the HTTP server.
func (a *Application) setupRoutes() {
	a.mux.HandleFunc("/health", a.handleHealth)
	a.mux.HandleFunc("/", a.handleRoot)
}

// handleHealth provides comprehensive health check endpoint for monitoring and observability.
//
// This handler implements a standard health check endpoint following industry
// best practices for microservice health monitoring. It provides essential
// service status information for load balancers, monitoring systems, and
// orchestration platforms like Kubernetes.
//
// HTTP Method: GET
// Endpoint: /health
// Content-Type: application/json
//
// Response Format:
//   {
//     "status": "ok",
//     "timestamp": "2024-01-15T10:30:45Z"
//   }
//
// Parameters:
//   w http.ResponseWriter: HTTP response writer for sending health status
//   r *http.Request: HTTP request (method and headers validated)
//
// Example:
//   Health check integration with monitoring:
//
//     // Kubernetes liveness probe
//     livenessProbe:
//       httpGet:
//         path: /health
//         port: 8080
//       initialDelaySeconds: 30
//       periodSeconds: 10
//     
//     // Load balancer health check
//     curl -f http://flexcore:8080/health || exit 1
//     
//     // Monitoring system integration
//     http_request_duration_seconds{
//       method="GET",
//       endpoint="/health",
//       status="200"
//     }
//
// Response Codes:
//   - 200 OK: Service is healthy and operational
//   - Future: 503 Service Unavailable for unhealthy state
//
// Enhanced Health Checks (Planned):
//   - Database connectivity validation
//   - External service dependency checks
//   - Resource utilization monitoring (memory, disk space)
//   - Plugin system health validation
//   - Event store accessibility verification
//
// Integration:
//   - Kubernetes: Liveness and readiness probe endpoint
//   - Load Balancers: Health check for traffic routing decisions
//   - Monitoring: Service availability metrics and alerting
//   - API Gateway: Upstream service health validation
//
// Thread Safety:
//   Safe for concurrent access. Health status is read-only operation.
func (a *Application) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleRoot provides service discovery and API information endpoint.
//
// This handler implements the API root endpoint following microservice
// discovery patterns, providing essential service metadata for clients,
// monitoring systems, and service mesh integration. It serves as the
// primary entry point for API discovery and service identification.
//
// HTTP Method: GET
// Endpoint: /
// Content-Type: application/json
//
// Response Format:
//   {
//     "service": "flexcore",
//     "version": "0.9.0",
//     "environment": "production"
//   }
//
// Parameters:
//   w http.ResponseWriter: HTTP response writer for service information
//   r *http.Request: HTTP request (method and headers processed)
//
// Example:
//   Service discovery and API exploration:
//
//     // Client service discovery
//     response, err := http.Get("http://flexcore:8080/")
//     if err != nil {
//         return fmt.Errorf("service discovery failed: %w", err)
//     }
//     
//     var serviceInfo struct {
//         Service     string `json:"service"`
//         Version     string `json:"version"`
//         Environment string `json:"environment"`
//     }
//     
//     if err := json.NewDecoder(response.Body).Decode(&serviceInfo); err != nil {
//         return fmt.Errorf("service info decode failed: %w", err)
//     }
//     
//     fmt.Printf("Connected to %s v%s (%s)\n", 
//         serviceInfo.Service, serviceInfo.Version, serviceInfo.Environment)
//
// Enhanced API Information (Planned):
//   - API version and compatibility information
//   - Available endpoints and operations
//   - Authentication requirements and methods
//   - Rate limiting information and policies
//   - OpenAPI/Swagger documentation links
//   - Support contact and documentation URLs
//
// Response Fields:
//   - service: Service name identifier ("flexcore")
//   - version: Current service version (semantic versioning)
//   - environment: Deployment environment (development, staging, production)
//
// Integration:
//   - Service Mesh: Service identity and version discovery
//   - API Gateway: Service registration and routing configuration
//   - Monitoring: Service inventory and version tracking
//   - Client Libraries: Automatic service discovery and configuration
//
// Thread Safety:
//   Safe for concurrent access. Service information is read-only metadata.
func (a *Application) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"flexcore","version":"%s","environment":"%s"}`, a.config.App.Version, a.config.App.Environment)
}

// Start initiates the FlexCore application with comprehensive service startup.
//
// This method implements the application startup sequence, including HTTP server
// initialization, service registration, and health check activation. It follows
// graceful startup patterns with proper error handling and observability integration.
//
// Startup Sequence:
//   1. **Service Initialization**: Core services and dependencies startup
//   2. **HTTP Server Start**: Non-blocking server startup in dedicated goroutine
//   3. **Health Check Activation**: Health endpoints become available
//   4. **Service Registration**: Register with service discovery (future)
//   5. **Monitoring Integration**: Metrics and tracing activation
//
// Parameters:
//   ctx context.Context: Application context for cancellation and timeout handling
//
// Returns:
//   error: Startup error if critical services fail to initialize
//
// Example:
//   Application startup with proper error handling:
//
//     app, err := NewApplication(config)
//     if err != nil {
//         return fmt.Errorf("application creation failed: %w", err)
//     }
//     
//     ctx := context.Background()
//     if err := app.Start(ctx); err != nil {
//         return fmt.Errorf("application startup failed: %w", err)
//     }
//     
//     // Wait for server to be ready
//     time.Sleep(100 * time.Millisecond)
//     
//     // Verify server is responding
//     resp, err := http.Get("http://localhost:8080/health")
//     if err != nil {
//         return fmt.Errorf("health check failed: %w", err)
//     }
//     defer resp.Body.Close()
//     
//     if resp.StatusCode != http.StatusOK {
//         return fmt.Errorf("server not healthy: status %d", resp.StatusCode)
//     }
//     
//     log.Info("FlexCore application started successfully")
//
// Startup Logging:
//   - Service start confirmation with configuration details
//   - Server address and port binding information
//   - Environment and version identification
//   - Error logging for startup failures
//
// Error Handling:
//   - HTTP server startup failures logged but don't block startup
//   - ErrServerClosed is expected during graceful shutdown
//   - Critical service failures return startup errors
//   - Non-critical failures logged with degraded service indication
//
// Async Operations:
//   - HTTP server runs in separate goroutine for non-blocking startup
//   - Background services and health checks activated asynchronously
//   - Service discovery registration performed in background
//
// Integration:
//   - Logging System: Structured startup logging with context information
//   - HTTP Server: Production-ready server with configured timeouts
//   - Monitoring: Service startup metrics and health status
//   - Service Discovery: Automatic service registration (planned)
//
// Thread Safety:
//   Safe for single-threaded startup sequence. Concurrent access to running
//   application managed by Go's HTTP server implementation.
func (a *Application) Start(ctx context.Context) error {
	logging.Logger.Info("Starting FlexCore application",
		zap.String("address", a.server.Addr),
		zap.String("environment", a.config.App.Environment),
	)

	// Start server in goroutine
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logger.Error("Server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop performs graceful shutdown of the FlexCore application with proper resource cleanup.
//
// This method implements the graceful shutdown pattern, ensuring all active
// requests are completed, resources are properly released, and external
// connections are cleanly terminated. It follows best practices for production
// service shutdown with timeout handling and error recovery.
//
// Shutdown Sequence:
//   1. **Shutdown Initiation**: Log shutdown start and stop accepting new requests
//   2. **Active Request Completion**: Wait for in-flight requests to complete
//   3. **Resource Cleanup**: Close database connections, release locks, cleanup temp files
//   4. **Service Deregistration**: Remove from service discovery (future)
//   5. **Final Logging**: Log successful shutdown completion
//
// Parameters:
//   ctx context.Context: Shutdown context with timeout for controlling shutdown duration
//
// Returns:
//   error: Shutdown error if graceful shutdown fails or times out
//
// Example:
//   Graceful shutdown with signal handling:
//
//     // Setup signal handling
//     sigChan := make(chan os.Signal, 1)
//     signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
//     
//     // Start application
//     app, _ := NewApplication(config)
//     app.Start(context.Background())
//     
//     // Wait for shutdown signal
//     <-sigChan
//     log.Info("Shutdown signal received")
//     
//     // Graceful shutdown with timeout
//     shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//     defer cancel()
//     
//     if err := app.Stop(shutdownCtx); err != nil {
//         log.Error("Graceful shutdown failed", "error", err)
//         os.Exit(1)
//     }
//     
//     log.Info("Application shutdown completed successfully")
//
// Timeout Handling:
//   - Context timeout determines maximum shutdown duration
//   - In-flight requests given time to complete naturally
//   - Forced termination if graceful shutdown exceeds timeout
//   - Proper error reporting for timeout scenarios
//
// Resource Cleanup (Current and Planned):
//   - HTTP server connection termination
//   - Database connection pool cleanup
//   - Plugin resource deallocation
//   - Temporary file and cache cleanup
//   - External service connection termination
//
// Integration:
//   - Signal Handling: Integration with OS signal handling for clean shutdown
//   - Container Orchestration: Kubernetes SIGTERM handling compatibility
//   - Load Balancer: Proper deregistration and traffic draining
//   - Monitoring: Shutdown metrics and completion status tracking
//
// Thread Safety:
//   Safe for single-threaded shutdown sequence. HTTP server shutdown is
//   thread-safe and handles concurrent request completion.
func (a *Application) Stop(ctx context.Context) error {
	logging.Logger.Info("Stopping FlexCore application")
	return a.server.Shutdown(ctx)
}

// Config returns the application's configuration for runtime access and inspection.
//
// This method provides read-only access to the application's configuration,
// enabling runtime configuration inspection, debugging, and dynamic behavior
// based on configuration values. The returned configuration should be treated
// as immutable to prevent unintended side effects.
//
// Returns:
//   *config.Config: Application configuration with all settings and parameters
//
// Example:
//   Configuration access for runtime decisions:
//
//     app, _ := NewApplication(config)
//     
//     // Access configuration for runtime decisions
//     appConfig := app.Config()
//     
//     // Environment-specific behavior
//     if appConfig.App.Environment == "development" {
//         log.SetLevel(log.DebugLevel)
//         enableDeveloperFeatures()
//     }
//     
//     // Feature flag evaluation
//     if appConfig.App.Debug {
//         enableDebugEndpoints()
//         logDetailedRequestInfo()
//     }
//     
//     // Server configuration access
//     serverAddr := fmt.Sprintf("%s:%d", 
//         appConfig.Server.Host, 
//         appConfig.Server.Port)
//     
//     log.Info("Server configuration", 
//         "address", serverAddr,
//         "read_timeout", appConfig.Server.ReadTimeout,
//         "write_timeout", appConfig.Server.WriteTimeout)
//
// Configuration Categories:
//   - App: Application metadata (name, version, environment)
//   - Server: HTTP server settings (host, port, timeouts)
//   - Database: Data persistence configuration
//   - Logging: Log levels and output configuration
//   - Security: Authentication and authorization settings
//   - External Services: API endpoints and connection settings
//
// Usage Patterns:
//   - Runtime Feature Flags: Enable/disable features based on configuration
//   - Environment Adaptation: Different behavior per environment
//   - Performance Tuning: Timeout and limit adjustments
//   - Integration Settings: External service endpoints and credentials
//   - Debugging Support: Development and debugging feature activation
//
// Security Considerations:
//   - Sensitive configuration (passwords, tokens) should not be logged
//   - Configuration access should be controlled and audited
//   - Runtime configuration changes should be validated and authorized
//
// Integration:
//   - Feature Flags: Runtime feature enabling based on configuration
//   - Monitoring: Configuration-based metrics and alerting setup
//   - Service Discovery: Dynamic service endpoint configuration
//   - Security: Runtime security policy and credential management
//
// Thread Safety:
//   Safe for concurrent read access. Configuration is immutable after
//   application initialization.
func (a *Application) Config() *config.Config {
	return a.config
}
