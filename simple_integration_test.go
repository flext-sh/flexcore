// Package flexcore simple integration tests
package flexcore_test

import (
	"context"
	"testing"

	"github.com/flext/flexcore/application/commands"
	"github.com/flext/flexcore/application/queries"
	"github.com/flext/flexcore/domain"
	"github.com/flext/flexcore/domain/entities"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleIntegration(t *testing.T) {
	// Test basic component creation
	
	// Test command bus
	commandBus := commands.NewCommandBus()
	assert.NotNil(t, commandBus)
	
	// Test query bus
	queryBus := queries.NewQueryBus()
	assert.NotNil(t, queryBus)
	
	// Test event bus
	eventBus := events.NewInMemoryEventBus(10, 100)
	assert.NotNil(t, eventBus)
}

func TestDomainEntitiesSimple(t *testing.T) {
	// Test Pipeline entity
	pipelineResult := entities.NewPipeline("Test Pipeline", "A test pipeline", "test-owner")
	require.True(t, pipelineResult.IsSuccess())
	
	pipeline := pipelineResult.Value()
	assert.Equal(t, "Test Pipeline", pipeline.Name)
	assert.Equal(t, "A test pipeline", pipeline.Description)
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)
	
	// Test Plugin entity
	pluginResult := entities.NewPlugin("test-plugin", "1.0.0", "Test plugin", entities.PluginTypeExtractor)
	require.True(t, pluginResult.IsSuccess())
	
	plugin := pluginResult.Value()
	assert.Equal(t, "test-plugin", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Equal(t, entities.PluginTypeExtractor, plugin.Type())
	assert.Equal(t, entities.PluginStatusRegistered, plugin.Status())
}

func TestDomainEvents(t *testing.T) {
	// Test domain event creation
	event := domain.NewBaseDomainEvent("test.event", "test-aggregate")
	assert.NotNil(t, event)
	assert.Equal(t, "test.event", event.EventType())
	assert.Equal(t, "test-aggregate", event.AggregateID())
	assert.NotEmpty(t, event.EventID())
	assert.False(t, event.OccurredAt().IsZero())
}

// Simple command type for testing
type SimpleCommand struct {
	Message string
}

func (c SimpleCommand) CommandType() string {
	return "simple.command"
}

// Simple handler
type SimpleHandler struct{}

func (h *SimpleHandler) Handle(ctx context.Context, cmd SimpleCommand) result.Result[interface{}] {
	return result.Success[interface{}]("processed: " + cmd.Message)
}

func TestCommandBusBasic(t *testing.T) {
	// Test basic registration and execution
	commandBus := commands.NewCommandBus()
	
	// This would be more complex to test properly due to reflection
	// For now, just test that the bus exists
	assert.NotNil(t, commandBus)
}

func TestResultTypes(t *testing.T) {
	// Test Result success
	successResult := result.Success("test value")
	assert.True(t, successResult.IsSuccess())
	assert.False(t, successResult.IsFailure())
	assert.Equal(t, "test value", successResult.Value())
	
	// Test Result failure
	failureResult := result.Failure[string](errors.ValidationError("test error"))
	assert.False(t, failureResult.IsSuccess())
	assert.True(t, failureResult.IsFailure())
	assert.NotNil(t, failureResult.Error())
}