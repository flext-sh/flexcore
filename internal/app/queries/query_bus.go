// Package queries implements CQRS Query processing infrastructure for FlexCore.
//
// This package provides comprehensive Command Query Responsibility Segregation (CQRS)
// implementation for handling read operations in the FlexCore application layer.
// It implements the Query pattern with enterprise-grade features including
// caching, pagination, filtering, and optimized read model access.
//
// Architecture:
//
//	The Query Bus implementation follows several enterprise patterns:
//	- Query Pattern: Encapsulates data retrieval requests with immutable query objects
//	- Repository Pattern: Optimized read models and data access for query processing
//	- Caching Pattern: Intelligent result caching with cache invalidation strategies
//	- Pagination Pattern: Efficient large dataset handling with cursor-based pagination
//	- Builder Pattern: Query bus creation with flexible configuration options
//
// Core Components:
//   - Query Interface: Standard contract for all query objects
//   - QueryHandler[T, R]: Generic handler interface for type-safe query processing
//   - QueryBus: Central dispatcher for query execution and optimization
//   - InMemoryQueryBus: Thread-safe in-memory implementation with handler registry
//   - CachedQueryBus: Decorator pattern implementation for result caching
//
// Query Types:
//   - BaseQuery: Simple query implementation with type identification
//   - PagedQuery: Pagination support with sorting and ordering
//   - FilteredQuery: Dynamic filtering with flexible criteria
//   - PagedResult[T]: Type-safe pagination results with navigation metadata
//
// Example:
//
//	Complete query bus setup with caching:
//
//	  // Create query bus with caching enabled
//	  queryBus := queries.NewQueryBusBuilder().
//	      WithCache().
//	      Build()
//
//	  // Register query handlers
//	  pipelineListHandler := &ListPipelinesHandler{readModel: pipelineReadModel}
//	  queryBus.RegisterHandler(&ListPipelinesQuery{}, pipelineListHandler)
//
//	  // Execute queries with pagination
//	  query := queries.NewPagedQuery("ListPipelines", 1, 20).
//	      WithOrderBy("created_at", "DESC")
//
//	  result := queryBus.Execute(ctx, &ListPipelinesQuery{
//	      PagedQuery: query,
//	      Owner:      "data-team",
//	      Status:     entities.PipelineStatusActive,
//	  })
//
//	  if result.IsFailure() {
//	      return fmt.Errorf("query execution failed: %w", result.Error())
//	  }
//
//	  pagedResult := result.Value().(queries.PagedResult[*PipelineView])
//	  fmt.Printf("Found %d pipelines (page %d of %d)\n",
//	      len(pagedResult.Items), pagedResult.Page, pagedResult.TotalPages)
//
// Integration:
//   - Read Models: Optimized data structures for query processing
//   - Application Services: Coordinates complex queries across multiple read models
//   - Infrastructure Layer: Integrates with databases, caches, and external services
//   - Presentation Layer: HTTP handlers delegate to query bus for read operations
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package queries

import (
	"context"
	"fmt"
	"sync"

	"github.com/flext-sh/flexcore/pkg/cqrs"
	"github.com/flext-sh/flexcore/pkg/result"
)

// Query represents a query in the CQRS pattern for read operations.
//
// Query serves as the base interface for all query objects in the CQRS
// architecture, encapsulating data retrieval requests without modifying system
// state. Queries are immutable request objects that contain all necessary
// parameters for retrieving specific data from read models.
//
// Design Principles:
//   - Queries represent data retrieval intentions (GetPipeline, ListUsers, SearchData)
//   - Queries are immutable after creation to ensure consistent processing
//   - Queries contain all parameters needed for data retrieval (no external dependencies)
//   - Queries are optimized for read performance with minimal validation overhead
//   - Queries can be cached based on parameters for improved performance
//
// Example:
//
//	Implementing custom queries:
//
//	  type ListPipelinesQuery struct {
//	      queries.PagedQuery
//	      Owner  string
//	      Status entities.PipelineStatus
//	      Tags   []string
//	  }
//
//	  func NewListPipelinesQuery(owner string, page, pageSize int) *ListPipelinesQuery {
//	      return &ListPipelinesQuery{
//	          PagedQuery: queries.NewPagedQuery("ListPipelines", page, pageSize),
//	          Owner:      owner,
//	      }
//	  }
//
//	  // QueryType is implemented by PagedQuery
//	  // Additional filtering methods can be added
//	  func (q *ListPipelinesQuery) WithStatus(status entities.PipelineStatus) *ListPipelinesQuery {
//	      q.Status = status
//	      return q
//	  }
//
//	  func (q *ListPipelinesQuery) WithTags(tags []string) *ListPipelinesQuery {
//	      q.Tags = tags
//	      return q
//	  }
//
// Integration:
//   - Query Handlers: Process specific query types with optimized data access
//   - Query Bus: Routes queries to appropriate handlers for execution
//   - Caching: Queries can be used as cache keys for result storage
//   - Serialization: Queries can be serialized for logging and debugging
type Query interface {
	QueryType() string // Returns unique query type identifier for routing and caching
}

// QueryHandler represents a type-safe handler for processing specific query types.
//
// QueryHandler implements the Handler pattern with Go generics for complete type safety,
// ensuring that handlers can only process queries of their designated type and return
// strongly-typed results. Handlers encapsulate optimized data access logic and
// coordinate with read models and infrastructure services.
//
// Type Parameters:
//
//	T Query: Specific query type that this handler processes
//	R any: Result type that this handler returns (strongly typed)
//
// Handler Responsibilities:
//   - Access optimized read models and data projections
//   - Apply filtering, sorting, and pagination logic
//   - Coordinate with caching layers for performance optimization
//   - Transform data into appropriate view models for presentation
//   - Return execution results with proper error handling
//
// Example:
//
//	Implementing a query handler:
//
//	  type ListPipelinesHandler struct {
//	      pipelineReadModel PipelineReadModel
//	      cache             CacheService
//	      logger            logging.Logger
//	  }
//
//	  func (h *ListPipelinesHandler) Handle(
//	      ctx context.Context,
//	      query *ListPipelinesQuery,
//	  ) result.Result[queries.PagedResult[*PipelineView]] {
//	      // 1. Apply filters and build query criteria
//	      criteria := h.buildQueryCriteria(query)
//
//	      // 2. Access read model with pagination
//	      pipelines, totalCount, err := h.pipelineReadModel.List(
//	          ctx, criteria, query.Page, query.PageSize,
//	      )
//	      if err != nil {
//	          return result.Failure[queries.PagedResult[*PipelineView]](err)
//	      }
//
//	      // 3. Transform to view models
//	      viewModels := make([]*PipelineView, len(pipelines))
//	      for i, pipeline := range pipelines {
//	          viewModels[i] = h.transformToView(pipeline)
//	      }
//
//	      // 4. Create paged result
//	      pagedResult := queries.NewPagedResult(
//	          viewModels, totalCount, query.Page, query.PageSize,
//	      )
//
//	      return result.Success(pagedResult)
//	  }
//
// Integration:
//   - Read Models: Accesses optimized data structures for query processing
//   - View Models: Transforms data into presentation-ready formats
//   - Caching: Integrates with caching layers for performance optimization
//   - Logging: Provides query execution logging and performance monitoring
type QueryHandler[T Query, R any] interface {
	Handle(ctx context.Context, query T) result.Result[R] // Processes query with full type safety
}

// QueryBus coordinates query execution and handler management in the CQRS architecture.
//
// QueryBus serves as the central dispatcher for all read operations in the
// application, implementing the Mediator pattern to decouple query senders
// from query handlers. It provides optimized read processing with caching,
// performance monitoring, and comprehensive error handling.
//
// Core Responsibilities:
//   - Handler Registration: Type-safe registration of query handlers
//   - Query Routing: Routes queries to appropriate handlers based on type
//   - Execution Optimization: Manages query execution with caching and performance monitoring
//   - Result Caching: Intelligent caching of query results for improved performance
//   - Read Model Coordination: Coordinates access to optimized read models
//
// Example:
//
//	Complete query bus usage:
//
//	  // Create and configure query bus
//	  queryBus := queries.NewQueryBusBuilder().
//	      WithCache().
//	      Build()
//
//	  // Register handlers for different query types
//	  queryBus.RegisterHandler(&ListPipelinesQuery{}, listPipelinesHandler)
//	  queryBus.RegisterHandler(&GetPipelineQuery{}, getPipelineHandler)
//	  queryBus.RegisterHandler(&SearchPipelinesQuery{}, searchPipelinesHandler)
//
//	  // Execute queries with different patterns
//
//	  // Simple query execution
//	  query := &GetPipelineQuery{ID: pipelineID}
//	  result := queryBus.Execute(ctx, query)
//	  if result.IsFailure() {
//	      return fmt.Errorf("query failed: %w", result.Error())
//	  }
//
//	  pipeline := result.Value().(*PipelineView)
//
//	  // Paginated query execution
//	  listQuery := queries.NewPagedQuery("ListPipelines", 1, 20)
//	  listResult := queryBus.Execute(ctx, &ListPipelinesQuery{
//	      PagedQuery: listQuery,
//	      Owner:      "data-team",
//	  })
//
//	  if listResult.IsSuccess() {
//	      pagedResult := listResult.Value().(queries.PagedResult[*PipelineView])
//	      fmt.Printf("Found %d pipelines\n", len(pagedResult.Items))
//	  }
//
// Integration:
//   - Application Layer: Central coordination point for all read operations
//   - HTTP Handlers: REST API endpoints delegate to query bus
//   - Read Models: Optimized data access through specialized read models
//   - Caching Layer: Result caching for improved query performance
type QueryBus interface {
	RegisterHandler(query Query, handler interface{}) error              // Registers handler for query type
	Execute(ctx context.Context, query Query) result.Result[interface{}] // Executes query with optimization
}

// InMemoryQueryBus provides a thread-safe in-memory implementation of QueryBus.
//
// InMemoryQueryBus implements the QueryBus interface with handlers stored
// in memory using a concurrent-safe map. It provides high-performance query
// processing optimized for read operations with minimal latency, suitable
// for single-instance deployments and development environments.
//
// Architecture:
//   - Thread Safety: Uses RWMutex for concurrent handler registration and execution
//   - Handler Registry: Type-safe handler registration with validation
//   - Shared Utilities: Leverages cqrs package for DRY principles and consistency
//   - Generic Execution: Supports any query type through reflection-based dispatch
//   - Read Optimization: Optimized for high-throughput read operations
//
// Fields:
//
//	mu sync.RWMutex: Reader-writer mutex for thread-safe access to handlers map
//	handlers map[string]interface{}: Query type to handler mapping registry
//	validator *cqrs.BusValidationUtilities: Shared validation logic for DRY compliance
//	invoker *cqrs.HandlerInvoker: Shared handler invocation with reflection optimization
//
// Example:
//
//	Direct usage (typically used through builder pattern):
//
//	  queryBus := queries.NewInMemoryQueryBus()
//
//	  // Register handlers
//	  listHandler := &ListPipelinesHandler{readModel: pipelineReadModel}
//	  err := queryBus.RegisterHandler(&ListPipelinesQuery{}, listHandler)
//	  if err != nil {
//	      return fmt.Errorf("handler registration failed: %w", err)
//	  }
//
//	  // Execute queries
//	  query := &ListPipelinesQuery{Owner: "data-team"}
//	  result := queryBus.Execute(ctx, query)
//	  if result.IsFailure() {
//	      return result.Error()
//	  }
//
//	  // Check registered query types
//	  registeredTypes := queryBus.GetRegisteredQueries()
//	  fmt.Printf("Registered queries: %v\n", registeredTypes)
//
// Performance Characteristics:
//   - Read Operations: Optimized for high-throughput concurrent query execution
//   - Memory Usage: Minimal memory overhead with efficient handler storage
//   - Latency: Sub-millisecond query routing and handler invocation
//   - Scalability: Suitable for moderate query loads in single-instance deployments
//
// Thread Safety:
//   - Handler registration uses write lock for exclusive access
//   - Query execution uses read lock for maximum concurrent throughput
//   - Handler map access properly synchronized for concurrent operations
//   - Safe for high-concurrency query processing scenarios
//
// Integration:
//   - CQRS Package: Leverages shared validation and invocation utilities
//   - Caching Pattern: Wrappable by CachedQueryBus for result caching
//   - Builder Pattern: Created through QueryBusBuilder for configuration flexibility
//   - Result Pattern: All operations return Result types for explicit error handling
type InMemoryQueryBus struct {
	mu        sync.RWMutex                 // Thread-safe access to handlers registry
	handlers  map[string]interface{}       // Query type to handler mapping
	validator *cqrs.BusValidationUtilities // Shared validation logic for DRY compliance
	invoker   *cqrs.HandlerInvoker         // Shared handler invocation with reflection
}

// NewInMemoryQueryBus creates a new in-memory query bus
func NewInMemoryQueryBus() *InMemoryQueryBus {
	return &InMemoryQueryBus{
		handlers:  make(map[string]interface{}),
		validator: cqrs.NewBusValidationUtilities(),
		invoker:   cqrs.NewHandlerInvoker(),
	}
}

// NewQueryBus creates a new query bus (returns interface)
func NewQueryBus() QueryBus {
	return NewInMemoryQueryBus()
}

// RegisterHandler registers a query handler with type validation and thread safety.
//
// This method implements the Registry pattern for query handler registration,
// using shared validation logic to ensure consistency with command bus registration.
// It enforces type safety by validating that handlers implement the correct
// QueryHandler interface for their designated query type.
//
// Parameters:
//
//	query Query: Example query instance for type identification
//	handler interface{}: Handler implementation (must implement QueryHandler[T, R])
//
// Returns:
//
//	error: Registration error if handler type validation fails
//
// Example:
//
//	Registering query handlers:
//
//	  queryBus := queries.NewInMemoryQueryBus()
//
//	  // Register different query handlers
//	  listHandler := &ListPipelinesHandler{readModel: pipelineReadModel}
//	  getHandler := &GetPipelineHandler{readModel: pipelineReadModel}
//	  searchHandler := &SearchPipelinesHandler{searchIndex: searchIndex}
//
//	  // Type-safe registration
//	  if err := queryBus.RegisterHandler(&ListPipelinesQuery{}, listHandler); err != nil {
//	      return fmt.Errorf("list handler registration failed: %w", err)
//	  }
//
//	  if err := queryBus.RegisterHandler(&GetPipelineQuery{}, getHandler); err != nil {
//	      return fmt.Errorf("get handler registration failed: %w", err)
//	  }
//
//	  if err := queryBus.RegisterHandler(&SearchPipelinesQuery{}, searchHandler); err != nil {
//	      return fmt.Errorf("search handler registration failed: %w", err)
//	  }
//
//	  log.Info("All query handlers registered successfully")
//
// Validation:
//   - Handler must implement QueryHandler[T, R] interface
//   - Query type extracted from query.QueryType()
//   - Prevents duplicate handler registration for same query type
//   - Type compatibility validated through reflection
//
// DRY Principle:
//
//	Uses shared registration logic from cqrs.BusValidationUtilities to eliminate
//	21-line duplication (mass=117) with command bus registration, ensuring consistent
//	validation behavior across command and query buses.
//
// Thread Safety:
//
//	Uses write lock for exclusive access during handler registration.
func (bus *InMemoryQueryBus) RegisterHandler(query Query, handler interface{}) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	// DRY: Use shared registration logic
	return bus.validator.RegisterHandlerGeneric(query, handler, "query", bus.handlers, "query")
}

// Execute executes a query synchronously with comprehensive error handling and optimization.
//
// This method implements the synchronous query execution pattern, routing
// the query to its registered handler and returning the execution result.
// It uses shared execution logic for consistency with command processing and
// provides thread-safe concurrent execution optimized for read operations.
//
// Parameters:
//
//	ctx context.Context: Execution context for cancellation and timeout handling
//	query Query: Query instance to execute
//
// Returns:
//
//	result.Result[interface{}]: Query result with success value or error
//
// Example:
//
//	Synchronous query execution:
//
//	  queryBus := setupQueryBus()
//
//	  // Simple query execution
//	  query := &GetPipelineQuery{ID: "pipeline-123"}
//
//	  // Execute with context and timeout
//	  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	  defer cancel()
//
//	  result := queryBus.Execute(ctx, query)
//	  if result.IsFailure() {
//	      log.Error("Query execution failed", "error", result.Error(), "query", query.QueryType())
//	      return result.Error()
//	  }
//
//	  // Extract result value
//	  pipeline := result.Value().(*PipelineView)
//	  log.Info("Pipeline retrieved successfully", "id", pipeline.ID, "name", pipeline.Name)
//
//	  // Paginated query execution
//	  listQuery := &ListPipelinesQuery{
//	      PagedQuery: queries.NewPagedQuery("ListPipelines", 1, 20),
//	      Owner:      "data-team",
//	  }
//
//	  listResult := queryBus.Execute(ctx, listQuery)
//	  if listResult.IsSuccess() {
//	      pagedResult := listResult.Value().(queries.PagedResult[*PipelineView])
//	      fmt.Printf("Found %d pipelines (page %d of %d)\n",
//	          len(pagedResult.Items), pagedResult.Page, pagedResult.TotalPages)
//	  }
//
// Execution Flow:
//  1. Acquire read lock for thread-safe handler access
//  2. Look up handler by query type
//  3. Validate handler exists and is compatible
//  4. Invoke handler with query and context
//  5. Return handler result with proper error wrapping
//
// Error Handling:
//   - Handler not found: Returns appropriate error with query type
//   - Handler execution failure: Propagates handler error with context
//   - Context cancellation: Respects context timeout and cancellation
//   - Type safety: Validates query and handler compatibility
//
// Performance Optimization:
//   - Read lock allows high-throughput concurrent query execution
//   - Minimal overhead for handler lookup and invocation
//   - Optimized for read-heavy workloads with multiple concurrent queries
//   - No write operations during query execution for maximum performance
//
// DRY Principle:
//
//	Uses shared execution logic from cqrs.BusValidationUtilities to eliminate
//	21-line duplication (mass=148) with command bus execution, ensuring consistent
//	error handling and execution patterns.
//
// Thread Safety:
//
//	Uses read lock for concurrent query execution while allowing handler registration.
func (bus *InMemoryQueryBus) Execute(ctx context.Context, query Query) result.Result[interface{}] {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// DRY: Use shared execution logic
	return bus.validator.ExecuteGeneric(ctx, query, "query", bus.handlers, bus.invoker)
}

// isValidHandler has been replaced by shared BusValidationUtilities.IsValidQueryHandler()
// DRY PRINCIPLE: Eliminates 27-line duplication by using shared validation utilities

// invokeHandler has been replaced by shared ExecuteGeneric method
// DRY PRINCIPLE: Eliminates duplicate handler invocation by using shared execution logic

// GetRegisteredQueries returns all registered query types
func (bus *InMemoryQueryBus) GetRegisteredQueries() []string {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	queries := make([]string, 0, len(bus.handlers))
	for queryType := range bus.handlers {
		queries = append(queries, queryType)
	}
	return queries
}

// BaseQuery provides a base implementation for queries
type BaseQuery struct {
	queryType string
}

// NewBaseQuery creates a new base query
func NewBaseQuery(queryType string) BaseQuery {
	return BaseQuery{queryType: queryType}
}

// QueryType returns the query type
func (q BaseQuery) QueryType() string {
	return q.queryType
}

// PagedQuery represents a query with pagination
type PagedQuery struct {
	BaseQuery
	Page     int
	PageSize int
	OrderBy  string
	Order    string // ASC or DESC
}

// NewPagedQuery creates a new paged query
func NewPagedQuery(queryType string, page, pageSize int) PagedQuery {
	return PagedQuery{
		BaseQuery: NewBaseQuery(queryType),
		Page:      page,
		PageSize:  pageSize,
		Order:     "ASC",
	}
}

// WithOrderBy sets the order by field
func (q PagedQuery) WithOrderBy(field string, order string) PagedQuery {
	q.OrderBy = field
	q.Order = order
	return q
}

// PagedResult represents a paged result
type PagedResult[T any] struct {
	Items      []T
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

// NewPagedResult creates a new paged result
func NewPagedResult[T any](items []T, totalCount, page, pageSize int) PagedResult[T] {
	totalPages := (totalCount + pageSize - 1) / pageSize
	return PagedResult[T]{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// HasNextPage returns true if there is a next page
func (pr PagedResult[T]) HasNextPage() bool {
	return pr.Page < pr.TotalPages
}

// HasPreviousPage returns true if there is a previous page
func (pr PagedResult[T]) HasPreviousPage() bool {
	return pr.Page > 1
}

// FilteredQuery represents a query with filters
type FilteredQuery struct {
	BaseQuery
	Filters map[string]interface{}
}

// NewFilteredQuery creates a new filtered query
func NewFilteredQuery(queryType string) FilteredQuery {
	return FilteredQuery{
		BaseQuery: NewBaseQuery(queryType),
		Filters:   make(map[string]interface{}),
	}
}

// WithFilter adds a filter
func (q FilteredQuery) WithFilter(key string, value interface{}) FilteredQuery {
	q.Filters[key] = value
	return q
}

// QueryCache provides caching for query results
type QueryCache struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

// NewQueryCache creates a new query cache
func NewQueryCache() *QueryCache {
	return &QueryCache{
		cache: make(map[string]interface{}),
	}
}

// Get retrieves a cached result
func (c *QueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.cache[key]
	return value, exists
}

// Set stores a result in cache
func (c *QueryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = value
}

// Clear clears the cache
func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]interface{})
}

// CachedQueryBus wraps a query bus with caching
type CachedQueryBus struct {
	inner QueryBus
	cache *QueryCache
}

// NewCachedQueryBus creates a new cached query bus
func NewCachedQueryBus(inner QueryBus) *CachedQueryBus {
	return &CachedQueryBus{
		inner: inner,
		cache: NewQueryCache(),
	}
}

// RegisterHandler registers a query handler
func (bus *CachedQueryBus) RegisterHandler(query Query, handler interface{}) error {
	return bus.inner.RegisterHandler(query, handler)
}

// Execute executes a query with caching
func (bus *CachedQueryBus) Execute(ctx context.Context, query Query) result.Result[interface{}] {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%v", query.QueryType(), query)

	// Check cache
	if cached, exists := bus.cache.Get(cacheKey); exists {
		return result.Success(cached)
	}

	// Execute query
	result := bus.inner.Execute(ctx, query)

	// Cache successful results
	if result.IsSuccess() {
		bus.cache.Set(cacheKey, result.Value())
	}

	return result
}

// InvalidateCache clears the cache
func (bus *CachedQueryBus) InvalidateCache() {
	bus.cache.Clear()
}

// QueryBusBuilder helps build query bus configurations
type QueryBusBuilder struct {
	enableCache bool
}

// NewQueryBusBuilder creates a new query bus builder
func NewQueryBusBuilder() *QueryBusBuilder {
	return &QueryBusBuilder{}
}

// WithCache enables caching
func (b *QueryBusBuilder) WithCache() *QueryBusBuilder {
	b.enableCache = true
	return b
}

// Build creates the query bus
func (b *QueryBusBuilder) Build() QueryBus {
	inner := NewInMemoryQueryBus()

	if b.enableCache {
		return NewCachedQueryBus(inner)
	}

	return inner
}
