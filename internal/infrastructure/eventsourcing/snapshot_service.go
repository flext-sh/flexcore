// Package eventsourcing provides snapshot management functionality
// SOLID SRP: Dedicated module for snapshot operations
package eventsourcing

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/result"
)

// Snapshot represents an aggregate snapshot
type Snapshot struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
}

// SnapshotService handles all snapshot operations
// SOLID SRP: Single responsibility for snapshot management
type SnapshotService struct {
	db        *sql.DB
	mu        sync.RWMutex
	snapshots map[string]*Snapshot
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(db *sql.DB) *SnapshotService {
	return &SnapshotService{
		db:        db,
		snapshots: make(map[string]*Snapshot),
	}
}

// SaveSnapshot saves a snapshot for an aggregate
// SOLID SRP: Single responsibility for snapshot saving
func (ss *SnapshotService) SaveSnapshot(snapshot *Snapshot) result.Result[bool] {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Save to memory cache
	ss.snapshots[snapshot.AggregateID] = snapshot

	// Persist to database
	query := `
		INSERT OR REPLACE INTO snapshots 
		(aggregate_id, aggregate_type, version, data, created_at) 
		VALUES (?, ?, ?, ?, ?)`

	dataBytes, err := json.Marshal(snapshot.Data)
	if err != nil {
		return result.Failure[bool](err)
	}

	_, err = ss.db.Exec(query,
		snapshot.AggregateID,
		snapshot.AggregateType,
		snapshot.Version,
		string(dataBytes),
		snapshot.CreatedAt,
	)

	if err != nil {
		return result.Failure[bool](err)
	}

	return result.Success(true)
}

// LoadSnapshot loads a snapshot for an aggregate
// SOLID SRP: Single responsibility for snapshot loading
func (ss *SnapshotService) LoadSnapshot(aggregateID string) result.Result[*Snapshot] {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Try memory cache first
	if snapshot, exists := ss.snapshots[aggregateID]; exists {
		return result.Success(snapshot)
	}

	// Load from database
	return ss.loadSnapshotFromDatabase(aggregateID)
}

// loadSnapshotFromDatabase loads snapshot from persistent storage
func (ss *SnapshotService) loadSnapshotFromDatabase(aggregateID string) result.Result[*Snapshot] {
	query := `
		SELECT aggregate_id, aggregate_type, version, data, created_at 
		FROM snapshots 
		WHERE aggregate_id = ? 
		ORDER BY version DESC 
		LIMIT 1`

	row := ss.db.QueryRow(query, aggregateID)

	var snapshot Snapshot
	var dataStr string

	err := row.Scan(
		&snapshot.AggregateID,
		&snapshot.AggregateType,
		&snapshot.Version,
		&dataStr,
		&snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return result.Failure[*Snapshot](nil) // No snapshot found
	}

	if err != nil {
		return result.Failure[*Snapshot](err)
	}

	// Unmarshal data
	err = json.Unmarshal([]byte(dataStr), &snapshot.Data)
	if err != nil {
		return result.Failure[*Snapshot](err)
	}

	// Cache in memory
	ss.snapshots[aggregateID] = &snapshot

	return result.Success(&snapshot)
}

// CreateSnapshotTable creates the snapshots table if it doesn't exist
func (ss *SnapshotService) CreateSnapshotTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			version INTEGER NOT NULL,
			data TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE(aggregate_id, version)
		)`

	_, err := ss.db.Exec(query)
	return err
}