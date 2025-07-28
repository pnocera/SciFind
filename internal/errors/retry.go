package errors

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []ErrorType   `json:"retryable_errors"`
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalAttempts     int64   `json:"total_attempts"`
	SuccessfulRetries int64   `json:"successful_retries"`
	FailedRetries     int64   `json:"failed_retries"`
	AverageAttempts   float64 `json:"average_attempts"`
}

// RetryExecutor implements intelligent retry logic
type RetryExecutor struct {
	config     RetryConfig
	classifier *ErrorClassifier
	stats      RetryStats
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(config RetryConfig, classifier *ErrorClassifier, logger *slog.Logger) *RetryExecutor {
	return &RetryExecutor{
		config:     config,
		classifier: classifier,
		logger:     logger,
	}
}

// Execute executes a function with retry logic
func (re *RetryExecutor) Execute(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	attempts := 0
	
	re.mutex.Lock()
	re.stats.TotalAttempts++
	re.mutex.Unlock()
	
	for attempts < re.config.MaxAttempts {
		attempts++
		
		err := fn()
		if err == nil {
			if attempts > 1 {
				re.mutex.Lock()
				re.stats.SuccessfulRetries++
				re.updateAverageAttempts(float64(attempts))
				re.mutex.Unlock()
				
				re.logger.Info("Operation succeeded after retries",
					slog.String("operation", operation),
					slog.Int("attempts", attempts))
			}
			return nil
		}
		
		lastErr = err
		classifiedErr := re.classifier.Classify(err)
		
		if !re.shouldRetry(classifiedErr, attempts) {
			break
		}
		
		delay := re.calculateDelay(attempts, classifiedErr)
		
		re.logger.Warn("Operation failed, retrying",
			slog.String("operation", operation),
			slog.Int("attempt", attempts),
			slog.String("error", err.Error()),
			slog.Duration("delay", delay))
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue with retry
		}
	}
	
	re.mutex.Lock()
	re.stats.FailedRetries++
	re.updateAverageAttempts(float64(attempts))
	re.mutex.Unlock()
	
	re.logger.Error("Operation failed after all retries",
		slog.String("operation", operation),
		slog.Int("attempts", attempts),
		slog.String("final_error", lastErr.Error()))
	
	return NewError(ErrorTypePermanent, "RETRY_EXHAUSTED", fmt.Sprintf("Operation failed after %d attempts", attempts)).
		WithCause(lastErr).
		WithComponent("retry_executor").
		WithOperation(operation).
		WithDetail("attempts", attempts).
		WithDetail("max_attempts", re.config.MaxAttempts).
		Retryable(false).
		WithStack().
		Build()
}

// ExecuteWithCallback executes a function with retry logic and callbacks
func (re *RetryExecutor) ExecuteWithCallback(ctx context.Context, operation string, fn func() error, onRetry func(attempt int, err error)) error {
	var lastErr error
	attempts := 0
	
	re.mutex.Lock()
	re.stats.TotalAttempts++
	re.mutex.Unlock()
	
	for attempts < re.config.MaxAttempts {
		attempts++
		
		err := fn()
		if err == nil {
			if attempts > 1 {
				re.mutex.Lock()
				re.stats.SuccessfulRetries++
				re.updateAverageAttempts(float64(attempts))
				re.mutex.Unlock()
				
				re.logger.Info("Operation succeeded after retries",
					slog.String("operation", operation),
					slog.Int("attempts", attempts))
			}
			return nil
		}
		
		lastErr = err
		classifiedErr := re.classifier.Classify(err)
		
		if !re.shouldRetry(classifiedErr, attempts) {
			break
		}
		
		if onRetry != nil {
			onRetry(attempts, err)
		}
		
		delay := re.calculateDelay(attempts, classifiedErr)
		
		re.logger.Warn("Operation failed, retrying",
			slog.String("operation", operation),
			slog.Int("attempt", attempts),
			slog.String("error", err.Error()),
			slog.Duration("delay", delay))
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue with retry
		}
	}
	
	re.mutex.Lock()
	re.stats.FailedRetries++
	re.updateAverageAttempts(float64(attempts))
	re.mutex.Unlock()
	
	re.logger.Error("Operation failed after all retries",
		slog.String("operation", operation),
		slog.Int("attempts", attempts),
		slog.String("final_error", lastErr.Error()))
	
	return NewError(ErrorTypePermanent, "RETRY_EXHAUSTED", fmt.Sprintf("Operation failed after %d attempts", attempts)).
		WithCause(lastErr).
		WithComponent("retry_executor").
		WithOperation(operation).
		WithDetail("attempts", attempts).
		WithDetail("max_attempts", re.config.MaxAttempts).
		Retryable(false).
		WithStack().
		Build()
}

// shouldRetry determines if an error should be retried
func (re *RetryExecutor) shouldRetry(err *SciFindError, attempt int) bool {
	if err == nil {
		return false
	}
	
	if attempt >= re.config.MaxAttempts {
		return false
	}
	
	if !err.Retryable {
		return false
	}
	
	// Check if error type is retryable
	for _, retryableType := range re.config.RetryableErrors {
		if err.Type == retryableType {
			return true
		}
	}
	
	return false
}

// calculateDelay calculates the delay for the next retry attempt
func (re *RetryExecutor) calculateDelay(attempt int, err *SciFindError) time.Duration {
	baseDelay := re.config.InitialDelay
	
	// Exponential backoff
	delay := time.Duration(float64(baseDelay) * math.Pow(re.config.BackoffFactor, float64(attempt-1)))
	
	// Cap at maximum delay
	if delay > re.config.MaxDelay {
		delay = re.config.MaxDelay
	}
	
	// Special handling for rate limit errors
	if err != nil && err.Type == ErrorTypeRateLimit {
		if retryAfterStr, ok := err.Details["retry_after"].(string); ok {
			if retryAfter, parseErr := time.ParseDuration(retryAfterStr); parseErr == nil {
				delay = retryAfter
			}
		} else {
			// More aggressive backoff for rate limits
			delay = delay * 2
		}
	}
	
	// Add jitter to avoid thundering herd
	if re.config.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay += jitter
	}
	
	return delay
}

// updateAverageAttempts updates the average attempts metric
func (re *RetryExecutor) updateAverageAttempts(attempts float64) {
	totalOps := re.stats.SuccessfulRetries + re.stats.FailedRetries
	if totalOps > 0 {
		re.stats.AverageAttempts = (re.stats.AverageAttempts*float64(totalOps-1) + attempts) / float64(totalOps)
	} else {
		re.stats.AverageAttempts = attempts
	}
}

// GetStats returns current retry statistics
func (re *RetryExecutor) GetStats() RetryStats {
	re.mutex.RLock()
	defer re.mutex.RUnlock()
	return re.stats
}

// ResetStats resets the retry statistics
func (re *RetryExecutor) ResetStats() {
	re.mutex.Lock()
	defer re.mutex.Unlock()
	re.stats = RetryStats{}
}

// WithExponentialBackoff creates a retry config with exponential backoff
func WithExponentialBackoff(maxAttempts int, initialDelay, maxDelay time.Duration) RetryConfig {
	return RetryConfig{
		MaxAttempts:   maxAttempts,
		InitialDelay:  initialDelay,
		MaxDelay:      maxDelay,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrorTypeTransient,
			ErrorTypeTimeout,
			ErrorTypeNetwork,
			ErrorTypeRateLimit,
		},
	}
}

// WithLinearBackoff creates a retry config with linear backoff
func WithLinearBackoff(maxAttempts int, delay time.Duration) RetryConfig {
	return RetryConfig{
		MaxAttempts:   maxAttempts,
		InitialDelay:  delay,
		MaxDelay:      delay * time.Duration(maxAttempts),
		BackoffFactor: 1.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrorTypeTransient,
			ErrorTypeTimeout,
			ErrorTypeNetwork,
		},
	}
}

// WithFixedDelay creates a retry config with fixed delay
func WithFixedDelay(maxAttempts int, delay time.Duration) RetryConfig {
	return RetryConfig{
		MaxAttempts:   maxAttempts,
		InitialDelay:  delay,
		MaxDelay:      delay,
		BackoffFactor: 1.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{
			ErrorTypeTransient,
			ErrorTypeTimeout,
			ErrorTypeNetwork,
		},
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation struct {
	Name        string
	Operation   func() error
	ShouldRetry func(error) bool
	OnRetry     func(attempt int, err error)
}

// ExecuteRetryableOperation executes a retryable operation
func (re *RetryExecutor) ExecuteRetryableOperation(ctx context.Context, op RetryableOperation) error {
	customFn := func() error {
		return op.Operation()
	}
	
	if op.ShouldRetry != nil {
		// Override the default retry logic with custom logic
		return re.executeWithCustomRetryLogic(ctx, op.Name, customFn, op.ShouldRetry, op.OnRetry)
	}
	
	return re.ExecuteWithCallback(ctx, op.Name, customFn, op.OnRetry)
}

// executeWithCustomRetryLogic executes with custom retry logic
func (re *RetryExecutor) executeWithCustomRetryLogic(ctx context.Context, operation string, fn func() error, shouldRetry func(error) bool, onRetry func(attempt int, err error)) error {
	var lastErr error
	attempts := 0
	
	re.mutex.Lock()
	re.stats.TotalAttempts++
	re.mutex.Unlock()
	
	for attempts < re.config.MaxAttempts {
		attempts++
		
		err := fn()
		if err == nil {
			if attempts > 1 {
				re.mutex.Lock()
				re.stats.SuccessfulRetries++
				re.updateAverageAttempts(float64(attempts))
				re.mutex.Unlock()
				
				re.logger.Info("Operation succeeded after retries",
					slog.String("operation", operation),
					slog.Int("attempts", attempts))
			}
			return nil
		}
		
		lastErr = err
		
		if !shouldRetry(err) {
			break
		}
		
		if onRetry != nil {
			onRetry(attempts, err)
		}
		
		classifiedErr := re.classifier.Classify(err)
		delay := re.calculateDelay(attempts, classifiedErr)
		
		re.logger.Warn("Operation failed, retrying",
			slog.String("operation", operation),
			slog.Int("attempt", attempts),
			slog.String("error", err.Error()),
			slog.Duration("delay", delay))
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue with retry
		}
	}
	
	re.mutex.Lock()
	re.stats.FailedRetries++
	re.updateAverageAttempts(float64(attempts))
	re.mutex.Unlock()
	
	re.logger.Error("Operation failed after all retries",
		slog.String("operation", operation),
		slog.Int("attempts", attempts),
		slog.String("final_error", lastErr.Error()))
	
	return NewError(ErrorTypePermanent, "RETRY_EXHAUSTED", fmt.Sprintf("Operation failed after %d attempts", attempts)).
		WithCause(lastErr).
		WithComponent("retry_executor").
		WithOperation(operation).
		WithDetail("attempts", attempts).
		WithDetail("max_attempts", re.config.MaxAttempts).
		Retryable(false).
		WithStack().
		Build()
}