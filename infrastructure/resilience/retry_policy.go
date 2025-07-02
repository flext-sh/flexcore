// Retry Policy - REAL Resilience Implementation
package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors func(error) bool
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(config RetryPolicyConfig) *RetryPolicy {
	policy := &RetryPolicy{
		MaxAttempts:     config.MaxAttempts,
		InitialDelay:    config.InitialDelay,
		MaxDelay:        config.MaxDelay,
		BackoffFactor:   config.BackoffFactor,
		Jitter:          config.Jitter,
		RetryableErrors: config.RetryableErrors,
	}

	// Set defaults
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 3
	}
	if policy.InitialDelay <= 0 {
		policy.InitialDelay = 100 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 10 * time.Second
	}
	if policy.BackoffFactor <= 0 {
		policy.BackoffFactor = 2.0
	}
	if policy.RetryableErrors == nil {
		policy.RetryableErrors = defaultRetryableErrors
	}

	return policy
}

// RetryPolicyConfig holds configuration for retry policy
type RetryPolicyConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors func(error) bool
}

// Execute executes a function with retry logic
func (rp *RetryPolicy) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return rp.ExecuteContext(context.Background(), func(ctx context.Context) (interface{}, error) {
		return fn()
	})
}

// ExecuteContext executes a function with retry logic and context
func (rp *RetryPolicy) ExecuteContext(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= rp.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !rp.RetryableErrors(err) {
			return nil, fmt.Errorf("non-retryable error on attempt %d: %w", attempt, err)
		}

		// Don't sleep after the last attempt
		if attempt == rp.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := rp.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("max retry attempts (%d) exceeded, last error: %w", rp.MaxAttempts, lastErr)
}

func (rp *RetryPolicy) calculateDelay(attempt int) time.Duration {
	delay := float64(rp.InitialDelay) * math.Pow(rp.BackoffFactor, float64(attempt-1))

	if delay > float64(rp.MaxDelay) {
		delay = float64(rp.MaxDelay)
	}

	if rp.Jitter {
		// Add random jitter (±25%)
		jitter := delay * 0.25 * (rand.Float64()*2 - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

func defaultRetryableErrors(err error) bool {
	// By default, retry all errors except context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	return true
}

// FailoverManager manages service failover
type FailoverManager struct {
	services        map[string][]ServiceEndpoint
	healthCheckers  map[string]HealthChecker
	currentEndpoint map[string]int
	circuitBreakers map[string]*CircuitBreaker
}

// ServiceEndpoint represents a service endpoint
type ServiceEndpoint struct {
	ID       string
	Address  string
	Priority int
	Healthy  bool
}

// HealthChecker defines health check interface
type HealthChecker interface {
	CheckHealth(ctx context.Context, endpoint ServiceEndpoint) error
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager() *FailoverManager {
	return &FailoverManager{
		services:        make(map[string][]ServiceEndpoint),
		healthCheckers:  make(map[string]HealthChecker),
		currentEndpoint: make(map[string]int),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// RegisterService registers a service with multiple endpoints
func (fm *FailoverManager) RegisterService(serviceName string, endpoints []ServiceEndpoint, healthChecker HealthChecker) {
	fm.services[serviceName] = endpoints
	fm.healthCheckers[serviceName] = healthChecker
	fm.currentEndpoint[serviceName] = 0

	// Create circuit breaker for each endpoint
	for _, endpoint := range endpoints {
		cbName := fmt.Sprintf("%s_%s", serviceName, endpoint.ID)
		fm.circuitBreakers[cbName] = NewCircuitBreaker(CircuitBreakerConfig{
			Name:        cbName,
			MaxRequests: 10,
			Interval:    time.Minute,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts Counts) bool {
				return counts.ConsecutiveFailures >= 3
			},
		})
	}
}

// GetHealthyEndpoint returns a healthy endpoint for the service
func (fm *FailoverManager) GetHealthyEndpoint(ctx context.Context, serviceName string) (*ServiceEndpoint, error) {
	endpoints, exists := fm.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not registered", serviceName)
	}

	healthChecker := fm.healthCheckers[serviceName]

	// Check current endpoint first
	currentIdx := fm.currentEndpoint[serviceName]
	if currentIdx < len(endpoints) {
		endpoint := &endpoints[currentIdx]
		cbName := fmt.Sprintf("%s_%s", serviceName, endpoint.ID)

		if cb, exists := fm.circuitBreakers[cbName]; exists && cb.State() != StateOpen {
			if healthChecker != nil {
				if err := healthChecker.CheckHealth(ctx, *endpoint); err == nil {
					endpoint.Healthy = true
					return endpoint, nil
				}
			} else {
				// No health checker, assume healthy if circuit breaker is not open
				endpoint.Healthy = true
				return endpoint, nil
			}
		}
	}

	// Find next healthy endpoint
	for i, endpoint := range endpoints {
		if i == currentIdx {
			continue // Already checked
		}

		cbName := fmt.Sprintf("%s_%s", serviceName, endpoint.ID)
		if cb, exists := fm.circuitBreakers[cbName]; exists && cb.State() == StateOpen {
			continue // Circuit breaker is open
		}

		if healthChecker != nil {
			if err := healthChecker.CheckHealth(ctx, endpoint); err != nil {
				continue // Endpoint is unhealthy
			}
		}

		// Found healthy endpoint, update current
		fm.currentEndpoint[serviceName] = i
		endpoint.Healthy = true
		return &endpoint, nil
	}

	return nil, fmt.Errorf("no healthy endpoints available for service %s", serviceName)
}

// ExecuteWithFailover executes a function with automatic failover
func (fm *FailoverManager) ExecuteWithFailover(ctx context.Context, serviceName string, fn func(endpoint ServiceEndpoint) (interface{}, error)) (interface{}, error) {
	endpoint, err := fm.GetHealthyEndpoint(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	cbName := fmt.Sprintf("%s_%s", serviceName, endpoint.ID)
	cb, exists := fm.circuitBreakers[cbName]
	if !exists {
		// No circuit breaker, execute directly
		return fn(*endpoint)
	}

	// Execute with circuit breaker
	return cb.ExecuteContext(ctx, func(ctx context.Context) (interface{}, error) {
		return fn(*endpoint)
	})
}

// GetServiceStats returns statistics for all services
func (fm *FailoverManager) GetServiceStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for serviceName, endpoints := range fm.services {
		serviceStats := map[string]interface{}{
			"current_endpoint": fm.currentEndpoint[serviceName],
			"total_endpoints":  len(endpoints),
			"endpoints":        make([]map[string]interface{}, 0, len(endpoints)),
		}

		for _, endpoint := range endpoints {
			cbName := fmt.Sprintf("%s_%s", serviceName, endpoint.ID)
			endpointStats := map[string]interface{}{
				"id":       endpoint.ID,
				"address":  endpoint.Address,
				"priority": endpoint.Priority,
				"healthy":  endpoint.Healthy,
			}

			if cb, exists := fm.circuitBreakers[cbName]; exists {
				counts := cb.Counts()
				endpointStats["circuit_breaker"] = map[string]interface{}{
					"state":               cb.State().String(),
					"requests":            counts.Requests,
					"successes":           counts.TotalSuccesses,
					"failures":            counts.TotalFailures,
					"consecutive_failures": counts.ConsecutiveFailures,
				}
			}

			serviceStats["endpoints"] = append(serviceStats["endpoints"].([]map[string]interface{}), endpointStats)
		}

		stats[serviceName] = serviceStats
	}

	return stats
}

// PerformHealthChecks runs health checks on all endpoints
func (fm *FailoverManager) PerformHealthChecks(ctx context.Context) map[string]map[string]bool {
	results := make(map[string]map[string]bool)

	for serviceName, endpoints := range fm.services {
		healthChecker := fm.healthCheckers[serviceName]
		if healthChecker == nil {
			continue
		}

		serviceHealth := make(map[string]bool)
		for _, endpoint := range endpoints {
			err := healthChecker.CheckHealth(ctx, endpoint)
			serviceHealth[endpoint.ID] = err == nil
		}
		results[serviceName] = serviceHealth
	}

	return results
}
