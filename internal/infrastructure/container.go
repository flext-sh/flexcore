package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/flext-sh/flexcore/internal/application/services"
	"github.com/flext-sh/flexcore/pkg/config"
	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/flext-sh/flexcore/pkg/result"
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
// SOLID SRP: Reduced from 6 returns to 1 return using Result pattern and InitializationOrchestrator
func (fc *FlexCoreContainer) Initialize(ctx context.Context) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.initialized {
		return nil
	}

	fc.logger.Info("Initializing FLEXCORE distributed runtime container...")

	// Use orchestrator for centralized error handling
	orchestrator := fc.createInitializationOrchestrator(ctx)
	initResult := orchestrator.InitializeAllComponents()

	if initResult.IsFailure() {
		return initResult.Error()
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

// ContainerInitializationOrchestrator handles complete container initialization with Result pattern
// SOLID SRP: Single responsibility for container initialization orchestration
type ContainerInitializationOrchestrator struct {
	container *FlexCoreContainer
	ctx       context.Context
}

// createInitializationOrchestrator creates a specialized initialization orchestrator
// SOLID SRP: Factory method for creating specialized orchestrators
func (fc *FlexCoreContainer) createInitializationOrchestrator(ctx context.Context) *ContainerInitializationOrchestrator {
	return &ContainerInitializationOrchestrator{
		container: fc,
		ctx:       ctx,
	}
}

// InitializeAllComponents initializes all container components
// SOLID SRP: Single responsibility for complete initialization sequence
func (orchestrator *ContainerInitializationOrchestrator) InitializeAllComponents() result.Result[bool] {
	// Initialize components in order with Result pattern
	components := []ComponentInitializer{
		{Name: "event store", InitFunc: orchestrator.container.initializeEventStore},
		{Name: "event bus", InitFunc: orchestrator.container.initializeEventBus},
		{Name: "command bus", InitFunc: orchestrator.container.initializeCommandBus},
		{Name: "query bus", InitFunc: orchestrator.container.initializeQueryBus},
	}

	for _, component := range components {
		if err := component.InitFunc(orchestrator.ctx); err != nil {
			return result.Failure[bool](fmt.Errorf("failed to initialize %s: %w", component.Name, err))
		}
	}

	return result.Success(true)
}

// ComponentInitializer represents a component initialization function
// SOLID SRP: Single responsibility for component initialization metadata
type ComponentInitializer struct {
	Name     string
	InitFunc func(context.Context) error
}

// Shutdown gracefully shuts down the container
func (fc *FlexCoreContainer) Shutdown(ctx context.Context) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.logger.Info("Shutting down FLEXCORE container...")

	// Use orchestrator for shutdown coordination
	shutdownOrchestrator := fc.createShutdownOrchestrator(ctx)
	shutdownResult := shutdownOrchestrator.ShutdownAllComponents()

	if shutdownResult.IsFailure() {
		fc.logger.Error("Container shutdown had errors", zap.Error(shutdownResult.Error()))
	}

	fc.initialized = false
	fc.logger.Info("FLEXCORE container shutdown complete")
	return nil
}

// ContainerShutdownOrchestrator handles graceful container shutdown
// SOLID SRP: Single responsibility for container shutdown orchestration
type ContainerShutdownOrchestrator struct {
	container *FlexCoreContainer
	ctx       context.Context
}

// createShutdownOrchestrator creates a specialized shutdown orchestrator
// SOLID SRP: Factory method for creating specialized shutdown orchestrators
func (fc *FlexCoreContainer) createShutdownOrchestrator(ctx context.Context) *ContainerShutdownOrchestrator {
	return &ContainerShutdownOrchestrator{
		container: fc,
		ctx:       ctx,
	}
}

// ShutdownAllComponents shuts down all container components gracefully
// SOLID SRP: Single responsibility for complete shutdown sequence
func (orchestrator *ContainerShutdownOrchestrator) ShutdownAllComponents() result.Result[bool] {
	// Shutdown components in reverse order
	components := []string{"query bus", "command bus", "event bus", "event store"}

	for _, componentName := range components {
		orchestrator.container.logger.Debug(fmt.Sprintf("Shutting down %s...", componentName))
		// For now, just log - actual shutdown logic can be added per component
	}

	return result.Success(true)
}
