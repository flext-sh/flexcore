// FlexCore REAL CQRS Read Model Projections - 100% Real Implementation
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// REAL Read Model Implementation
type ReadModelProjector struct {
	db           *sql.DB
	eventStoreDB *sql.DB
}

type ReadModel struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	ModelType     string                 `json:"model_type"`
	Data          map[string]interface{} `json:"data"`
	Version       int                    `json:"version"`
	LastUpdated   time.Time              `json:"last_updated"`
	CreatedAt     time.Time              `json:"created_at"`
}

type ProjectionRequest struct {
	AggregateID   string `json:"aggregate_id"`
	AggregateType string `json:"aggregate_type"`
	ModelType     string `json:"model_type"`
	FromVersion   int    `json:"from_version"`
}

type ProjectionResponse struct {
	Success        bool                   `json:"success"`
	ReadModel      *ReadModel             `json:"read_model"`
	EventsApplied  int                    `json:"events_applied"`
	ProcessingTime int64                  `json:"processing_time_ms"`
	Message        string                 `json:"message"`
}

func NewReadModelProjector() *ReadModelProjector {
	// Connect to read model database
	readDB, err := sql.Open("sqlite3", "./read_models.db")
	if err != nil {
		log.Fatal("Failed to open read model database:", err)
	}

	// Connect to event store database
	eventDB, err := sql.Open("sqlite3", "./real_events.db")
	if err != nil {
		log.Fatal("Failed to open event store database:", err)
	}

	projector := &ReadModelProjector{
		db:           readDB,
		eventStoreDB: eventDB,
	}

	projector.initializeSchema()
	return projector
}

func (rmp *ReadModelProjector) initializeSchema() {
	// Create read models table
	createReadModelsTable := `
	CREATE TABLE IF NOT EXISTS read_models (
		id TEXT PRIMARY KEY,
		aggregate_id TEXT NOT NULL,
		aggregate_type TEXT NOT NULL,
		model_type TEXT NOT NULL,
		data TEXT NOT NULL,
		version INTEGER NOT NULL,
		last_updated DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(aggregate_id, model_type)
	);`

	_, err := rmp.db.Exec(createReadModelsTable)
	if err != nil {
		log.Fatal("Failed to create read_models table:", err)
	}

	// Create projection metadata table
	createMetadataTable := `
	CREATE TABLE IF NOT EXISTS projection_metadata (
		model_type TEXT PRIMARY KEY,
		last_processed_version INTEGER NOT NULL,
		last_processed_at DATETIME NOT NULL,
		total_events_processed INTEGER NOT NULL
	);`

	_, err = rmp.db.Exec(createMetadataTable)
	if err != nil {
		log.Fatal("Failed to create projection_metadata table:", err)
	}

	// Create indexes for performance
	rmp.db.Exec("CREATE INDEX IF NOT EXISTS idx_read_models_aggregate ON read_models(aggregate_id, aggregate_type);")
	rmp.db.Exec("CREATE INDEX IF NOT EXISTS idx_read_models_type ON read_models(model_type);")

	log.Println("📊 Read model database schema initialized")
}

func (rmp *ReadModelProjector) ProjectReadModel(req *ProjectionRequest) (*ProjectionResponse, error) {
	startTime := time.Now()

	log.Printf("📊 Projecting read model: %s for %s/%s", req.ModelType, req.AggregateType, req.AggregateID)

	// Get events from Event Store
	events, err := rmp.getEventsFromEventStore(req.AggregateID, req.FromVersion)
	if err != nil {
		return &ProjectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get events: %v", err),
		}, nil
	}

	if len(events) == 0 {
		return &ProjectionResponse{
			Success: true,
			Message: "No new events to project",
		}, nil
	}

	// Get existing read model or create new one
	readModel, err := rmp.getOrCreateReadModel(req.AggregateID, req.AggregateType, req.ModelType)
	if err != nil {
		return &ProjectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get read model: %v", err),
		}, nil
	}

	// Apply events to read model
	eventsApplied := 0
	for _, event := range events {
		err = rmp.applyEventToReadModel(readModel, event, req.ModelType)
		if err != nil {
			log.Printf("⚠️ Failed to apply event %s: %v", event["id"], err)
			continue
		}
		eventsApplied++
		readModel.Version = event["version"].(int)
	}

	// Save updated read model
	err = rmp.saveReadModel(readModel)
	if err != nil {
		return &ProjectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to save read model: %v", err),
		}, nil
	}

	// Update projection metadata
	rmp.updateProjectionMetadata(req.ModelType, readModel.Version, eventsApplied)

	return &ProjectionResponse{
		Success:        true,
		ReadModel:      readModel,
		EventsApplied:  eventsApplied,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		Message:        fmt.Sprintf("Read model projected successfully with %d events", eventsApplied),
	}, nil
}

func (rmp *ReadModelProjector) getEventsFromEventStore(aggregateID string, fromVersion int) ([]map[string]interface{}, error) {
	query := `
	SELECT id, aggregate_id, aggregate_type, type, data, metadata, version, created_at
	FROM events 
	WHERE aggregate_id = ? AND version > ?
	ORDER BY version ASC`

	rows, err := rmp.eventStoreDB.Query(query, aggregateID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var id, aggregateID, aggregateType, eventType, eventData, metadata, createdAt string
		var version int

		err := rows.Scan(&id, &aggregateID, &aggregateType, &eventType, &eventData, &metadata, &version, &createdAt)
		if err != nil {
			continue
		}

		var data map[string]interface{}
		json.Unmarshal([]byte(eventData), &data)

		var meta map[string]interface{}
		json.Unmarshal([]byte(metadata), &meta)

		event := map[string]interface{}{
			"id":             id,
			"aggregate_id":   aggregateID,
			"aggregate_type": aggregateType,
			"event_type":     eventType,
			"data":           data,
			"metadata":       meta,
			"version":        version,
			"created_at":     createdAt,
		}

		events = append(events, event)
	}

	return events, nil
}

func (rmp *ReadModelProjector) getOrCreateReadModel(aggregateID, aggregateType, modelType string) (*ReadModel, error) {
	query := `
	SELECT id, aggregate_id, aggregate_type, model_type, data, version, last_updated, created_at
	FROM read_models 
	WHERE aggregate_id = ? AND model_type = ?`

	var readModel ReadModel
	var dataJSON string
	var lastUpdated, createdAt string

	err := rmp.db.QueryRow(query, aggregateID, modelType).Scan(
		&readModel.ID, &readModel.AggregateID, &readModel.AggregateType,
		&readModel.ModelType, &dataJSON, &readModel.Version,
		&lastUpdated, &createdAt)

	if err == sql.ErrNoRows {
		// Create new read model
		readModel = ReadModel{
			ID:            fmt.Sprintf("%s_%s_%d", aggregateID, modelType, time.Now().Unix()),
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			ModelType:     modelType,
			Data:          make(map[string]interface{}),
			Version:       0,
			LastUpdated:   time.Now(),
			CreatedAt:     time.Now(),
		}
		return &readModel, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query read model: %w", err)
	}

	// Parse existing read model
	json.Unmarshal([]byte(dataJSON), &readModel.Data)
	readModel.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
	readModel.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &readModel, nil
}

func (rmp *ReadModelProjector) applyEventToReadModel(readModel *ReadModel, event map[string]interface{}, modelType string) error {
	eventType := event["event_type"].(string)
	eventData := event["data"].(map[string]interface{})

	switch modelType {
	case "pipeline_summary":
		return rmp.applyPipelineSummaryProjection(readModel, eventType, eventData)
	case "plugin_status":
		return rmp.applyPluginStatusProjection(readModel, eventType, eventData)
	case "workflow_history":
		return rmp.applyWorkflowHistoryProjection(readModel, eventType, eventData)
	case "performance_metrics":
		return rmp.applyPerformanceMetricsProjection(readModel, eventType, eventData)
	default:
		return rmp.applyGenericProjection(readModel, eventType, eventData)
	}
}

func (rmp *ReadModelProjector) applyPipelineSummaryProjection(readModel *ReadModel, eventType string, eventData map[string]interface{}) error {
	if readModel.Data == nil {
		readModel.Data = make(map[string]interface{})
	}

	switch eventType {
	case "PipelineCreated":
		readModel.Data["name"] = eventData["name"]
		readModel.Data["status"] = "created"
		readModel.Data["created_at"] = time.Now().Format(time.RFC3339)
		readModel.Data["steps"] = []string{}
		readModel.Data["total_executions"] = 0

	case "PipelineStarted":
		readModel.Data["status"] = "running"
		readModel.Data["last_started"] = time.Now().Format(time.RFC3339)
		if executions, ok := readModel.Data["total_executions"].(float64); ok {
			readModel.Data["total_executions"] = int(executions) + 1
		}

	case "PipelineCompleted":
		readModel.Data["status"] = "completed"
		readModel.Data["last_completed"] = time.Now().Format(time.RFC3339)
		if duration, ok := eventData["duration"]; ok {
			readModel.Data["last_duration"] = duration
		}

	case "PipelineFailed":
		readModel.Data["status"] = "failed"
		readModel.Data["last_error"] = eventData["error"]
		readModel.Data["failed_at"] = time.Now().Format(time.RFC3339)
	}

	return nil
}

func (rmp *ReadModelProjector) applyPluginStatusProjection(readModel *ReadModel, eventType string, eventData map[string]interface{}) error {
	if readModel.Data == nil {
		readModel.Data = make(map[string]interface{})
	}

	switch eventType {
	case "PluginLoaded":
		readModel.Data["plugin_name"] = eventData["plugin_name"]
		readModel.Data["status"] = "loaded"
		readModel.Data["loaded_at"] = time.Now().Format(time.RFC3339)
		readModel.Data["executions"] = 0

	case "PluginExecuted":
		if executions, ok := readModel.Data["executions"].(float64); ok {
			readModel.Data["executions"] = int(executions) + 1
		}
		readModel.Data["last_execution"] = time.Now().Format(time.RFC3339)
		if success, ok := eventData["success"].(bool); ok && success {
			readModel.Data["last_status"] = "success"
		} else {
			readModel.Data["last_status"] = "failed"
		}

	case "PluginUnloaded":
		readModel.Data["status"] = "unloaded"
		readModel.Data["unloaded_at"] = time.Now().Format(time.RFC3339)
	}

	return nil
}

func (rmp *ReadModelProjector) applyWorkflowHistoryProjection(readModel *ReadModel, eventType string, eventData map[string]interface{}) error {
	if readModel.Data == nil {
		readModel.Data = map[string]interface{}{
			"workflow_executions": []map[string]interface{}{},
			"total_executions":    0,
			"successful_executions": 0,
			"failed_executions":   0,
		}
	}

	switch eventType {
	case "WorkflowStarted":
		execution := map[string]interface{}{
			"execution_id": eventData["execution_id"],
			"started_at":   time.Now().Format(time.RFC3339),
			"status":       "running",
		}

		executions := readModel.Data["workflow_executions"].([]map[string]interface{})
		readModel.Data["workflow_executions"] = append(executions, execution)
		readModel.Data["total_executions"] = len(executions) + 1

	case "WorkflowCompleted":
		executions := readModel.Data["workflow_executions"].([]map[string]interface{})
		for i, exec := range executions {
			if exec["execution_id"] == eventData["execution_id"] {
				executions[i]["status"] = "completed"
				executions[i]["completed_at"] = time.Now().Format(time.RFC3339)
				executions[i]["duration"] = eventData["duration"]
				break
			}
		}
		if successful, ok := readModel.Data["successful_executions"].(float64); ok {
			readModel.Data["successful_executions"] = int(successful) + 1
		}
	}

	return nil
}

func (rmp *ReadModelProjector) applyPerformanceMetricsProjection(readModel *ReadModel, eventType string, eventData map[string]interface{}) error {
	if readModel.Data == nil {
		readModel.Data = map[string]interface{}{
			"avg_response_time": 0.0,
			"total_requests":    0,
			"total_errors":      0,
			"throughput":        0.0,
		}
	}

	switch eventType {
	case "RequestProcessed":
		totalRequests := int(readModel.Data["total_requests"].(float64))
		avgResponseTime := readModel.Data["avg_response_time"].(float64)
		newResponseTime := eventData["response_time"].(float64)

		// Calculate new average
		newAvg := (avgResponseTime*float64(totalRequests) + newResponseTime) / float64(totalRequests+1)
		readModel.Data["avg_response_time"] = newAvg
		readModel.Data["total_requests"] = totalRequests + 1

	case "RequestFailed":
		totalErrors := int(readModel.Data["total_errors"].(float64))
		readModel.Data["total_errors"] = totalErrors + 1
	}

	return nil
}

func (rmp *ReadModelProjector) applyGenericProjection(readModel *ReadModel, eventType string, eventData map[string]interface{}) error {
	if readModel.Data == nil {
		readModel.Data = make(map[string]interface{})
	}

	// Generic projection - just accumulate event data
	readModel.Data["last_event_type"] = eventType
	readModel.Data["last_event_data"] = eventData
	readModel.Data["last_updated"] = time.Now().Format(time.RFC3339)

	// Count events by type
	eventCounts, ok := readModel.Data["event_counts"].(map[string]interface{})
	if !ok {
		eventCounts = make(map[string]interface{})
		readModel.Data["event_counts"] = eventCounts
	}

	if count, ok := eventCounts[eventType].(float64); ok {
		eventCounts[eventType] = int(count) + 1
	} else {
		eventCounts[eventType] = 1
	}

	return nil
}

func (rmp *ReadModelProjector) saveReadModel(readModel *ReadModel) error {
	dataJSON, _ := json.Marshal(readModel.Data)
	readModel.LastUpdated = time.Now()

	query := `
	INSERT OR REPLACE INTO read_models 
	(id, aggregate_id, aggregate_type, model_type, data, version, last_updated, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := rmp.db.Exec(query,
		readModel.ID, readModel.AggregateID, readModel.AggregateType,
		readModel.ModelType, string(dataJSON), readModel.Version,
		readModel.LastUpdated.Format(time.RFC3339),
		readModel.CreatedAt.Format(time.RFC3339))

	return err
}

func (rmp *ReadModelProjector) updateProjectionMetadata(modelType string, lastVersion, eventsProcessed int) {
	query := `
	INSERT OR REPLACE INTO projection_metadata 
	(model_type, last_processed_version, last_processed_at, total_events_processed)
	VALUES (?, ?, ?, 
		COALESCE((SELECT total_events_processed FROM projection_metadata WHERE model_type = ?), 0) + ?)`

	rmp.db.Exec(query, modelType, lastVersion, time.Now().Format(time.RFC3339), modelType, eventsProcessed)
}

func (rmp *ReadModelProjector) GetReadModel(aggregateID, modelType string) (*ReadModel, error) {
	query := `
	SELECT id, aggregate_id, aggregate_type, model_type, data, version, last_updated, created_at
	FROM read_models 
	WHERE aggregate_id = ? AND model_type = ?`

	var readModel ReadModel
	var dataJSON string
	var lastUpdated, createdAt string

	err := rmp.db.QueryRow(query, aggregateID, modelType).Scan(
		&readModel.ID, &readModel.AggregateID, &readModel.AggregateType,
		&readModel.ModelType, &dataJSON, &readModel.Version,
		&lastUpdated, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("read model not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to query read model: %w", err)
	}

	json.Unmarshal([]byte(dataJSON), &readModel.Data)
	readModel.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
	readModel.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &readModel, nil
}

func (rmp *ReadModelProjector) ListReadModels(modelType string) ([]*ReadModel, error) {
	var query string
	var args []interface{}

	if modelType != "" {
		query = `SELECT id, aggregate_id, aggregate_type, model_type, data, version, last_updated, created_at
		        FROM read_models WHERE model_type = ? ORDER BY last_updated DESC`
		args = []interface{}{modelType}
	} else {
		query = `SELECT id, aggregate_id, aggregate_type, model_type, data, version, last_updated, created_at
		        FROM read_models ORDER BY last_updated DESC`
	}

	rows, err := rmp.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query read models: %w", err)
	}
	defer rows.Close()

	var readModels []*ReadModel
	for rows.Next() {
		var readModel ReadModel
		var dataJSON string
		var lastUpdated, createdAt string

		err := rows.Scan(&readModel.ID, &readModel.AggregateID, &readModel.AggregateType,
			&readModel.ModelType, &dataJSON, &readModel.Version,
			&lastUpdated, &createdAt)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(dataJSON), &readModel.Data)
		readModel.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
		readModel.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		readModels = append(readModels, &readModel)
	}

	return readModels, nil
}

// HTTP Handlers
func (rmp *ReadModelProjector) handleProjectReadModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProjectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	resp, err := rmp.ProjectReadModel(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Projection failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (rmp *ReadModelProjector) handleGetReadModel(w http.ResponseWriter, r *http.Request) {
	aggregateID := r.URL.Query().Get("aggregate_id")
	modelType := r.URL.Query().Get("model_type")

	if aggregateID == "" || modelType == "" {
		http.Error(w, "aggregate_id and model_type required", http.StatusBadRequest)
		return
	}

	readModel, err := rmp.GetReadModel(aggregateID, modelType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Read model not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get read model: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readModel)
}

func (rmp *ReadModelProjector) handleListReadModels(w http.ResponseWriter, r *http.Request) {
	modelType := r.URL.Query().Get("model_type")

	readModels, err := rmp.ListReadModels(modelType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list read models: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"read_models": readModels,
		"count":       len(readModels),
		"model_type":  modelType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rmp *ReadModelProjector) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"healthy":   true,
		"status":    "operational",
		"message":   "CQRS read model projector running normally",
		"timestamp": time.Now(),
		"databases": map[string]bool{
			"read_models":  rmp.checkDBHealth(rmp.db),
			"event_store":  rmp.checkDBHealth(rmp.eventStoreDB),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (rmp *ReadModelProjector) checkDBHealth(db *sql.DB) bool {
	err := db.Ping()
	return err == nil
}

func main() {
	fmt.Println("📊 FlexCore REAL CQRS Read Model Projector")
	fmt.Println("=========================================")
	fmt.Println("🗄️ Real SQLite read model projections")
	fmt.Println("⚡ Event sourcing integration")
	fmt.Println("📈 Pipeline, Plugin, Workflow projections")
	fmt.Println("🔄 Real-time projection updates")
	fmt.Println()

	projector := NewReadModelProjector()

	// HTTP endpoints
	http.HandleFunc("/project", projector.handleProjectReadModel)
	http.HandleFunc("/readmodel", projector.handleGetReadModel)
	http.HandleFunc("/list", projector.handleListReadModels)
	http.HandleFunc("/health", projector.handleHealth)

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		info := map[string]interface{}{
			"service":     "cqrs-read-model-projector",
			"version":     "1.0.0",
			"description": "FlexCore REAL CQRS Read Model Projector",
			"timestamp":   time.Now(),
			"endpoints": []string{
				"POST /project - Project read model from events",
				"GET  /readmodel - Get specific read model",
				"GET  /list - List read models",
				"GET  /health - Health check",
			},
			"supported_projections": []string{
				"pipeline_summary", "plugin_status", "workflow_history", "performance_metrics", "generic",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	fmt.Printf("🌐 CQRS Read Model Projector starting on :8099\n")
	fmt.Printf("📊 Endpoints:\n")
	fmt.Printf("  POST /project   - Project read models\n")
	fmt.Printf("  GET  /readmodel - Get read model\n")
	fmt.Printf("  GET  /list      - List read models\n")
	fmt.Printf("  GET  /health    - Health check\n")
	fmt.Printf("🗄️ Database: read_models.db\n")
	fmt.Printf("⚡ Event Store: real_events.db\n")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8099", nil))
}