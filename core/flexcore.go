// Package core provides the main FlexCore distributed event-driven architecture
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flext/flexcore/infrastructure/plugins"
	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// FlexCore represents the main distributed event processing engine
type FlexCore struct {
	// Windmill integration
	windmillClient  *windmill.Client
	workflowManager *windmill.WorkflowManager
	
	// Plugin system
	pluginManager *plugins.PluginManager
	
	// Event processing
	eventRouter     *EventRouter
	messageQueue    *DistributedMessageQueue
	scheduler       *DistributedScheduler
	realScheduler   *RealDistributedScheduler
	clusterManager  *ClusterManager
	
	// Configuration
	config *FlexCoreConfig
	
	// Lifecycle
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
}

// FlexCoreConfig represents FlexCore configuration
type FlexCoreConfig struct {
	// Windmill settings
	WindmillURL       string            `json:"windmill_url"`
	WindmillToken     string            `json:"windmill_token"`
	WindmillWorkspace string            `json:"windmill_workspace"`
	
	// Cluster settings
	ClusterName       string            `json:"cluster_name"`
	NodeID            string            `json:"node_id"`
	ClusterNodes      []string          `json:"cluster_nodes"`
	
	// Event routing
	EventRoutes       []EventRoute      `json:"event_routes"`
	MessageQueues     []QueueConfig     `json:"message_queues"`
	
	// Scheduling
	Schedulers        []SchedulerConfig `json:"schedulers"`
	
	// Plugin settings
	PluginDirectory   string            `json:"plugin_directory"`
	EnabledPlugins    []string          `json:"enabled_plugins"`
	
	// Performance settings
	MaxConcurrentJobs int               `json:"max_concurrent_jobs"`
	EventBufferSize   int               `json:"event_buffer_size"`
	RetryPolicy       RetryPolicy       `json:"retry_policy"`
	
	// Custom parameters
	CustomParams      map[string]interface{} `json:"custom_params"`
}

// EventRoute defines how events are routed between adapters
type EventRoute struct {
	Name            string            `json:"name"`
	SourceAdapter   string            `json:"source_adapter"`
	TargetAdapters  []string          `json:"target_adapters"`
	EventFilter     EventFilter       `json:"event_filter"`
	Transform       *TransformConfig  `json:"transform,omitempty"`
	Async           bool              `json:"async"`
	MaxRetries      int               `json:"max_retries"`
	CustomParams    map[string]interface{} `json:"custom_params"`
}

// EventFilter defines filtering criteria for events
type EventFilter struct {
	EventTypes      []string          `json:"event_types"`
	SourceFilters   map[string]string `json:"source_filters"`
	ContentFilters  []ContentFilter   `json:"content_filters"`
	TimeWindow      *TimeWindow       `json:"time_window,omitempty"`
}

// ContentFilter defines content-based filtering
type ContentFilter struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value     interface{} `json:"value"`
}

// TimeWindow defines time-based filtering
type TimeWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Timezone  string    `json:"timezone"`
}

// TransformConfig defines event transformation
type TransformConfig struct {
	Type       string                 `json:"type"` // script, mapping, plugin
	Script     string                 `json:"script,omitempty"`
	Language   string                 `json:"language,omitempty"`
	Mapping    map[string]interface{} `json:"mapping,omitempty"`
	PluginName string                 `json:"plugin_name,omitempty"`
}

// QueueConfig defines message queue configuration
type QueueConfig struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"` // fifo, priority, delayed
	MaxSize        int               `json:"max_size"`
	TTL            time.Duration     `json:"ttl"`
	DeadLetterQueue string           `json:"dead_letter_queue,omitempty"`
	CustomParams   map[string]interface{} `json:"custom_params"`
}

// SchedulerConfig defines scheduler configuration
type SchedulerConfig struct {
	Name         string            `json:"name"`
	CronExpression string          `json:"cron_expression"`
	Timezone     string            `json:"timezone"`
	WorkflowPath string            `json:"workflow_path"`
	Input        map[string]interface{} `json:"input"`
	Singleton    bool              `json:"singleton"`    // Only one instance runs at a time
	MaxInstances int               `json:"max_instances"` // Max concurrent instances
	CustomParams map[string]interface{} `json:"custom_params"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// NewFlexCore creates a new FlexCore instance
func NewFlexCore(config *FlexCoreConfig) result.Result[*FlexCore] {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return result.Failure[*FlexCore](err)
	}

	// Create Windmill client
	windmillConfig := windmill.Config{
		BaseURL:   config.WindmillURL,
		Token:     config.WindmillToken,
		Workspace: config.WindmillWorkspace,
		Timeout:   30 * time.Second,
	}
	windmillClient := windmill.NewClient(windmillConfig)
	
	// Create workflow manager
	workflowManager := windmill.NewWorkflowManager(windmillClient)
	
	// Create plugin manager
	pluginManager := plugins.NewPluginManager(config.PluginDirectory)
	
	// Create core components
	eventRouter := NewEventRouter(windmillClient, config)
	messageQueue := NewDistributedMessageQueue(windmillClient, config)
	scheduler := NewDistributedScheduler(windmillClient, config)
	realScheduler := NewRealDistributedScheduler(windmillClient, config, config.NodeID)
	clusterManager := NewClusterManager(windmillClient, config)

	flexCore := &FlexCore{
		windmillClient:  windmillClient,
		workflowManager: workflowManager,
		pluginManager:   pluginManager,
		eventRouter:     eventRouter,
		messageQueue:    messageQueue,
		scheduler:       scheduler,
		realScheduler:   realScheduler,
		clusterManager:  clusterManager,
		config:          config,
		stopChan:        make(chan struct{}),
	}

	return result.Success(flexCore)
}

// Start starts the FlexCore engine
func (fc *FlexCore) Start(ctx context.Context) result.Result[bool] {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.running {
		return result.Failure[bool](errors.ValidationError("FlexCore is already running"))
	}

	// Initialize Windmill workflows for internal operations
	if err := fc.initializeWindmillWorkflows(ctx); err != nil {
		return result.Failure[bool](err)
	}

	// Start plugin manager
	if err := fc.pluginManager.LoadAllPlugins(ctx); err != nil {
		return result.Failure[bool](err)
	}

	// Start core components
	if err := fc.startCoreComponents(ctx); err != nil {
		return result.Failure[bool](err)
	}

	fc.running = true
	
	// Start background processes
	fc.waitGroup.Add(1)
	go fc.runBackgroundProcesses(ctx)

	return result.Success(true)
}

// Stop stops the FlexCore engine
func (fc *FlexCore) Stop(ctx context.Context) result.Result[bool] {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if !fc.running {
		return result.Success(true)
	}

	// Signal stop
	close(fc.stopChan)
	
	// Wait for background processes to finish
	done := make(chan struct{})
	go func() {
		fc.waitGroup.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// Graceful shutdown
	case <-time.After(30 * time.Second):
		// Force shutdown
	}

	// Stop components
	fc.scheduler.Stop(ctx)
	fc.clusterManager.Stop(ctx)
	fc.pluginManager.Shutdown()

	fc.running = false
	return result.Success(true)
}

// RegisterAdapter registers an adapter with the event system
func (fc *FlexCore) RegisterAdapter(ctx context.Context, adapter *AdapterConfig) result.Result[bool] {
	// Create Windmill workflow for adapter
	workflowDef := fc.createAdapterWorkflow(adapter)
	
	// Register workflow
	registerResult := fc.workflowManager.RegisterWorkflow(ctx, workflowDef)
	if registerResult.IsFailure() {
		return registerResult
	}

	// Register event routes for this adapter
	for _, route := range fc.config.EventRoutes {
		if route.SourceAdapter == adapter.Name {
			if err := fc.eventRouter.RegisterRoute(ctx, &route); err != nil {
				return result.Failure[bool](err)
			}
		}
	}

	return result.Success(true)
}

// SendEvent sends an event through the distributed event system
func (fc *FlexCore) SendEvent(ctx context.Context, event *Event) result.Result[bool] {
	return fc.eventRouter.RouteEvent(ctx, event)
}

// SendMessage sends a message through the distributed message queue
func (fc *FlexCore) SendMessage(ctx context.Context, queueName string, message *Message) result.Result[bool] {
	return fc.messageQueue.SendMessage(ctx, queueName, message)
}

// ReceiveMessages receives messages from a distributed queue
func (fc *FlexCore) ReceiveMessages(ctx context.Context, queueName string, maxMessages int) result.Result[[]*Message] {
	return fc.messageQueue.ReceiveMessages(ctx, queueName, maxMessages)
}

// ScheduleJob schedules a job using the distributed scheduler
func (fc *FlexCore) ScheduleJob(ctx context.Context, schedulerName string, input map[string]interface{}) result.Result[string] {
	return fc.scheduler.ScheduleJob(ctx, schedulerName, input)
}

// GetClusterStatus gets the current cluster status
func (fc *FlexCore) GetClusterStatus(ctx context.Context) result.Result[*ClusterStatus] {
	return fc.clusterManager.GetStatus(ctx)
}

// ExecuteWorkflow executes a workflow with distributed orchestration
func (fc *FlexCore) ExecuteWorkflow(ctx context.Context, workflowPath string, input map[string]interface{}) result.Result[string] {
	return fc.workflowManager.ExecuteWorkflowAsync(ctx, workflowPath, input)
}

// GetWorkflowStatus gets the status of a workflow execution
func (fc *FlexCore) GetWorkflowStatus(ctx context.Context, jobID string) result.Result[*windmill.ExecutionResult] {
	return fc.workflowManager.GetExecutionStatus(ctx, jobID)
}

// RegisterCustomWorkflow registers a custom workflow
func (fc *FlexCore) RegisterCustomWorkflow(ctx context.Context, workflow *windmill.WorkflowDefinition) result.Result[bool] {
	return fc.workflowManager.RegisterWorkflow(ctx, workflow)
}

// SetCustomParameter sets a custom parameter
func (fc *FlexCore) SetCustomParameter(key string, value interface{}) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	if fc.config.CustomParams == nil {
		fc.config.CustomParams = make(map[string]interface{})
	}
	fc.config.CustomParams[key] = value
}

// GetCustomParameter gets a custom parameter
func (fc *FlexCore) GetCustomParameter(key string) (interface{}, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	
	if fc.config.CustomParams == nil {
		return nil, false
	}
	value, exists := fc.config.CustomParams[key]
	return value, exists
}

// GetMetrics gets FlexCore metrics
func (fc *FlexCore) GetMetrics(ctx context.Context) result.Result[*Metrics] {
	metrics := &Metrics{
		EventsProcessed:   fc.eventRouter.GetMetrics().EventsProcessed,
		MessagesQueued:    fc.messageQueue.GetMetrics().MessagesQueued,
		ScheduledJobs:     fc.scheduler.GetMetrics().ScheduledJobs,
		ActivePlugins:     int64(fc.pluginManager.GetActivePluginCount()),
		ClusterNodes:      int64(len(fc.config.ClusterNodes)),
		UptimeSeconds:     time.Since(fc.getStartTime()).Seconds(),
		CustomMetrics:     make(map[string]interface{}),
	}
	
	return result.Success(metrics)
}

// Helper methods

func (fc *FlexCore) initializeWindmillWorkflows(ctx context.Context) error {
	// Create core system workflows
	workflows := []*windmill.WorkflowDefinition{
		fc.createEventRoutingWorkflow(),
		fc.createMessageQueueWorkflow(),
		fc.createSchedulerWorkflow(),
		fc.createClusterCoordinationWorkflow(),
	}
	
	for _, workflow := range workflows {
		if result := fc.workflowManager.RegisterWorkflow(ctx, workflow); result.IsFailure() {
			return result.Error()
		}
	}
	
	return nil
}

func (fc *FlexCore) createEventRoutingWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        "system/event_routing",
		Name:        "System Event Routing",
		Description: "Core event routing for FlexCore",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "route_events",
				Name:     "Route Events",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(events: list):
    """Route events through the system"""
    return {"routed_events": len(events)}
`,
				Input: map[string]interface{}{
					"events": "{{ input.events }}",
				},
			},
		},
		Timeout: time.Minute * 5,
	}
}

func (fc *FlexCore) createMessageQueueWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        "system/message_queue",
		Name:        "System Message Queue",
		Description: "Core message queuing for FlexCore",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "process_messages",
				Name:     "Process Messages",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(messages: list):
    """Process messages through the queue"""
    return {"processed_messages": len(messages)}
`,
				Input: map[string]interface{}{
					"messages": "{{ input.messages }}",
				},
			},
		},
		Timeout: time.Minute * 5,
	}
}

func (fc *FlexCore) createSchedulerWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        "system/scheduler",
		Name:        "System Scheduler",
		Description: "Core scheduling for FlexCore",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "schedule_jobs",
				Name:     "Schedule Jobs",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(jobs: list):
    """Schedule jobs in the system"""
    return {"scheduled_jobs": len(jobs)}
`,
				Input: map[string]interface{}{
					"jobs": "{{ input.jobs }}",
				},
			},
		},
		Timeout: time.Minute * 5,
	}
}

func (fc *FlexCore) createClusterCoordinationWorkflow() *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        "system/cluster_coordination",
		Name:        "System Cluster Coordination",
		Description: "Core cluster coordination for FlexCore",
		Steps: []windmill.WorkflowStep{
			{
				ID:       "coordinate_cluster",
				Name:     "Coordinate Cluster",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: `
def main(cluster_info: dict):
    """Coordinate cluster operations"""
    return {"cluster_status": "coordinated"}
`,
				Input: map[string]interface{}{
					"cluster_info": "{{ input.cluster_info }}",
				},
			},
		},
		Timeout: time.Minute * 5,
	}
}

func (fc *FlexCore) startCoreComponents(ctx context.Context) error {
	// Start event router
	if err := fc.eventRouter.Start(ctx); err != nil {
		return err
	}
	
	// Start message queue
	if err := fc.messageQueue.Start(ctx); err != nil {
		return err
	}
	
	// Start scheduler
	if schedulerResult := fc.scheduler.Start(ctx); schedulerResult.IsFailure() {
		return schedulerResult.Error()
	}
	
	// Start REAL distributed scheduler with timer-based singletons
	if realSchedulerResult := fc.realScheduler.Start(ctx); realSchedulerResult.IsFailure() {
		return realSchedulerResult.Error()
	}
	
	// Start cluster manager
	if err := fc.clusterManager.Start(ctx); err != nil {
		return err
	}
	
	return nil
}

func (fc *FlexCore) runBackgroundProcesses(ctx context.Context) {
	defer fc.waitGroup.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-fc.stopChan:
			return
		case <-ticker.C:
			// Perform periodic maintenance
			fc.performMaintenance(ctx)
		}
	}
}

func (fc *FlexCore) performMaintenance(ctx context.Context) {
	// Health checks, cleanup, metrics collection, etc.
	fc.eventRouter.PerformMaintenance(ctx)
	fc.messageQueue.PerformMaintenance(ctx)
	fc.scheduler.PerformMaintenance(ctx)
	fc.clusterManager.PerformMaintenance(ctx)
}

func (fc *FlexCore) getStartTime() time.Time {
	// This would be stored during Start()
	return time.Now() // Placeholder
}

func (fc *FlexCore) createAdapterWorkflow(adapter *AdapterConfig) *windmill.WorkflowDefinition {
	return &windmill.WorkflowDefinition{
		Path:        fmt.Sprintf("adapters/%s/main", adapter.Name),
		Name:        fmt.Sprintf("Adapter: %s", adapter.Name),
		Description: fmt.Sprintf("Main workflow for %s adapter", adapter.Name),
		Steps: []windmill.WorkflowStep{
			{
				ID:       "receive_event",
				Name:     "Receive Event",
				Type:     windmill.StepTypeScript,
				Language: "python3",
				Script: fmt.Sprintf(`
def main(event: dict, route_name: str, source: str, target: str, custom_params: dict):
    """Handle event for %s adapter"""
    import json
    import time
    
    # Process event using adapter configuration
    adapter_config = %s
    
    result = {
        "adapter_name": "%s",
        "event_id": event.get("id"),
        "processed_at": time.time(),
        "status": "processed"
    }
    
    return result
`, adapter.Name, fmt.Sprintf("%+v", adapter.Config), adapter.Name),
				Input: map[string]interface{}{
					"event":         "{{ input.event }}",
					"route_name":    "{{ input.route_name }}",
					"source":        "{{ input.source }}",
					"target":        "{{ input.target }}",
					"custom_params": "{{ input.custom_params }}",
				},
			},
		},
		Timeout: time.Minute * 10,
	}
}

func validateConfig(config *FlexCoreConfig) error {
	if config.WindmillURL == "" {
		return errors.ValidationError("windmill_url is required")
	}
	if config.WindmillToken == "" {
		return errors.ValidationError("windmill_token is required")
	}
	if config.WindmillWorkspace == "" {
		return errors.ValidationError("windmill_workspace is required")
	}
	if config.ClusterName == "" {
		return errors.ValidationError("cluster_name is required")
	}
	if config.NodeID == "" {
		return errors.ValidationError("node_id is required")
	}
	return nil
}

// Additional types

// AdapterConfig represents adapter configuration
type AdapterConfig struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	PluginName   string                 `json:"plugin_name"`
	Config       map[string]interface{} `json:"config"`
	EventTypes   []string               `json:"event_types"`
	CustomParams map[string]interface{} `json:"custom_params"`
}

// Event represents a distributed event
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Timestamp   time.Time             `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Headers     map[string]string      `json:"headers"`
	RoutingKey  string                 `json:"routing_key"`
	Priority    int                    `json:"priority"`
}

// Message represents a distributed message
type Message struct {
	ID          string                 `json:"id"`
	Queue       string                 `json:"queue"`
	Content     interface{}            `json:"content"`
	Headers     map[string]string      `json:"headers"`
	CreatedAt   time.Time             `json:"created_at"`
	ExpiresAt   *time.Time            `json:"expires_at"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Priority    int                    `json:"priority"`
}

// ClusterStatus represents cluster status
type ClusterStatus struct {
	ClusterName   string              `json:"cluster_name"`
	NodeID        string              `json:"node_id"`
	NodeCount     int                 `json:"node_count"`
	ActiveNodes   []NodeInfo          `json:"active_nodes"`
	LeaderNode    string              `json:"leader_node"`
	LastHeartbeat time.Time           `json:"last_heartbeat"`
	CustomStatus  map[string]interface{} `json:"custom_status"`
}

// NodeInfo represents cluster node information
type NodeInfo struct {
	ID            string              `json:"id"`
	Address       string              `json:"address"`
	Status        string              `json:"status"`
	LastSeen      time.Time           `json:"last_seen"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// Metrics represents FlexCore metrics
type Metrics struct {
	EventsProcessed int64                  `json:"events_processed"`
	MessagesQueued  int64                  `json:"messages_queued"`
	ScheduledJobs   int64                  `json:"scheduled_jobs"`
	ActivePlugins   int64                  `json:"active_plugins"`
	ClusterNodes    int64                  `json:"cluster_nodes"`
	UptimeSeconds   float64                `json:"uptime_seconds"`
	CustomMetrics   map[string]interface{} `json:"custom_metrics"`
}