// Package handlers - Retry Strategy Pattern Implementation
// SOLID OCP: Retry behavior is extensible through strategy pattern
// SOLID SRP: Separated retry logic from middleware concerns
package handlers

import (
	"context"
	"time"

	"github.com/flext/flexcore/shared/errors"
)

// RetryStrategy defines how retries should be executed
// SOLID DIP: High-level modules depend on abstractions
type RetryStrategy interface {
	ShouldRetry(attempt int, err error) bool
	CalculateDelay(attempt int) time.Duration
	MaxAttempts() int
}

// RetryResult represents the outcome of a retry operation
type RetryResult struct {
	Success     bool
	FinalError  error
	AttemptsMade int
	TotalDelay   time.Duration
}

// RetryExecutor handles the execution logic for retries
// SOLID SRP: Single responsibility for executing retry operations
type RetryExecutor struct {
	strategy RetryStrategy
}

// NewRetryExecutor creates a new retry executor with given strategy
func NewRetryExecutor(strategy RetryStrategy) *RetryExecutor {
	return &RetryExecutor{
		strategy: strategy,
	}
}

// ExecuteWithRetry executes a function with retry logic
// DRY PRINCIPLE: Eliminates retry logic duplication across different functions
func (re *RetryExecutor) ExecuteWithRetry(
	ctx context.Context,
	operation func() error,
) *RetryResult {
	var lastErr error
	startTime := time.Now()
	
	for attempt := 0; attempt < re.strategy.MaxAttempts(); attempt++ {
		// Apply delay before retry (except first attempt)
		if attempt > 0 {
			delay := re.strategy.CalculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return &RetryResult{
					Success:      false,
					FinalError:   ctx.Err(),
					AttemptsMade: attempt,
					TotalDelay:   time.Since(startTime),
				}
			}
		}
		
		// Execute operation
		err := operation()
		if err == nil {
			return &RetryResult{
				Success:      true,
				FinalError:   nil,
				AttemptsMade: attempt + 1,
				TotalDelay:   time.Since(startTime),
			}
		}
		
		lastErr = err
		
		// Check if we should retry
		if !re.strategy.ShouldRetry(attempt, err) {
			break
		}
	}
	
	return &RetryResult{
		Success:      false,
		FinalError:   lastErr,
		AttemptsMade: re.strategy.MaxAttempts(),
		TotalDelay:   time.Since(startTime),
	}
}

// StandardRetryStrategy implements a standard retry strategy
// SOLID SRP: Single responsibility for standard retry logic
type StandardRetryStrategy struct {
	maxRetries   int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
	backoffMultiplier float64
}

// NewStandardRetryStrategy creates a standard retry strategy
func NewStandardRetryStrategy(maxRetries int, baseBackoff time.Duration) *StandardRetryStrategy {
	return &StandardRetryStrategy{
		maxRetries:        maxRetries,
		baseBackoff:       baseBackoff,
		maxBackoff:        time.Minute * 5, // Reasonable maximum
		backoffMultiplier: 2.0,             // Exponential backoff
	}
}

// ShouldRetry determines if an error should trigger a retry
func (s *StandardRetryStrategy) ShouldRetry(attempt int, err error) bool {
	// Don't retry client errors (4xx status codes)
	if httpErr, ok := err.(*errors.FlexError); ok {
		switch httpErr.Code() {
		case errors.CodeValidation, errors.CodeUnauthorized, errors.CodeForbidden:
			return false
		}
	}
	
	return attempt < s.maxRetries
}

// CalculateDelay calculates exponential backoff delay
func (s *StandardRetryStrategy) CalculateDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	
	delay := time.Duration(float64(s.baseBackoff) * 
		(s.backoffMultiplier * float64(attempt)))
	
	if delay > s.maxBackoff {
		delay = s.maxBackoff
	}
	
	return delay
}

// MaxAttempts returns maximum number of attempts
func (s *StandardRetryStrategy) MaxAttempts() int {
	return s.maxRetries + 1 // +1 for initial attempt
}

// ExponentialRetryStrategy implements exponential backoff with jitter
// SOLID OCP: Open for extension - can add jitter, circuit breaker, etc.
type ExponentialRetryStrategy struct {
	*StandardRetryStrategy
	jitterEnabled bool
}

// NewExponentialRetryStrategy creates exponential retry strategy with jitter
func NewExponentialRetryStrategy(maxRetries int, baseBackoff time.Duration, jitter bool) *ExponentialRetryStrategy {
	return &ExponentialRetryStrategy{
		StandardRetryStrategy: NewStandardRetryStrategy(maxRetries, baseBackoff),
		jitterEnabled:         jitter,
	}
}

// CalculateDelay calculates delay with optional jitter
func (e *ExponentialRetryStrategy) CalculateDelay(attempt int) time.Duration {
	delay := e.StandardRetryStrategy.CalculateDelay(attempt)
	
	if e.jitterEnabled && delay > 0 {
		// Add up to 10% jitter to avoid thundering herd
		jitter := time.Duration(float64(delay) * 0.1)
		delay += time.Duration(time.Now().UnixNano() % int64(jitter))
	}
	
	return delay
}

// NoRetryStrategy implements a no-retry strategy for testing/debugging
type NoRetryStrategy struct{}

// NewNoRetryStrategy creates a strategy that never retries
func NewNoRetryStrategy() *NoRetryStrategy {
	return &NoRetryStrategy{}
}

// ShouldRetry always returns false
func (n *NoRetryStrategy) ShouldRetry(attempt int, err error) bool {
	return false
}

// CalculateDelay always returns 0
func (n *NoRetryStrategy) CalculateDelay(attempt int) time.Duration {
	return 0
}

// MaxAttempts returns 1 (no retries)
func (n *NoRetryStrategy) MaxAttempts() int {
	return 1
}