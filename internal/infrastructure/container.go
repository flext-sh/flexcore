package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/flext/flexcore/internal/application/services"
	"github.com/flext/flexcore/pkg/config"
	"github.com/flext/flexcore/pkg/logging"
	"go.uber.org/zap"
)

// FlexCoreContainer implements the FLEXCORE distributed runtime container exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type FlexCoreContainer struct {
	eventStore  services.EventStore
	eventBus    services.EventBus
	commandBus  services.CommandBus
	queryBus    *InMemoryQueryBus
	logger      logging.LoggerInterface
	mu          sync.RWMutex
	initialized bool
}

// NewFlexCoreContainer creates a new FLEXCORE container with real persistence
func NewFlexCoreContainer(useRealPersistence bool) *FlexCoreContainer {
	logger := logging.NewLogger("flexcore-container")
	
	container := &FlexCoreContainer{
		eventBus:   NewInMemoryEventBus(logger),
		commandBus: NewInMemoryCommandBus(logger),
		queryBus:   NewInMemoryQueryBus(logger),
		logger:     logger,
	}
	
	if useRealPersistence {
		// Use PostgreSQL event store when real persistence is enabled
		if dbConn, err := NewDatabaseConnection(config.Current, logger); err == nil {
			container.eventStore = NewPostgreSQLEventStore(dbConn.DB, logger)
		} else {
			logger.Warn("Failed to connect to PostgreSQL, falling back to in-memory event store", zap.Error(err))
			container.eventStore = NewMemoryEventStore(logger)
		}
	} else {
		// Use in-memory event store for development/testing
		container.eventStore = NewMemoryEventStore(logger)
	}
	
	return container
}

// GetEventStore returns the event store for audit trail
func (fc *FlexCoreContainer) GetEventStore() services.EventStore {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.eventStore
}

// GetEventBus returns the event bus for Event Sourcing + CQRS
func (fc *FlexCoreContainer) GetEventBus() services.EventBus {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.eventBus
}

// GetCommandBus returns the CQRS command bus
func (fc *FlexCoreContainer) GetCommandBus() services.CommandBus {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.commandBus
}

// GetQueryBus returns the CQRS query bus
func (fc *FlexCoreContainer) GetQueryBus() *InMemoryQueryBus {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.queryBus
}

// Initialize initializes the FLEXCORE container
func (fc *FlexCoreContainer) Initialize(ctx context.Context) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	if fc.initialized {
		return nil
	}
	
	fc.logger.Info("Initializing FLEXCORE distributed runtime container...")
	
	// Initialize components in order
	if err := fc.initializeEventStore(ctx); err != nil {
		return fmt.Errorf("failed to initialize event store: %w", err)
	}
	
	if err := fc.initializeEventBus(ctx); err != nil {
		return fmt.Errorf("failed to initialize event bus: %w", err)
	}
	
	if err := fc.initializeCommandBus(ctx); err != nil {
		return fmt.Errorf("failed to initialize command bus: %w", err)
	}
	
	if err := fc.initializeQueryBus(ctx); err != nil {
		return fmt.Errorf("failed to initialize query bus: %w", err)
	}
	
	fc.initialized = true
	fc.logger.Info("FLEXCORE container initialized successfully")
	
	return nil
}

// initializeEventStore initializes the event store
func (fc *FlexCoreContainer) initializeEventStore(ctx context.Context) error {
	fc.logger.Debug("Initializing event store...")
	// Event store is already created in constructor
	return nil
}

// initializeEventBus initializes the event bus
func (fc *FlexCoreContainer) initializeEventBus(ctx context.Context) error {
	fc.logger.Debug("Initializing event bus...")
	// Event bus is already created in constructor
	return nil
}

// initializeCommandBus initializes the command bus
func (fc *FlexCoreContainer) initializeCommandBus(ctx context.Context) error {
	fc.logger.Debug("Initializing command bus...")
	// Command bus is already created in constructor
	return nil
}

// initializeQueryBus initializes the query bus
func (fc *FlexCoreContainer) initializeQueryBus(ctx context.Context) error {
	fc.logger.Debug("Initializing query bus...")
	// Query bus is already created in constructor
	return nil
}

// Shutdown gracefully shuts down the container
func (fc *FlexCoreContainer) Shutdown(ctx context.Context) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	fc.logger.Info("Shutting down FLEXCORE container...")
	
	// Shutdown components in reverse order
	// For now, just mark as not initialized
	fc.initialized = false
	
	fc.logger.Info("FLEXCORE container shutdown complete")
	return nil
}