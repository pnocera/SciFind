package errors

import (
	"log/slog"
	"sync"
	"time"
)

// CircuitBreakerState represents the current state
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// String returns the string representation of the state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half_open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration parameters
type CircuitBreakerConfig struct {
	Name                string        `json:"name"`
	FailureThreshold    int           `json:"failure_threshold"`    // Failures to trigger open
	SuccessThreshold    int           `json:"success_threshold"`    // Successes to close from half-open
	Timeout             time.Duration `json:"timeout"`              // Time to wait before half-open
	MaxRequests         int           `json:"max_requests"`         // Max requests in half-open
	ExpectedFailureRate float64       `json:"expected_failure_rate"` // Normal failure rate
	MinRequestCount     int           `json:"min_request_count"`    // Min requests before considering state change
	SlidingWindow       time.Duration `json:"sliding_window"`       // Window for failure rate calculation
}

// CircuitBreakerMetrics tracks operational metrics
type CircuitBreakerMetrics struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulReqs     int64   `json:"successful_requests"`
	FailedReqs         int64   `json:"failed_requests"`
	TimeoutReqs        int64   `json:"timeout_requests"`
	CircuitOpenReqs    int64   `json:"circuit_open_requests"`
	LastFailureTime    int64   `json:"last_failure_time"`
	LastSuccessTime    int64   `json:"last_success_time"`
	StateChanges       int64   `json:"state_changes"`
	CurrentFailureRate float64 `json:"current_failure_rate"`
}

// CircuitBreaker implements an intelligent circuit breaker pattern
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	metrics      CircuitBreakerMetrics
	failures     *RollingWindow
	mutex        sync.RWMutex
	stateChanged time.Time
	logger       *slog.Logger
	
	// Callbacks
	onStateChange func(from, to CircuitBreakerState)
}

// RollingWindow tracks failures over a time window
type RollingWindow struct {
	window   time.Duration
	buckets  []TimeBucket
	current  int
	mutex    sync.RWMutex
}

// TimeBucket represents a time bucket for tracking metrics
type TimeBucket struct {
	timestamp time.Time
	failures  int
	requests  int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	cb := &CircuitBreaker{
		config:       config,
		state:        StateClosed,
		failures:     NewRollingWindow(config.SlidingWindow),
		stateChanged: time.Now(),
		logger:       logger,
	}
	
	return cb
}

// Execute wraps a function call with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		cb.recordCircuitOpen()
		return NewCircuitBreakerError(cb.config.Name)
	}
	
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	cb.Record(err == nil, duration)
	return err
}

// Allow checks if requests should be allowed through
func (cb *CircuitBreaker) Allow() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return cb.shouldAttemptReset()
	case StateHalfOpen:
		return cb.canProcessHalfOpenRequest()
	default:
		return false
	}
}

// Record records the result of a request
func (cb *CircuitBreaker) Record(success bool, duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	now := time.Now()
	cb.metrics.TotalRequests++
	
	if success {
		cb.metrics.SuccessfulReqs++
		cb.metrics.LastSuccessTime = now.Unix()
		cb.onSuccess()
	} else {
		cb.metrics.FailedReqs++
		cb.metrics.LastFailureTime = now.Unix()
		cb.onFailure()
	}
	
	cb.failures.Record(!success)
	cb.updateFailureRate()
	cb.evaluateStateChange()
	
	cb.logger.Debug("Circuit breaker recorded result",
		slog.String("name", cb.config.Name),
		slog.Bool("success", success),
		slog.Duration("duration", duration),
		slog.String("state", cb.state.String()),
		slog.Float64("failure_rate", cb.metrics.CurrentFailureRate))
}

// recordCircuitOpen records when a request is rejected due to open circuit
func (cb *CircuitBreaker) recordCircuitOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.metrics.CircuitOpenReqs++
}

// shouldAttemptReset checks if we should attempt to reset the circuit breaker
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	return time.Since(cb.stateChanged) >= cb.config.Timeout
}

// canProcessHalfOpenRequest checks if we can process requests in half-open state
func (cb *CircuitBreaker) canProcessHalfOpenRequest() bool {
	return cb.metrics.TotalRequests < int64(cb.config.MaxRequests)
}

// onSuccess handles successful requests
func (cb *CircuitBreaker) onSuccess() {
	if cb.state == StateHalfOpen {
		successCount := cb.failures.GetSuccessCount()
		if successCount >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
		}
	}
}

// onFailure handles failed requests
func (cb *CircuitBreaker) onFailure() {
	if cb.state == StateHalfOpen {
		cb.setState(StateOpen)
	}
}

// updateFailureRate updates the current failure rate
func (cb *CircuitBreaker) updateFailureRate() {
	totalRequests := cb.failures.GetTotalCount()
	if totalRequests > 0 {
		failures := cb.failures.GetFailureCount()
		cb.metrics.CurrentFailureRate = float64(failures) / float64(totalRequests)
	}
}

// evaluateStateChange evaluates whether the state should change
func (cb *CircuitBreaker) evaluateStateChange() {
	if cb.state != StateClosed {
		return
	}
	
	totalRequests := cb.failures.GetTotalCount()
	if totalRequests < cb.config.MinRequestCount {
		return
	}
	
	if cb.metrics.CurrentFailureRate > cb.config.ExpectedFailureRate {
		failures := cb.failures.GetFailureCount()
		if failures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
		}
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	cb.stateChanged = time.Now()
	cb.metrics.StateChanges++
	
	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
	
	cb.logger.Info("Circuit breaker state changed",
		slog.String("name", cb.config.Name),
		slog.String("from", oldState.String()),
		slog.String("to", newState.String()),
		slog.Float64("failure_rate", cb.metrics.CurrentFailureRate))
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.metrics
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// SetOnStateChange sets the state change callback
func (cb *CircuitBreaker) SetOnStateChange(callback func(from, to CircuitBreakerState)) {
	cb.onStateChange = callback
}

// NewRollingWindow creates a new rolling window for tracking failures
func NewRollingWindow(window time.Duration) *RollingWindow {
	bucketCount := 10 // 10 buckets for granular tracking
	buckets := make([]TimeBucket, bucketCount)
	now := time.Now()
	
	for i := range buckets {
		buckets[i] = TimeBucket{
			timestamp: now.Add(-window + time.Duration(i)*window/time.Duration(bucketCount)),
		}
	}
	
	return &RollingWindow{
		window:  window,
		buckets: buckets,
	}
}

// Record records a success or failure
func (rw *RollingWindow) Record(isFailure bool) {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()
	
	now := time.Now()
	rw.evictOldBuckets(now)
	
	bucket := rw.getCurrentBucket(now)
	bucket.requests++
	if isFailure {
		bucket.failures++
	}
}

// GetFailureCount returns the number of failures in the current window
func (rw *RollingWindow) GetFailureCount() int {
	rw.mutex.RLock()
	defer rw.mutex.RUnlock()
	
	rw.evictOldBuckets(time.Now())
	
	failures := 0
	for _, bucket := range rw.buckets {
		failures += bucket.failures
	}
	return failures
}

// GetTotalCount returns the total number of requests in the current window
func (rw *RollingWindow) GetTotalCount() int {
	rw.mutex.RLock()
	defer rw.mutex.RUnlock()
	
	rw.evictOldBuckets(time.Now())
	
	total := 0
	for _, bucket := range rw.buckets {
		total += bucket.requests
	}
	return total
}

// GetSuccessCount returns the number of successes in the current window
func (rw *RollingWindow) GetSuccessCount() int {
	return rw.GetTotalCount() - rw.GetFailureCount()
}

// evictOldBuckets removes buckets that are outside the time window
func (rw *RollingWindow) evictOldBuckets(now time.Time) {
	cutoff := now.Add(-rw.window)
	for i := range rw.buckets {
		if rw.buckets[i].timestamp.Before(cutoff) {
			rw.buckets[i] = TimeBucket{timestamp: now}
		}
	}
}

// getCurrentBucket gets the current time bucket
func (rw *RollingWindow) getCurrentBucket(now time.Time) *TimeBucket {
	for i := range rw.buckets {
		if rw.buckets[i].timestamp.After(now.Add(-rw.window / time.Duration(len(rw.buckets)))) {
			return &rw.buckets[i]
		}
	}
	
	// Create new bucket if none found
	rw.buckets[rw.current] = TimeBucket{timestamp: now}
	bucket := &rw.buckets[rw.current]
	rw.current = (rw.current + 1) % len(rw.buckets)
	return bucket
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *slog.Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger *slog.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()
	
	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}
	
	config.Name = name
	cb := NewCircuitBreaker(config, cbm.logger)
	cbm.breakers[name] = cb
	
	return cb
}

// Get gets an existing circuit breaker
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	
	return result
}

// GetMetrics returns metrics for all circuit breakers
func (cbm *CircuitBreakerManager) GetMetrics() map[string]CircuitBreakerMetrics {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	result := make(map[string]CircuitBreakerMetrics)
	for name, cb := range cbm.breakers {
		result[name] = cb.GetMetrics()
	}
	
	return result
}