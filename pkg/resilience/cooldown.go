package resilience

import (
	"strings"
	"sync"
	"time"
)

type CooldownTarget struct {
	ProviderID   string    `json:"provider_id"`
	Model        string    `json:"model"`
	Reason       string    `json:"reason"`
	StatusCode   int       `json:"status_code"`
	CoolingUntil time.Time `json:"cooling_until"`
	Attempts     int       `json:"attempts"`
	Exhausted    bool      `json:"exhausted,omitempty"`
}

type CooldownManager struct {
	mu         sync.RWMutex
	cooldowns  map[string]*CooldownTarget
	defaultTTL time.Duration
}

var GlobalCooldown = NewCooldownManager(30 * time.Second)

func NewCooldownManager(defaultTTL time.Duration) *CooldownManager {
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Second
	}
	return &CooldownManager{
		cooldowns:  make(map[string]*CooldownTarget),
		defaultTTL: defaultTTL,
	}
}

// IsDailyQuotaExhausted checks error text for daily quota exhaustion signals
func IsDailyQuotaExhausted(text string) bool {
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.Contains(lower, "today's quota") ||
		strings.Contains(lower, "daily quota") ||
		strings.Contains(lower, "try again tomorrow") ||
		strings.Contains(lower, "quota_exhausted") ||
		strings.Contains(lower, "exceeded your current quota") ||
		strings.Contains(lower, "resource_exhausted") ||
		strings.Contains(lower, "usage limit reached")
}

func (cm *CooldownManager) MarkCooldown(providerID, model string, statusCode int, reason string, ttl time.Duration) {
	exhausted := IsDailyQuotaExhausted(reason)
	if exhausted {
		// Daily quota exhausted: lock out for 1 hour instead of transient 10m
		ttl = 1 * time.Hour
	} else if ttl <= 0 {
		ttl = cm.defaultTTL
	}
	key := providerID + "/" + model
	cm.mu.Lock()
	defer cm.mu.Unlock()

	target, exists := cm.cooldowns[key]
	if !exists {
		target = &CooldownTarget{
			ProviderID: providerID,
			Model:      model,
		}
		cm.cooldowns[key] = target
	}
	target.Reason = reason
	target.StatusCode = statusCode
	target.CoolingUntil = time.Now().Add(ttl)
	target.Attempts++
	target.Exhausted = exhausted
}

func (cm *CooldownManager) IsCooling(providerID, model string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check model level
	key := providerID + "/" + model
	if target, exists := cm.cooldowns[key]; exists {
		if time.Now().Before(target.CoolingUntil) {
			return true
		}
	}

	// Check provider level
	if target, exists := cm.cooldowns[providerID]; exists {
		if time.Now().Before(target.CoolingUntil) {
			return true
		}
	}

	return false
}

func (cm *CooldownManager) Reset(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if key == "" {
		cm.cooldowns = make(map[string]*CooldownTarget)
		return
	}
	delete(cm.cooldowns, key)
}

func (cm *CooldownManager) List() []*CooldownTarget {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var res []*CooldownTarget
	now := time.Now()
	for _, target := range cm.cooldowns {
		if now.Before(target.CoolingUntil) {
			res = append(res, target)
		}
	}
	return res
}
