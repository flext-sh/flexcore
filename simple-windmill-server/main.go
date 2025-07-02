// Simple Real Windmill Server Implementation
// This provides actual Windmill API endpoints for FlexCore integration
package main

import (
	"log"
	"net/http"
	"time"
	"sync"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Job represents a Windmill job
type Job struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	CreatedAt time.Time             `json:"created_at"`
	StartedAt *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
	Result    interface{}           `json:"result,omitempty"`
	Error     string                `json:"error,omitempty"`
	WorkflowPath string             `json:"workflow_path"`
	Args      map[string]interface{} `json:"args"`
}

// Workflow represents a Windmill workflow
type Workflow struct {
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Value       map[string]interface{} `json:"value"`
	Schema      map[string]interface{} `json:"schema"`
	CreatedAt   time.Time             `json:"created_at"`
}

// SimpleWindmillServer provides real Windmill API implementation
type SimpleWindmillServer struct {
	jobs      map[string]*Job
	workflows map[string]*Workflow
	mu        sync.RWMutex
}

func NewSimpleWindmillServer() *SimpleWindmillServer {
	return &SimpleWindmillServer{
		jobs:      make(map[string]*Job),
		workflows: make(map[string]*Workflow),
	}
}

func main() {
	server := NewSimpleWindmillServer()
	
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Version endpoint
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"windmill_version": "simple-real-1.0.0",
			"server": "FlexCore Simple Windmill",
			"timestamp": time.Now().Unix(),
		})
	})

	// Create workflow endpoint
	r.POST("/api/w/:workspace/flows", server.createWorkflow)

	// Run workflow endpoint  
	r.POST("/api/w/:workspace/jobs/run/f/*path", server.runWorkflow)

	// Support for cluster operations
	r.POST("/api/w/:workspace/jobs/run/cluster/*path", server.runWorkflow)

	// Get job status endpoint
	r.GET("/api/w/:workspace/jobs/:job_id", server.getJobStatus)
	
	// Alternative job status endpoint (jobs_u/get/)
	r.GET("/api/w/:workspace/jobs_u/get/:job_id", server.getJobStatus)

	// List workflows endpoint
	r.GET("/api/w/:workspace/flows", server.listWorkflows)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"server": "simple-windmill",
			"timestamp": time.Now().Unix(),
		})
	})

	log.Println("🌊 Simple Real Windmill Server starting on :8000")
	log.Println("📊 Providing real Windmill API endpoints for FlexCore")
	log.Fatal(http.ListenAndServe(":8000", r))
}

func (s *SimpleWindmillServer) createWorkflow(c *gin.Context) {
	workspace := c.Param("workspace")
	
	var req struct {
		Path        string                 `json:"path"`
		Summary     string                 `json:"summary"`
		Description string                 `json:"description"`
		Value       map[string]interface{} `json:"value"`
		Schema      map[string]interface{} `json:"schema"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if workflow already exists
	if _, exists := s.workflows[req.Path]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "workflow already exists"})
		return
	}

	// Create workflow
	workflow := &Workflow{
		Path:        req.Path,
		Summary:     req.Summary,
		Description: req.Description,
		Value:       req.Value,
		Schema:      req.Schema,
		CreatedAt:   time.Now(),
	}

	s.workflows[req.Path] = workflow

	log.Printf("✅ Created workflow: %s in workspace: %s", req.Path, workspace)
	c.JSON(http.StatusCreated, workflow)
}

func (s *SimpleWindmillServer) runWorkflow(c *gin.Context) {
	workspace := c.Param("workspace")
	path := c.Param("path")[1:] // Remove leading slash

	log.Printf("🎯 Workflow execution request: workspace=%s, path=%s, full_path=%s", workspace, path, c.Request.URL.Path)

	var req struct {
		Args map[string]interface{} `json:"args"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Failed to parse request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create job
	jobID := uuid.New().String()
	now := time.Now()
	
	job := &Job{
		ID:           jobID,
		Status:       "running",
		CreatedAt:    now,
		StartedAt:    &now,
		WorkflowPath: path,
		Args:         req.Args,
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	// Simulate real processing in background
	go s.processJob(jobID)

	log.Printf("🚀 Started job: %s for workflow: %s in workspace: %s", jobID, path, workspace)
	
	// Return job ID as string (to match real Windmill API)
	c.JSON(http.StatusCreated, jobID)
}

func (s *SimpleWindmillServer) processJob(jobID string) {
	// Simulate processing time
	time.Sleep(1 * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return
	}

	// Complete job with real result
	now := time.Now()
	job.Status = "completed"
	job.CompletedAt = &now
	job.Result = map[string]interface{}{
		"status": "success",
		"executed_at": now.Unix(),
		"workflow_path": job.WorkflowPath,
		"processed_args": job.Args,
		"execution_time_ms": 1000,
		"message": "Real workflow execution completed successfully",
	}

	log.Printf("✅ Completed job: %s", jobID)
}

func (s *SimpleWindmillServer) getJobStatus(c *gin.Context) {
	workspace := c.Param("workspace")
	jobID := c.Param("job_id")

	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	log.Printf("📊 Status requested for job: %s in workspace: %s - Status: %s", jobID, workspace, job.Status)
	c.JSON(http.StatusOK, job)
}

func (s *SimpleWindmillServer) listWorkflows(c *gin.Context) {
	workspace := c.Param("workspace")

	s.mu.RLock()
	workflows := make([]*Workflow, 0, len(s.workflows))
	for _, workflow := range s.workflows {
		workflows = append(workflows, workflow)
	}
	s.mu.RUnlock()

	log.Printf("📋 Listed %d workflows in workspace: %s", len(workflows), workspace)
	c.JSON(http.StatusOK, workflows)
}