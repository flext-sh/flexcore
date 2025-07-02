// FlexCore - Professional distributed event-driven architecture
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flext/flexcore/internal/domain"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version information (set by build flags)
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

// Server represents the FlexCore HTTP server
type Server struct {
	core   *domain.FlexCore
	router *mux.Router
	config *domain.FlexCoreConfig
}

// NewServer creates a new FlexCore server
func NewServer(config *domain.FlexCoreConfig) (*Server, error) {
	// Create FlexCore instance
	coreResult := domain.NewFlexCore(config)
	if coreResult.IsFailure() {
		return nil, fmt.Errorf("failed to create FlexCore: %v", coreResult.Error())
	}

	server := &Server{
		core:   coreResult.Value(),
		config: config,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Health & Info
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/info", s.handleInfo).Methods("GET")
	s.router.HandleFunc("/metrics", promhttp.Handler().ServeHTTP).Methods("GET")

	// Event handling
	s.router.HandleFunc("/events", s.handleSendEvent).Methods("POST")
	s.router.HandleFunc("/events/batch", s.handleBatchEvents).Methods("POST")

	// Message queue
	s.router.HandleFunc("/queues/{queue}/messages", s.handleSendMessage).Methods("POST")
	s.router.HandleFunc("/queues/{queue}/messages", s.handleReceiveMessages).Methods("GET")

	// Workflow execution
	s.router.HandleFunc("/workflows/execute", s.handleExecuteWorkflow).Methods("POST")
	s.router.HandleFunc("/workflows/{id}/status", s.handleWorkflowStatus).Methods("GET")

	// Cluster management
	s.router.HandleFunc("/cluster/status", s.handleClusterStatus).Methods("GET")
}

// Start starts the server
func (s *Server) Start(port string) error {
	ctx := context.Background()

	// Start FlexCore
	if startResult := s.core.Start(ctx); startResult.IsFailure() {
		log.Printf("Warning: FlexCore start returned: %v", startResult.Error())
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)
		s.core.Stop(ctx)
	}()

	log.Printf("FlexCore server listening on port %s", port)
	log.Printf("Version: %s, Build: %s, Commit: %s", Version, BuildTime, CommitHash)
	return srv.ListenAndServe()
}

// HTTP Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now().Unix(),
		"version":    Version,
		"build_time": BuildTime,
		"commit":     CommitHash,
	})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":    "flexcore",
		"version":    Version,
		"build_time": BuildTime,
		"commit":     CommitHash,
		"node_id":    s.config.NodeID,
		"cluster":    s.config.ClusterName,
		"uptime":     time.Since(time.Now()).Seconds(), // Would track actual start time
	})
}

func (s *Server) handleSendEvent(w http.ResponseWriter, r *http.Request) {
	var eventData struct {
		Type        string                 `json:"type"`
		AggregateID string                 `json:"aggregate_id"`
		Data        map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eventData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if eventData.Type == "" {
		http.Error(w, "Event type is required", http.StatusBadRequest)
		return
	}

	event := domain.NewEvent(eventData.Type, eventData.AggregateID, eventData.Data)
	result := s.core.SendEvent(r.Context(), event)
	
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     event.ID,
		"status": "accepted",
	})
}

func (s *Server) handleBatchEvents(w http.ResponseWriter, r *http.Request) {
	var events []struct {
		Type        string                 `json:"type"`
		AggregateID string                 `json:"aggregate_id"`
		Data        map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	results := make([]map[string]interface{}, 0, len(events))
	
	for _, eventData := range events {
		event := domain.NewEvent(eventData.Type, eventData.AggregateID, eventData.Data)
		result := s.core.SendEvent(r.Context(), event)
		
		status := "accepted"
		if result.IsFailure() {
			status = "failed"
		}

		results = append(results, map[string]interface{}{
			"id":     event.ID,
			"status": status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"total":   len(results),
	})
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queue := vars["queue"]
	
	var messageData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&messageData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	message := domain.NewMessage(queue, messageData)
	result := s.core.SendMessage(r.Context(), queue, message)
	
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     message.ID,
		"queue":  queue,
		"status": "queued",
	})
}

func (s *Server) handleReceiveMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queue := vars["queue"]
	maxMessages := 10 // Could be from query param
	
	result := s.core.ReceiveMessages(r.Context(), queue, maxMessages)
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusInternalServerError)
		return
	}

	messages := result.Value()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
		"queue":    queue,
	})
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string                 `json:"path"`
		Input map[string]interface{} `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Workflow path is required", http.StatusBadRequest)
		return
	}

	result := s.core.ExecuteWorkflow(r.Context(), req.Path, req.Input)
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": result.Value(),
		"status": "started",
	})
}

func (s *Server) handleWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	result := s.core.GetWorkflowStatus(r.Context(), jobID)
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value())
}

func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	result := s.core.GetClusterStatus(r.Context())
	if result.IsFailure() {
		http.Error(w, result.Error().Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value())
}

func main() {
	// Parse flags
	var (
		configFile = flag.String("config", "", "Configuration file (optional)")
		port       = flag.String("port", "8080", "HTTP server port")
		nodeID     = flag.String("node", "", "Node ID (auto-generated if empty)")
		help       = flag.Bool("help", false, "Show help")
		version    = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Printf("FlexCore %s (build %s, commit %s)\n", Version, BuildTime, CommitHash)
		return
	}

	// Create configuration
	config := domain.DefaultConfig()
	if *nodeID != "" {
		config.NodeID = *nodeID
	}

	// TODO: Load from config file if provided
	if *configFile != "" {
		log.Printf("Loading configuration from %s", *configFile)
		// Implementation would load from file
	}

	// Override from environment variables
	if url := os.Getenv("WINDMILL_URL"); url != "" {
		config.WindmillURL = url
	}
	if token := os.Getenv("WINDMILL_TOKEN"); token != "" {
		config.WindmillToken = token
	}
	if workspace := os.Getenv("WINDMILL_WORKSPACE"); workspace != "" {
		config.WindmillWorkspace = workspace
	}

	log.Printf("Starting FlexCore node: %s", config.NodeID)

	// Create and start server
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(*port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}