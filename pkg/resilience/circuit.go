package resilience

import (
	"sync"
	"time"
)

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"    // Normal operation
	CircuitOpen     CircuitState = "open"      // Tripped, failing fast
	CircuitHalfOpen CircuitState = "half_open" // Testing recovery
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitState
	failures     int
	threshold    int
	lastFailure  time.Time
	cooldown     time.Duration
	successCount int
}

type CircuitManager struct {
	mu       sync.RWMutex
	circuits map[string]*CircuitBreaker
}

var GlobalCircuits = &CircuitManager{
	circuits: make(map[string]*CircuitBreaker),
}

func (cm *CircuitManager) Get(providerID string) *CircuitBreaker {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cb, ok := cm.circuits[providerID]
	if !ok {
		cb = &CircuitBreaker{
			state:     CircuitClosed,
			threshold: 5,
			cooldown:  30 * time.Second,
		}
		cm.circuits[providerID] = cb
	}
	return cb
}

func (cm *CircuitManager) Reset(providerID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if providerID == "" {
		cm.circuits = make(map[string]*CircuitBreaker)
		return
	}
	delete(cm.circuits, providerID)
}

func (cm *CircuitManager) ResetAll() {
	cm.Reset("")
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	if cb.state == CircuitOpen {
		if now.Sub(cb.lastFailure) > cb.cooldown {
			cb.state = CircuitHalfOpen
			return true // Allow one probe request
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= 2 {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successCount = 0
		}
	} else if cb.state == CircuitClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen || cb.failures >= cb.threshold {
		cb.state = CircuitOpen
		cb.successCount = 0
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
