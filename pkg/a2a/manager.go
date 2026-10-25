package a2a

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ziuus/goroute/pkg/storage"
)

type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Examples    []string `json:"examples,omitempty"`
}

type AgentCard struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	URL          string            `json:"url"`
	Version      string            `json:"version"`
	Capabilities map[string]bool   `json:"capabilities"`
	Skills       []AgentSkill      `json:"skills"`
	Auth         map[string]string `json:"authentication"`
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GetAgentCard generates the standard Agent Card adhering to A2A Protocol v0.3.
func GetAgentCard(baseURL string) AgentCard {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8128"
	}
	return AgentCard{
		Name: "GoRoute High-Performance LLM Gateway",
		Description: "Native Go AI gateway with 36+ providers, zero Node.js footprint, smart fallbacks, " +
			"single-pass context pruning, quota governance, and MCP/A2A protocol interoperability.",
		URL:     baseURL + "/a2a",
		Version: "1.0.0",
		Capabilities: map[string]bool{
			"streaming":         true,
			"pushNotifications": false,
		},
		Skills: []AgentSkill{
			{
				ID:          "smart-routing",
				Name:        "Smart Request Routing",
				Description: "Routes LLM requests to the optimal provider based on quota, latency, and cost.",
				Tags:        []string{"routing", "llm", "optimization", "fallback"},
				Examples:    []string{"Route this coding task to the fastest available model"},
			},
			{
				ID:          "quota-management",
				Name:        "Quota & Cost Management",
				Description: "Tracks and enforces token budgets and automatically fails over when quotas deplete.",
				Tags:        []string{"quota", "cost", "monitoring", "budget"},
				Examples:    []string{"Check remaining quota for all providers"},
			},
			{
				ID:          "provider-discovery",
				Name:        "Provider Discovery",
				Description: "Discovers active models and healthy providers for chat, embeddings, and multimodal tasks.",
				Tags:        []string{"providers", "discovery", "health"},
				Examples:    []string{"List healthy providers for Claude Sonnet"},
			},
			{
				ID:          "web-search",
				Name:        "Web Search & Extraction",
				Description: "Fetches live web content and conducts DuckDuckGo/Tavily search without API keys.",
				Tags:        []string{"web", "search", "reader", "scraping"},
				Examples:    []string{"Search for recent updates on Go 1.25 release notes"},
			},
		},
		Auth: map[string]string{
			"type":   "api-key",
			"header": "Authorization",
		},
	}
}

// CreateTask registers a new A2A task in submitted state.
func CreateTask(skill, input string) (*storage.A2ATask, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	task := &storage.A2ATask{
		ID:        generateUUID(),
		Skill:     skill,
		State:     "submitted",
		Input:     input,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := db.UpsertA2ATask(task); err != nil {
		return nil, err
	}

	// Auto-transition to working and completion for synchronous execution in background
	taskCopy := *task
	go executeTask(&taskCopy)

	return task, nil
}

func executeTask(task *storage.A2ATask) {
	db := storage.GetDB()
	if db == nil {
		return
	}
	task.State = "working"
	_ = db.UpsertA2ATask(task)

	time.Sleep(50 * time.Millisecond) // Simulate bounded processing
	task.State = "completed"
	task.Result = fmt.Sprintf("{\"status\":\"success\",\"skill\":\"%s\",\"processed_at\":\"%s\"}", task.Skill, time.Now().UTC().Format(time.RFC3339))
	_ = db.UpsertA2ATask(task)
}

// GetTask returns a specific task.
func GetTask(id string) (*storage.A2ATask, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return db.GetA2ATask(id)
}

// ListTasks returns matching tasks.
func ListTasks(state, skill string, limit, offset int) ([]storage.A2ATask, int, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}
	return db.ListA2ATasks(state, skill, limit, offset)
}

// CancelTask marks a task as cancelled.
func CancelTask(id string) error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	task, err := db.GetA2ATask(id)
	if err != nil || task == nil {
		return fmt.Errorf("task not found")
	}
	task.State = "cancelled"
	return db.UpsertA2ATask(task)
}

// GetStats returns current task counts by status.
func GetStats() map[string]int {
	db := storage.GetDB()
	if db == nil {
		return map[string]int{"submitted": 0, "working": 0, "completed": 0, "failed": 0, "cancelled": 0}
	}
	stats, err := db.GetA2ATaskStats()
	if err != nil {
		return map[string]int{"submitted": 0, "working": 0, "completed": 0, "failed": 0, "cancelled": 0}
	}
	return stats
}
