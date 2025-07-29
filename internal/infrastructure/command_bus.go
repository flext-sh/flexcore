package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// CommandHandler represents a function that handles a specific command
type CommandHandler func(context.Context, interface{}) error

// InMemoryCommandBus implements an in-memory command bus for CQRS exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type InMemoryCommandBus struct {
	handlers map[string]CommandHandler
	mu       sync.RWMutex
	logger   logging.LoggerInterface
}

// NewInMemoryCommandBus creates a new in-memory command bus
func NewInMemoryCommandBus(logger logging.LoggerInterface) *InMemoryCommandBus {
	return &InMemoryCommandBus{
		handlers: make(map[string]CommandHandler),
		logger:   logger,
	}
}

// Send sends a command to its registered handler exactly as specified in the architecture document
func (cb *InMemoryCommandBus) Send(ctx context.Context, command interface{}) error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	commandType := fmt.Sprintf("%T", command)
	handler, exists := cb.handlers[commandType]

	if !exists {
		return fmt.Errorf("no handler registered for command type: %s", commandType)
	}

	cb.logger.Debug("Executing command",
		zap.String("command_type", commandType))

	// Execute the command handler
	if err := handler(ctx, command); err != nil {
		cb.logger.Error("Command handler failed",
			zap.String("command_type", commandType),
			zap.Error(err))
		return fmt.Errorf("command handler failed for %s: %w", commandType, err)
	}

	cb.logger.Debug("Command executed successfully", zap.String("command_type", commandType))
	return nil
}

// RegisterHandler registers a handler for a specific command type
func (cb *InMemoryCommandBus) RegisterHandler(commandType string, handler CommandHandler) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if _, exists := cb.handlers[commandType]; exists {
		cb.logger.Warn("Overriding existing handler", zap.String("command_type", commandType))
	}

	cb.handlers[commandType] = handler
	cb.logger.Debug("Command handler registered", zap.String("command_type", commandType))
}

// QueryHandler represents a function that handles a specific query
type QueryHandler func(context.Context, interface{}) (interface{}, error)

// InMemoryQueryBus implements an in-memory query bus for CQRS exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type InMemoryQueryBus struct {
	handlers map[string]QueryHandler
	mu       sync.RWMutex
	logger   logging.LoggerInterface
}

// NewInMemoryQueryBus creates a new in-memory query bus
func NewInMemoryQueryBus(logger logging.LoggerInterface) *InMemoryQueryBus {
	return &InMemoryQueryBus{
		handlers: make(map[string]QueryHandler),
		logger:   logger,
	}
}

// Send sends a query to its registered handler exactly as specified in the architecture document
func (qb *InMemoryQueryBus) Send(ctx context.Context, query interface{}) (interface{}, error) {
	qb.mu.RLock()
	defer qb.mu.RUnlock()

	queryType := fmt.Sprintf("%T", query)
	handler, exists := qb.handlers[queryType]

	if !exists {
		return nil, fmt.Errorf("no handler registered for query type: %s", queryType)
	}

	qb.logger.Debug("Executing query",
		zap.String("query_type", queryType))

	// Execute the query handler
	result, err := handler(ctx, query)
	if err != nil {
		qb.logger.Error("Query handler failed",
			zap.String("query_type", queryType),
			zap.Error(err))
		return nil, fmt.Errorf("query handler failed for %s: %w", queryType, err)
	}

	qb.logger.Debug("Query executed successfully", zap.String("query_type", queryType))
	return result, nil
}

// RegisterHandler registers a handler for a specific query type
func (qb *InMemoryQueryBus) RegisterHandler(queryType string, handler QueryHandler) {
	qb.mu.Lock()
	defer qb.mu.Unlock()

	if _, exists := qb.handlers[queryType]; exists {
		qb.logger.Warn("Overriding existing handler", zap.String("query_type", queryType))
	}

	qb.handlers[queryType] = handler
	qb.logger.Debug("Query handler registered", zap.String("query_type", queryType))
}
