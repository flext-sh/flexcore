// Package queries provides CQRS query handling infrastructure
package queries

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
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
	mu       sync.RWMutex
	handlers map[string]interface{}
}

// NewInMemoryQueryBus creates a new in-memory query bus
func NewInMemoryQueryBus() *InMemoryQueryBus {
	return &InMemoryQueryBus{
		handlers: make(map[string]interface{}),
	}
}

// NewQueryBus creates a new query bus (returns interface)
func NewQueryBus() QueryBus {
	return NewInMemoryQueryBus()
}

// RegisterHandler registers a query handler
func (bus *InMemoryQueryBus) RegisterHandler(query Query, handler interface{}) error {
	if query == nil {
		return errors.ValidationError("query cannot be nil")
	}

	if handler == nil {
		return errors.ValidationError("handler cannot be nil")
	}

	// Validate handler implements QueryHandler interface
	handlerType := reflect.TypeOf(handler)
	if !bus.isValidHandler(handlerType) {
		return errors.ValidationError("handler must implement QueryHandler interface")
	}

	queryType := query.QueryType()
	
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.handlers[queryType]; exists {
		return errors.AlreadyExistsError("handler for query type " + queryType)
	}

	bus.handlers[queryType] = handler
	return nil
}

// Execute executes a query
func (bus *InMemoryQueryBus) Execute(ctx context.Context, query Query) result.Result[interface{}] {
	if query == nil {
		return result.Failure[interface{}](errors.ValidationError("query cannot be nil"))
	}

	bus.mu.RLock()
	handler, exists := bus.handlers[query.QueryType()]
	bus.mu.RUnlock()

	if !exists {
		return result.Failure[interface{}](errors.NotFoundError("handler for query type " + query.QueryType()))
	}

	return bus.invokeHandler(ctx, handler, query)
}

// isValidHandler checks if a type implements the QueryHandler interface
func (bus *InMemoryQueryBus) isValidHandler(handlerType reflect.Type) bool {
	// Check if it has a Handle method with correct signature
	method, exists := handlerType.MethodByName("Handle")
	if !exists {
		return false
	}

	// Check method signature: Handle(ctx context.Context, query T) result.Result[R]
	methodType := method.Type
	if methodType.NumIn() != 3 || methodType.NumOut() != 1 {
		return false
	}

	// Check context parameter
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !methodType.In(1).Implements(contextType) {
		return false
	}

	// Check query parameter implements Query interface
	queryType := reflect.TypeOf((*Query)(nil)).Elem()
	if !methodType.In(2).Implements(queryType) {
		return false
	}

	return true
}

// invokeHandler invokes a query handler using reflection
func (bus *InMemoryQueryBus) invokeHandler(ctx context.Context, handler interface{}, query Query) result.Result[interface{}] {
	handlerValue := reflect.ValueOf(handler)
	handlerType := reflect.TypeOf(handler)

	method, exists := handlerType.MethodByName("Handle")
	if !exists {
		return result.Failure[interface{}](errors.InternalError("handler does not have Handle method"))
	}

	// Prepare arguments
	args := []reflect.Value{
		handlerValue,
		reflect.ValueOf(ctx),
		reflect.ValueOf(query),
	}

	// Call the handler method
	results := method.Func.Call(args)
	if len(results) != 1 {
		return result.Failure[interface{}](errors.InternalError("handler returned unexpected number of values"))
	}

	// Extract the result
	resultValue := results[0].Interface()
	
	// Handle different result types
	switch v := resultValue.(type) {
	case result.Result[interface{}]:
		return v
	case error:
		if v != nil {
			return result.Failure[interface{}](v)
		}
		return result.Success[interface{}](nil)
	default:
		// If it's not a Result type, wrap it
		return result.Success[interface{}](v)
	}
}

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