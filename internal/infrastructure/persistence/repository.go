// Package persistence provides repository implementations
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/flext/flexcore/internal/domain/entities"
	"github.com/flext/flexcore/pkg/errors"
	"github.com/lib/pq"
)

// InMemoryPipelineRepository provides an in-memory implementation of pipeline repository
type InMemoryPipelineRepository struct {
	mu        sync.RWMutex
	pipelines map[entities.PipelineID]*entities.Pipeline
	nameIndex map[string]entities.PipelineID
}

// NewInMemoryPipelineRepository creates a new in-memory pipeline repository
func NewInMemoryPipelineRepository() *InMemoryPipelineRepository {
	return &InMemoryPipelineRepository{
		pipelines: make(map[entities.PipelineID]*entities.Pipeline),
		nameIndex: make(map[string]entities.PipelineID),
	}
}

// Save saves a pipeline
func (r *InMemoryPipelineRepository) Save(ctx context.Context, pipeline *entities.Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Update name index
	r.nameIndex[pipeline.Name] = pipeline.ID

	// Deep copy to avoid mutations
	r.pipelines[pipeline.ID] = pipeline
	return nil
}

// FindByID finds a pipeline by ID
func (r *InMemoryPipelineRepository) FindByID(ctx context.Context, id entities.PipelineID) (*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pipeline, exists := r.pipelines[id]
	if !exists {
		return nil, nil
	}
	return pipeline, nil
}

// FindByName finds a pipeline by name
func (r *InMemoryPipelineRepository) FindByName(ctx context.Context, name string) (*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.nameIndex[name]
	if !exists {
		return nil, nil
	}

	return r.pipelines[id], nil
}

// Delete deletes a pipeline
func (r *InMemoryPipelineRepository) Delete(ctx context.Context, id entities.PipelineID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipeline, exists := r.pipelines[id]
	if !exists {
		return errors.NotFoundError("pipeline")
	}

	delete(r.nameIndex, pipeline.Name)
	delete(r.pipelines, id)
	return nil
}

// List lists pipelines with pagination
func (r *InMemoryPipelineRepository) List(ctx context.Context, limit, offset int) ([]*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice
	all := make([]*entities.Pipeline, 0, len(r.pipelines))
	for _, p := range r.pipelines {
		all = append(all, p)
	}

	// Apply pagination
	start := offset
	if start > len(all) {
		return []*entities.Pipeline{}, nil
	}

	end := start + limit
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], nil
}

// Count counts total pipelines
func (r *InMemoryPipelineRepository) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pipelines), nil
}

// FindByOwner finds pipelines by owner
func (r *InMemoryPipelineRepository) FindByOwner(ctx context.Context, owner string, limit, offset int) ([]*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]*entities.Pipeline, 0)
	for _, p := range r.pipelines {
		if p.Owner == owner {
			filtered = append(filtered, p)
		}
	}

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []*entities.Pipeline{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// FindByTag finds pipelines by tag
func (r *InMemoryPipelineRepository) FindByTag(ctx context.Context, tag string, limit, offset int) ([]*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]*entities.Pipeline, 0)
	for _, p := range r.pipelines {
		for _, t := range p.Tags {
			if t == tag {
				filtered = append(filtered, p)
				break
			}
		}
	}

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []*entities.Pipeline{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// FindByStatus finds pipelines by status
func (r *InMemoryPipelineRepository) FindByStatus(ctx context.Context, status entities.PipelineStatus, limit, offset int) ([]*entities.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]*entities.Pipeline, 0)
	for _, p := range r.pipelines {
		if p.Status == status {
			filtered = append(filtered, p)
		}
	}

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []*entities.Pipeline{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// Exists checks if a pipeline exists
func (r *InMemoryPipelineRepository) Exists(id entities.PipelineID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.pipelines[id]
	return exists
}

// PostgreSQLPipelineRepository provides a PostgreSQL implementation of pipeline repository
type PostgreSQLPipelineRepository struct {
	db *sql.DB
}

// NewPostgreSQLPipelineRepository creates a new PostgreSQL pipeline repository
func NewPostgreSQLPipelineRepository(db *sql.DB) *PostgreSQLPipelineRepository {
	return &PostgreSQLPipelineRepository{db: db}
}

// Save saves a pipeline to the database
func (r *PostgreSQLPipelineRepository) Save(ctx context.Context, pipeline *entities.Pipeline) error {
	// Serialize steps and schedule as JSON
	stepsJSON, err := json.Marshal(pipeline.Steps)
	if err != nil {
		return errors.Wrap(err, "failed to marshal steps")
	}

	var scheduleJSON []byte
	if pipeline.Schedule != nil {
		scheduleJSON, err = json.Marshal(pipeline.Schedule)
		if err != nil {
			return errors.Wrap(err, "failed to marshal schedule")
		}
	}

	// Upsert pipeline
	query := `
		INSERT INTO pipelines (
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			owner = EXCLUDED.owner,
			tags = EXCLUDED.tags,
			steps = EXCLUDED.steps,
			schedule = EXCLUDED.schedule,
			last_run_at = EXCLUDED.last_run_at,
			next_run_at = EXCLUDED.next_run_at,
			updated_at = EXCLUDED.updated_at,
			version = EXCLUDED.version
	`

	_, err = r.db.ExecContext(ctx, query,
		pipeline.ID.String(),
		pipeline.Name,
		pipeline.Description,
		int(pipeline.Status),
		pipeline.Owner,
		pq.Array(pipeline.Tags),
		stepsJSON,
		scheduleJSON,
		pipeline.LastRunAt,
		pipeline.NextRunAt,
		pipeline.CreatedAt,
		pipeline.UpdatedAt,
		pipeline.Version,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save pipeline")
	}

	return nil
}

// FindByID finds a pipeline by ID
func (r *PostgreSQLPipelineRepository) FindByID(ctx context.Context, id entities.PipelineID) (*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanPipeline(row)
}

// FindByName finds a pipeline by name
func (r *PostgreSQLPipelineRepository) FindByName(ctx context.Context, name string) (*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		WHERE name = $1
	`

	row := r.db.QueryRowContext(ctx, query, name)
	return r.scanPipeline(row)
}

// Delete deletes a pipeline
func (r *PostgreSQLPipelineRepository) Delete(ctx context.Context, id entities.PipelineID) error {
	query := `DELETE FROM pipelines WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return errors.Wrap(err, "failed to delete pipeline")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NotFoundError("pipeline")
	}

	return nil
}

// List lists pipelines with pagination
func (r *PostgreSQLPipelineRepository) List(ctx context.Context, limit, offset int) ([]*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pipelines")
	}
	defer rows.Close()

	return r.scanPipelines(rows)
}

// Count counts total pipelines
func (r *PostgreSQLPipelineRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM pipelines`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count pipelines")
	}

	return count, nil
}

// FindByOwner finds pipelines by owner
func (r *PostgreSQLPipelineRepository) FindByOwner(ctx context.Context, owner string, limit, offset int) ([]*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		WHERE owner = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, owner, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipelines by owner")
	}
	defer rows.Close()

	return r.scanPipelines(rows)
}

// FindByTag finds pipelines by tag
func (r *PostgreSQLPipelineRepository) FindByTag(ctx context.Context, tag string, limit, offset int) ([]*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		WHERE $1 = ANY(tags)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, tag, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipelines by tag")
	}
	defer rows.Close()

	return r.scanPipelines(rows)
}

// FindByStatus finds pipelines by status
func (r *PostgreSQLPipelineRepository) FindByStatus(ctx context.Context, status entities.PipelineStatus, limit, offset int) ([]*entities.Pipeline, error) {
	query := `
		SELECT 
			id, name, description, status, owner, tags,
			steps, schedule, last_run_at, next_run_at,
			created_at, updated_at, version
		FROM pipelines
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, int(status), limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find pipelines by status")
	}
	defer rows.Close()

	return r.scanPipelines(rows)
}

// Exists checks if a pipeline exists
func (r *PostgreSQLPipelineRepository) Exists(id entities.PipelineID) bool {
	query := `SELECT EXISTS(SELECT 1 FROM pipelines WHERE id = $1)`
	
	var exists bool
	err := r.db.QueryRow(query, id.String()).Scan(&exists)
	if err != nil {
		return false
	}

	return exists
}

// scanPipeline scans a single pipeline from a row
func (r *PostgreSQLPipelineRepository) scanPipeline(row *sql.Row) (*entities.Pipeline, error) {
	var (
		idStr       string
		status      int
		stepsJSON   []byte
		scheduleJSON sql.NullString
		tags        pq.StringArray
		lastRunAt   sql.NullTime
		nextRunAt   sql.NullTime
	)

	pipeline := &entities.Pipeline{}

	err := row.Scan(
		&idStr,
		&pipeline.Name,
		&pipeline.Description,
		&status,
		&pipeline.Owner,
		&tags,
		&stepsJSON,
		&scheduleJSON,
		&lastRunAt,
		&nextRunAt,
		&pipeline.CreatedAt,
		&pipeline.UpdatedAt,
		&pipeline.Version,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to scan pipeline")
	}

	// Convert ID
	pipeline.ID = entities.PipelineID(idStr)
	
	// Convert status
	pipeline.Status = entities.PipelineStatus(status)
	
	// Convert tags
	pipeline.Tags = []string(tags)
	
	// Unmarshal steps
	if err := json.Unmarshal(stepsJSON, &pipeline.Steps); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal steps")
	}
	
	// Unmarshal schedule
	if scheduleJSON.Valid {
		var schedule entities.PipelineSchedule
		if err := json.Unmarshal([]byte(scheduleJSON.String), &schedule); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal schedule")
		}
		pipeline.Schedule = &schedule
	}
	
	// Set optional times
	if lastRunAt.Valid {
		pipeline.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		pipeline.NextRunAt = &nextRunAt.Time
	}

	return pipeline, nil
}

// scanPipelines scans multiple pipelines from rows
func (r *PostgreSQLPipelineRepository) scanPipelines(rows *sql.Rows) ([]*entities.Pipeline, error) {
	pipelines := make([]*entities.Pipeline, 0)

	for rows.Next() {
		var (
			idStr       string
			status      int
			stepsJSON   []byte
			scheduleJSON sql.NullString
			tags        pq.StringArray
			lastRunAt   sql.NullTime
			nextRunAt   sql.NullTime
		)

		pipeline := &entities.Pipeline{}

		err := rows.Scan(
			&idStr,
			&pipeline.Name,
			&pipeline.Description,
			&status,
			&pipeline.Owner,
			&tags,
			&stepsJSON,
			&scheduleJSON,
			&lastRunAt,
			&nextRunAt,
			&pipeline.CreatedAt,
			&pipeline.UpdatedAt,
			&pipeline.Version,
		)

		if err != nil {
			return nil, errors.Wrap(err, "failed to scan pipeline")
		}

		// Convert fields (same as scanPipeline)
		pipeline.ID = entities.PipelineID(idStr)
		pipeline.Status = entities.PipelineStatus(status)
		pipeline.Tags = []string(tags)
		
		if err := json.Unmarshal(stepsJSON, &pipeline.Steps); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal steps")
		}
		
		if scheduleJSON.Valid {
			var schedule entities.PipelineSchedule
			if err := json.Unmarshal([]byte(scheduleJSON.String), &schedule); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal schedule")
			}
			pipeline.Schedule = &schedule
		}
		
		if lastRunAt.Valid {
			pipeline.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			pipeline.NextRunAt = &nextRunAt.Time
		}

		pipelines = append(pipelines, pipeline)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return pipelines, nil
}

// CreateSchema creates the database schema
func CreateSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS pipelines (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			status INTEGER NOT NULL DEFAULT 0,
			owner VARCHAR(255) NOT NULL,
			tags TEXT[],
			steps JSONB NOT NULL DEFAULT '[]',
			schedule JSONB,
			last_run_at TIMESTAMP,
			next_run_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			version BIGINT NOT NULL DEFAULT 1
		);

		CREATE INDEX IF NOT EXISTS idx_pipelines_name ON pipelines(name);
		CREATE INDEX IF NOT EXISTS idx_pipelines_owner ON pipelines(owner);
		CREATE INDEX IF NOT EXISTS idx_pipelines_status ON pipelines(status);
		CREATE INDEX IF NOT EXISTS idx_pipelines_tags ON pipelines USING GIN(tags);
		CREATE INDEX IF NOT EXISTS idx_pipelines_created_at ON pipelines(created_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return errors.Wrap(err, "failed to create schema")
	}

	return nil
}