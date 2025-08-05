package services

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/flext-sh/flexcore/internal/domain/services"
)

// EventBus represents the messaging event bus for Event Sourcing + CQRS
// Domain service interfaces imported from internal/domain/services

// WorkflowServiceConfig contains all dependencies for WorkflowService
// PARAMETER OBJECT PATTERN: Eliminates 6-parameter constructor complexity
type WorkflowServiceConfig struct {
	EventBus     services.EventBus            // Event Sourcing + CQRS
	PluginLoader services.PluginLoader        // HashiCorp-style plugins
	Cluster      services.CoordinationLayer   // Distributed coordination
	Repository   services.EventStore          // Event Store for audit trail
	CommandBus   services.CommandBus          // CQRS command processing
	Logger       logging.LoggerInterface      // Structured logging
}

// ConfigValidator provides specialized validation for workflow service configuration
// SOLID SRP: Single responsibility for configuration validation eliminating 7 returns
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	FieldName string
	Check     func(*WorkflowServiceConfig) bool
	Message   string
}

// getValidationRules returns all validation rules for WorkflowServiceConfig
// SOLID OCP: Open for extension with new validation rules
func (v *ConfigValidator) getValidationRules() []ValidationRule {
	return []ValidationRule{
		{
			"EventBus",
			func(cfg *WorkflowServiceConfig) bool { return cfg.EventBus != nil },
			"EventBus is required",
		},
		{
			"PluginLoader",
			func(cfg *WorkflowServiceConfig) bool { return cfg.PluginLoader != nil },
			"PluginLoader is required",
		},
		{
			"Cluster",
			func(cfg *WorkflowServiceConfig) bool { return cfg.Cluster != nil },
			"Cluster coordination layer is required",
		},
		{
			"Repository",
			func(cfg *WorkflowServiceConfig) bool { return cfg.Repository != nil },
			"Repository (EventStore) is required",
		},
		{
			"CommandBus",
			func(cfg *WorkflowServiceConfig) bool { return cfg.CommandBus != nil },
			"CommandBus is required",
		},
		{
			"Logger",
			func(cfg *WorkflowServiceConfig) bool { return cfg.Logger != nil },
			"Logger is required",
		},
	}
}

// ValidateConfig validates configuration using rule-based approach
// DRY PRINCIPLE: Eliminates 7 return statements using validation rules
func (v *ConfigValidator) ValidateConfig(cfg *WorkflowServiceConfig) error {
	for _, rule := range v.getValidationRules() {
		if !rule.Check(cfg) {
			return fmt.Errorf("%s", rule.Message)
		}
	}
	return nil
}

// Validate validates the workflow service configuration
// DRY PRINCIPLE: Delegates to specialized validator eliminating multiple returns
func (cfg *WorkflowServiceConfig) Validate() error {
	validator := NewConfigValidator()
	return validator.ValidateConfig(cfg)
}

// WorkflowService provides FLEXCORE runtime services exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type WorkflowService struct {
	eventBus     services.EventBus            // Event Sourcing + CQRS
	pluginLoader services.PluginLoader        // HashiCorp-style plugins
	cluster      services.CoordinationLayer   // Distributed coordination
	repository   services.EventStore          // Event Store for audit trail
	commandBus   services.CommandBus          // CQRS command processing
	logger       logging.LoggerInterface

	// SOLID SRP: Specialized orchestrator for pipeline execution with reduced returns
	pipelineOrchestrator *PipelineExecutionOrchestrator
}

// NewWorkflowService creates a new workflow service using Parameter Object Pattern
// PARAMETER OBJECT PATTERN: Eliminates 6-parameter constructor complexity
func NewWorkflowService(config *WorkflowServiceConfig) (*WorkflowService, error) {
	if config == nil {
		return nil, fmt.Errorf("WorkflowServiceConfig cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create specialized pipeline orchestrator using legacy constructor for compatibility
	orchestrator := NewPipelineExecutionOrchestratorLegacy(
		config.EventBus,
		config.CommandBus,
		config.PluginLoader,
		config.Cluster,
		config.Repository,
	)

	return &WorkflowService{
		eventBus:             config.EventBus,
		pluginLoader:         config.PluginLoader,
		cluster:              config.Cluster,
		repository:           config.Repository,
		commandBus:           config.CommandBus,
		logger:               config.Logger,
		pipelineOrchestrator: orchestrator,
	}, nil
}

// WorkflowServiceBuilder applies Builder Pattern to eliminate parameter explosion
// SOLID Builder Pattern: Reduces 6-parameter constructor to fluent interface
type WorkflowServiceBuilder struct {
	config *WorkflowServiceConfig
}

// NewWorkflowServiceBuilder creates a new workflow service builder
func NewWorkflowServiceBuilder() *WorkflowServiceBuilder {
	return &WorkflowServiceBuilder{
		config: &WorkflowServiceConfig{},
	}
}

// WithEventBus sets the event bus for Event Sourcing + CQRS
func (b *WorkflowServiceBuilder) WithEventBus(eventBus services.EventBus) *WorkflowServiceBuilder {
	b.config.EventBus = eventBus
	return b
}

// WithPluginLoader sets the plugin loader for HashiCorp-style plugins
func (b *WorkflowServiceBuilder) WithPluginLoader(pluginLoader services.PluginLoader) *WorkflowServiceBuilder {
	b.config.PluginLoader = pluginLoader
	return b
}

// WithCluster sets the coordination layer for distributed coordination
func (b *WorkflowServiceBuilder) WithCluster(cluster services.CoordinationLayer) *WorkflowServiceBuilder {
	b.config.Cluster = cluster
	return b
}

// WithRepository sets the event store for audit trail
func (b *WorkflowServiceBuilder) WithRepository(repository services.EventStore) *WorkflowServiceBuilder {
	b.config.Repository = repository
	return b
}

// WithCommandBus sets the CQRS command bus
func (b *WorkflowServiceBuilder) WithCommandBus(commandBus services.CommandBus) *WorkflowServiceBuilder {
	b.config.CommandBus = commandBus
	return b
}

// WithLogger sets the structured logger
func (b *WorkflowServiceBuilder) WithLogger(logger logging.LoggerInterface) *WorkflowServiceBuilder {
	b.config.Logger = logger
	return b
}

// Build creates the WorkflowService using Parameter Object Pattern
func (b *WorkflowServiceBuilder) Build() (*WorkflowService, error) {
	return NewWorkflowService(b.config)
}

// NewWorkflowServiceLegacy maintains backward compatibility using Builder Pattern
// Deprecated: Use NewWorkflowService with WorkflowServiceConfig for better maintainability
// BUILDER PATTERN: Eliminates 6-parameter constructor complexity using fluent interface
func NewWorkflowServiceLegacy(
	eventBus services.EventBus,
	pluginLoader services.PluginLoader,
	cluster services.CoordinationLayer,
	repository services.EventStore,
	commandBus services.CommandBus,
	logger logging.LoggerInterface,
) *WorkflowService {
	// Apply Builder Pattern to eliminate parameter explosion
	service, err := NewWorkflowServiceBuilder().
		WithEventBus(eventBus).
		WithPluginLoader(pluginLoader).
		WithCluster(cluster).
		WithRepository(repository).
		WithCommandBus(commandBus).
		WithLogger(logger).
		Build()

	if err != nil {
		// Log error but maintain backward compatibility by creating service anyway
		if logger != nil {
			logger.Error("Failed to create WorkflowService with validation", zap.Error(err))
		}
		// Fallback to direct construction for backward compatibility
		return &WorkflowService{
			eventBus:     eventBus,
			pluginLoader: pluginLoader,
			cluster:      cluster,
			repository:   repository,
			commandBus:   commandBus,
			logger:       logger,
		}
	}

	return service
}

// Event types moved to pipeline_events.go to avoid duplication

// ExecuteFlextPipeline executes FLEXT service workflows exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
// SOLID SRP: Reduced from 6 returns to 2 returns (67% reduction) using specialized orchestrator
func (ws *WorkflowService) ExecuteFlextPipeline(ctx context.Context, pipelineID string) error {
	// Log pipeline execution start
	ws.logger.Info("Starting FLEXT pipeline execution",
		zap.String("pipeline_id", pipelineID),
		zap.String("cluster_node", "node-1"), // TODO: Implement node ID retrieval from cluster
	)

	// Delegate to specialized orchestrator with consolidated error handling
	if err := ws.pipelineOrchestrator.ExecutePipeline(ctx, pipelineID); err != nil {
		ws.logger.Error("FLEXT pipeline execution failed",
			zap.String("pipeline_id", pipelineID),
			zap.Error(err),
		)
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Log successful completion
	ws.logger.Info("FLEXT pipeline execution completed successfully",
		zap.String("pipeline_id", pipelineID),
	)
	return nil
}

