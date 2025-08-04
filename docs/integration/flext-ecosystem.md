# FLEXT Ecosystem Integration Guide

**Version**: 0.9.0 | **Status**: Development | **Last Updated**: 2025-08-01

This document provides comprehensive guidance for integrating FlexCore with the broader FLEXT data integration ecosystem, including service communication patterns, data flow orchestration, and cross-project coordination.

> ⚠️ **Integration Notice**: Due to ongoing architectural refactoring, integration patterns may change significantly. See [TODO.md](../TODO.md) for current limitations and planned improvements.

## 🌐 FLEXT Ecosystem Overview

### Ecosystem Architecture

FlexCore serves as the **central orchestration engine** for 33 interconnected FLEXT projects organized into distinct architectural layers:

```
┌─────────────────────────────────────────────────────┐
│                FLEXT Ecosystem                      │
├─────────────────────────────────────────────────────┤
│  🎯 FlexCore (Runtime Container)                    │
│     • Plugin orchestration engine                  │
│     • Event sourcing coordinator                   │
│     • Service integration hub                      │
├─────────────────────────────────────────────────────┤
│  Singer Ecosystem (15 projects)                    │
│  ├─ Taps (5): Oracle, LDAP, LDIF, OIC, WMS        │
│  ├─ Targets (5): Oracle, LDAP, LDIF, OIC, WMS     │
│  └─ DBT (4): Data transformation projects          │
├─────────────────────────────────────────────────────┤
│  Application Services (5 projects)                 │
│  ├─ flext-api: REST API with FastAPI               │
│  ├─ flext-auth: Authentication & authorization     │
│  ├─ flext-web: Web interface & dashboard           │
│  ├─ flext-quality: Code quality & analysis         │
│  └─ flext-cli: Command-line interface              │
├─────────────────────────────────────────────────────┤
│  Infrastructure Services (6 projects)              │
│  ├─ flext-db-oracle: Oracle database connectivity  │
│  ├─ flext-ldap: LDAP directory services           │
│  ├─ flext-ldif: LDIF file processing              │
│  ├─ flext-oracle-wms: WMS API connectivity        │
│  ├─ flext-grpc: gRPC communication protocols      │
│  └─ flext-meltano: Singer/Meltano orchestration   │
├─────────────────────────────────────────────────────┤
│  Foundation Libraries (2 projects)                 │
│  ├─ flext-core: Python base patterns & DI         │
│  └─ flext-observability: Monitoring foundation    │
└─────────────────────────────────────────────────────┘
```

### FlexCore's Role

- **Orchestration Hub**: Coordinates execution across all ecosystem services
- **Plugin Runtime**: Secure execution environment for data processing plugins
- **Event Coordinator**: Central event sourcing and CQRS implementation
- **Integration Bridge**: Connects Go performance layer with Python business logic

## 🔗 Service Integration Patterns

### 1. FLEXT Service Integration (Primary)

#### Service Communication

**FLEXT Service** (port 8081) is the primary Python-based data processing service that FlexCore orchestrates:

```
FlexCore (8080) ←→ FLEXT Service (8081)
        ↕
    PostgreSQL (5433) + Redis (6380)
```

#### Integration Configuration

```yaml
# FlexCore configuration
flext_service:
  host: "localhost"
  port: 8081
  base_url: "http://localhost:8081"
  timeout_seconds: 30
  retry_attempts: 3
  health_check_interval: 60

# Connection settings
connection:
  max_connections: 10
  connection_timeout: 5
  read_timeout: 30
```

#### API Integration Examples

**Health Check Integration**:

```bash
# FlexCore → FLEXT Service health check
curl -X POST http://localhost:8080/api/v1/flexcore/plugins/flext-service/execute \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "health",
    "parameters": {},
    "metadata": {
      "correlation_id": "health-check-001"
    }
  }'
```

**Data Processing Pipeline**:

```bash
# FlexCore orchestrating FLEXT Service data processing
curl -X POST http://localhost:8080/api/v1/flexcore/plugins/flext-service/execute \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "process_data",
    "parameters": {
      "source": "oracle-db-prod",
      "target": "data-warehouse",
      "transform": "customer-normalization"
    },
    "metadata": {
      "correlation_id": "pipeline-exec-123",
      "user_id": "system",
      "priority": "high"
    }
  }'
```

### 2. flext-core Integration (Foundation)

#### Pattern Integration

FlexCore integrates with **flext-core** Python foundation library for consistent patterns:

```python
# flext-core patterns used in FlexCore
from flext.core import ServiceResult, DIContainer, LoggingContext
from flext.core.events import EventBus
from flext.core.patterns import ResultPattern

# Cross-language integration
class FlexCoreIntegration:
    def __init__(self):
        self.result_pattern = ResultPattern()
        self.di_container = DIContainer()
        self.event_bus = EventBus()
        self.logging = LoggingContext("flexcore")
```

#### Go-Python Bridge

```go
// Go implementation using flext-core patterns
type FlextCoreAdapter struct {
    httpClient   *http.Client
    resultParser ResultParser
    eventBridge  EventBridge
    logger       LoggingInterface
}

func (f *FlextCoreAdapter) ExecuteFlextOperation(ctx context.Context, operation FlextOperation) result.Result[FlextResponse] {
    // Convert Go request to flext-core compatible format
    request := f.convertToFlextFormat(operation)

    // Execute via HTTP bridge
    response, err := f.httpClient.Post(f.buildURL(operation.Type), "application/json", request)
    if err != nil {
        return result.Failure[FlextResponse](errors.Wrap(err, "flext-core integration failed"))
    }

    // Parse response using flext-core Result pattern
    return f.parseFlextResponse(response)
}
```

### 3. Singer Ecosystem Orchestration

#### Singer Pipeline Coordination

FlexCore orchestrates all 15 Singer ecosystem projects through standardized patterns:

```
FlexCore Orchestration
    ├─ Singer Taps (Data Extraction)
    │  ├─ flext-tap-oracle: Oracle database extraction
    │  ├─ flext-tap-ldap: LDAP directory extraction
    │  ├─ flext-tap-ldif: LDIF file processing
    │  ├─ flext-tap-oracle-oic: Oracle Integration Cloud
    │  └─ flext-tap-oracle-wms: Warehouse Management System
    │
    ├─ DBT Transformations (Data Processing)
    │  ├─ flext-dbt-oracle: Oracle data models
    │  ├─ flext-dbt-ldap: Directory data models
    │  ├─ flext-dbt-ldif: File processing models
    │  └─ flext-dbt-oracle-wms: WMS analytics models
    │
    └─ Singer Targets (Data Loading)
       ├─ flext-target-oracle: Oracle database loading
       ├─ flext-target-ldap: LDAP directory updates
       ├─ flext-target-ldif: LDIF file generation
       ├─ flext-target-oracle-oic: OIC integration
       └─ flext-target-oracle-wms: WMS data synchronization
```

#### Singer Pipeline Execution

```bash
# FlexCore orchestrating complete Singer pipeline
curl -X POST http://localhost:8080/api/v1/flexcore/pipelines/singer-etl-001/execute \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_config": {
      "tap": "flext-tap-oracle",
      "target": "flext-target-oracle-wms",
      "transform": "flext-dbt-oracle-wms"
    },
    "parameters": {
      "source_connection": "oracle-prod",
      "target_connection": "wms-staging",
      "batch_size": 1000,
      "parallel_jobs": 4
    },
    "metadata": {
      "correlation_id": "singer-pipeline-456",
      "execution_mode": "incremental",
      "data_classification": "customer-data"
    }
  }'
```

### 4. Infrastructure Services Integration

#### Database Connectivity

**flext-db-oracle** integration for Oracle database operations:

```yaml
# Oracle database integration via flext-db-oracle
oracle_integration:
  service: "flext-db-oracle"
  connection_pools:
    - name: "oracle-prod"
      host: "oracle.prod.company.com"
      port: 1521
      service_name: "PRODDB"
      pool_size: 20
    - name: "oracle-staging"
      host: "oracle.staging.company.com"
      port: 1521
      service_name: "STAGINGDB"
      pool_size: 10
```

#### Directory Services

**flext-ldap** integration for LDAP operations:

```yaml
# LDAP integration via flext-ldap
ldap_integration:
  service: "flext-ldap"
  connections:
    - name: "active-directory"
      host: "ldap.company.com"
      port: 389
      bind_dn: "cn=service,ou=applications,dc=company,dc=com"
      search_base: "dc=company,dc=com"
```

### 5. Application Services Integration

#### API Gateway Integration

**flext-api** serves as the primary API gateway with FlexCore as the backend orchestrator:

```
Client Request → flext-api (FastAPI) → FlexCore (Orchestration) → Services
     ↓                  ↓                      ↓
Authentication   →  Authorization    →   Execution
(flext-auth)       (RBAC rules)        (Plugin runtime)
```

#### Web Interface Integration

**flext-web** provides the management interface for FlexCore operations:

```typescript
// flext-web integration with FlexCore APIs
class FlexCoreApiClient {
  async executePlugin(
    pluginId: string,
    operation: PluginOperation,
  ): Promise<PluginResult> {
    const response = await fetch(
      `/api/v1/flexcore/plugins/${pluginId}/execute`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(operation),
      },
    );

    if (!response.ok) {
      throw new FlexCoreApiError(await response.json());
    }

    return await response.json();
  }

  async monitorPipelineExecution(pipelineId: string): Promise<PipelineStatus> {
    // WebSocket connection for real-time monitoring
    const ws = new WebSocket(`/ws/pipelines/${pipelineId}/status`);

    return new Promise((resolve, reject) => {
      ws.onmessage = (event) => {
        const status = JSON.parse(event.data);
        if (status.state === "completed" || status.state === "failed") {
          resolve(status);
        }
      };
    });
  }
}
```

## 🔄 Data Flow Patterns

### 1. Event-Driven Data Flow

#### Event Sourcing Coordination

FlexCore coordinates event-driven data flows across the ecosystem:

```
Data Source → Singer Tap → FlexCore Event Store → Event Handlers
    ↓              ↓              ↓                    ↓
Oracle DB    →  Extract     →  PipelineStarted   →  Notification
LDAP Dir     →  Transform   →  DataExtracted     →  Quality Check
WMS API      →  Load        →  PipelineCompleted →  Audit Log
```

#### Event-Driven Integration Example

```go
// FlexCore event handling for ecosystem coordination
type EcosystemEventHandler struct {
    eventBus      EventBus
    pluginManager PluginManager
    logger        LoggingInterface
}

func (h *EcosystemEventHandler) HandleDataExtractionCompleted(ctx context.Context, event DataExtractionCompletedEvent) error {
    // Trigger DBT transformation
    transformCommand := TransformDataCommand{
        SourceData:     event.ExtractedData,
        TransformType:  event.TransformationRules,
        TargetFormat:   event.TargetSchema,
    }

    // Execute via plugin system
    result := h.pluginManager.ExecutePlugin(ctx, "flext-dbt-processor", transformCommand)
    if result.IsFailure() {
        return result.Error()
    }

    // Publish transformation completed event
    transformedEvent := DataTransformationCompletedEvent{
        OriginalEvent: event,
        TransformedData: result.Value(),
        CompletedAt: time.Now(),
    }

    return h.eventBus.Publish(ctx, transformedEvent)
}
```

### 2. Synchronous Pipeline Orchestration

#### Multi-Service Pipeline

```yaml
# Pipeline definition for multi-service coordination
pipeline:
  name: "customer-data-synchronization"
  description: "Synchronize customer data across Oracle, LDAP, and WMS"

  steps:
    - name: "extract-oracle-customers"
      service: "flext-tap-oracle"
      config:
        connection: "oracle-prod"
        query: "SELECT * FROM customers WHERE updated_at > ?"

    - name: "transform-customer-data"
      service: "flext-dbt-oracle"
      depends_on: ["extract-oracle-customers"]
      config:
        model: "customer_normalization"

    - name: "update-ldap-directory"
      service: "flext-target-ldap"
      depends_on: ["transform-customer-data"]
      config:
        connection: "active-directory"
        operation: "upsert"

    - name: "sync-wms-customers"
      service: "flext-target-oracle-wms"
      depends_on: ["transform-customer-data"]
      config:
        connection: "wms-staging"
        batch_size: 500
```

### 3. Asynchronous Message Patterns

#### Message Queue Integration

```go
// FlexCore integration with message queues for async processing
type MessageQueueIntegration struct {
    publisher  MessagePublisher
    subscriber MessageSubscriber
    router     MessageRouter
}

func (mq *MessageQueueIntegration) PublishDataProcessingRequest(ctx context.Context, request DataProcessingRequest) error {
    message := Message{
        Type:      "data.processing.requested",
        Payload:   request,
        Metadata: map[string]string{
            "correlation_id": request.CorrelationID,
            "source_service": "flexcore",
            "target_service": request.TargetService,
        },
    }

    return mq.publisher.Publish(ctx, "data-processing-queue", message)
}

func (mq *MessageQueueIntegration) HandleProcessingResults(ctx context.Context, message Message) error {
    var result DataProcessingResult
    if err := json.Unmarshal(message.Payload, &result); err != nil {
        return err
    }

    // Route result to appropriate handler based on correlation ID
    return mq.router.RouteMessage(ctx, result.CorrelationID, result)
}
```

## 🔧 Configuration Management

### 1. Environment-Specific Configuration

#### Development Configuration

```yaml
# config/development.yaml
flexcore:
  environment: "development"
  debug: true
  port: 8080

services:
  flext_service:
    host: "localhost"
    port: 8081
    timeout: 30

  database:
    host: "localhost"
    port: 5433
    database: "flexcore_dev"

  redis:
    host: "localhost"
    port: 6380

ecosystem_integration:
  singer_projects:
    base_path: "/home/user/flext"
    meltano_path: "/home/user/flext/flext-meltano"

  infrastructure_services:
    oracle:
      connection_string: "oracle://dev:password@localhost:1521/XEPDB1"
    ldap:
      url: "ldap://localhost:389"
      bind_dn: "cn=REDACTED_LDAP_BIND_PASSWORD,dc=example,dc=com"
```

#### Production Configuration

```yaml
# config/production.yaml
flexcore:
  environment: "production"
  debug: false
  port: 8080
  max_connections: 1000

services:
  flext_service:
    host: "internal.invalid"
    port: 8081
    timeout: 60
    connection_pool_size: 50

  database:
    host: "internal.invalid"
    port: 5432
    database: "flexcore_prod"
    ssl_mode: "require"
    connection_pool_size: 100

  redis:
    cluster_endpoints:
      - "internal.invalid:6379"
      - "internal.invalid:6379"
      - "internal.invalid:6379"

security:
  tls_enabled: true
  certificate_path: "/etc/ssl/certs/flexcore.crt"
  key_path: "/etc/ssl/private/flexcore.key"

observability:
  metrics_enabled: true
  tracing_enabled: true
  log_level: "info"
```

### 2. Service Discovery Integration

#### Kubernetes Service Discovery

```yaml
# kubernetes/service-discovery.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: flexcore-service-discovery
data:
  services.yaml: |
    services:
      flext-service:
        namespace: "flext-system"
        service: "flext-service"
        port: 8081
        
      flext-api:
        namespace: "flext-system"
        service: "flext-api"
        port: 8000
        
      singer-projects:
        namespace: "flext-data"
        services:
          - name: "flext-tap-oracle"
            port: 8090
          - name: "flext-target-oracle"
            port: 8091
```

## 🚀 Deployment Integration

### 1. Docker Compose Integration

#### Complete Ecosystem Stack

```yaml
# docker-compose.ecosystem.yml
version: "3.8"

services:
  # FlexCore (Orchestration Engine)
  flexcore:
    build: ./flexcore
    ports:
      - "8080:8080"
    environment:
      - FLEXCORE_ENV=development
      - POSTGRES_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
      - flext-service
    networks:
      - flext-network

  # FLEXT Service (Python Data Processing)
  flext-service:
    build: ./cmd/flext
    ports:
      - "8081:8081"
    environment:
      - FLEXT_ENV=development
      - POSTGRES_HOST=postgres
    depends_on:
      - postgres
    networks:
      - flext-network

  # Application Services
  flext-api:
    build: ./flext-api
    ports:
      - "8000:8000"
    environment:
      - FLEXCORE_URL=http://flexcore:8080
    depends_on:
      - flexcore
    networks:
      - flext-network

  flext-web:
    build: ./flext-web
    ports:
      - "3000:3000"
    environment:
      - FLEXT_API_URL=http://flext-api:8000
    depends_on:
      - flext-api
    networks:
      - flext-network

  # Infrastructure
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: flext_ecosystem
      POSTGRES_USER: flext
      POSTGRES_PASSWORD: flext_dev_2025
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - flext-network

  redis:
    image: redis:7-alpine
    networks:
      - flext-network

volumes:
  postgres_data:

networks:
  flext-network:
    driver: bridge
```

### 2. Kubernetes Integration

#### FlexCore Deployment

```yaml
# kubernetes/flexcore-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flexcore
  namespace: flext-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flexcore
  template:
    metadata:
      labels:
        app: flexcore
    spec:
      containers:
        - name: flexcore
          image: flext-sh/flexcore:0.9.0
          ports:
            - containerPort: 8080
          env:
            - name: FLEXCORE_ENV
              value: "production"
            - name: POSTGRES_HOST
              value: "internal.invalid"
            - name: REDIS_HOST
              value: "internal.invalid"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

## 📊 Monitoring and Observability

### 1. Cross-Service Monitoring

#### Prometheus Configuration

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "flexcore"
    static_configs:
      - targets: ["flexcore:8080"]
    metrics_path: "/metrics"

  - job_name: "flext-service"
    static_configs:
      - targets: ["flext-service:8081"]
    metrics_path: "/metrics"

  - job_name: "ecosystem-services"
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ["flext-system", "flext-data"]
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_flext_component]
        action: keep
        regex: "singer-(tap|target|dbt)"
```

#### Grafana Dashboard Integration

```json
{
  "dashboard": {
    "title": "FLEXT Ecosystem Overview",
    "panels": [
      {
        "title": "FlexCore Plugin Executions",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(flexcore_plugin_executions_total[5m])",
            "legendFormat": "{{plugin_name}}"
          }
        ]
      },
      {
        "title": "Cross-Service Response Times",
        "type": "heatmap",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{service}}"
          }
        ]
      }
    ]
  }
}
```

### 2. Distributed Tracing

#### Jaeger Integration

```go
// FlexCore distributed tracing setup
func InitializeTracing() {
    tracer, closer := jaeger.NewTracer(
        "flexcore",
        jaeger.NewConstSampler(true),
        jaeger.NewRemoteReporter(
            jaeger.NewUDPTransport("jaeger:14268", 0),
        ),
    )

    opentracing.SetGlobalTracer(tracer)
    return closer
}

// Cross-service trace correlation
func (p *PluginExecutor) ExecuteWithTracing(ctx context.Context, pluginID string, operation PluginOperation) result.Result[interface{}] {
    span, ctx := opentracing.StartSpanFromContext(ctx, "plugin.execute")
    defer span.Finish()

    span.SetTag("plugin.id", pluginID)
    span.SetTag("operation.type", operation.Type)

    // Inject trace context into HTTP headers for downstream services
    headers := make(http.Header)
    opentracing.GlobalTracer().Inject(
        span.Context(),
        opentracing.HTTPHeaders,
        opentracing.HTTPHeadersCarrier(headers),
    )

    return p.executePlugin(ctx, pluginID, operation, headers)
}
```

## ⚠️ Current Limitations and Future Enhancements

### Current Integration Limitations

1. **Authentication Gap**: No secure authentication between services
2. **Limited Error Propagation**: Error context not fully preserved across services
3. **Manual Configuration**: Service discovery requires manual configuration
4. **Synchronous Only**: Limited async operation support
5. **Basic Monitoring**: Limited cross-service observability

### Planned Enhancements

1. **Service Mesh Integration**: Istio/Linkerd for secure service communication
2. **Advanced Error Handling**: Rich error context propagation
3. **Automatic Service Discovery**: Kubernetes-native service discovery
4. **Async Operation Support**: Webhook callbacks and message queues
5. **Enhanced Observability**: Complete distributed tracing and metrics

### Migration Considerations

- **Backward Compatibility**: Integration patterns will evolve during refactoring
- **Configuration Changes**: Service configurations may change significantly
- **API Evolution**: Integration APIs will be versioned for stability
- **Testing Requirements**: Integration tests needed for all cross-service communication

---

## 📖 Related Documentation

- [TODO.md](../TODO.md) - **Critical integration issues and refactoring impact**
- [Architecture Overview](../architecture/overview.md) - System architecture and integration patterns
- [API Reference](../api-reference.md) - FlexCore API for service integration
- [Plugin Development](../development/plugins.md) - Creating ecosystem-compatible plugins

**For the most current integration status and limitations, always refer to [TODO.md](../TODO.md).**
