// Package server provides gRPC server implementation for FlexCore Control Panel communication
package server

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/durationpb"
	"go.uber.org/zap"

	pb "github.com/flext-sh/flexcore/api/grpc/v1"
	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/flext-sh/flexcore/pkg/windmill/engine"
	"github.com/flext-sh/flexcore/pkg/runtimes"
)

// FlexCoreServer implements the FlexCore gRPC service
type FlexCoreServer struct {
	pb.UnimplementedFlexCoreManagementServer
	windmillEngine *engine.WindmillEngine
	runtimeManager runtimes.RuntimeManager
	logger         *zap.Logger
	instanceID     string
	startTime      time.Time
}

// NewFlexCoreServer creates a new FlexCore gRPC server instance
func NewFlexCoreServer(windmillEngine *engine.WindmillEngine, runtimeManager runtimes.RuntimeManager, instanceID string) *FlexCoreServer {
	return &FlexCoreServer{
		windmillEngine: windmillEngine,
		runtimeManager: runtimeManager,
		logger:         logging.GetLogger().With(zap.String("component", "flexcore_grpc_server")),
		instanceID:     instanceID,
		startTime:      time.Now(),
	}
}

// Runtime Management Methods

// StartRuntime starts a specific runtime type
func (s *FlexCoreServer) StartRuntime(ctx context.Context, req *pb.StartRuntimeRequest) (*pb.RuntimeResponse, error) {
	s.logger.Info("Starting runtime",
		logging.F("runtime_type", req.RuntimeType),
		logging.F("capabilities", req.Capabilities))

	// Validate runtime type
	if req.RuntimeType == "" {
		return nil, status.Error(codes.InvalidArgument, "runtime_type is required")
	}

	// Start runtime via runtime manager
	runtimeStatus, err := s.runtimeManager.StartRuntime(ctx, req.RuntimeType, req.Config, req.Capabilities)
	if err != nil {
		s.logger.Error("Failed to start runtime",
			logging.F("runtime_type", req.RuntimeType),
			logging.F("error", err))
		return &pb.RuntimeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	s.logger.Info("Runtime started successfully",
		logging.F("runtime_type", req.RuntimeType),
		logging.F("status", runtimeStatus.Status))

	return &pb.RuntimeResponse{
		Success: true,
		Message: fmt.Sprintf("Runtime %s started successfully", req.RuntimeType),
		Status:  convertRuntimeStatus(runtimeStatus),
	}, nil
}

// StopRuntime stops a specific runtime type
func (s *FlexCoreServer) StopRuntime(ctx context.Context, req *pb.StopRuntimeRequest) (*pb.RuntimeResponse, error) {
	s.logger.Info("Stopping runtime",
		logging.F("runtime_type", req.RuntimeType),
		logging.F("force", req.Force),
		logging.F("timeout_seconds", req.TimeoutSeconds))

	// Stop runtime via runtime manager
	err := s.runtimeManager.StopRuntime(ctx, req.RuntimeType, req.Force, time.Duration(req.TimeoutSeconds)*time.Second)
	if err != nil {
		s.logger.Error("Failed to stop runtime",
			logging.F("runtime_type", req.RuntimeType),
			logging.F("error", err))
		return &pb.RuntimeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	s.logger.Info("Runtime stopped successfully", logging.F("runtime_type", req.RuntimeType))

	return &pb.RuntimeResponse{
		Success: true,
		Message: fmt.Sprintf("Runtime %s stopped successfully", req.RuntimeType),
		Status: &pb.RuntimeStatus{
			RuntimeType: req.RuntimeType,
			Status:      "stopped",
			Health:      "unknown",
		},
	}, nil
}

// GetRuntimeStatus gets the status of a specific runtime
func (s *FlexCoreServer) GetRuntimeStatus(ctx context.Context, req *pb.RuntimeStatusRequest) (*pb.RuntimeStatusResponse, error) {
	s.logger.Debug("Getting runtime status", logging.F("runtime_type", req.RuntimeType))

	runtimeStatus, err := s.runtimeManager.GetRuntimeStatus(ctx, req.RuntimeType)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Runtime %s not found: %v", req.RuntimeType, err))
	}

	return &pb.RuntimeStatusResponse{
		Status: convertRuntimeStatus(runtimeStatus),
	}, nil
}

// ListRuntimes lists all available runtimes
func (s *FlexCoreServer) ListRuntimes(ctx context.Context, req *pb.ListRuntimesRequest) (*pb.ListRuntimesResponse, error) {
	s.logger.Debug("Listing runtimes", logging.F("include_inactive", req.IncludeInactive))

	runtimeStatuses, err := s.runtimeManager.ListRuntimes(ctx, req.IncludeInactive)
	if err != nil {
		return nil, status.Error(internal.invalid, fmt.Sprintf("Failed to list runtimes: %v", err))
	}

	var pbStatuses []*pb.RuntimeStatus
	for _, runtimeStatus := range runtimeStatuses {
		pbStatuses = append(pbStatuses, convertRuntimeStatus(runtimeStatus))
	}

	return &pb.ListRuntimesResponse{
		Runtimes: pbStatuses,
	}, nil
}

// Workflow Management Methods

// ExecuteWorkflow executes a workflow via Windmill
func (s *FlexCoreServer) ExecuteWorkflow(ctx context.Context, req *pb.WorkflowExecutionRequest) (*pb.WorkflowExecutionResponse, error) {
	s.logger.Info("Executing workflow",
		logging.F("workflow_id", req.WorkflowId),
		logging.F("runtime_type", req.RuntimeType),
		logging.F("command", req.Command))

	// Create Windmill workflow request
	windmillReq := engine.WorkflowRequest{
		ID:          req.WorkflowId,
		WorkflowID:  req.WorkflowId,
		RuntimeType: req.RuntimeType,
		Command:     req.Command,
		Args:        req.Args,
		Config:      convertConfigToMap(req.Config),
	}

	// Execute via Windmill engine
	response, err := s.windmillEngine.ExecuteWorkflow(ctx, windmillReq)
	if err != nil {
		s.logger.Error("Failed to execute workflow",
			logging.F("workflow_id", req.WorkflowId),
			logging.F("error", err))
		return &pb.WorkflowExecutionResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	s.logger.Info("Workflow executed successfully",
		logging.F("workflow_id", req.WorkflowId),
		logging.F("job_id", response.JobID))

	return &pb.WorkflowExecutionResponse{
		Success: true,
		JobId:   response.JobID,
		Message: "Workflow executed successfully",
		Status:  convertWorkflowStatus(response),
	}, nil
}

// GetWorkflowStatus gets the status of a workflow
func (s *FlexCoreServer) GetWorkflowStatus(ctx context.Context, req *pb.WorkflowStatusRequest) (*pb.WorkflowStatusResponse, error) {
	s.logger.Debug("Getting workflow status", logging.F("job_id", req.JobId))

	workflowStatus, err := s.windmillEngine.GetWorkflowStatus(ctx, req.JobId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Workflow %s not found: %v", req.JobId, err))
	}

	return &pb.WorkflowStatusResponse{
		Status: convertWindmillWorkflowStatus(workflowStatus),
	}, nil
}

// ListWorkflows lists workflows
func (s *FlexCoreServer) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	s.logger.Debug("Listing workflows",
		logging.F("runtime_type", req.RuntimeType),
		logging.F("status_filter", req.StatusFilter))

	workflows, err := s.windmillEngine.ListWorkflows(ctx)
	if err != nil {
		return nil, status.Error(internal.invalid, fmt.Sprintf("Failed to list workflows: %v", err))
	}

	var pbWorkflows []*pb.WorkflowStatus
	for _, workflow := range workflows {
		// Apply filters
		if req.RuntimeType != "" && workflow.RuntimeType != req.RuntimeType {
			continue
		}
		if req.StatusFilter != "" && workflow.Status != req.StatusFilter {
			continue
		}
		if !req.IncludeCompleted && (workflow.Status == "completed" || workflow.Status == "failed") {
			continue
		}

		pbWorkflows = append(pbWorkflows, convertWindmillWorkflowStatus(&workflow))
	}

	// Apply pagination
	total := len(pbWorkflows)
	start := int(req.Offset)
	end := start + int(req.Limit)
	
	if start > total {
		pbWorkflows = []*pb.WorkflowStatus{}
	} else {
		if end > total {
			end = total
		}
		if req.Limit > 0 {
			pbWorkflows = pbWorkflows[start:end]
		}
	}

	return &pb.ListWorkflowsResponse{
		Workflows:  pbWorkflows,
		TotalCount: int32(total),
	}, nil
}

// CancelWorkflow cancels a workflow
func (s *FlexCoreServer) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*pb.CancelWorkflowResponse, error) {
	s.logger.Info("Cancelling workflow",
		logging.F("job_id", req.JobId),
		logging.F("reason", req.Reason))

	err := s.windmillEngine.CancelWorkflow(ctx, req.JobId)
	if err != nil {
		s.logger.Error("Failed to cancel workflow",
			logging.F("job_id", req.JobId),
			logging.F("error", err))
		return &pb.CancelWorkflowResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	s.logger.Info("Workflow cancelled successfully", logging.F("job_id", req.JobId))

	return &pb.CancelWorkflowResponse{
		Success: true,
		Message: "Workflow cancelled successfully",
	}, nil
}

// Health and Monitoring Methods

// GetHealthStatus gets overall health status
func (s *FlexCoreServer) GetHealthStatus(ctx context.Context, req *pb.HealthStatusRequest) (*pb.HealthStatusResponse, error) {
	s.logger.Debug("Getting health status",
		logging.F("include_runtimes", req.IncludeRuntimes),
		logging.F("include_windmill", req.IncludeWindmill))

	response := &pb.HealthStatusResponse{
		FlexcoreStatus: "healthy",
		CheckTime:      timestamppb.Now(),
	}

	// Check Windmill health
	if req.IncludeWindmill {
		if err := s.windmillEngine.HealthCheck(ctx); err != nil {
			response.WindmillStatus = "unhealthy"
			response.Errors = append(response.Errors, fmt.Sprintf("Windmill health check failed: %v", err))
			response.OverallStatus = "degraded"
		} else {
			response.WindmillStatus = "healthy"
		}
	}

	// Check runtime health
	if req.IncludeRuntimes {
		runtimeStatuses, err := s.runtimeManager.ListRuntimes(ctx, false)
		if err != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("Failed to get runtime status: %v", err))
			response.OverallStatus = "degraded"
		} else {
			for _, runtimeStatus := range runtimeStatuses {
				healthStatus := &pb.RuntimeHealthStatus{
					RuntimeType: runtimeStatus.RuntimeType,
					Status:      runtimeStatus.Status,
					LastCheck:   timestamppb.New(time.Now()), // Use current time as placeholder
				}
				if runtimeStatus.Status != "healthy" {
					healthStatus.Message = "Runtime health check failed"
					response.OverallStatus = "degraded"
				}
				response.RuntimeHealth = append(response.RuntimeHealth, healthStatus)
			}
		}
	}

	// Set overall status if not already set
	if response.OverallStatus == "" {
		if len(response.Errors) > 0 {
			response.OverallStatus = "degraded"
		} else {
			response.OverallStatus = "healthy"
		}
	}

	return response, nil
}

// GetMetrics gets performance metrics
func (s *FlexCoreServer) GetMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	s.logger.Debug("Getting metrics")

	// TODO: Implement actual metrics collection
	// For now, return mock metrics
	return &pb.MetricsResponse{
		Metrics: []*pb.Metric{
			{
				Name:      "flexcore_uptime_seconds",
				Type:      "gauge",
				Value:     time.Since(s.startTime).Seconds(),
				Unit:      "seconds",
				Timestamp: timestamppb.Now(),
			},
			{
				Name:      "flexcore_active_workflows",
				Type:      "gauge",
				Value:     5, // Mock value
				Unit:      "count",
				Timestamp: timestamppb.Now(),
			},
		},
		CollectionTime: timestamppb.Now(),
	}, nil
}

// StreamLogs streams log entries
func (s *FlexCoreServer) StreamLogs(req *pb.LogStreamRequest, stream pb.FlexCoreManagement_StreamLogsServer) error {
	s.logger.Debug("Starting log stream",
		logging.F("level_filter", req.LevelFilter),
		logging.F("component_filter", req.ComponentFilter))

	// TODO: Implement actual log streaming
	// For now, send some mock log entries
	mockLogs := []*pb.LogEntry{
		{
			Timestamp: timestamppb.Now(),
			Level:     "info",
			Component: "flexcore",
			Message:   "FlexCore service started",
			Fields: map[string]string{
				"instance_id": s.instanceID,
			},
		},
		{
			Timestamp: timestamppb.Now(),
			Level:     "info",
			Component: "windmill",
			Message:   "Windmill engine initialized",
		},
	}

	for _, logEntry := range mockLogs {
		if err := stream.Send(logEntry); err != nil {
			return status.Error(internal.invalid, fmt.Sprintf("Failed to send log entry: %v", err))
		}
	}

	return nil
}

// Configuration Management Methods

// GetConfiguration gets component configuration
func (s *FlexCoreServer) GetConfiguration(ctx context.Context, req *pb.ConfigurationRequest) (*pb.ConfigurationResponse, error) {
	s.logger.Debug("Getting configuration", logging.F("component", req.Component))

	// TODO: Implement actual configuration retrieval
	// For now, return mock configuration
	return &pb.ConfigurationResponse{
		Component: req.Component,
		Configuration: map[string]string{
			"version":    "2.0.0",
			"debug_mode": "false",
			"log_level":  "info",
		},
		Version:     "1.0",
		LastUpdated: timestamppb.Now(),
	}, nil
}

// UpdateConfiguration updates component configuration
func (s *FlexCoreServer) UpdateConfiguration(ctx context.Context, req *pb.UpdateConfigurationRequest) (*pb.ConfigurationResponse, error) {
	s.logger.Info("Updating configuration",
		logging.F("component", req.Component),
		logging.F("validate_only", req.ValidateOnly))

	if req.ValidateOnly {
		s.logger.Debug("Configuration validation requested")
		// TODO: Validate configuration
	} else {
		s.logger.Info("Applying configuration changes")
		// TODO: Apply configuration changes
	}

	return &pb.ConfigurationResponse{
		Component:     req.Component,
		Configuration: req.Configuration,
		Version:       "1.1",
		LastUpdated:   timestamppb.Now(),
	}, nil
}

// Instance Management Methods

// RegisterInstance registers a FlexCore instance
func (s *FlexCoreServer) RegisterInstance(ctx context.Context, req *pb.RegisterInstanceRequest) (*pb.RegisterInstanceResponse, error) {
	s.logger.Info("Registering FlexCore instance",
		logging.F("instance_id", req.InstanceId),
		logging.F("hostname", req.Hostname),
		logging.F("grpc_address", req.GrpcAddress))

	// TODO: Implement instance registration logic
	return &pb.RegisterInstanceResponse{
		Success:    true,
		Message:    "Instance registered successfully",
		AssignedId: req.InstanceId,
	}, nil
}

// DeregisterInstance deregisters a FlexCore instance
func (s *FlexCoreServer) DeregisterInstance(ctx context.Context, req *pb.DeregisterInstanceRequest) (*emptypb.Empty, error) {
	s.logger.Info("Deregistering FlexCore instance",
		logging.F("instance_id", req.InstanceId),
		logging.F("reason", req.Reason))

	// TODO: Implement instance deregistration logic
	return &emptypb.Empty{}, nil
}

// GetInstanceInfo gets information about a FlexCore instance
func (s *FlexCoreServer) GetInstanceInfo(ctx context.Context, req *pb.InstanceInfoRequest) (*pb.InstanceInfoResponse, error) {
	s.logger.Debug("Getting instance info", logging.F("instance_id", req.InstanceId))

	// TODO: Implement actual instance info retrieval
	return &pb.InstanceInfoResponse{
		InstanceId:    req.InstanceId,
		Hostname:      "flexcore-001",
		GrpcAddress:   "localhost:8080",
		Status:        "active",
		Capabilities:  []string{"meltano", "windmill"},
		RegisteredAt:  timestamppb.New(s.startTime),
		LastHeartbeat: timestamppb.Now(),
		Metadata: map[string]string{
			"version": "2.0.0",
			"region":  "us-east-1",
		},
		Stats: &pb.InstanceStats{
			ActiveWorkflows:    3,
			CompletedWorkflows: 25,
			FailedWorkflows:    2,
			CpuUtilization:     0.45,
			MemoryUtilization:  0.67,
			Uptime:             durationpb.New(time.Since(s.startTime)),
		},
	}, nil
}

// Helper Functions

// convertRuntimeStatus converts internal runtime status to protobuf
func convertRuntimeStatus(status *runtimes.RuntimeStatus) *pb.RuntimeStatus {
	return &pb.RuntimeStatus{
		RuntimeType: status.RuntimeType,
		Status:      status.Status,
		Health:      status.Health,
		StartTime:   timestamppb.New(status.StartTime),
		Uptime:      durationpb.New(status.Uptime),
		LastCheck:   timestamppb.New(status.LastCheck),
		Version:     status.Version,
		Metadata:    status.Metadata,
		ResourceUsage: &pb.ResourceUsage{
			CpuPercent:  status.ResourceUsage.CPUPercent,
			MemoryBytes: status.ResourceUsage.MemoryBytes,
			DiskBytes:   status.ResourceUsage.DiskBytes,
			ActiveJobs:  int32(status.ResourceUsage.ActiveJobs),
			QueuedJobs:  int32(status.ResourceUsage.QueuedJobs),
		},
	}
}

// convertWorkflowStatus converts Windmill response to protobuf
func convertWorkflowStatus(response *engine.WorkflowResponse) *pb.WorkflowStatus {
	pbStatus := &pb.WorkflowStatus{
		JobId:       response.JobID,
		Status:      response.Status,
		StartTime:   timestamppb.New(response.StartTime),
		Logs:        response.Logs,
	}

	if response.EndTime != nil {
		pbStatus.EndTime = timestamppb.New(*response.EndTime)
	}

	return pbStatus
}

// convertWindmillWorkflowStatus converts Windmill workflow status to protobuf
func convertWindmillWorkflowStatus(status *engine.WorkflowStatus) *pb.WorkflowStatus {
	pbStatus := &pb.WorkflowStatus{
		JobId:       status.JobID,
		Status:      status.Status,
		Progress:    status.Progress,
		RuntimeType: status.RuntimeType,
		StartTime:   timestamppb.New(status.StartTime),
		Duration:    durationpb.New(status.Duration),
		Logs:        status.Logs,
		Metadata:    convertMapToStringMap(status.Metadata),
	}

	if status.EndTime != nil {
		pbStatus.EndTime = timestamppb.New(*status.EndTime)
	}

	if status.Result != nil {
		// Convert result to string representation
		pbStatus.Result = fmt.Sprintf("%v", status.Result)
	}

	if status.Error != "" {
		pbStatus.Error = status.Error
	}

	return pbStatus
}

// convertConfigToMap converts string map to interface map
func convertConfigToMap(config map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}
	return result
}

// convertMapToStringMap converts interface map to string map
func convertMapToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
