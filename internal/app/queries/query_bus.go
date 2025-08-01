// Package queries provides CQRS query handling infrastructure
package queries

import (
	"context"
	"fmt"
	"sync"

	"github.com/flext/flexcore/pkg/cqrs"
	"github.com/flext/flexcore/pkg/result"
)

// Query represents a query in the CQRS pattern
type Query interface {
	QueryType() string
}

// QueryHandler represents a handler for a specific query type
type QueryHandler[T Query, R any] interface {
	Handle(ctx context.Context, query T) result.Result[R]
}

// QueryBus coordinates query execution
type QueryBus interface {
	RegisterHandler(query Query, handler interface{}) error
	Execute(ctx context.Context, query Query) result.Result[interface{}]
}

// InMemoryQueryBus provides an in-memory implementation of QueryBus
type InMemoryQueryBus struct {
	mu        sync.RWMutex
	handlers  map[string]interface{}
	validator *cqrs.BusValidationUtilities
	invoker   *cqrs.HandlerInvoker
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

// RegisterHandler registers a query handler
// DRY PRINCIPLE: Uses shared registration logic eliminating 21-line duplication (mass=117)
func (bus *InMemoryQueryBus) RegisterHandler(query Query, handler interface{}) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	// DRY: Use shared registration logic
	return bus.validator.RegisterHandlerGeneric(query, handler, "query", bus.handlers, "query")
}

// Execute executes a query synchronously
// DRY PRINCIPLE: Uses shared execution logic eliminating 21-line duplication (mass=148)
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
