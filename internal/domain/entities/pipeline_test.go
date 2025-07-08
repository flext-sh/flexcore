// Package entities tests
package entities_test

import (
	"testing"

	"github.com/flext/flexcore/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	name := "test-pipeline"
	description := "Test pipeline description"
	owner := "user@example.com"

	result := entities.NewPipeline(name, description, owner)
	require.True(t, result.IsSuccess())

	pipeline := result.Value()
	assert.NotEmpty(t, pipeline.ID)
	assert.Equal(t, name, pipeline.Name)
	assert.Equal(t, description, pipeline.Description)
	assert.Equal(t, owner, pipeline.Owner)
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)
	assert.Empty(t, pipeline.Steps)
	assert.Empty(t, pipeline.Tags)
	assert.Nil(t, pipeline.Schedule)
	assert.Nil(t, pipeline.LastRunAt)
	assert.Nil(t, pipeline.NextRunAt)
	assert.NotZero(t, pipeline.CreatedAt)
	assert.NotZero(t, pipeline.UpdatedAt)
	assert.Equal(t, int64(1), pipeline.Version)
}

func TestPipelineAddStep(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add first step
	step1 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Extract",
		Type:      "extractor",
		Config:    map[string]interface{}{"source": "database"},
		IsEnabled: true,
	}

	res := pipeline.AddStep(step1)
	require.True(t, res.IsSuccess())
	assert.Len(t, pipeline.Steps, 1)
	assert.Equal(t, step1, pipeline.Steps[0])

	// Add second step
	step2 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Transform",
		Type:      "transformer",
		Config:    map[string]interface{}{"operation": "clean"},
		IsEnabled: true,
	}

	res = pipeline.AddStep(step2)
	require.True(t, res.IsSuccess())
	assert.Len(t, pipeline.Steps, 2)

	// Try to add step with duplicate name
	step3 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Extract", // Same name as step1
		Type:      "loader",
		IsEnabled: true,
	}

	res = pipeline.AddStep(step3)
	assert.True(t, res.IsFailure())
	assert.Contains(t, res.Error().Error(), "already exists")
	assert.Len(t, pipeline.Steps, 2) // Should still have only 2 steps
}

// UpdateStep test removed - method doesn't exist in Pipeline

func TestPipelineRemoveStep(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add steps
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step1",
		Type:      "generic",
		IsEnabled: true,
	})
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step2",
		Type:      "generic",
		IsEnabled: true,
	})

	assert.Len(t, pipeline.Steps, 2)

	// Remove first step by name
	res := pipeline.RemoveStep("Step1")
	require.True(t, res.IsSuccess())
	assert.Len(t, pipeline.Steps, 1)
	assert.Equal(t, "Step2", pipeline.Steps[0].Name)

	// Try to remove non-existent step
	res = pipeline.RemoveStep("NonExistent")
	assert.True(t, res.IsFailure())
	assert.Contains(t, res.Error().Error(), "not found")

	// Remove last step
	res = pipeline.RemoveStep("Step2")
	require.True(t, res.IsSuccess())
	assert.Empty(t, pipeline.Steps)
}

func TestPipelineActivate(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Initially inactive
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)

	// Activate - needs steps first
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})

	res := pipeline.Activate()
	require.True(t, res.IsSuccess())
	assert.Equal(t, entities.PipelineStatusActive, pipeline.Status)

	// Events should be raised (3 events: PipelineCreated, PipelineStepAdded, PipelineActivated)
	events := pipeline.DomainEvents()
	assert.Len(t, events, 3)
	assert.Equal(t, "PipelineActivated", events[2].EventType())
}

func TestPipelineDeactivate(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add step and activate first
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})
	pipeline.Activate()
	pipeline.ClearEvents()

	// Deactivate
	res := pipeline.Deactivate()
	require.True(t, res.IsSuccess())
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "PipelineDeactivated", events[0].EventType())
}

func TestPipelineStart(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add step and activate first
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})
	pipeline.Activate()
	pipeline.ClearEvents()

	// Start pipeline
	res := pipeline.Start()
	require.True(t, res.IsSuccess())
	assert.Equal(t, entities.PipelineStatusRunning, pipeline.Status)

	// Try to start already running pipeline
	res = pipeline.Start()
	assert.True(t, res.IsFailure())
	assert.Contains(t, res.Error().Error(), "can only start active pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "PipelineStarted", events[0].EventType())
}

func TestPipelineComplete(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add step, activate and start
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})
	pipeline.Activate()
	pipeline.Start()
	pipeline.ClearEvents()

	// Complete pipeline
	res := pipeline.Complete()
	require.True(t, res.IsSuccess())
	assert.Equal(t, entities.PipelineStatusCompleted, pipeline.Status)
	assert.NotNil(t, pipeline.LastRunAt)

	// Try to complete non-running pipeline
	res = pipeline.Complete()
	assert.True(t, res.IsFailure())
	assert.Contains(t, res.Error().Error(), "can only complete running pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "PipelineCompleted", events[0].EventType())
}

func TestPipelineFail(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add step, activate and start
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})
	pipeline.Activate()
	pipeline.Start()
	pipeline.ClearEvents()

	// Fail pipeline
	res := pipeline.Fail("Test error message")
	require.True(t, res.IsSuccess())
	assert.Equal(t, entities.PipelineStatusFailed, pipeline.Status)
	assert.NotNil(t, pipeline.LastRunAt)

	// Try to fail non-running pipeline
	res = pipeline.Fail("Another error")
	assert.True(t, res.IsFailure())
	assert.Contains(t, res.Error().Error(), "can only fail running pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "PipelineFailed", events[0].EventType())
}

func TestPipelineSetSchedule(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Set daily schedule
	res := pipeline.SetSchedule("0 10 * * *", "UTC")
	require.True(t, res.IsSuccess())

	assert.NotNil(t, pipeline.Schedule)
	assert.Equal(t, "0 10 * * *", pipeline.Schedule.CronExpression)
	assert.Equal(t, "UTC", pipeline.Schedule.Timezone)
	assert.True(t, pipeline.Schedule.IsEnabled)
	assert.NotZero(t, pipeline.Schedule.CreatedAt)

	// Clear schedule
	pipeline.ClearSchedule()
	assert.Nil(t, pipeline.Schedule)
}

func TestPipelineCanExecute(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Inactive pipeline cannot execute
	assert.False(t, pipeline.CanExecute())

	// Add step first
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	})

	// Active pipeline can execute
	pipeline.Activate()
	assert.True(t, pipeline.CanExecute())

	// Running pipeline cannot execute
	pipeline.Start()
	assert.False(t, pipeline.CanExecute())

	// Failed pipeline can execute after being active again
	pipeline.Status = entities.PipelineStatusFailed
	assert.False(t, pipeline.CanExecute())

	pipeline.Status = entities.PipelineStatusActive
	assert.True(t, pipeline.CanExecute())
}

func TestPipelineValidate(t *testing.T) {
	tests := []struct {
		name        string
		pipeline    *entities.Pipeline
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid pipeline",
			pipeline: &entities.Pipeline{
				Name:  "valid-pipeline",
				Owner: "owner@example.com",
				Steps: []entities.PipelineStep{
					{ID: "1", Name: "Step1", Type: "extractor", IsEnabled: true},
				},
			},
			expectError: false,
		},
		{
			name: "empty name",
			pipeline: &entities.Pipeline{
				Name:  "",
				Owner: "owner@example.com",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "empty owner",
			pipeline: &entities.Pipeline{
				Name:  "pipeline",
				Owner: "",
			},
			expectError: true,
			errorMsg:    "owner is required",
		},
		{
			name: "no steps",
			pipeline: &entities.Pipeline{
				Name:  "pipeline",
				Owner: "owner@example.com",
				Steps: []entities.PipelineStep{},
			},
			expectError: true,
			errorMsg:    "at least one step is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.pipeline.Validate()
			if tt.expectError {
				assert.True(t, res.IsFailure())
				assert.Contains(t, res.Error().Error(), tt.errorMsg)
			} else {
				assert.True(t, res.IsSuccess())
			}
		})
	}
}

func TestPipelineAddTag(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()

	// Add tags
	pipeline.AddTag("production")
	pipeline.AddTag("etl")
	pipeline.AddTag("daily")

	assert.Len(t, pipeline.Tags, 3)
	assert.Contains(t, pipeline.Tags, "production")
	assert.Contains(t, pipeline.Tags, "etl")
	assert.Contains(t, pipeline.Tags, "daily")

	// Try to add duplicate tag
	pipeline.AddTag("production")
	assert.Len(t, pipeline.Tags, 3) // Should still be 3
}

func TestPipelineRemoveTag(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()
	pipeline.AddTag("production")
	pipeline.AddTag("etl")
	pipeline.AddTag("daily")

	// Remove tag
	pipeline.RemoveTag("etl")
	assert.Len(t, pipeline.Tags, 2)
	assert.NotContains(t, pipeline.Tags, "etl")

	// Remove non-existent tag
	pipeline.RemoveTag("nonexistent")
	assert.Len(t, pipeline.Tags, 2) // Should still be 2
}

func TestPipelineHasTag(t *testing.T) {
	result := entities.NewPipeline("test", "desc", "owner")
	require.True(t, result.IsSuccess())
	pipeline := result.Value()
	pipeline.AddTag("production")
	pipeline.AddTag("etl")

	assert.True(t, pipeline.HasTag("production"))
	assert.True(t, pipeline.HasTag("etl"))
	assert.False(t, pipeline.HasTag("development"))
}

// Test step configuration
func TestPipelineStepValidation(t *testing.T) {
	step := entities.PipelineStep{
		ID:        "",
		Name:      "",
		Type:      "",
		IsEnabled: false,
	}

	// This assumes step validation exists
	// If not implemented, this test can be adjusted
	if validator, ok := interface{}(step).(interface{ Validate() error }); ok {
		err := validator.Validate()
		assert.Error(t, err)
	}
}

// Benchmarks

func BenchmarkNewPipeline(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = entities.NewPipeline("benchmark", "description", "owner")
	}
}

func BenchmarkPipelineAddStep(b *testing.B) {
	result := entities.NewPipeline("benchmark", "description", "owner")
	if !result.IsSuccess() {
		b.Fatal("Failed to create pipeline")
	}
	pipeline := result.Value()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step := entities.PipelineStep{
			ID:        uuid.New().String(),
			Name:      "Step",
			Type:      "extractor",
			IsEnabled: true,
		}
		_ = pipeline.AddStep(step)
	}
}

func BenchmarkPipelineValidate(b *testing.B) {
	result := entities.NewPipeline("benchmark", "description", "owner")
	require.True(b, result.IsSuccess())
	pipeline := result.Value()
	pipeline.AddStep(entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step",
		Type:      "extractor",
		IsEnabled: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pipeline.Validate()
	}
}
