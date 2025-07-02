// Circuit Breaker - REAL Resilience Implementation
package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int32

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name               string
	MaxRequests        uint32
	Interval           time.Duration
	Timeout            time.Duration
	ReadyToTrip        func(counts Counts) bool
	OnStateChange      func(name string, from CircuitBreakerState, to CircuitBreakerState)
	ShouldTrip         func(counts Counts) bool
}

// Counts holds the statistics for circuit breaker
type Counts struct {
	Requests              uint32
	TotalSuccesses        uint32
	TotalFailures         uint32
	ConsecutiveSuccesses  uint32
	ConsecutiveFailures   uint32
}

// CircuitBreaker implements the circuit breaker pattern for resilience
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState)

	mutex      sync.Mutex
	state      CircuitBreakerState
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          config.Name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		onStateChange: config.OnStateChange,
	}

	if cb.readyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	}

	cb.toNewGeneration(time.Now())
	return cb
}

// Execute runs the given function within the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := fn()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// ExecuteContext runs the given function within the circuit breaker with context
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := fn(ctx)
	cb.afterRequest(generation, err == nil)
	return result, err
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, fmt.Errorf("circuit breaker '%s' is open", cb.name)
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, fmt.Errorf("circuit breaker '%s' is half-open and max requests exceeded", cb.name)
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state CircuitBreakerState, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == StateHalfOpen {
		cb.setState(StateClosed, now)
	}
}

func (cb *CircuitBreaker) onFailure(state CircuitBreakerState, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (CircuitBreakerState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

// ResilienceManager manages multiple circuit breakers and retry policies
type ResilienceManager struct {
	circuitBreakers map[string]*CircuitBreaker
	retryPolicies   map[string]*RetryPolicy
	mutex           sync.RWMutex
}

// NewResilienceManager creates a new resilience manager
func NewResilienceManager() *ResilienceManager {
	return &ResilienceManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryPolicies:   make(map[string]*RetryPolicy),
	}
}

// RegisterCircuitBreaker registers a circuit breaker
func (rm *ResilienceManager) RegisterCircuitBreaker(name string, config CircuitBreakerConfig) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	config.Name = name
	rm.circuitBreakers[name] = NewCircuitBreaker(config)
}

// GetCircuitBreaker returns a circuit breaker by name
func (rm *ResilienceManager) GetCircuitBreaker(name string) (*CircuitBreaker, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	cb, exists := rm.circuitBreakers[name]
	return cb, exists
}

// ExecuteWithResilience executes a function with circuit breaker and retry
func (rm *ResilienceManager) ExecuteWithResilience(ctx context.Context, name string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	cb, exists := rm.GetCircuitBreaker(name)
	if !exists {
		// No circuit breaker, execute directly
		return fn(ctx)
	}

	retry, hasRetry := rm.getRetryPolicy(name)
	if !hasRetry {
		// Only circuit breaker
		return cb.ExecuteContext(ctx, fn)
	}

	// Both circuit breaker and retry
	return cb.ExecuteContext(ctx, func(ctx context.Context) (interface{}, error) {
		return retry.ExecuteContext(ctx, fn)
	})
}

func (rm *ResilienceManager) getRetryPolicy(name string) (*RetryPolicy, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	retry, exists := rm.retryPolicies[name]
	return retry, exists
}

// GetStats returns statistics for all circuit breakers
func (rm *ResilienceManager) GetStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := make(map[string]interface{})

	for name, cb := range rm.circuitBreakers {
		counts := cb.Counts()
		stats[name] = map[string]interface{}{
			"state":                  cb.State().String(),
			"requests":               counts.Requests,
			"total_successes":        counts.TotalSuccesses,
			"total_failures":         counts.TotalFailures,
			"consecutive_successes":  counts.ConsecutiveSuccesses,
			"consecutive_failures":   counts.ConsecutiveFailures,
		}
	}

	return stats
}

// HealthCheck performs health check on all circuit breakers
func (rm *ResilienceManager) HealthCheck() map[string]bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	health := make(map[string]bool)

	for name, cb := range rm.circuitBreakers {
		health[name] = cb.State() != StateOpen
	}

	return health
}
