package resilience

import (
	"regexp"
	"strings"
	"time"
)

type FailureType string

const (
	FailureAuth           FailureType = "authentication_error"
	FailureRateLimit      FailureType = "rate_limit"
	FailureQuotaExhausted FailureType = "quota_exhausted"
	FailureTimeout        FailureType = "timeout"
	FailureNetwork        FailureType = "network_error"
	FailureProvider5xx    FailureType = "provider_5xx"
	FailureInvalidRequest FailureType = "invalid_request"
	FailureModelNotFound  FailureType = "model_unavailable"
	FailurePermission     FailureType = "permission_error"
	FailureUnknown        FailureType = "unknown"
)

type ProviderFailure struct {
	Type         FailureType   `json:"type"`
	Retryable    bool          `json:"retryable"`
	ProviderID   string        `json:"provider_id"`
	StatusCode   int           `json:"status_code"`
	RetryAfter   time.Duration `json:"retry_after,omitempty"`
	Message      string        `json:"message"`
	LockoutDelay time.Duration `json:"lockout_delay"`
}

var (
	authRegex    = regexp.MustCompile(`(?i)(invalid.*(key|token)|unauthori[sz]ed|forbidden|bad credentials)`)
	quotaRegex   = regexp.MustCompile(`(?i)(quota|insufficient balance|credits exhausted|balance is \$0|resource_exhausted|daily quota)`)
	rateRegex    = regexp.MustCompile(`(?i)(rate.?limit|too many requests|retry.?after|429)`)
	timeoutRegex = regexp.MustCompile(`(?i)(timeout|timed out|etimedout|context deadline exceeded)`)
)

// ClassifyFailure implements OmniRoute failureClassification.ts
func ClassifyFailure(providerID string, statusCode int, message string, retryAfterSec int) ProviderFailure {
	msg := strings.TrimSpace(message)
	lower := strings.ToLower(msg)

	pf := ProviderFailure{
		ProviderID:   providerID,
		StatusCode:   statusCode,
		Message:      msg,
		Type:         FailureUnknown,
		Retryable:    false,
		LockoutDelay: 30 * time.Second,
	}

	if retryAfterSec > 0 {
		pf.RetryAfter = time.Duration(retryAfterSec) * time.Second
		pf.LockoutDelay = pf.RetryAfter
	}

	if statusCode == 401 || statusCode == 403 || authRegex.MatchString(lower) {
		if statusCode == 403 || strings.Contains(lower, "permission") {
			pf.Type = FailurePermission
		} else {
			pf.Type = FailureAuth
		}
		pf.Retryable = false
		pf.LockoutDelay = 30 * time.Minute // Auth failures lock out account from pool
	} else if statusCode == 408 || statusCode == 504 || timeoutRegex.MatchString(lower) {
		pf.Type = FailureTimeout
		pf.Retryable = true
		pf.LockoutDelay = 15 * time.Second
	} else if statusCode == 429 || rateRegex.MatchString(lower) {
		if quotaRegex.MatchString(lower) {
			pf.Type = FailureQuotaExhausted
			pf.Retryable = false
			pf.LockoutDelay = 1 * time.Hour // Quota exhaustion locks out for 1 hour
		} else {
			pf.Type = FailureRateLimit
			pf.Retryable = true
			if pf.LockoutDelay <= 30*time.Second {
				pf.LockoutDelay = 60 * time.Second
			}
		}
	} else if statusCode >= 500 {
		pf.Type = FailureProvider5xx
		pf.Retryable = true
		pf.LockoutDelay = 20 * time.Second
	} else if statusCode == 400 {
		pf.Type = FailureInvalidRequest
		pf.Retryable = false
		pf.LockoutDelay = 0
	} else if statusCode == 404 || strings.Contains(lower, "model") {
		pf.Type = FailureModelNotFound
		pf.Retryable = false
		pf.LockoutDelay = 10 * time.Minute
	}

	return pf
}
