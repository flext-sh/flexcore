// Package memory provides in-memory persistence adapters for FlexCore
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flext-sh/flexcore/internal/domain/entities"
	"github.com/flext-sh/flexcore/pkg/result"
)

// PipelineRepository provides in-memory storage for pipelines
type PipelineRepository struct {
	mu        sync.RWMutex
	pipelines map[string]*entities.Pipeline
}

// NewPipelineRepository creates a new in-memory pipeline repository
func NewPipelineRepository() *PipelineRepository {
	return &PipelineRepository{
		pipelines: make(map[string]*entities.Pipeline),
	}
}

// Save saves a pipeline to memory
func (r *PipelineRepository) Save(ctx context.Context, pipeline *entities.Pipeline) result.Result[*entities.Pipeline] {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pipeline == nil {
		return result.Failure[*entities.Pipeline](fmt.Errorf("pipeline cannot be nil"))
	}

	// Clone the pipeline to avoid external mutations
	cloned := &entities.Pipeline{
		AggregateRoot: pipeline.AggregateRoot,
		Name:          pipeline.Name,
		Description:   pipeline.Description,
		Status:        pipeline.Status,
		Steps:         pipeline.Steps,
		Tags:          pipeline.Tags,
	}
	cloned.UpdatedAt = time.Now()

	r.pipelines[pipeline.ID.String()] = cloned
	return result.Success(cloned)
}

// FindByID finds a pipeline by ID
func (r *PipelineRepository) FindByID(ctx context.Context, id string) result.Result[*entities.Pipeline] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pipeline, exists := r.pipelines[id]
	if !exists {
		return result.Failure[*entities.Pipeline](fmt.Errorf("pipeline with ID %s not found", id))
	}

	// Clone to avoid external mutations
	cloned := &entities.Pipeline{
		AggregateRoot: pipeline.AggregateRoot,
		Name:          pipeline.Name,
		Description:   pipeline.Description,
		Status:        pipeline.Status,
		Steps:         pipeline.Steps,
		Tags:          pipeline.Tags,
	}

	return result.Success(cloned)
}

// FindAll finds all pipelines
func (r *PipelineRepository) FindAll(ctx context.Context) result.Result[[]*entities.Pipeline] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pipelines := make([]*entities.Pipeline, 0, len(r.pipelines))
	for _, pipeline := range r.pipelines {
		// Clone each pipeline
		cloned := &entities.Pipeline{
			AggregateRoot: pipeline.AggregateRoot,
			Name:          pipeline.Name,
			Description:   pipeline.Description,
			Status:        pipeline.Status,
			Steps:         pipeline.Steps,
			Tags:          pipeline.Tags,
		}
		pipelines = append(pipelines, cloned)
	}

	return result.Success(pipelines)
}

// Delete removes a pipeline from memory
func (r *PipelineRepository) Delete(ctx context.Context, id string) result.Result[bool] {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pipelines[id]; !exists {
		return result.Failure[bool](fmt.Errorf("pipeline with ID %s not found", id))
	}

	delete(r.pipelines, id)
	return result.Success(true)
}

// Count returns the number of pipelines
func (r *PipelineRepository) Count(ctx context.Context) result.Result[int] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return result.Success(len(r.pipelines))
}

// Clear removes all pipelines (useful for testing)
func (r *PipelineRepository) Clear(ctx context.Context) result.Result[bool] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pipelines = make(map[string]*entities.Pipeline)
	return result.Success(true)
}
