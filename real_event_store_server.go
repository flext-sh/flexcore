// FlexCore REAL Event Store Server - 100% SQLite Persistence
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

// REAL Event with complete persistence
type Event struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	OccurredAt    time.Time              `json:"occurred_at"`
	CreatedAt     time.Time              `json:"created_at"`
}

// REAL Snapshot with persistence
type Snapshot struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
}

// REAL Event Store with SQLite
type RealEventStore struct {
	db *sql.DB
}

func NewRealEventStore(dbPath string) (*RealEventStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &RealEventStore{db: db}
	if err := store.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

func (es *RealEventStore) initializeDatabase() error {
	// Create events table
	eventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL,
		aggregate_type TEXT NOT NULL,
		version INTEGER NOT NULL,
		data TEXT NOT NULL,
		metadata TEXT NOT NULL,
		occurred_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(aggregate_id, version)
	);
	CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_id);
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
	CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
	`

	// Create snapshots table
	snapshotsTable := `
	CREATE TABLE IF NOT EXISTS snapshots (
		aggregate_id TEXT PRIMARY KEY,
		aggregate_type TEXT NOT NULL,
		version INTEGER NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_snapshots_type ON snapshots(aggregate_type);
	`

	if _, err := es.db.Exec(eventsTable); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	if _, err := es.db.Exec(snapshotsTable); err != nil {
		return fmt.Errorf("failed to create snapshots table: %w", err)
	}

	return nil
}

func (es *RealEventStore) AppendEvent(event *Event) error {
	// Get next version for aggregate
	nextVersion, err := es.getNextVersion(event.AggregateID)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	event.ID = uuid.New().String()
	event.Version = nextVersion
	event.CreatedAt = time.Now()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}

	// Serialize data and metadata
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal event metadata: %w", err)
	}

	// Insert event
	query := `
	INSERT INTO events (id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = es.db.Exec(query,
		event.ID, event.Type, event.AggregateID, event.AggregateType,
		event.Version, string(dataJSON), string(metadataJSON),
		event.OccurredAt, event.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

func (es *RealEventStore) getNextVersion(aggregateID string) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) + 1 FROM events WHERE aggregate_id = ?`
	var nextVersion int
	err := es.db.QueryRow(query, aggregateID).Scan(&nextVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get next version: %w", err)
	}
	return nextVersion, nil
}

func (es *RealEventStore) GetEventStream(aggregateID string) ([]*Event, error) {
	query := `
	SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at
	FROM events 
	WHERE aggregate_id = ? 
	ORDER BY version ASC
	`

	rows, err := es.db.Query(query, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var event Event
		var dataJSON, metadataJSON string

		err := rows.Scan(
			&event.ID, &event.Type, &event.AggregateID, &event.AggregateType,
			&event.Version, &dataJSON, &metadataJSON, &event.OccurredAt, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Deserialize data and metadata
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event metadata: %w", err)
		}

		events = append(events, &event)
	}

	return events, nil
}

func (es *RealEventStore) GetEventsByType(eventType string) ([]*Event, error) {
	query := `
	SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at
	FROM events 
	WHERE type = ? 
	ORDER BY occurred_at DESC
	`

	rows, err := es.db.Query(query, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var event Event
		var dataJSON, metadataJSON string

		err := rows.Scan(
			&event.ID, &event.Type, &event.AggregateID, &event.AggregateType,
			&event.Version, &dataJSON, &metadataJSON, &event.OccurredAt, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event metadata: %w", err)
		}

		events = append(events, &event)
	}

	return events, nil
}

func (es *RealEventStore) SaveSnapshot(snapshot *Snapshot) error {
	dataJSON, err := json.Marshal(snapshot.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO snapshots (aggregate_id, aggregate_type, version, data, created_at)
	VALUES (?, ?, ?, ?, ?)
	`

	snapshot.CreatedAt = time.Now()
	_, err = es.db.Exec(query,
		snapshot.AggregateID, snapshot.AggregateType, snapshot.Version,
		string(dataJSON), snapshot.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

func (es *RealEventStore) GetSnapshot(aggregateID string) (*Snapshot, error) {
	query := `
	SELECT aggregate_id, aggregate_type, version, data, created_at
	FROM snapshots 
	WHERE aggregate_id = ?
	`

	var snapshot Snapshot
	var dataJSON string

	err := es.db.QueryRow(query, aggregateID).Scan(
		&snapshot.AggregateID, &snapshot.AggregateType, &snapshot.Version,
		&dataJSON, &snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No snapshot found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if err := json.Unmarshal([]byte(dataJSON), &snapshot.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot data: %w", err)
	}

	return &snapshot, nil
}

func (es *RealEventStore) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total events
	var totalEvents int
	err := es.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}
	stats["total_events"] = totalEvents

	// Total aggregates
	var totalAggregates int
	err = es.db.QueryRow("SELECT COUNT(DISTINCT aggregate_id) FROM events").Scan(&totalAggregates)
	if err != nil {
		return nil, fmt.Errorf("failed to get total aggregates: %w", err)
	}
	stats["total_aggregates"] = totalAggregates

	// Total snapshots
	var totalSnapshots int
	err = es.db.QueryRow("SELECT COUNT(*) FROM snapshots").Scan(&totalSnapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to get total snapshots: %w", err)
	}
	stats["total_snapshots"] = totalSnapshots

	// Event types
	rows, err := es.db.Query("SELECT type, COUNT(*) FROM events GROUP BY type ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to get event types: %w", err)
	}
	defer rows.Close()

	eventTypes := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan event type: %w", err)
		}
		eventTypes[eventType] = count
	}
	stats["event_types"] = eventTypes

	return stats, nil
}

// HTTP Server for Real Event Store
type EventStoreServer struct {
	store *RealEventStore
}

func NewEventStoreServer(dbPath string) (*EventStoreServer, error) {
	store, err := NewRealEventStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &EventStoreServer{store: store}, nil
}

func (ess *EventStoreServer) handleAppendEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AggregateID   string                 `json:"aggregate_id"`
		AggregateType string                 `json:"aggregate_type"`
		EventType     string                 `json:"event_type"`
		EventData     map[string]interface{} `json:"event_data"`
		Metadata      map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	event := &Event{
		Type:          req.EventType,
		AggregateID:   req.AggregateID,
		AggregateType: req.AggregateType,
		Data:          req.EventData,
		Metadata:      req.Metadata,
		OccurredAt:    time.Now(),
	}

	if err := ess.store.AppendEvent(event); err != nil {
		http.Error(w, fmt.Sprintf("Failed to append event: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"event_id":     event.ID,
		"success":      true,
		"message":      "Event appended successfully",
		"created_at":   event.CreatedAt,
		"version":      event.Version,
		"aggregate_id": event.AggregateID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleGetEventStream(w http.ResponseWriter, r *http.Request) {
	aggregateID := r.URL.Query().Get("aggregate_id")
	if aggregateID == "" {
		http.Error(w, "aggregate_id parameter required", http.StatusBadRequest)
		return
	}

	events, err := ess.store.GetEventStream(aggregateID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get event stream: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"aggregate_id":  aggregateID,
		"events":        events,
		"total_events":  len(events),
		"stream_size":   len(events),
		"retrieved_at":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleGetEventsByType(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("event_type")
	if eventType == "" {
		http.Error(w, "event_type parameter required", http.StatusBadRequest)
		return
	}

	events, err := ess.store.GetEventsByType(eventType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events by type: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"event_type":   eventType,
		"events":       events,
		"total_events": len(events),
		"retrieved_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	aggregateID := r.URL.Query().Get("aggregate_id")
	if aggregateID == "" {
		http.Error(w, "aggregate_id parameter required", http.StatusBadRequest)
		return
	}

	// Get all events for the aggregate
	events, err := ess.store.GetEventStream(aggregateID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	if len(events) == 0 {
		http.Error(w, "No events found for aggregate", http.StatusNotFound)
		return
	}

	// Replay events to build current state
	state := make(map[string]interface{})
	var aggregateType string
	var version int

	for _, event := range events {
		aggregateType = event.AggregateType
		version = event.Version

		// Apply event to state (simplified event sourcing)
		switch event.Type {
		case "PipelineCreated":
			if name, ok := event.Data["name"]; ok {
				state["name"] = name
			}
			if status, ok := event.Data["status"]; ok {
				state["status"] = status
			}
			if steps, ok := event.Data["steps"]; ok {
				state["steps"] = steps
			}
		case "PipelineStarted":
			state["status"] = "running"
			if execID, ok := event.Data["execution_id"]; ok {
				state["execution_id"] = execID
			}
			if startedBy, ok := event.Data["started_by"]; ok {
				state["started_by"] = startedBy
			}
		case "PipelineCompleted":
			state["status"] = "completed"
			if duration, ok := event.Data["duration_ms"]; ok {
				state["duration_ms"] = duration
			}
			if records, ok := event.Data["records_processed"]; ok {
				state["records_processed"] = records
			}
		}

		state["id"] = aggregateID
		state["version"] = version
	}

	// Create snapshot
	snapshot := &Snapshot{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       version,
		Data:          state,
	}

	if err := ess.store.SaveSnapshot(snapshot); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"snapshot_created": true,
		"aggregate_id":     aggregateID,
		"version":          version,
		"events_replayed":  len(events),
		"snapshot_data":    state,
		"created_at":       snapshot.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	aggregateID := r.URL.Query().Get("aggregate_id")
	if aggregateID == "" {
		http.Error(w, "aggregate_id parameter required", http.StatusBadRequest)
		return
	}

	snapshot, err := ess.store.GetSnapshot(aggregateID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	if snapshot == nil {
		http.Error(w, "Snapshot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func (ess *EventStoreServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := ess.store.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"event_store_stats": stats,
		"database_path":     "./real_events.db",
		"retrieved_at":      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleReplayAggregate(w http.ResponseWriter, r *http.Request) {
	aggregateID := r.URL.Query().Get("aggregate_id")
	if aggregateID == "" {
		http.Error(w, "aggregate_id parameter required", http.StatusBadRequest)
		return
	}

	toVersionStr := r.URL.Query().Get("to_version")
	toVersion := -1 // Default: replay all events
	if toVersionStr != "" {
		if v, err := strconv.Atoi(toVersionStr); err == nil {
			toVersion = v
		}
	}

	// Get events
	events, err := ess.store.GetEventStream(aggregateID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	if len(events) == 0 {
		http.Error(w, "No events found for aggregate", http.StatusNotFound)
		return
	}

	// Replay events up to specified version
	state := make(map[string]interface{})
	eventsReplayed := 0

	for _, event := range events {
		if toVersion > 0 && event.Version > toVersion {
			break
		}

		// Apply event to state
		switch event.Type {
		case "PipelineCreated":
			for key, value := range event.Data {
				state[key] = value
			}
		case "PipelineStarted":
			state["status"] = "running"
			for key, value := range event.Data {
				state[key] = value
			}
		case "PipelineCompleted":
			state["status"] = "completed"
			for key, value := range event.Data {
				state[key] = value
			}
		default:
			// Generic event application
			for key, value := range event.Data {
				state[key] = value
			}
		}

		state["id"] = aggregateID
		state["version"] = event.Version
		eventsReplayed++
	}

	response := map[string]interface{}{
		"aggregate_id":     aggregateID,
		"events_replayed":  eventsReplayed,
		"target_version":   toVersion,
		"final_state":      state,
		"replayed_at":      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ess *EventStoreServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats, _ := ess.store.GetStats()

	health := map[string]interface{}{
		"service":     "real-event-store",
		"status":      "healthy",
		"version":     "1.0.0",
		"timestamp":   time.Now(),
		"database":    "sqlite",
		"stats":       stats,
		"features": []string{
			"Real SQLite Persistence",
			"Event Append & Replay",
			"Snapshot Creation",
			"Event Type Queries",
			"Aggregate Reconstruction",
			"Time-Travel Queries",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func main() {
	fmt.Println("📝 FlexCore REAL Event Store Server - 100% SQLite Persistence")
	fmt.Println("============================================================")
	fmt.Println("🗄️ Real SQLite database persistence")
	fmt.Println("⚡ Event append, replay, and snapshots")
	fmt.Println("🔍 Event type queries and aggregate reconstruction")
	fmt.Println()

	server, err := NewEventStoreServer("./real_events.db")
	if err != nil {
		log.Fatal("Failed to create event store server:", err)
	}

	// Event operations
	http.HandleFunc("/events/append", server.handleAppendEvent)
	http.HandleFunc("/events/stream", server.handleGetEventStream)
	http.HandleFunc("/events/by-type", server.handleGetEventsByType)

	// Snapshot operations
	http.HandleFunc("/snapshots/create", server.handleCreateSnapshot)
	http.HandleFunc("/snapshots/get", server.handleGetSnapshot)

	// Replay and stats
	http.HandleFunc("/replay", server.handleReplayAggregate)
	http.HandleFunc("/stats", server.handleGetStats)
	http.HandleFunc("/health", server.handleHealth)

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		info := map[string]interface{}{
			"service":     "real-event-store",
			"version":     "1.0.0",
			"description": "FlexCore Real Event Store with SQLite persistence",
			"timestamp":   time.Now(),
			"endpoints": []string{
				"POST /events/append - Append event to store",
				"GET  /events/stream?aggregate_id=X - Get event stream",
				"GET  /events/by-type?event_type=X - Get events by type",
				"POST /snapshots/create?aggregate_id=X - Create snapshot",
				"GET  /snapshots/get?aggregate_id=X - Get snapshot",
				"GET  /replay?aggregate_id=X&to_version=N - Replay aggregate",
				"GET  /stats - Event store statistics",
				"GET  /health - Health check",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	fmt.Println("🌐 Real Event Store Server starting on :8095")
	fmt.Println("📊 Endpoints:")
	fmt.Println("  POST /events/append       - Append event")
	fmt.Println("  GET  /events/stream       - Get event stream")
	fmt.Println("  GET  /events/by-type      - Get events by type")
	fmt.Println("  POST /snapshots/create    - Create snapshot")
	fmt.Println("  GET  /snapshots/get       - Get snapshot")
	fmt.Println("  GET  /replay              - Replay aggregate")
	fmt.Println("  GET  /stats               - Statistics")
	fmt.Println("  GET  /health              - Health check")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8095", nil))
}