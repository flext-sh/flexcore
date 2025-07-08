// Package commands tests
package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flext/flexcore/internal/app/commands"
	"github.com/flext/flexcore/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test command
type TestCommand struct {
	commands.BaseCommand
	Value string
}

func NewTestCommand(value string) TestCommand {
	return TestCommand{
		BaseCommand: commands.NewBaseCommand("TestCommand"),
		Value:       value,
	}
}

// Test command with result
type CalculateCommand struct {
	commands.BaseCommand
	A, B int
}

func NewCalculateCommand(a, b int) CalculateCommand {
	return CalculateCommand{
		BaseCommand: commands.NewBaseCommand("CalculateCommand"),
		A:           a,
		B:           b,
	}
}

// Test handlers
type TestCommandHandler struct {
	executedCommands []TestCommand
	shouldError      bool
}

func NewTestCommandHandler() *TestCommandHandler {
	return &TestCommandHandler{
		executedCommands: make([]TestCommand, 0),
	}
}

func (h *TestCommandHandler) Handle(ctx context.Context, cmd TestCommand) result.Result[interface{}] {
	if h.shouldError {
		return result.Failure[interface{}](errors.New("handler error"))
	}
	h.executedCommands = append(h.executedCommands, cmd)
	return result.Success[interface{}]("success")
}

type CalculateCommandHandler struct{}

func (h *CalculateCommandHandler) Handle(ctx context.Context, cmd CalculateCommand) result.Result[interface{}] {
	if cmd.B == 0 {
		return result.Failure[interface{}](errors.New("division by zero"))
	}
	return result.Success[interface{}](cmd.A / cmd.B)
}

// Tests

func TestCommandBusRegisterAndExecute(t *testing.T) {
	bus := commands.NewCommandBus()
	handler := NewTestCommandHandler()

	// Register handler
	cmd := NewTestCommand("test value")
	err := bus.RegisterHandler(cmd, handler)
	require.NoError(t, err)

	// Execute command
	result := bus.Execute(context.Background(), cmd)

	require.True(t, result.IsSuccess())
	assert.Equal(t, "success", result.Value())
	assert.Len(t, handler.executedCommands, 1)
	assert.Equal(t, cmd.Value, handler.executedCommands[0].Value)
}

func TestCommandBusHandlerNotFound(t *testing.T) {
	bus := commands.NewCommandBus()

	// Try to handle unregistered command
	cmd := NewTestCommand("test")
	result := bus.Execute(context.Background(), cmd)

	assert.True(t, result.IsFailure())
	assert.Contains(t, result.Error().Error(), "TestCommand")
}

func TestCommandBusHandlerError(t *testing.T) {
	bus := commands.NewCommandBus()
	handler := NewTestCommandHandler()
	handler.shouldError = true

	// Register handler that returns error
	cmd := NewTestCommand("test")
	err := bus.RegisterHandler(cmd, handler)
	require.NoError(t, err)

	// Execute command
	result := bus.Execute(context.Background(), cmd)

	assert.True(t, result.IsFailure())
	assert.Equal(t, "handler error", result.Error().Error())
}

func TestCommandBusWithTypedResult(t *testing.T) {
	bus := commands.NewCommandBus()
	handler := &CalculateCommandHandler{}

	// Register handler
	cmd1 := NewCalculateCommand(10, 2)
	err := bus.RegisterHandler(cmd1, handler)
	require.NoError(t, err)

	// Test successful calculation
	result1 := bus.Execute(context.Background(), cmd1)

	require.True(t, result1.IsSuccess())
	assert.Equal(t, 5, result1.Value())

	// Test division by zero
	cmd2 := NewCalculateCommand(10, 0)
	result2 := bus.Execute(context.Background(), cmd2)

	assert.True(t, result2.IsFailure())
	assert.Contains(t, result2.Error().Error(), "division by zero")
}

func TestCommandBusExecuteAsync(t *testing.T) {
	bus := commands.NewCommandBus()
	handler := NewTestCommandHandler()

	// Register handler
	cmd := NewTestCommand("async test")
	err := bus.RegisterHandler(cmd, handler)
	require.NoError(t, err)

	// Execute command asynchronously
	resultChan := bus.ExecuteAsync(context.Background(), cmd)

	require.True(t, resultChan.IsSuccess())

	// Get result from channel
	asyncResult := <-resultChan.Value()

	require.True(t, asyncResult.IsSuccess())
	assert.Equal(t, "success", asyncResult.Value())
	assert.Len(t, handler.executedCommands, 1)
}

func TestCommandBusBuilderPattern(t *testing.T) {
	// Test that we can create command bus using builder
	bus := commands.NewCommandBusBuilder().Build()

	handler := NewTestCommandHandler()
	cmd := NewTestCommand("builder test")

	err := bus.RegisterHandler(cmd, handler)
	require.NoError(t, err)

	result := bus.Execute(context.Background(), cmd)

	require.True(t, result.IsSuccess())
	assert.Equal(t, "success", result.Value())
}
