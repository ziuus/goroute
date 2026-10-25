package router

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ziuus/goroute/pkg/fast"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/resilience"
	"github.com/ziuus/goroute/pkg/translator"
)

type Target struct {
	ProviderID   string  `json:"provider_id"`
	Model        string  `json:"model"`
	QuotaPercent float64 `json:"quota_percent,omitempty"`
}

type Combo struct {
	Name     string   `json:"name"`
	Strategy string   `json:"strategy"` // "failover", "round_robin", "lowest_latency", "quota_weighted"
	Targets  []Target `json:"targets"`
	counter  uint64
}

type TargetMetrics struct {
	EWMALatencyMs float64   `json:"ewma_latency_ms"`
	SampleCount   int64     `json:"sample_count"`
	LastLatencyMs int64     `json:"last_latency_ms"`
	LastSuccessAt time.Time `json:"last_success_at"`
	QuotaPercent  float64   `json:"quota_percent"`
}

type Router struct {
	mu        sync.RWMutex
	providers map[string]providers.Provider
	combos    map[string]*Combo

	metricsMu sync.RWMutex
	metrics   map[string]*TargetMetrics
}

func NewRouter() *Router {
	return &Router{
		providers: make(map[string]providers.Provider),
		combos:    make(map[string]*Combo),
		metrics:   make(map[string]*TargetMetrics),
	}
}

func (r *Router) RegisterProvider(p providers.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

func (r *Router) RegisterCombo(name, strategy string, targets []Target) {
	r.mu.Lock()
	r.combos[name] = &Combo{
		Name:     name,
		Strategy: strategy,
		Targets:  targets,
	}
	r.mu.Unlock()

	for _, t := range targets {
		if t.QuotaPercent > 0 {
			r.SetTargetQuota(t.ProviderID, t.Model, t.QuotaPercent)
		}
	}
}

func (r *Router) Clear() {
	r.mu.Lock()
	r.providers = make(map[string]providers.Provider)
	r.combos = make(map[string]*Combo)
	r.mu.Unlock()

	r.metricsMu.Lock()
	r.metrics = make(map[string]*TargetMetrics)
	r.metricsMu.Unlock()
}

func (r *Router) GetProvider(id string) (providers.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

func (r *Router) GetAllProviders() []providers.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []providers.Provider
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}

func (r *Router) GetCombos() map[string]*Combo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make(map[string]*Combo, len(r.combos))
	for k, v := range r.combos {
		res[k] = v
	}
	return res
}

func targetKey(providerID, model string) string {
	return providerID + "/" + model
}

func (r *Router) RecordTargetLatency(providerID, model string, latencyMs int64) {
	key := targetKey(providerID, model)
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()

	m, ok := r.metrics[key]
	if !ok {
		m = &TargetMetrics{
			QuotaPercent: 100.0,
		}
		r.metrics[key] = m
	}

	const alpha = 0.25
	if m.SampleCount == 0 {
		m.EWMALatencyMs = float64(latencyMs)
	} else {
		m.EWMALatencyMs = alpha*float64(latencyMs) + (1.0-alpha)*m.EWMALatencyMs
	}
	m.SampleCount++
	m.LastLatencyMs = latencyMs
	m.LastSuccessAt = time.Now()
}

func (r *Router) SetTargetQuota(providerID, model string, percent float64) {
	key := targetKey(providerID, model)
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()

	m, ok := r.metrics[key]
	if !ok {
		m = &TargetMetrics{
			QuotaPercent: percent,
		}
		r.metrics[key] = m
		return
	}
	m.QuotaPercent = percent
}

func (r *Router) GetTargetMetrics(providerID, model string) TargetMetrics {
	key := targetKey(providerID, model)
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	if m, ok := r.metrics[key]; ok {
		return *m
	}
	return TargetMetrics{
		QuotaPercent: 100.0,
	}
}

func (r *Router) GetAllMetrics() map[string]TargetMetrics {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	res := make(map[string]TargetMetrics, len(r.metrics))
	for k, v := range r.metrics {
		res[k] = *v
	}
	return res
}

func (r *Router) ResolveTarget(model string) (Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if strings.HasPrefix(model, "gr/") {
		model = strings.TrimPrefix(model, "gr/")
	} else if strings.HasPrefix(model, "goroute/") {
		model = strings.TrimPrefix(model, "goroute/")
	}
	if strings.HasPrefix(model, "agy/") {
		model = "antigravity/" + strings.TrimPrefix(model, "agy/")
	}

	if combo, ok := r.combos[model]; ok && len(combo.Targets) > 0 {
		ordered := r.orderTargets(combo)
		for _, t := range ordered {
			if !resilience.GlobalCooldown.IsCooling(t.ProviderID, t.Model) {
				return t, true
			}
		}
		// If all are cooling down, return first target per strategy order
		return ordered[0], true
	}

	parts := strings.SplitN(model, "/", 2)
	if len(parts) == 2 {
		return Target{ProviderID: parts[0], Model: parts[1]}, true
	}

	var activeProvs []string
	for pid := range r.providers {
		activeProvs = append(activeProvs, pid)
	}

	if res, ok := fast.ResolveModel(model, activeProvs); ok && res != nil {
		return Target{ProviderID: res.ProviderID, Model: res.TargetModel}, true
	}

	return Target{}, false
}

func (r *Router) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var models []providers.ModelInfo

	// 1. Add configured combos
	for name := range r.combos {
		if !seen[name] {
			seen[name] = true
			models = append(models, providers.ModelInfo{
				ID:      name,
				Object:  "model",
				OwnedBy: "goroute",
			})
		}
	}

	// 2. Add standard curated aliases and gateway virtual models
	standardAliases := []string{
		"gemini-flash",
		"gemini-flash-lite",
		"gemini-2.5-flash",
		"gemini-3.8-flash-high",
		"claude-sonnet-ag",
		"opus",
		"haiku",
		"sonnet",
		"claude-sonnet-4.5",
		"claude-opus-5.5",
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"gpt-4o",
		"gpt-4o-mini",
	}
	for _, alias := range standardAliases {
		if !seen[alias] {
			seen[alias] = true
			models = append(models, providers.ModelInfo{
				ID:      alias,
				Object:  "model",
				OwnedBy: "goroute",
			})
		}
	}

	// 3. Add provider direct models (e.g. antigravity/gemini-2.5-flash, kiro/claude-3-7-sonnet)
	for _, p := range r.providers {
		pCtx, pCancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		pModels, err := p.Models(pCtx)
		pCancel()
		if err == nil {
			for _, m := range pModels {
				fullID := fmt.Sprintf("%s/%s", p.ID(), m.ID)
				if !seen[fullID] {
					seen[fullID] = true
					models = append(models, providers.ModelInfo{
						ID:      fullID,
						Object:  "model",
						OwnedBy: p.ID(),
					})
				}
			}
		}
	}

	return models, nil
}

// orderTargets reorders combo targets according to strategy
func (r *Router) orderTargets(combo *Combo) []Target {
	targets := make([]Target, len(combo.Targets))
	copy(targets, combo.Targets)

	switch combo.Strategy {
	case "round_robin":
		if len(targets) > 1 {
			offset := atomic.AddUint64(&combo.counter, 1) % uint64(len(targets))
			rotated := make([]Target, len(targets))
			for i := range targets {
				rotated[i] = targets[(int(offset)+i)%len(targets)]
			}
			return rotated
		}
	case "lowest_latency":
		// Sort targets by EWMA latency ascending
		// Cold-start targets (0 samples) are given an exploratory benchmark 5.0ms
		sort.SliceStable(targets, func(i, j int) bool {
			mi := r.GetTargetMetrics(targets[i].ProviderID, targets[i].Model)
			mj := r.GetTargetMetrics(targets[j].ProviderID, targets[j].Model)

			latI := mi.EWMALatencyMs
			if mi.SampleCount == 0 {
				latI = 5.0 // exploration priority
			}
			latJ := mj.EWMALatencyMs
			if mj.SampleCount == 0 {
				latJ = 5.0
			}
			if latI != latJ {
				return latI < latJ
			}
			return targets[i].ProviderID < targets[j].ProviderID
		})
	case "quota_weighted":
		// Sort targets by QuotaPercent descending
		// Targets with higher remaining quota percentage are tried first
		sort.SliceStable(targets, func(i, j int) bool {
			mi := r.GetTargetMetrics(targets[i].ProviderID, targets[i].Model)
			mj := r.GetTargetMetrics(targets[j].ProviderID, targets[j].Model)

			if mi.QuotaPercent != mj.QuotaPercent {
				return mi.QuotaPercent > mj.QuotaPercent
			}
			// Secondary sort: lowest EWMA latency
			return mi.EWMALatencyMs < mj.EWMALatencyMs
		})
	default:
		// "failover" or unspecified: preserves sequential fallback
	}

	return targets
}

// Execute routes the request, executing automatic failover or load balancing
func (r *Router) Execute(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	model := req.Model
	if strings.HasPrefix(model, "gr/") {
		model = strings.TrimPrefix(model, "gr/")
	} else if strings.HasPrefix(model, "goroute/") {
		model = strings.TrimPrefix(model, "goroute/")
	}

	if strings.HasPrefix(model, "pool1/") {
		ctx = context.WithValue(ctx, "goroute-pool", "pool1")
		model = strings.TrimPrefix(model, "pool1/")
	} else if strings.HasPrefix(model, "pool2/") {
		ctx = context.WithValue(ctx, "goroute-pool", "pool2")
		model = strings.TrimPrefix(model, "pool2/")
	} else if strings.HasPrefix(model, "pool3/") {
		ctx = context.WithValue(ctx, "goroute-pool", "pool3")
		model = strings.TrimPrefix(model, "pool3/")
	}

	if strings.HasPrefix(model, "agy/") {
		model = "antigravity/" + strings.TrimPrefix(model, "agy/")
	}

	normReq := *req
	normReq.Model = model

	r.mu.RLock()
	combo, isCombo := r.combos[normReq.Model]
	r.mu.RUnlock()

	// 2. High-speed Rust Model & Family Resolver if not exact combo match
	if !isCombo && !strings.Contains(normReq.Model, "/") {
		r.mu.RLock()
		var activeProvs []string
		for pid := range r.providers {
			activeProvs = append(activeProvs, pid)
		}
		r.mu.RUnlock()

		if res, ok := fast.ResolveModel(normReq.Model, activeProvs); ok && res != nil && res.ProviderID != "" {
			var targets []Target
			targets = append(targets, Target{
				ProviderID: res.ProviderID,
				Model:      res.TargetModel,
			})
			for _, fb := range res.Fallbacks {
				targets = append(targets, Target{
					ProviderID: fb.ProviderID,
					Model:      fb.TargetModel,
				})
			}
			combo = &Combo{
				Name:     normReq.Model,
				Strategy: "failover",
				Targets:  targets,
			}
			isCombo = true
		} else {
			// Substring heuristic alias lookup across combos
			lower := strings.ToLower(normReq.Model)
			r.mu.RLock()
			for name, c := range r.combos {
				cLower := strings.ToLower(name)
				if strings.Contains(lower, "opus") && strings.Contains(cLower, "opus") {
					combo = c
					isCombo = true
					break
				}
				if strings.Contains(lower, "haiku") && strings.Contains(cLower, "haiku") {
					combo = c
					isCombo = true
					break
				}
				if (strings.Contains(lower, "sonnet") || strings.Contains(lower, "claude")) && (strings.Contains(cLower, "sonnet") || strings.Contains(cLower, "claude")) {
					combo = c
					isCombo = true
					break
				}
				if strings.Contains(lower, "gemini") && strings.Contains(cLower, "gemini") {
					combo = c
					isCombo = true
					break
				}
			}
			r.mu.RUnlock()
		}
	}

	if isCombo && len(combo.Targets) > 0 {
		return r.executeCombo(ctx, combo, &normReq)
	}

	// Direct model or provider/model lookup
	return r.executeDirect(ctx, &normReq)
}

func (r *Router) executeCombo(ctx context.Context, combo *Combo, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	targets := r.orderTargets(combo)

	var lastErr error
	var lastStatus int

	for _, target := range targets {
		r.mu.RLock()
		prov, ok := r.providers[target.ProviderID]
		r.mu.RUnlock()

		if !ok || !prov.IsHealthy() {
			continue
		}

		if resilience.GlobalCooldown.IsCooling(target.ProviderID, target.Model) {
			log.Printf("[GoRoute] Skipping %s/%s (currently cooling down)", target.ProviderID, target.Model)
			continue
		}

		// Pre-emptive Quota Fallback (OmniRoute Parity)
		// Skip targets automatically if remaining quota drops below 5% to avoid upstream 429
		targetMetrics := r.GetTargetMetrics(target.ProviderID, target.Model)
		if targetMetrics.QuotaPercent > 0.0 && targetMetrics.QuotaPercent < 5.0 {
			log.Printf("[GoRoute] Pre-emptive Quota Fallback: Skipping %s/%s (remaining quota %.1f%% < 5%% limit)", target.ProviderID, target.Model, targetMetrics.QuotaPercent)
			continue
		}

		// Clone request with target model and sanitize parameters for provider/model
		targetReq := *req
		targetReq.Model = target.Model
		translator.SanitizeChatRequest(prov.ID(), prov.Type(), &targetReq)

		start := time.Now()
		body, isStream, statusCode, err := prov.ChatCompletion(ctx, &targetReq)
		elapsedMs := time.Since(start).Milliseconds()

		if err != nil {
			log.Printf("[GoRoute] Failover: target %s/%s returned error: %v (status %d), cooling down...", target.ProviderID, target.Model, err, statusCode)
			resilience.GlobalCooldown.MarkCooldown(target.ProviderID, target.Model, statusCode, err.Error(), 30*time.Second)
			lastErr = err
			if statusCode != 0 {
				lastStatus = statusCode
			}
			continue
		}

		// If success (2xx or 3xx), record latency telemetry and return immediately
		if statusCode >= 200 && statusCode < 400 {
			r.RecordTargetLatency(target.ProviderID, target.Model, elapsedMs)
			return body, isStream, statusCode, nil
		}

		// Upstream target failed (4xx or 5xx): close body, mark cooldown, and cascade!
		var errReason string
		if body != nil {
			errBytes, _ := io.ReadAll(io.LimitReader(body, 2048))
			errReason = string(errBytes)
			body.Close()
		}
		if errReason == "" {
			errReason = fmt.Sprintf("HTTP %d", statusCode)
		}
		log.Printf("[GoRoute] Failover: target %s/%s returned HTTP %d (%s), cooling down and trying next target...", target.ProviderID, target.Model, statusCode, errReason)

		// For 429 quota exhaustion or 400 schema mismatches, mark cooldown and try next target
		cooldownDuration := 10 * time.Minute
		if statusCode == 400 {
			cooldownDuration = 3 * time.Minute
		} else if statusCode == 429 && resilience.IsDailyQuotaExhausted(errReason) {
			cooldownDuration = 1 * time.Hour
		}
		resilience.GlobalCooldown.MarkCooldown(target.ProviderID, target.Model, statusCode, errReason, cooldownDuration)
		lastStatus = statusCode
		lastErr = fmt.Errorf("target %s/%s returned HTTP %d: %s", target.ProviderID, target.Model, statusCode, errReason)
		// Continue cascade loop to next target!
		continue
	}

	if lastErr != nil {
		return nil, false, lastStatus, lastErr
	}
	if lastStatus != 0 {
		return nil, false, lastStatus, fmt.Errorf("all combo targets failed, last status: %d", lastStatus)
	}

	return nil, false, http.StatusServiceUnavailable, fmt.Errorf("no available targets for combo %s", combo.Name)
}

// rankProvidersForModel returns healthy providers sorted deterministically by model affinity
func (r *Router) rankProvidersForModel(model string) []providers.Provider {
	lower := strings.ToLower(model)
	var list []providers.Provider
	for _, p := range r.providers {
		if p.IsHealthy() {
			list = append(list, p)
		}
	}

	scoreProvider := func(p providers.Provider) int {
		pType := strings.ToLower(p.Type())
		pID := strings.ToLower(p.ID())

		// Claude / Anthropic affinity
		if strings.Contains(lower, "claude") || strings.Contains(lower, "anthropic") || strings.Contains(lower, "haiku") || strings.Contains(lower, "sonnet") || strings.Contains(lower, "opus") {
			if pType == "anthropic" || pID == "anthropic" {
				return 100
			}
			if pType == "kiro" || pID == "kiro" {
				return 90
			}
			if pType == "antigravity" || pID == "antigravity" {
				return 85
			}
		}

		// Gemini affinity
		if strings.Contains(lower, "gemini") {
			if pType == "antigravity" || pID == "antigravity" {
				return 100
			}
			if pType == "google" || pID == "gemini" {
				return 90
			}
		}

		// OpenAI / GPT / Codex affinity
		if strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "o3") || strings.Contains(lower, "codex") {
			if pType == "codex" || pID == "codex" {
				return 100
			}
			if pType == "openai" {
				return 90
			}
		}

		// DeepSeek affinity
		if strings.Contains(lower, "deepseek") {
			if strings.Contains(pID, "deepseek") {
				return 100
			}
			if pType == "kiro" || pID == "kiro" {
				return 80
			}
		}

		return 10
	}

	sort.SliceStable(list, func(i, j int) bool {
		scoreI := scoreProvider(list[i])
		scoreJ := scoreProvider(list[j])
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return list[i].ID() < list[j].ID()
	})

	return list
}

func (r *Router) executeDirect(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if model contains provider prefix, e.g. "antigravity/claude-3-7-sonnet"
	parts := strings.SplitN(req.Model, "/", 2)
	if len(parts) == 2 {
		provID := parts[0]
		modelName := parts[1]
		if prov, ok := r.providers[provID]; ok {
			targetReq := *req
			targetReq.Model = modelName
			translator.SanitizeChatRequest(prov.ID(), prov.Type(), &targetReq)
			start := time.Now()
			body, isStream, statusCode, err := prov.ChatCompletion(ctx, &targetReq)
			if err == nil && statusCode >= 200 && statusCode < 400 {
				r.RecordTargetLatency(provID, modelName, time.Since(start).Milliseconds())
			}
			return body, isStream, statusCode, err
		}
	}

	// Deterministic model-affinity failover across healthy, non-cooling providers
	var lastErr error
	var lastStatus int
	candidates := r.rankProvidersForModel(req.Model)
	for _, p := range candidates {
		if resilience.GlobalCooldown.IsCooling(p.ID(), req.Model) {
			continue
		}
		targetReq := *req
		translator.SanitizeChatRequest(p.ID(), p.Type(), &targetReq)
		start := time.Now()
		body, isStream, statusCode, err := p.ChatCompletion(ctx, &targetReq)
		if err == nil && statusCode >= 200 && statusCode < 400 {
			r.RecordTargetLatency(p.ID(), req.Model, time.Since(start).Milliseconds())
			return body, isStream, statusCode, nil
		}
		log.Printf("[GoRoute] executeDirect: provider %s failed with status %d (err: %v), cooling down and trying next...", p.ID(), statusCode, err)
		resilience.GlobalCooldown.MarkCooldown(p.ID(), req.Model, statusCode, "executeDirect failover", 1*time.Minute)
		lastErr = err
		lastStatus = statusCode
		if body != nil {
			body.Close()
		}
	}

	if lastErr != nil {
		return nil, false, lastStatus, lastErr
	}
	return nil, false, http.StatusNotFound, fmt.Errorf("no provider found for model: %s", req.Model)
}

func (r *Router) ExecuteEmbeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if model contains provider prefix, e.g. "openai/text-embedding-3-small"
	if strings.Contains(req.Model, "/") {
		parts := strings.SplitN(req.Model, "/", 2)
		provID := parts[0]
		modelName := parts[1]
		if prov, ok := r.providers[provID]; ok && prov.IsHealthy() {
			targetReq := *req
			targetReq.Model = modelName
			return prov.Embeddings(ctx, &targetReq)
		}
	}

	// Try each healthy provider
	var lastErr error
	for _, prov := range r.providers {
		if !prov.IsHealthy() {
			continue
		}
		resp, err := prov.Embeddings(ctx, req)
		if err == nil && resp != nil {
			return resp, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no provider available for embeddings")
}
