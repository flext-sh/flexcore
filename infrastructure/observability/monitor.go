// Package observability provides real-time monitoring capabilities
package observability

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// Monitor provides real-time system monitoring
type Monitor struct {
	mu              sync.RWMutex
	metrics         *MetricsCollector
	tracer          *Tracer
	healthCheckers  map[string]HealthChecker
	alerts          []Alert
	config          MonitorConfig
	running         bool
	stopCh          chan struct{}
	alertCh         chan Alert
	subscribers     []MonitorSubscriber
}

// MonitorConfig configures the monitor
type MonitorConfig struct {
	MetricsInterval   time.Duration
	HealthInterval    time.Duration
	ResourceInterval  time.Duration
	MaxAlerts         int
	AlertThresholds   AlertThresholds
}

// AlertThresholds defines alert thresholds
type AlertThresholds struct {
	CPUPercent     float64
	MemoryMB       float64
	GoroutinesMax  int
	LatencyMS      float64
	ErrorRatePercent float64
}

// HealthChecker interface for health checks
type HealthChecker interface {
	HealthCheck(ctx context.Context) result.Result[bool]
	Name() string
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeSystem    AlertType = "system"
	AlertTypeHealth    AlertType = "health"
	AlertTypeLatency   AlertType = "latency"
	AlertTypeErrorRate AlertType = "error_rate"
	AlertTypeCustom    AlertType = "custom"
)

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// MonitorSubscriber receives monitoring events
type MonitorSubscriber interface {
	OnAlert(alert Alert)
	OnMetricsUpdate(metrics []MetricPoint)
	OnHealthUpdate(status HealthStatus)
}

// HealthStatus represents overall system health
type HealthStatus struct {
	Overall   HealthState            `json:"overall"`
	Services  map[string]HealthState `json:"services"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// HealthState represents health state
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateWarning   HealthState = "warning"
	HealthStateUnhealthy HealthState = "unhealthy"
	HealthStateUnknown   HealthState = "unknown"
)

// SystemMetrics represents system resource metrics
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	Goroutines    int     `json:"goroutines"`
	HeapMB        float64 `json:"heap_mb"`
	GCPauseMS     float64 `json:"gc_pause_ms"`
	Uptime        time.Duration `json:"uptime"`
}

// NewMonitor creates a new monitor
func NewMonitor(metrics *MetricsCollector, tracer *Tracer, config MonitorConfig) *Monitor {
	if config.MetricsInterval == 0 {
		config.MetricsInterval = 10 * time.Second
	}
	if config.HealthInterval == 0 {
		config.HealthInterval = 30 * time.Second
	}
	if config.ResourceInterval == 0 {
		config.ResourceInterval = 5 * time.Second
	}
	if config.MaxAlerts == 0 {
		config.MaxAlerts = 1000
	}

	// Set default thresholds
	if config.AlertThresholds.CPUPercent == 0 {
		config.AlertThresholds.CPUPercent = 80.0
	}
	if config.AlertThresholds.MemoryMB == 0 {
		config.AlertThresholds.MemoryMB = 1024.0
	}
	if config.AlertThresholds.GoroutinesMax == 0 {
		config.AlertThresholds.GoroutinesMax = 10000
	}
	if config.AlertThresholds.LatencyMS == 0 {
		config.AlertThresholds.LatencyMS = 1000.0
	}
	if config.AlertThresholds.ErrorRatePercent == 0 {
		config.AlertThresholds.ErrorRatePercent = 5.0
	}

	return &Monitor{
		metrics:        metrics,
		tracer:         tracer,
		healthCheckers: make(map[string]HealthChecker),
		alerts:         make([]Alert, 0),
		config:         config,
		stopCh:         make(chan struct{}),
		alertCh:        make(chan Alert, 100),
		subscribers:    make([]MonitorSubscriber, 0),
	}
}

// Start starts the monitor
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return errors.ValidationError("monitor is already running")
	}
	m.running = true
	m.mu.Unlock()

	// Start monitoring goroutines
	go m.metricsLoop(ctx)
	go m.healthLoop(ctx)
	go m.resourceLoop(ctx)
	go m.alertLoop(ctx)

	return nil
}

// Stop stops the monitor
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopCh)
}

// AddHealthChecker adds a health checker
func (m *Monitor) AddHealthChecker(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCheckers[checker.Name()] = checker
}

// RemoveHealthChecker removes a health checker
func (m *Monitor) RemoveHealthChecker(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.healthCheckers, name)
}

// Subscribe adds a monitor subscriber
func (m *Monitor) Subscribe(subscriber MonitorSubscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, subscriber)
}

// CreateAlert creates a new alert
func (m *Monitor) CreateAlert(alertType AlertType, severity AlertSeverity, title, message string, metadata map[string]interface{}) {
	alert := Alert{
		ID:        m.generateAlertID(),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  metadata,
		Resolved:  false,
	}

	select {
	case m.alertCh <- alert:
	default:
		// Alert channel is full, drop the alert
	}
}

// GetAlerts returns current alerts
func (m *Monitor) GetAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]Alert, len(m.alerts))
	copy(alerts, m.alerts)
	return alerts
}

// GetHealthStatus returns current health status
func (m *Monitor) GetHealthStatus(ctx context.Context) HealthStatus {
	m.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range m.healthCheckers {
		checkers[name] = checker
	}
	m.mu.RUnlock()

	status := HealthStatus{
		Services:  make(map[string]HealthState),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	overall := HealthStateHealthy
	healthyCount := 0
	totalCount := len(checkers)

	for name, checker := range checkers {
		result := checker.HealthCheck(ctx)
		if result.IsSuccess() && result.Value() {
			status.Services[name] = HealthStateHealthy
			healthyCount++
		} else {
			status.Services[name] = HealthStateUnhealthy
			if overall == HealthStateHealthy {
				overall = HealthStateWarning
			}
			if result.IsFailure() {
				status.Details[name+"_error"] = result.Error().Error()
			}
		}
	}

	// Determine overall health
	if totalCount == 0 {
		overall = HealthStateUnknown
	} else if healthyCount == 0 {
		overall = HealthStateUnhealthy
	} else if healthyCount < totalCount {
		overall = HealthStateWarning
	}

	status.Overall = overall
	return status
}

// GetSystemMetrics returns current system metrics
func (m *Monitor) GetSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemMetrics{
		CPUPercent: 0.0, // Would need additional CPU monitoring
		MemoryMB:   float64(memStats.Alloc) / 1024 / 1024,
		Goroutines: runtime.NumGoroutine(),
		HeapMB:     float64(memStats.HeapAlloc) / 1024 / 1024,
		GCPauseMS:  float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1000000,
		Uptime:     time.Since(time.Now()), // Would track actual uptime
	}
}

// Monitor loops

func (m *Monitor) metricsLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

func (m *Monitor) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkHealth(ctx)
		}
	}
}

func (m *Monitor) resourceLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.ResourceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.monitorResources()
		}
	}
}

func (m *Monitor) alertLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case alert := <-m.alertCh:
			m.processAlert(alert)
		}
	}
}

func (m *Monitor) collectMetrics() {
	if m.metrics == nil || !m.metrics.IsEnabled() {
		return
	}

	// Collect system metrics
	sysMetrics := m.GetSystemMetrics()
	
	m.metrics.SetGauge("system_memory_mb", sysMetrics.MemoryMB, nil)
	m.metrics.SetGauge("system_goroutines", float64(sysMetrics.Goroutines), nil)
	m.metrics.SetGauge("system_heap_mb", sysMetrics.HeapMB, nil)
	m.metrics.SetGauge("system_gc_pause_ms", sysMetrics.GCPauseMS, nil)

	// Check thresholds
	if sysMetrics.MemoryMB > m.config.AlertThresholds.MemoryMB {
		m.CreateAlert(AlertTypeSystem, AlertSeverityWarning, 
			"High Memory Usage", 
			fmt.Sprintf("Memory usage: %.2f MB", sysMetrics.MemoryMB), 
			map[string]interface{}{"memory_mb": sysMetrics.MemoryMB})
	}

	if sysMetrics.Goroutines > m.config.AlertThresholds.GoroutinesMax {
		m.CreateAlert(AlertTypeSystem, AlertSeverityWarning, 
			"High Goroutine Count", 
			fmt.Sprintf("Goroutines: %d", sysMetrics.Goroutines), 
			map[string]interface{}{"goroutines": sysMetrics.Goroutines})
	}

	// Notify subscribers
	m.notifyMetricsUpdate()
}

func (m *Monitor) checkHealth(ctx context.Context) {
	status := m.GetHealthStatus(ctx)
	
	// Create alerts for unhealthy services
	for service, state := range status.Services {
		if state == HealthStateUnhealthy {
			m.CreateAlert(AlertTypeHealth, AlertSeverityError, 
				"Service Unhealthy", 
				fmt.Sprintf("Service %s is unhealthy", service), 
				map[string]interface{}{"service": service})
		}
	}

	// Notify subscribers
	m.notifyHealthUpdate(status)
}

func (m *Monitor) monitorResources() {
	// Additional resource monitoring would go here
	// For now, this is handled in collectMetrics
}

func (m *Monitor) processAlert(alert Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to alerts list
	m.alerts = append(m.alerts, alert)

	// Limit alerts to prevent memory leak
	if len(m.alerts) > m.config.MaxAlerts {
		m.alerts = m.alerts[len(m.alerts)-m.config.MaxAlerts:]
	}

	// Notify subscribers
	for _, subscriber := range m.subscribers {
		go subscriber.OnAlert(alert)
	}
}

func (m *Monitor) notifyMetricsUpdate() {
	if m.metrics == nil {
		return
	}

	metrics := m.metrics.GetAllMetrics()
	
	m.mu.RLock()
	subscribers := make([]MonitorSubscriber, len(m.subscribers))
	copy(subscribers, m.subscribers)
	m.mu.RUnlock()

	for _, subscriber := range subscribers {
		go subscriber.OnMetricsUpdate(metrics)
	}
}

func (m *Monitor) notifyHealthUpdate(status HealthStatus) {
	m.mu.RLock()
	subscribers := make([]MonitorSubscriber, len(m.subscribers))
	copy(subscribers, m.subscribers)
	m.mu.RUnlock()

	for _, subscriber := range subscribers {
		go subscriber.OnHealthUpdate(status)
	}
}

func (m *Monitor) generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// IsRunning returns whether the monitor is running
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}