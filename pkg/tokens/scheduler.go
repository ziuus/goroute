package tokens

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/providers/antigravity"
	"github.com/ziuus/goroute/pkg/providers/kiro"
)

type AccountHealth struct {
	ProviderID string    `json:"provider_id"`
	AccountID  string    `json:"account_id"`
	Status     string    `json:"status"` // "healthy", "refreshing", "expired", "error"
	LastCheck  time.Time `json:"last_check"`
	LastError  string    `json:"last_error,omitempty"`
	LatencyMs  int64     `json:"latency_ms"`
}

type Scheduler struct {
	mu          sync.RWMutex
	interval    time.Duration
	stopCh      chan struct{}
	running     bool
	healthState map[string]*AccountHealth
	router      ProviderRegistry
}

type ProviderRegistry interface {
	GetProvider(id string) (providers.Provider, bool)
	GetAllProviders() []providers.Provider
}

var GlobalScheduler *Scheduler

func InitScheduler(registry ProviderRegistry, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 3 * time.Minute
	}
	s := &Scheduler{
		interval:    interval,
		stopCh:      make(chan struct{}),
		healthState: make(map[string]*AccountHealth),
		router:      registry,
	}
	GlobalScheduler = s
	s.Start()
	return s
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("[tokens] Background proactive token refresh scheduler started (interval: %v)", s.interval)
	go s.loop()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

func (s *Scheduler) loop() {
	// Run initial probe 2 seconds after startup
	select {
	case <-time.After(2 * time.Second):
		s.RunHealthCheck(context.Background())
	case <-s.stopCh:
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.RunHealthCheck(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) RunHealthCheck(ctx context.Context) {
	if s.router == nil {
		return
	}

	provs := s.router.GetAllProviders()
	for _, p := range provs {
		switch prov := p.(type) {
		case *kiro.Provider:
			s.checkKiroProvider(ctx, prov)
		case *antigravity.Provider:
			s.checkAntigravityProvider(ctx, prov)
		}
	}
}

func (s *Scheduler) checkKiroProvider(ctx context.Context, p *kiro.Provider) {
	for _, acc := range p.Accounts() {
		key := fmt.Sprintf("%s:%s", p.ID(), acc.ID)
		start := time.Now()

		s.mu.Lock()
		rec, ok := s.healthState[key]
		if !ok {
			rec = &AccountHealth{
				ProviderID: p.ID(),
				AccountID:  acc.ID,
			}
			s.healthState[key] = rec
		}
		s.mu.Unlock()

		err := p.TestAccount(ctx, acc.ID)
		elapsed := time.Since(start).Milliseconds()

		s.mu.Lock()
		rec.LastCheck = time.Now()
		rec.LatencyMs = elapsed
		if err != nil {
			rec.Status = "error"
			rec.LastError = err.Error()
			log.Printf("[tokens] Proactive check failed for Kiro account %s: %v", acc.ID, err)
		} else {
			rec.Status = "healthy"
			rec.LastError = ""
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) checkAntigravityProvider(ctx context.Context, p *antigravity.Provider) {
	for _, acc := range p.Accounts() {
		key := fmt.Sprintf("%s:%s", p.ID(), acc.ID)
		start := time.Now()

		s.mu.Lock()
		rec, ok := s.healthState[key]
		if !ok {
			rec = &AccountHealth{
				ProviderID: p.ID(),
				AccountID:  acc.ID,
			}
			s.healthState[key] = rec
		}
		s.mu.Unlock()

		err := p.TestAccount(ctx, acc.ID)
		elapsed := time.Since(start).Milliseconds()

		s.mu.Lock()
		rec.LastCheck = time.Now()
		rec.LatencyMs = elapsed
		if err != nil {
			rec.Status = "error"
			rec.LastError = err.Error()
			log.Printf("[tokens] Proactive check failed for Antigravity account %s: %v", acc.ID, err)
		} else {
			rec.Status = "healthy"
			rec.LastError = ""
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) ListHealth() []*AccountHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*AccountHealth
	for _, v := range s.healthState {
		list = append(list, v)
	}
	return list
}
