// Package flexcore integration tests
package flexcore_test

import (
	"context"
	"testing"
	"time"

	"github.com/flext/flexcore/application/commands"
	"github.com/flext/flexcore/application/queries"
	"github.com/flext/flexcore/domain"
	"github.com/flext/flexcore/domain/entities"
	"github.com/flext/flexcore/infrastructure/di"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/shared/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicIntegration(t *testing.T) {
	// Test basic component integration
	container := di.NewContainer()
	
	// Register components using correct API
	container.RegisterSingleton(func() *events.InMemoryEventBus {
		return events.NewInMemoryEventBus(100, 10) // buffered and concurrent workers
	})
	
	container.RegisterSingleton(func() commands.CommandBus {
		return commands.NewCommandBus()
	})
	
	// Test resolution using generic Resolve
	eventBusResult := di.Resolve[*events.InMemoryEventBus](container)
	require.True(t, eventBusResult.IsSuccess())
	
	commandBusResult := di.Resolve[commands.CommandBus](container)
	require.True(t, commandBusResult.IsSuccess())
	
	// Test event bus
	eventBus := eventBusResult.Value()
	assert.NotNil(t, eventBus)
	
	// Test command bus
	commandBus := commandBusResult.Value()
	assert.NotNil(t, commandBus)
}

func TestDomainEntities(t *testing.T) {
	// Test Pipeline entity with correct parameters
	pipelineResult := entities.NewPipeline("Test Pipeline", "A test pipeline", "test-owner")
	require.True(t, pipelineResult.IsSuccess())
	
	pipeline := pipelineResult.Value()
	assert.Equal(t, "Test Pipeline", pipeline.Name)
	assert.Equal(t, "A test pipeline", pipeline.Description)
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)
	
	// Test Pipeline activation
	activateResult := pipeline.Activate()
	require.True(t, activateResult.IsSuccess())
	assert.Equal(t, entities.PipelineStatusActive, pipeline.Status)
	
	// Test Plugin entity
	pluginResult := entities.NewPlugin("test-plugin", "1.0.0", "Test plugin", entities.PluginTypeExtractor)
	require.True(t, pluginResult.IsSuccess())
	
	plugin := pluginResult.Value()
	assert.Equal(t, "test-plugin", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Equal(t, entities.PluginTypeExtractor, plugin.Type())
	assert.Equal(t, entities.PluginStatusRegistered, plugin.Status())
	
	// Test Plugin activation
	activatePluginResult := plugin.Activate()
	require.True(t, activatePluginResult.IsSuccess())
	assert.Equal(t, entities.PluginStatusActive, plugin.Status())
}

func TestEventSystem(t *testing.T) {
	// Test event publication and subscription
	eventBus := events.NewInMemoryEventBus(10, 100) // 10 workers, 100 buffer
	
	received := make(chan string, 1)
	
	// Subscribe to events
	err := eventBus.Subscribe("test.event", func(ctx context.Context, event domain.DomainEvent) error {
		received <- event.EventType()
		return nil
	})
	require.NoError(t, err)
	
	// Start the event bus
	ctx := context.Background()
	err = eventBus.Start(ctx)
	require.NoError(t, err)
	defer eventBus.Stop()
	
	// Create and publish event
	event := domain.NewBaseDomainEvent("test.event", "test-aggregate")
	
	err = eventBus.Publish(ctx, event)
	require.NoError(t, err)
	
	// Check event was received
	select {
	case eventType := <-received:
		assert.Equal(t, "test.event", eventType)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Event not received within timeout")
	}
}

func TestDependencyInjection(t *testing.T) {
	container := di.NewContainer()
	
	// Test singleton registration
	container.RegisterSingleton(func() string {
		return "singleton-value"
	})
	
	// Test factory registration (transient)
	container.Register(func() string {
		return "factory-value"
	})
	
	// Test instance registration
	container.RegisterInstance("static-value")
	
	// Test resolution using generics
	singletonResult1 := di.Resolve[string](container)
	require.True(t, singletonResult1.IsSuccess())
	
	singletonResult2 := di.Resolve[string](container)
	require.True(t, singletonResult2.IsSuccess())
	
	// Should get the same value since string is the registered type
	assert.Equal(t, singletonResult1.Value(), singletonResult2.Value())
}

// Test command and handler types
type TestCommand struct {
	Name string
}

func (c TestCommand) CommandType() string {
	return "test.command"
}

type TestCommandHandler struct{}

func (h *TestCommandHandler) Handle(ctx context.Context, cmd TestCommand) result.Result[interface{}] {
	return result.Success[interface{}]("handled: " + cmd.Name)
}

func TestCommandHandling(t *testing.T) {
	
	// Setup
	commandBus := commands.NewCommandBus()
	handler := &TestCommandHandler{}
	
	// Register handler
	err := commandBus.RegisterHandler(TestCommand{}, handler)
	require.NoError(t, err)
	
	// Execute command
	cmd := TestCommand{Name: "test"}
	result := commandBus.Execute(context.Background(), cmd)
	
	require.True(t, result.IsSuccess())
	assert.Equal(t, "handled: test", result.Value())
}

func TestQueryHandling(t *testing.T) {
	// Test query system
	queryBus := queries.NewQueryBus()
	assert.NotNil(t, queryBus)
	
	// Basic query bus functionality
	// Note: Specific query tests would depend on query implementations
}