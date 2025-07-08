# FlexCore Enterprise Observability Guide

## Overview

FlexCore implements enterprise-grade observability with comprehensive logging, metrics, distributed tracing, and health monitoring. This guide covers all observability features and their configuration.

## 📊 Metrics (Prometheus Integration)

### System Metrics

FlexCore automatically collects system-level metrics:

```prometheus
# HTTP Request Metrics
flexcore_http_requests_total{method="GET",endpoint="/health",status_code="200"} 15
flexcore_http_request_duration_seconds{method="GET",endpoint="/health"} 0.001234
flexcore_http_response_size_bytes{method="GET",endpoint="/health"} 1234
flexcore_http_requests_in_flight 3

# Event Processing Metrics
flexcore_events_processed_total{event_type="user_created",status="success"} 50
flexcore_events_processing_duration_seconds{event_type="user_created"} 0.045

# Pipeline Metrics
flexcore_pipelines_total{status="active"} 5
flexcore_pipelines_executions_total{pipeline_id="data-pipeline-1",status="success"} 25
flexcore_pipelines_execution_duration_seconds{pipeline_id="data-pipeline-1"} 12.34

# System Resource Metrics
flexcore_system_memory_usage_bytes{type="heap"} 52428800
flexcore_system_goroutines 150
flexcore_system_start_time_seconds 1625097600
```

### Health Check Metrics

All health checks automatically generate metrics:

```prometheus
# Health Check Status (1=healthy, 0=unhealthy)
flexcore_health_check_status{check_name="system_resources",component="system"} 1
flexcore_health_check_status{check_name="flexcore_event_store",component="flexcore"} 1

# Health Check Duration
flexcore_health_check_duration_seconds{check_name="system_resources",component="system"} 0.025
```

### Custom Business Metrics

Register custom metrics for business logic:

```go
// Register custom counter
counter := metricsCollector.RegisterCustomCounter(
    "orders_processed_total",
    "Total number of orders processed",
    []string{"status", "payment_method"},
)

// Use the metric
counter.WithLabelValues("completed", "credit_card").Inc()

// Register custom histogram
histogram := metricsCollector.RegisterCustomHistogram(
    "order_processing_duration_seconds",
    "Time to process orders",
    []string{"order_type"},
    []float64{0.1, 0.5, 1.0, 5.0, 10.0},
)

// Record timing
start := time.Now()
// ... process order ...
histogram.WithLabelValues("premium").Observe(time.Since(start).Seconds())
```

## 🔍 Distributed Tracing (OpenTelemetry + Jaeger)

### Automatic HTTP Tracing

All HTTP requests are automatically traced:

```json
{
  "traceID": "4bf92f3577b34da6a3ce929d0e0e4736",
  "spanID": "00f067aa0ba902b7",
  "operationName": "GET /api/pipelines",
  "duration": 45.123,
  "tags": {
    "http.method": "GET",
    "http.url": "/api/pipelines",
    "http.status_code": 200,
    "component": "http-server",
    "span.kind": "server"
  }
}
```

### Business Operation Tracing

Trace business operations with context:

```go
// Event processing tracing
err := businessTracing.TraceEventProcessingWithMetrics(ctx, "user_created", eventID, func(ctx context.Context) error {
    // Process the event
    return processUserCreatedEvent(ctx, event)
})

// Pipeline execution tracing
err := businessTracing.TracePipelineExecutionWithMetrics(ctx, pipelineID, "data-pipeline", 5, func(ctx context.Context) error {
    // Execute pipeline steps
    return executePipelineSteps(ctx, pipeline)
})

// Database operation tracing
err := businessTracing.TraceDatabaseWithMetrics(ctx, "SELECT", "SELECT * FROM users", "main", func(ctx context.Context) error {
    // Execute database query
    return db.QueryContext(ctx, query)
})
```

### Custom Tracing

Create custom spans for specific operations:

```go
ctx, span := tracing.StartSpan(ctx, "custom_operation")
defer span.End()

// Add attributes
tracingManager.AddSpanAttributes(span, map[string]interface{}{
    "user.id": userID,
    "operation.type": "data_processing",
    "batch.size": batchSize,
})

// Add events
tracingManager.AddSpanEvent(span, "processing_started", map[string]interface{}{
    "timestamp": time.Now().Unix(),
})

// Record errors
if err != nil {
    tracingManager.RecordError(span, err, map[string]interface{}{
        "error.type": "validation_error",
    })
}
```

## 📝 Enterprise Logging

### Structured JSON Logging

All logs are structured JSON with enterprise features:

```json
{
  "timestamp": "2025-07-05T10:52:47.577Z",
  "level": "info",
  "message": "Pipeline executed successfully",
  "service": "flexcore-server",
  "version": "dev",
  "environment": "production",
  "component": "pipeline-executor",
  "correlation_id": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "user_id": "user-12345",
  "pipeline_id": "data-pipeline-1",
  "execution_time_ms": 1234.56,
  "steps_executed": 5
}
```

### Audit Logging

Compliance-ready audit trail:

```json
{
  "timestamp": "2025-07-05T10:52:47.577Z",
  "level": "info",
  "message": "Audit Event",
  "log_type": "audit",
  "audit_event": "pipeline_execution",
  "correlation_id": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "user_id": "user-12345",
  "pipeline_id": "data-pipeline-1",
  "action": "execute_pipeline",
  "resource": "data-pipeline-1",
  "result": "success"
}
```

### Security Logging

Automatic security event detection:

```json
{
  "timestamp": "2025-07-05T10:52:47.577Z",
  "level": "warn",
  "message": "Security Event",
  "log_type": "security",
  "security_level": "medium",
  "security_event": "suspicious_request",
  "correlation_id": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "remote_addr": "192.168.1.100",
  "user_agent": "curl/7.68.0",
  "reason": "unusual_pattern",
  "method": "POST",
  "path": "/admin/config"
}
```

### Performance Logging

Built-in performance metrics:

```json
{
  "timestamp": "2025-07-05T10:52:47.577Z",
  "level": "info",
  "message": "Performance Metric",
  "log_type": "performance",
  "operation": "database_query",
  "duration": 0.025,
  "duration_ms": 25.0,
  "correlation_id": "4bf92f35-77b3-4da6-a3ce-929d0e0e4736",
  "query_type": "SELECT",
  "table": "users"
}
```

## 🏥 Health Monitoring

### Health Check Endpoints

#### GET /health

Comprehensive system health with all component details:

```json
{
  "status": "healthy",
  "message": "All health checks passing",
  "checks": {
    "system_resources": {
      "status": "healthy",
      "message": "System resources within normal limits",
      "duration": "25ms",
      "timestamp": "2025-07-05T10:52:47.577Z",
      "metadata": {
        "memory_mb": 45.2,
        "goroutines": 150,
        "gc_cycles": 12
      }
    },
    "flexcore_event_store": {
      "status": "healthy",
      "message": "Event store is operational",
      "duration": "5ms",
      "timestamp": "2025-07-05T10:52:47.577Z"
    }
  },
  "summary": {
    "total": 4,
    "healthy": 4,
    "unhealthy": 0,
    "degraded": 0,
    "critical": 3
  },
  "uptime": "2h30m45s",
  "version": "1.0.0",
  "node_id": "node-12345",
  "environment": "production"
}
```

#### GET /health/live

Simple liveness probe for Kubernetes:

```json
{
  "status": "alive",
  "timestamp": 1625097600,
  "message": "Service is running",
  "node_id": "node-12345",
  "version": "1.0.0"
}
```

#### GET /health/ready

Readiness probe checking only critical components:

```json
{
  "status": "ready",
  "ready": true,
  "timestamp": 1625097600,
  "message": "Service is ready to accept traffic",
  "failed_critical": [],
  "node_id": "node-12345",
  "version": "1.0.0"
}
```

### Custom Health Checks

Register custom health checks:

```go
// Simple health check
check := health.NewCustomHealthCheck("database", "database").
    WithDescription("PostgreSQL database connectivity").
    WithInterval(60 * time.Second).
    WithTimeout(10 * time.Second).
    WithCritical(true).
    WithChecker(health.SimpleChecker(func(ctx context.Context) error {
        return db.PingContext(ctx)
    })).
    Build()

healthChecker.RegisterCheck(check)

// Advanced health check with custom logic
advancedCheck := &health.HealthCheck{
    Name:        "api_dependencies",
    Component:   "external",
    Description: "Check all external API dependencies",
    Interval:    120 * time.Second,
    Timeout:     30 * time.Second,
    Critical:    false,
    Checker: func(ctx context.Context) health.HealthResult {
        // Custom health check logic
        failures := 0
        total := 3

        // Check each external dependency
        for _, api := range externalAPIs {
            if err := checkAPI(ctx, api); err != nil {
                failures++
            }
        }

        healthPercentage := float64(total-failures) / float64(total) * 100

        if healthPercentage >= 100 {
            return health.HealthResult{
                Status:  health.HealthStatusHealthy,
                Message: "All external APIs healthy",
                Metadata: map[string]interface{}{
                    "healthy_apis": total - failures,
                    "total_apis":   total,
                    "health_percentage": healthPercentage,
                },
            }
        } else if healthPercentage >= 50 {
            return health.HealthResult{
                Status:  health.HealthStatusDegraded,
                Message: fmt.Sprintf("Some external APIs failing (%d/%d)", failures, total),
                Metadata: map[string]interface{}{
                    "healthy_apis": total - failures,
                    "total_apis":   total,
                    "health_percentage": healthPercentage,
                },
            }
        } else {
            return health.HealthResult{
                Status:  health.HealthStatusUnhealthy,
                Message: fmt.Sprintf("Most external APIs failing (%d/%d)", failures, total),
                Metadata: map[string]interface{}{
                    "healthy_apis": total - failures,
                    "total_apis":   total,
                    "health_percentage": healthPercentage,
                },
            }
        }
    },
}

healthChecker.RegisterCheck(advancedCheck)
```

## 🚀 Deployment Configuration

### Docker Compose for Development

```yaml
version: "3.8"

services:
  flexcore:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=debug
      - METRICS_ENABLED=true
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - jaeger
      - prometheus

  # Jaeger for distributed tracing
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686" # UI
      - "14268:14268" # HTTP collector
    environment:
      - COLLECTOR_OTLP_ENABLED=true

  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
      - "--web.console.libraries=/etc/prometheus/console_libraries"
      - "--web.console.templates=/etc/prometheus/consoles"

  # Grafana for dashboards
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "flexcore"
    static_configs:
      - targets: ["flexcore:8080"]
    metrics_path: "/metrics"
    scrape_interval: 10s

  - job_name: "flexcore-health"
    static_configs:
      - targets: ["flexcore:8080"]
    metrics_path: "/health"
    scrape_interval: 30s
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flexcore
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flexcore
  template:
    metadata:
      labels:
        app: flexcore
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        - name: flexcore
          image: flexcore:latest
          ports:
            - containerPort: 8080
          env:
            - name: LOG_LEVEL
              value: "info"
            - name: JAEGER_ENDPOINT
              value: "http://jaeger-collector:14268/api/traces"
          livenessProbe:
            httpGet:
              path: /health/live
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

## 📈 Observability Best Practices

### 1. Correlation ID Propagation

Always propagate correlation IDs across service boundaries:

```go
// Extract correlation ID from HTTP headers
correlationID := r.Header.Get("X-Correlation-ID")
if correlationID == "" {
    correlationID = uuid.New().String()
}

// Add to context
ctx := context.WithValue(r.Context(), "correlation_id", correlationID)

// Use context-aware logger
logger := logger.WithContext(ctx)
logger.Info("Processing request")

// Propagate in outgoing requests
req.Header.Set("X-Correlation-ID", correlationID)
```

### 2. Error Handling with Observability

```go
func processOrder(ctx context.Context, order Order) error {
    start := time.Now()
    logger := logger.WithContext(ctx).WithComponent("order-processor")

    // Start tracing span
    ctx, span := tracing.StartSpan(ctx, "process_order")
    defer span.End()

    // Add order attributes
    span.SetAttributes(
        attribute.String("order.id", order.ID),
        attribute.String("order.type", order.Type),
    )

    logger.Info("Starting order processing",
        logging.String("order_id", order.ID),
        logging.String("order_type", order.Type),
    )

    // Process order
    if err := validateOrder(ctx, order); err != nil {
        // Record error in all observability systems
        logger.Error("Order validation failed",
            logging.String("order_id", order.ID),
            logging.String("error", err.Error()),
        )

        span.RecordError(err)
        span.SetStatus(codes.Error, "Validation failed")

        metricsCollector.RecordError("order-processor", "validation_failed", "high")

        return fmt.Errorf("validation failed: %w", err)
    }

    // Record success metrics
    duration := time.Since(start)
    logger.Performance("order_processing", duration,
        logging.String("order_id", order.ID),
        logging.String("status", "success"),
    )

    businessMetrics.RecordCustomBusinessMetric("orders_processed_total", 1, map[string]string{
        "status": "success",
        "type":   order.Type,
    })

    span.SetStatus(codes.Ok, "Order processed successfully")

    return nil
}
```

### 3. Dashboard and Alerting

Create Grafana dashboards and Prometheus alerts:

```yaml
# alerts.yml
groups:
  - name: flexcore
    rules:
      - alert: FlexCoreHighErrorRate
        expr: rate(flexcore_http_requests_total{status_code=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "FlexCore high error rate"
          description: "Error rate is above 10% for 2 minutes"

      - alert: FlexCoreHealthCheckFailing
        expr: flexcore_health_check_status == 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "FlexCore health check failing"
          description: "Health check {{ $labels.check_name }} is failing"

      - alert: FlexCoreHighLatency
        expr: histogram_quantile(0.95, rate(flexcore_http_request_duration_seconds_bucket[5m])) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "FlexCore high latency"
          description: "95th percentile latency is above 1 second"
```

This comprehensive observability setup provides enterprise-grade visibility into FlexCore's operation, performance, and health status.
