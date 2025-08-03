# FlexCore API Reference

**Version**: 0.9.0 | **Status**: Development | **Last Updated**: 2025-08-02

This document provides comprehensive API reference for FlexCore's REST API endpoints, including current implementation and planned endpoints for the post-refactoring architecture.

> ⚠️ **API Stability Warning**: FlexCore APIs are currently unstable due to ongoing architectural refactoring. Expect breaking changes. See [TODO.md](TODO.md) for details on architectural issues.

## 🌐 Base Configuration

### Service Endpoints

- **Primary API**: `http://localhost:8080` (Development)
- **Health Endpoints**: `http://localhost:8080/health`
- **Metrics**: `http://localhost:8080/metrics`
- **gRPC** (Planned): `localhost:50051`

### Content Types

- **Request**: `application/json`
- **Response**: `application/json`
- **Error Format**: RFC 7807 Problem Details

### Authentication

> ⚠️ **Security Note**: Current implementation lacks proper authentication. This will be addressed in the security hardening phase.

**Planned Authentication**:

- **Bearer Token**: JWT-based authentication
- **API Keys**: Service-to-service authentication
- **RBAC**: Role-based access control

## 🏥 Health & System Endpoints

### GET /health

Basic health check endpoint for monitoring and load balancers.

**Response**:

```json
{
  "status": "ok",
  "timestamp": "2025-08-02T10:30:00Z"
}
```

**Status Codes**:

- `200`: Service is healthy
- `503`: Service is unhealthy

**Example**:

```bash
curl http://localhost:8080/health
```

### GET /metrics

Prometheus-compatible metrics endpoint for monitoring.

**Response**: Prometheus metrics format

```
# HELP flexcore_requests_total Total number of requests
# TYPE flexcore_requests_total counter
flexcore_requests_total{method="GET",endpoint="/health"} 42

# HELP flexcore_request_duration_seconds Request duration
# TYPE flexcore_request_duration_seconds histogram
flexcore_request_duration_seconds_bucket{le="0.1"} 35
```

**Status Codes**:

- `200`: Metrics available

### GET /api/v1/flexcore/status

Detailed system status with dependency health checks.

**Response**:

```json
{
  "service": "flexcore",
  "version": "0.9.0",
  "status": "healthy",
  "timestamp": "2025-08-02T10:30:00Z",
  "dependencies": {
    "postgresql": {
      "status": "healthy",
      "response_time_ms": 5,
      "last_check": "2025-08-02T10:30:00Z"
    },
    "redis": {
      "status": "healthy",
      "response_time_ms": 2,
      "last_check": "2025-08-02T10:30:00Z"
    }
  },
  "metrics": {
    "plugins_active": 3,
    "pipelines_running": 1,
    "events_processed": 1520,
    "uptime_seconds": 3600
  }
}
```

**Status Codes**:

- `200`: Status retrieved successfully
- `503`: System unhealthy

## 🔌 Plugin Management API

### GET /api/v1/flexcore/plugins

List all registered plugins with their status and metadata.

**Query Parameters**:

- `status` (optional): Filter by plugin status (`active`, `inactive`, `error`)
- `type` (optional): Filter by plugin type
- `limit` (optional): Maximum number of results (default: 50)
- `offset` (optional): Offset for pagination (default: 0)

**Response**:

```json
{
  "plugins": [
    {
      "id": "flext-service",
      "name": "FLEXT Service Plugin",
      "version": "0.9.0",
      "type": "service-integration",
      "status": "active",
      "description": "Integration with FLEXT Python service",
      "endpoints": ["http://localhost:8081"],
      "health_status": "healthy",
      "last_execution": "2025-08-02T10:25:00Z",
      "execution_count": 45,
      "average_duration_ms": 120,
      "created_at": "2025-08-02T09:00:00Z",
      "updated_at": "2025-08-02T10:25:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

**Status Codes**:

- `200`: Plugins retrieved successfully
- `400`: Invalid query parameters

### POST /api/v1/flexcore/plugins/{plugin_id}/execute

Execute a specific plugin with provided parameters.

**Path Parameters**:

- `plugin_id`: Unique plugin identifier

**Request Body**:

```json
{
  "operation": "health",
  "parameters": {
    "timeout_seconds": 30,
    "data": {
      "key": "value"
    }
  },
  "metadata": {
    "correlation_id": "req-123456",
    "user_id": "user-789"
  }
}
```

**Response**:

```json
{
  "execution_id": "exec-abc123",
  "plugin_id": "flext-service",
  "status": "completed",
  "result": {
    "success": true,
    "data": {
      "response": "Plugin executed successfully"
    }
  },
  "execution_time_ms": 150,
  "started_at": "2025-08-02T10:30:00Z",
  "completed_at": "2025-08-02T10:30:00.150Z",
  "metadata": {
    "correlation_id": "req-123456"
  }
}
```

**Status Codes**:

- `200`: Plugin executed successfully
- `400`: Invalid request parameters
- `404`: Plugin not found
- `409`: Plugin not available for execution
- `500`: Plugin execution failed

### GET /api/v1/flexcore/plugins/{plugin_id}/status

Get detailed status and metrics for a specific plugin.

**Response**:

```json
{
  "id": "flext-service",
  "name": "FLEXT Service Plugin",
  "status": "active",
  "health_status": "healthy",
  "metrics": {
    "total_executions": 45,
    "successful_executions": 43,
    "failed_executions": 2,
    "average_duration_ms": 120,
    "last_execution": "2025-08-02T10:25:00Z",
    "uptime_seconds": 3600
  },
  "resource_usage": {
    "cpu_percent": 15.5,
    "memory_mb": 128,
    "active_connections": 2
  }
}
```

**Status Codes**:

- `200`: Plugin status retrieved successfully
- `404`: Plugin not found

## 🔄 Pipeline Management API (Current)

> ⚠️ **Implementation Note**: Pipeline API is partially implemented and will be significantly enhanced during the refactoring process.

### GET /api/v1/flexcore/pipelines

List all pipelines with their current status.

**Response**:

```json
{
  "pipelines": [
    {
      "id": "pipeline-123",
      "name": "Data Processing Pipeline",
      "status": "running",
      "owner": "user-789",
      "created_at": "2025-08-02T09:00:00Z",
      "last_run": "2025-08-02T10:00:00Z",
      "steps": [
        {
          "id": "step-1",
          "name": "Data Extraction",
          "type": "plugin-execution",
          "status": "completed"
        }
      ]
    }
  ]
}
```

**Status Codes**:

- `200`: Pipelines retrieved successfully

### POST /api/v1/flexcore/pipelines/{pipeline_id}/execute

Start execution of a specific pipeline.

**Request Body**:

```json
{
  "parameters": {
    "source": "oracle-db",
    "target": "data-warehouse"
  },
  "metadata": {
    "correlation_id": "req-456789"
  }
}
```

**Response**:

```json
{
  "execution_id": "exec-def456",
  "pipeline_id": "pipeline-123",
  "status": "started",
  "started_at": "2025-08-02T10:30:00Z",
  "estimated_duration_seconds": 300
}
```

**Status Codes**:

- `202`: Pipeline execution started
- `404`: Pipeline not found
- `409`: Pipeline already running

## 📊 Event Sourcing API (Planned)

> ⚠️ **Implementation Status**: Event Sourcing API is planned for implementation after architectural refactoring. Current implementation is inadequate for production use.

### GET /api/v1/flexcore/events

Retrieve events from the event store with filtering and pagination.

**Query Parameters**:

- `aggregate_id` (optional): Filter by aggregate ID
- `event_type` (optional): Filter by event type
- `from_timestamp` (optional): Start timestamp (ISO 8601)
- `to_timestamp` (optional): End timestamp (ISO 8601)
- `limit` (optional): Maximum events to return (default: 100, max: 1000)
- `offset` (optional): Offset for pagination

**Planned Response**:

```json
{
  "events": [
    {
      "id": "evt-123456",
      "type": "PipelineStarted",
      "aggregate_id": "pipeline-123",
      "aggregate_type": "Pipeline",
      "version": 1,
      "data": {
        "pipeline_name": "Data Processing Pipeline",
        "started_by": "user-789"
      },
      "metadata": {
        "correlation_id": "req-456789",
        "causation_id": "cmd-789012"
      },
      "timestamp": "2025-08-02T10:30:00Z"
    }
  ],
  "total": 1,
  "has_more": false
}
```

### POST /api/v1/flexcore/events

Publish events to the event store (for testing and integration purposes).

**Request Body**:

```json
{
  "type": "TestEvent",
  "aggregate_id": "test-123",
  "aggregate_type": "Test",
  "data": {
    "message": "Test event data"
  },
  "metadata": {
    "correlation_id": "test-correlation"
  }
}
```

## 🎯 CQRS API (Planned)

### POST /api/v1/flexcore/commands

Execute commands through the command bus.

**Request Body**:

```json
{
  "command_type": "CreatePipeline",
  "command_id": "cmd-789012",
  "aggregate_id": "pipeline-new",
  "payload": {
    "name": "New Data Pipeline",
    "description": "Pipeline for processing customer data",
    "owner": "user-789"
  },
  "metadata": {
    "correlation_id": "req-new-pipeline",
    "user_id": "user-789"
  }
}
```

**Planned Response**:

```json
{
  "command_id": "cmd-789012",
  "status": "accepted",
  "result": {
    "aggregate_id": "pipeline-new",
    "events_generated": ["PipelineCreated"]
  },
  "execution_time_ms": 25,
  "timestamp": "2025-08-02T10:30:00Z"
}
```

### POST /api/v1/flexcore/queries

Execute queries through the query bus.

**Request Body**:

```json
{
  "query_type": "GetPipelineStatus",
  "query_id": "qry-456789",
  "parameters": {
    "pipeline_id": "pipeline-123"
  }
}
```

**Planned Response**:

```json
{
  "query_id": "qry-456789",
  "data": {
    "pipeline_id": "pipeline-123",
    "name": "Data Processing Pipeline",
    "status": "running",
    "current_step": "Data Transformation",
    "progress_percent": 65,
    "started_at": "2025-08-02T10:00:00Z",
    "estimated_completion": "2025-08-02T10:35:00Z"
  },
  "execution_time_ms": 15,
  "timestamp": "2025-08-02T10:30:00Z"
}
```

## 🔒 Error Handling

### Standard Error Response

All API endpoints follow RFC 7807 Problem Details format for errors:

```json
{
  "type": "https://flexcore.flext.sh/problems/plugin-not-found",
  "title": "Plugin Not Found",
  "status": 404,
  "detail": "The requested plugin 'invalid-plugin' was not found in the registry",
  "instance": "/api/v1/flexcore/plugins/invalid-plugin",
  "timestamp": "2025-08-02T10:30:00Z",
  "correlation_id": "req-123456"
}
```

### Common Error Types

#### Validation Errors (400)

```json
{
  "type": "https://flexcore.flext.sh/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request validation failed",
  "errors": [
    {
      "field": "parameters.timeout_seconds",
      "message": "Must be between 1 and 300 seconds"
    }
  ]
}
```

#### Plugin Execution Errors (500)

```json
{
  "type": "https://flexcore.flext.sh/problems/plugin-execution-error",
  "title": "Plugin Execution Failed",
  "status": 500,
  "detail": "Plugin execution failed due to timeout",
  "plugin_id": "flext-service",
  "execution_id": "exec-failed-123"
}
```

#### System Errors (503)

```json
{
  "type": "https://flexcore.flext.sh/problems/system-unavailable",
  "title": "System Unavailable",
  "status": 503,
  "detail": "FlexCore is currently undergoing maintenance",
  "retry_after": 300
}
```

## 🔧 Integration Examples

### Plugin Execution with Error Handling

```bash
#!/bin/bash
# Execute plugin with proper error handling

PLUGIN_ID="flext-service"
CORRELATION_ID="req-$(uuidgen)"

response=$(curl -s -w "%{http_code}" \
  -X POST "http://localhost:8080/api/v1/flexcore/plugins/${PLUGIN_ID}/execute" \
  -H "Content-Type: application/json" \
  -d "{
    \"operation\": \"health\",
    \"metadata\": {
      \"correlation_id\": \"${CORRELATION_ID}\"
    }
  }")

http_code="${response: -3}"
body="${response%???}"

if [[ "$http_code" == "200" ]]; then
  echo "Plugin executed successfully"
  echo "$body" | jq '.result'
else
  echo "Plugin execution failed with status: $http_code"
  echo "$body" | jq '.detail'
fi
```

### Health Check with Retry Logic

```python
import requests
import time
from typing import Dict, Any

def check_flexcore_health(max_retries: int = 3) -> Dict[str, Any]:
    """Check FlexCore health with retry logic."""

    for attempt in range(max_retries):
        try:
            response = requests.get(
                "http://localhost:8080/health",
                timeout=10
            )

            if response.status_code == 200:
                return response.json()
            else:
                print(f"Health check failed: {response.status_code}")

        except requests.RequestException as e:
            print(f"Attempt {attempt + 1} failed: {e}")

        if attempt < max_retries - 1:
            time.sleep(2 ** attempt)  # Exponential backoff

    raise Exception("FlexCore health check failed after all retries")
```

### Event Stream Monitoring (Planned)

```javascript
// JavaScript example for event stream monitoring
async function monitorEvents(aggregateId) {
  const params = new URLSearchParams({
    aggregate_id: aggregateId,
    limit: "100",
  });

  try {
    const response = await fetch(
      `http://localhost:8080/api/v1/flexcore/events?${params}`,
      {
        headers: {
          Accept: "application/json",
        },
      },
    );

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();
    return data.events;
  } catch (error) {
    console.error("Failed to retrieve events:", error);
    throw error;
  }
}
```

## 📚 SDK and Client Libraries

### Official SDKs (Planned)

- **Go SDK**: Native Go client for high-performance integrations
- **Python SDK**: Pythonic client with async support
- **JavaScript SDK**: Browser and Node.js compatible client
- **Java SDK**: Enterprise Java client with Spring integration

### Community Libraries

> Community libraries will be available after API stabilization.

## 🚀 API Versioning Strategy

### Current Version: v1

- **Base Path**: `/api/v1/flexcore`
- **Stability**: Development (breaking changes expected)
- **Sunset**: TBD (after architectural refactoring)

### Planned Versioning

- **v2**: Post-refactoring stable API
- **Backward Compatibility**: v1 will be maintained during transition period
- **Migration Guide**: Will be provided for v1 to v2 transition

## ⚠️ Known Limitations

### Current API Limitations

1. **No Authentication**: Security model not implemented
2. **Limited Error Context**: Error messages lack detailed context
3. **No Rate Limiting**: API can be overwhelmed by high request rates
4. **Synchronous Only**: No async/webhook support for long-running operations
5. **Basic Validation**: Limited input validation and sanitization

### Planned Improvements

1. **JWT Authentication**: Secure token-based authentication
2. **Rich Error Context**: Detailed error messages with troubleshooting hints
3. **Rate Limiting**: Configurable rate limits per client/endpoint
4. **Async Support**: Webhook callbacks for long-running operations
5. **Enhanced Validation**: Comprehensive input validation with custom rules

---

## 📖 Related Documentation

- [TODO.md](TODO.md) - **Critical architectural issues affecting API stability**
- [Architecture Overview](architecture/overview.md) - System architecture and design patterns
- [Integration Guide](integration/flext-ecosystem.md) - FLEXT ecosystem integration
- [Plugin Development](development/plugins.md) - Plugin development guide

**For the most current API status and stability information, always refer to [TODO.md](TODO.md).**
