package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlexCore(t *testing.T) {
	tests := []struct {
		name    string
		config  *FlexCoreConfig
		wantErr bool
	}{
		{
			name:    "nil config should fail",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config should succeed",
			config:  DefaultConfig(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewFlexCore(tt.config)
			
			if tt.wantErr {
				assert.True(t, result.IsFailure())
				assert.NotNil(t, result.Error())
			} else {
				assert.True(t, result.IsSuccess())
				assert.NotNil(t, result.Value())
			}
		})
	}
}

func TestFlexCore_StartStop(t *testing.T) {
	config := DefaultConfig()
	result := NewFlexCore(config)
	require.True(t, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()

	// Test start
	startResult := core.Start(ctx)
	assert.True(t, startResult.IsSuccess())

	// Test double start should fail
	doubleStartResult := core.Start(ctx)
	assert.True(t, doubleStartResult.IsFailure())

	// Test stop
	stopResult := core.Stop(ctx)
	assert.True(t, stopResult.IsSuccess())

	// Test double stop should succeed
	doubleStopResult := core.Stop(ctx)
	assert.True(t, doubleStopResult.IsSuccess())
}

func TestFlexCore_SendEvent(t *testing.T) {
	config := DefaultConfig()
	result := NewFlexCore(config)
	require.True(t, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()

	// Start core
	startResult := core.Start(ctx)
	require.True(t, startResult.IsSuccess())
	defer core.Stop(ctx)

	tests := []struct {
		name    string
		event   *Event
		wantErr bool
	}{
		{
			name:    "nil event should fail",
			event:   nil,
			wantErr: true,
		},
		{
			name: "valid event should succeed",
			event: NewEvent("test-event", "test-aggregate", map[string]interface{}{
				"data": "test",
			}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.SendEvent(ctx, tt.event)
			
			if tt.wantErr {
				assert.True(t, result.IsFailure())
			} else {
				assert.True(t, result.IsSuccess())
			}
		})
	}
}

func TestFlexCore_SendReceiveMessage(t *testing.T) {
	config := DefaultConfig()
	result := NewFlexCore(config)
	require.True(t, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()

	// Start core
	startResult := core.Start(ctx)
	require.True(t, startResult.IsSuccess())
	defer core.Stop(ctx)

	// Test send message
	message := NewMessage("test-queue", map[string]interface{}{
		"test": "data",
	})

	sendResult := core.SendMessage(ctx, "test-queue", message)
	assert.True(t, sendResult.IsSuccess())

	// Test receive messages
	receiveResult := core.ReceiveMessages(ctx, "test-queue", 10)
	assert.True(t, receiveResult.IsSuccess())
	
	messages := receiveResult.Value()
	assert.Len(t, messages, 1)
	assert.Equal(t, message.ID, messages[0].ID)
}

func TestFlexCore_ExecuteWorkflow(t *testing.T) {
	config := DefaultConfig()
	result := NewFlexCore(config)
	require.True(t, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()

	// Execute workflow
	executeResult := core.ExecuteWorkflow(ctx, "test-workflow", map[string]interface{}{
		"input": "test",
	})
	
	assert.True(t, executeResult.IsSuccess())
	jobID := executeResult.Value()
	assert.NotEmpty(t, jobID)

	// Get workflow status
	statusResult := core.GetWorkflowStatus(ctx, jobID)
	assert.True(t, statusResult.IsSuccess())
	
	status := statusResult.Value()
	assert.Equal(t, "test-workflow", status["path"])
	assert.Equal(t, "running", status["status"])
}

func TestFlexCore_ClusterStatus(t *testing.T) {
	config := DefaultConfig()
	config.ClusterName = "test-cluster"
	config.NodeID = "test-node"
	
	result := NewFlexCore(config)
	require.True(t, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()

	statusResult := core.GetClusterStatus(ctx)
	assert.True(t, statusResult.IsSuccess())
	
	status := statusResult.Value()
	assert.Equal(t, "test-cluster", status["cluster_name"])
	assert.Equal(t, "test-node", status["node_id"])
	assert.Equal(t, "healthy", status["status"])
}

func TestEvent_Creation(t *testing.T) {
	event := NewEvent("test-event", "test-aggregate", map[string]interface{}{
		"key": "value",
	})

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "test-event", event.Type)
	assert.Equal(t, "test-aggregate", event.AggregateID)
	assert.Equal(t, "value", event.Data["key"])
	assert.Equal(t, 1, event.Version)
	assert.NotZero(t, event.Timestamp)
}

func TestMessage_Creation(t *testing.T) {
	message := NewMessage("test-queue", map[string]interface{}{
		"key": "value",
	})

	assert.NotEmpty(t, message.ID)
	assert.Equal(t, "test-queue", message.Queue)
	assert.Equal(t, "value", message.Data["key"])
	assert.Equal(t, "pending", message.Status)
	assert.Equal(t, 0, message.Attempts)
	assert.NotZero(t, message.CreatedAt)
}

func TestResult_Success(t *testing.T) {
	result := NewSuccess("test-value")
	
	assert.True(t, result.IsSuccess())
	assert.False(t, result.IsFailure())
	assert.Equal(t, "test-value", result.Value())
	assert.Nil(t, result.Error())
}

func TestResult_Failure(t *testing.T) {
	err := assert.AnError
	result := NewFailure[string](err)
	
	assert.False(t, result.IsSuccess())
	assert.True(t, result.IsFailure())
	assert.Equal(t, err, result.Error())
	
	// Value() should panic on failure
	assert.Panics(t, func() {
		result.Value()
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, "http://localhost:8000", config.WindmillURL)
	assert.Equal(t, "default", config.WindmillWorkspace)
	assert.Equal(t, "flexcore-cluster", config.ClusterName)
	assert.NotEmpty(t, config.NodeID)
	assert.Equal(t, "./plugins", config.PluginDirectory)
	assert.Equal(t, 100, config.MaxConcurrentJobs)
	assert.Equal(t, 10000, config.EventBufferSize)
}

// Benchmarks
func BenchmarkFlexCore_SendEvent(b *testing.B) {
	config := DefaultConfig()
	result := NewFlexCore(config)
	require.True(b, result.IsSuccess())
	
	core := result.Value()
	ctx := context.Background()
	
	startResult := core.Start(ctx)
	require.True(b, startResult.IsSuccess())
	defer core.Stop(ctx)

	event := NewEvent("benchmark-event", "benchmark-aggregate", map[string]interface{}{
		"data": "benchmark",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.SendEvent(ctx, event)
	}
}

func BenchmarkEvent_Creation(b *testing.B) {
	data := map[string]interface{}{
		"benchmark": "data",
		"number":    42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewEvent("benchmark-event", "benchmark-aggregate", data)
	}
}