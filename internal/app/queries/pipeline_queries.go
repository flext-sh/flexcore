// Package queries provides pipeline-specific queries
package queries

import (
	"context"
	"strings"
	"time"

	"github.com/flext/flexcore/internal/domain/entities"
	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/shared/errors"
)

// PipelineRepository represents a repository for pipelines (read operations)
type PipelineRepository interface {
	FindByID(ctx context.Context, id entities.PipelineID) (*entities.Pipeline, error)
	FindByName(ctx context.Context, name string) (*entities.Pipeline, error)
	List(ctx context.Context, limit, offset int) ([]*entities.Pipeline, error)
	Count(ctx context.Context) (int, error)
	FindByOwner(ctx context.Context, owner string, limit, offset int) ([]*entities.Pipeline, error)
	FindByTag(ctx context.Context, tag string, limit, offset int) ([]*entities.Pipeline, error)
	FindByStatus(ctx context.Context, status entities.PipelineStatus, limit, offset int) ([]*entities.Pipeline, error)
}

// GetPipelineQuery represents a query to get a pipeline by ID
type GetPipelineQuery struct {
	BaseQuery
	PipelineID entities.PipelineID
}

// NewGetPipelineQuery creates a new get pipeline query
func NewGetPipelineQuery(pipelineID entities.PipelineID) GetPipelineQuery {
	return GetPipelineQuery{
		BaseQuery:  NewBaseQuery("GetPipeline"),
		PipelineID: pipelineID,
	}
}

// GetPipelineQueryHandler handles get pipeline queries
type GetPipelineQueryHandler struct {
	repository PipelineRepository
}

// NewGetPipelineQueryHandler creates a new get pipeline query handler
func NewGetPipelineQueryHandler(repository PipelineRepository) *GetPipelineQueryHandler {
	return &GetPipelineQueryHandler{
		repository: repository,
	}
}

// Handle handles the get pipeline query
func (h *GetPipelineQueryHandler) Handle(ctx context.Context, query GetPipelineQuery) result.Result[*entities.Pipeline] {
	pipeline, err := h.repository.FindByID(ctx, query.PipelineID)
	if err != nil {
		return result.Failure[*entities.Pipeline](errors.Wrap(err, "failed to find pipeline"))
	}

	if pipeline == nil {
		return result.Failure[*entities.Pipeline](errors.NotFoundError("pipeline"))
	}

	return result.Success(pipeline)
}

// GetPipelineByNameQuery represents a query to get a pipeline by name
type GetPipelineByNameQuery struct {
	BaseQuery
	Name string
}

// NewGetPipelineByNameQuery creates a new get pipeline by name query
func NewGetPipelineByNameQuery(name string) GetPipelineByNameQuery {
	return GetPipelineByNameQuery{
		BaseQuery: NewBaseQuery("GetPipelineByName"),
		Name:      name,
	}
}

// GetPipelineByNameQueryHandler handles get pipeline by name queries
type GetPipelineByNameQueryHandler struct {
	repository PipelineRepository
}

// NewGetPipelineByNameQueryHandler creates a new handler
func NewGetPipelineByNameQueryHandler(repository PipelineRepository) *GetPipelineByNameQueryHandler {
	return &GetPipelineByNameQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *GetPipelineByNameQueryHandler) Handle(ctx context.Context, query GetPipelineByNameQuery) result.Result[*entities.Pipeline] {
	if query.Name == "" {
		return result.Failure[*entities.Pipeline](errors.ValidationError("pipeline name cannot be empty"))
	}

	pipeline, err := h.repository.FindByName(ctx, query.Name)
	if err != nil {
		return result.Failure[*entities.Pipeline](errors.Wrap(err, "failed to find pipeline"))
	}

	if pipeline == nil {
		return result.Failure[*entities.Pipeline](errors.NotFoundError("pipeline"))
	}

	return result.Success(pipeline)
}

// ListPipelinesQuery represents a query to list pipelines
type ListPipelinesQuery struct {
	PagedQuery
}

// NewListPipelinesQuery creates a new list pipelines query
func NewListPipelinesQuery(page, pageSize int) ListPipelinesQuery {
	return ListPipelinesQuery{
		PagedQuery: NewPagedQuery("ListPipelines", page, pageSize),
	}
}

// ListPipelinesQueryHandler handles list pipelines queries
type ListPipelinesQueryHandler struct {
	repository PipelineRepository
}

// NewListPipelinesQueryHandler creates a new handler
func NewListPipelinesQueryHandler(repository PipelineRepository) *ListPipelinesQueryHandler {
	return &ListPipelinesQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *ListPipelinesQueryHandler) Handle(ctx context.Context, query ListPipelinesQuery) result.Result[PagedResult[*entities.Pipeline]] {
	// Calculate offset
	offset := (query.Page - 1) * query.PageSize

	// Get pipelines
	pipelines, err := h.repository.List(ctx, query.PageSize, offset)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to list pipelines"))
	}

	// Get total count
	totalCount, err := h.repository.Count(ctx)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to count pipelines"))
	}

	// Create paged result
	pagedResult := NewPagedResult(pipelines, totalCount, query.Page, query.PageSize)
	return result.Success(pagedResult)
}

// ListPipelinesByOwnerQuery represents a query to list pipelines by owner
type ListPipelinesByOwnerQuery struct {
	PagedQuery
	Owner string
}

// NewListPipelinesByOwnerQuery creates a new query
func NewListPipelinesByOwnerQuery(owner string, page, pageSize int) ListPipelinesByOwnerQuery {
	return ListPipelinesByOwnerQuery{
		PagedQuery: NewPagedQuery("ListPipelinesByOwner", page, pageSize),
		Owner:      owner,
	}
}

// ListPipelinesByOwnerQueryHandler handles the query
type ListPipelinesByOwnerQueryHandler struct {
	repository PipelineRepository
}

// NewListPipelinesByOwnerQueryHandler creates a new handler
func NewListPipelinesByOwnerQueryHandler(repository PipelineRepository) *ListPipelinesByOwnerQueryHandler {
	return &ListPipelinesByOwnerQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *ListPipelinesByOwnerQueryHandler) Handle(ctx context.Context, query ListPipelinesByOwnerQuery) result.Result[PagedResult[*entities.Pipeline]] {
	if query.Owner == "" {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.ValidationError("owner cannot be empty"))
	}

	offset := (query.Page - 1) * query.PageSize
	pipelines, err := h.repository.FindByOwner(ctx, query.Owner, query.PageSize, offset)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to list pipelines by owner"))
	}

	// For simplicity, using the same count. In production, implement CountByOwner
	totalCount, err := h.repository.Count(ctx)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to count pipelines"))
	}

	pagedResult := NewPagedResult(pipelines, totalCount, query.Page, query.PageSize)
	return result.Success(pagedResult)
}

// ListPipelinesByTagQuery represents a query to list pipelines by tag
type ListPipelinesByTagQuery struct {
	PagedQuery
	Tag string
}

// NewListPipelinesByTagQuery creates a new query
func NewListPipelinesByTagQuery(tag string, page, pageSize int) ListPipelinesByTagQuery {
	return ListPipelinesByTagQuery{
		PagedQuery: NewPagedQuery("ListPipelinesByTag", page, pageSize),
		Tag:        tag,
	}
}

// ListPipelinesByTagQueryHandler handles the query
type ListPipelinesByTagQueryHandler struct {
	repository PipelineRepository
}

// NewListPipelinesByTagQueryHandler creates a new handler
func NewListPipelinesByTagQueryHandler(repository PipelineRepository) *ListPipelinesByTagQueryHandler {
	return &ListPipelinesByTagQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *ListPipelinesByTagQueryHandler) Handle(ctx context.Context, query ListPipelinesByTagQuery) result.Result[PagedResult[*entities.Pipeline]] {
	if query.Tag == "" {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.ValidationError("tag cannot be empty"))
	}

	offset := (query.Page - 1) * query.PageSize
	pipelines, err := h.repository.FindByTag(ctx, query.Tag, query.PageSize, offset)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to list pipelines by tag"))
	}

	totalCount, err := h.repository.Count(ctx)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to count pipelines"))
	}

	pagedResult := NewPagedResult(pipelines, totalCount, query.Page, query.PageSize)
	return result.Success(pagedResult)
}

// ListPipelinesByStatusQuery represents a query to list pipelines by status
type ListPipelinesByStatusQuery struct {
	PagedQuery
	Status entities.PipelineStatus
}

// NewListPipelinesByStatusQuery creates a new query
func NewListPipelinesByStatusQuery(status entities.PipelineStatus, page, pageSize int) ListPipelinesByStatusQuery {
	return ListPipelinesByStatusQuery{
		PagedQuery: NewPagedQuery("ListPipelinesByStatus", page, pageSize),
		Status:     status,
	}
}

// ListPipelinesByStatusQueryHandler handles the query
type ListPipelinesByStatusQueryHandler struct {
	repository PipelineRepository
}

// NewListPipelinesByStatusQueryHandler creates a new handler
func NewListPipelinesByStatusQueryHandler(repository PipelineRepository) *ListPipelinesByStatusQueryHandler {
	return &ListPipelinesByStatusQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *ListPipelinesByStatusQueryHandler) Handle(ctx context.Context, query ListPipelinesByStatusQuery) result.Result[PagedResult[*entities.Pipeline]] {
	offset := (query.Page - 1) * query.PageSize
	pipelines, err := h.repository.FindByStatus(ctx, query.Status, query.PageSize, offset)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to list pipelines by status"))
	}

	totalCount, err := h.repository.Count(ctx)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to count pipelines"))
	}

	pagedResult := NewPagedResult(pipelines, totalCount, query.Page, query.PageSize)
	return result.Success(pagedResult)
}

// PipelineStatistics represents pipeline statistics
type PipelineStatistics struct {
	TotalPipelines       int
	ActivePipelines      int
	RunningPipelines     int
	FailedPipelines      int
	CompletedPipelines   int
	PipelinesLastHour    int
	PipelinesLastDay     int
	AverageExecutionTime time.Duration
}

// GetPipelineStatisticsQuery represents a query to get pipeline statistics
type GetPipelineStatisticsQuery struct {
	BaseQuery
}

// NewGetPipelineStatisticsQuery creates a new query
func NewGetPipelineStatisticsQuery() GetPipelineStatisticsQuery {
	return GetPipelineStatisticsQuery{
		BaseQuery: NewBaseQuery("GetPipelineStatistics"),
	}
}

// GetPipelineStatisticsQueryHandler handles the query
type GetPipelineStatisticsQueryHandler struct {
	repository PipelineRepository
}

// NewGetPipelineStatisticsQueryHandler creates a new handler
func NewGetPipelineStatisticsQueryHandler(repository PipelineRepository) *GetPipelineStatisticsQueryHandler {
	return &GetPipelineStatisticsQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *GetPipelineStatisticsQueryHandler) Handle(ctx context.Context, query GetPipelineStatisticsQuery) result.Result[PipelineStatistics] {
	// This is a simplified implementation
	// In production, you would have specific repository methods for statistics

	totalCount, err := h.repository.Count(ctx)
	if err != nil {
		return result.Failure[PipelineStatistics](errors.Wrap(err, "failed to get pipeline count"))
	}

	// Get counts by status
	activePipelines, _ := h.repository.FindByStatus(ctx, entities.PipelineStatusActive, 1000, 0)
	runningPipelines, _ := h.repository.FindByStatus(ctx, entities.PipelineStatusRunning, 1000, 0)
	failedPipelines, _ := h.repository.FindByStatus(ctx, entities.PipelineStatusFailed, 1000, 0)
	completedPipelines, _ := h.repository.FindByStatus(ctx, entities.PipelineStatusCompleted, 1000, 0)

	stats := PipelineStatistics{
		TotalPipelines:       totalCount,
		ActivePipelines:      len(activePipelines),
		RunningPipelines:     len(runningPipelines),
		FailedPipelines:      len(failedPipelines),
		CompletedPipelines:   len(completedPipelines),
		PipelinesLastHour:    0,               // Would need time-based queries
		PipelinesLastDay:     0,               // Would need time-based queries
		AverageExecutionTime: time.Minute * 5, // Would need execution data
	}

	return result.Success(stats)
}

// SearchPipelinesQuery represents a query to search pipelines
type SearchPipelinesQuery struct {
	FilteredQuery
	PagedQuery
	SearchTerm string
}

// NewSearchPipelinesQuery creates a new query
func NewSearchPipelinesQuery(searchTerm string, page, pageSize int) SearchPipelinesQuery {
	return SearchPipelinesQuery{
		FilteredQuery: NewFilteredQuery("SearchPipelines"),
		PagedQuery:    NewPagedQuery("SearchPipelines", page, pageSize),
		SearchTerm:    searchTerm,
	}
}

// SearchPipelinesQueryHandler handles the query
type SearchPipelinesQueryHandler struct {
	repository PipelineRepository
}

// NewSearchPipelinesQueryHandler creates a new handler
func NewSearchPipelinesQueryHandler(repository PipelineRepository) *SearchPipelinesQueryHandler {
	return &SearchPipelinesQueryHandler{
		repository: repository,
	}
}

// Handle handles the query
func (h *SearchPipelinesQueryHandler) Handle(ctx context.Context, query SearchPipelinesQuery) result.Result[PagedResult[*entities.Pipeline]] {
	if query.SearchTerm == "" {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.ValidationError("search term cannot be empty"))
	}

	// In a real implementation, you would have a Search method in the repository
	// For now, we'll use the basic List method
	offset := (query.Page - 1) * query.PageSize
	pipelines, err := h.repository.List(ctx, query.PageSize, offset)
	if err != nil {
		return result.Failure[PagedResult[*entities.Pipeline]](errors.Wrap(err, "failed to search pipelines"))
	}

	// Filter by search term (simple implementation)
	filteredPipelines := make([]*entities.Pipeline, 0)
	for _, p := range pipelines {
		if contains(p.Name, query.SearchTerm) || contains(p.Description, query.SearchTerm) {
			filteredPipelines = append(filteredPipelines, p)
		}
	}

	totalCount := len(filteredPipelines)
	pagedResult := NewPagedResult(filteredPipelines, totalCount, query.Page, query.PageSize)
	return result.Success(pagedResult)
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}
