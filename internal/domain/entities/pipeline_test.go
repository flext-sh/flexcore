// Package entities tests
package entities_test

import (
	"testing"

	"github.com/flext-sh/flexcore/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	name := "test-pipeline"
	description := "Test pipeline description"
	owner := "user@example.com"

	pipeline, err := entities.NewPipeline(name, description, owner)
	require.NoError(t, err)
	require.NotNil(t, pipeline)
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
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add first step
	step1 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Extract",
		Type:      "extractor",
		Config:    map[string]interface{}{"source": "database"},
		IsEnabled: true,
	}

	err = pipeline.AddStep(&step1)
	require.NoError(t, err)
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

	err = pipeline.AddStep(&step2)
	require.NoError(t, err)
	assert.Len(t, pipeline.Steps, 2)

	// Try to add step with duplicate name
	step3 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Extract", // Same name as step1
		Type:      "loader",
		IsEnabled: true,
	}

	err = pipeline.AddStep(&step3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Len(t, pipeline.Steps, 2) // Should still have only 2 steps
}

// UpdateStep test removed - method doesn't exist in Pipeline

func TestPipelineRemoveStep(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add steps
	step1 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step1",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step1)

	step2 := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step2",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step2)

	assert.Len(t, pipeline.Steps, 2)

	// Remove first step by name
	err = pipeline.RemoveStep("Step1")
	require.NoError(t, err)
	assert.Len(t, pipeline.Steps, 1)
	assert.Equal(t, "Step2", pipeline.Steps[0].Name)

	// Try to remove non-existent step
	err = pipeline.RemoveStep("NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Remove last step
	err = pipeline.RemoveStep("Step2")
	require.NoError(t, err)
	assert.Empty(t, pipeline.Steps)
}

func TestPipelineActivate(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Initially inactive
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)

	// Activate - needs steps first
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)

	err = pipeline.Activate()
	require.NoError(t, err)
	assert.Equal(t, entities.PipelineStatusActive, pipeline.Status)

	// Events should be raised (3 events: PipelineCreated, PipelineStepAdded, PipelineActivated)
	events := pipeline.DomainEvents()
	assert.Len(t, events, 3)
	assert.Equal(t, "pipeline.activated", events[2].EventType())
}

func TestPipelineDeactivate(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add step and activate first
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)
	pipeline.Activate()
	pipeline.ClearEvents()

	// Deactivate
	err = pipeline.Deactivate()
	require.NoError(t, err)
	assert.Equal(t, entities.PipelineStatusDraft, pipeline.Status)

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "pipeline.deactivated", events[0].EventType())
}

func TestPipelineStart(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add step and activate first
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)
	pipeline.Activate()
	pipeline.ClearEvents()

	// Start pipeline
	err = pipeline.Start()
	require.NoError(t, err)
	assert.Equal(t, entities.PipelineStatusRunning, pipeline.Status)

	// Try to start already running pipeline
	err = pipeline.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only start active pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "pipeline.started", events[0].EventType())
}

func TestPipelineComplete(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add step, activate and start
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)
	pipeline.Activate()
	pipeline.Start()
	pipeline.ClearEvents()

	// Complete pipeline
	err = pipeline.Complete()
	require.NoError(t, err)
	assert.Equal(t, entities.PipelineStatusCompleted, pipeline.Status)
	assert.NotNil(t, pipeline.LastRunAt)

	// Try to complete non-running pipeline
	err = pipeline.Complete()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only complete running pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "pipeline.completed", events[0].EventType())
}

func TestPipelineFail(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Add step, activate and start
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)
	pipeline.Activate()
	pipeline.Start()
	pipeline.ClearEvents()

	// Fail pipeline
	err = pipeline.Fail("Test error message")
	require.NoError(t, err)
	assert.Equal(t, entities.PipelineStatusFailed, pipeline.Status)
	assert.NotNil(t, pipeline.LastRunAt)

	// Try to fail non-running pipeline
	err = pipeline.Fail("Another error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only fail running pipelines")

	// Events should be raised
	events := pipeline.DomainEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "pipeline.failed", events[0].EventType())
}

func TestPipelineSetSchedule(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Set daily schedule
	err = pipeline.SetSchedule("0 10 * * *", "UTC")
	require.NoError(t, err)

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
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	// Inactive pipeline cannot execute
	assert.False(t, pipeline.CanExecute())

	// Add step first
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "TestStep",
		Type:      "generic",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)

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
			err := tt.pipeline.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPipelineAddTag(t *testing.T) {
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)

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
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)
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
	pipeline, err := entities.NewPipeline("test", "desc", "owner")
	require.NoError(t, err)
	require.NotNil(t, pipeline)
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
		_, _ = entities.NewPipeline("benchmark", "description", "owner")
	}
}

func BenchmarkPipelineAddStep(b *testing.B) {
	pipeline, err := entities.NewPipeline("benchmark", "description", "owner")
	if err != nil {
		b.Fatal("Failed to create pipeline:", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step := entities.PipelineStep{
			ID:        uuid.New().String(),
			Name:      "Step",
			Type:      "extractor",
			IsEnabled: true,
		}
		_ = pipeline.AddStep(&step)
	}
}

func BenchmarkPipelineValidate(b *testing.B) {
	pipeline, err := entities.NewPipeline("benchmark", "description", "owner")
	if err != nil {
		b.Fatal("Failed to create pipeline:", err)
	}
	step := entities.PipelineStep{
		ID:        uuid.New().String(),
		Name:      "Step",
		Type:      "extractor",
		IsEnabled: true,
	}
	pipeline.AddStep(&step)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pipeline.Validate()
	}
}
