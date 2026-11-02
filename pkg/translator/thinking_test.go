package translator

import (
	"testing"

	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/storage"
)

func TestTranslateThinking_AdaptiveBudgetCap(t *testing.T) {
	// Ensure default adaptive config
	_ = storage.SetThinkingBudgetConfig(nil, storage.ThinkingBudgetConfig{
		Mode:         "adaptive",
		CustomBudget: 4000,
		EffortLevel:  "medium",
	})

	// 1. Client attempts to request 64,000 thinking tokens (token burn risk)
	req := &providers.ChatRequest{
		Model: "claude-3-7-sonnet",
		Thinking: map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": 64000,
		},
	}

	TranslateThinking("antigravity", "antigravity", req)

	// Verify budget was capped to 4000
	thMap, ok := req.Thinking.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Thinking to be map, got %T", req.Thinking)
	}
	budget := thMap["budget_tokens"]
	if budget != 4000 {
		t.Errorf("expected budget_tokens to be capped at 4000, got %v", budget)
	}

	// 2. Empty reasoning effort defaults to medium
	req2 := &providers.ChatRequest{
		Model: "claude-3-7-sonnet",
	}
	TranslateThinking("antigravity", "antigravity", req2)
	if req2.ReasoningEffort != "medium" {
		t.Errorf("expected empty reasoning_effort to default to medium, got %q", req2.ReasoningEffort)
	}
}
