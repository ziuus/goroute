package translator

import (
	"strings"

	"github.com/ziuus/goroute/pkg/providers"
)

// NormalizeReasoningEffort ensures reasoning_effort conforms to standard tiers: minimal, low, medium, high
func NormalizeReasoningEffort(effort string) string {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "max", "xhigh":
		return "high"
	case "minimal", "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(effort))
	default:
		return "medium"
	}
}

// BudgetToEffort converts a numeric token budget to standard reasoning_effort
func BudgetToEffort(budget int) string {
	if budget >= 16000 {
		return "high"
	}
	if budget >= 4000 {
		return "medium"
	}
	return "low"
}

// EffortToBudget converts standard reasoning_effort to a token budget
func EffortToBudget(effort string) int {
	switch strings.ToLower(effort) {
	case "high", "xhigh", "max":
		return 16000
	case "medium":
		return 4000
	case "low":
		return 2048
	case "minimal":
		return 1024
	default:
		return 4000
	}
}

// TranslateThinking normalizes reasoning parameters across OpenAI, Claude, and Gemini
func TranslateThinking(providerID, providerType string, req *providers.ChatRequest) {
	if req == nil {
		return
	}

	modelLower := strings.ToLower(req.Model)

	// 1. If client sent Anthropic thinking object: thinking: { type: "enabled", budget_tokens: N }
	if req.Thinking != nil {
		if thMap, ok := req.Thinking.(map[string]interface{}); ok {
			if bVal, hasBudget := thMap["budget_tokens"]; hasBudget {
				var budget int
				switch v := bVal.(type) {
				case float64:
					budget = int(v)
				case int:
					budget = v
				}
				if budget > 0 && req.ReasoningEffort == "" {
					req.ReasoningEffort = BudgetToEffort(budget)
				}
			}
		}
	}

	// 2. If client sent OpenAI reasoning_effort, translate for Anthropic / Gemini
	if req.ReasoningEffort != "" {
		req.ReasoningEffort = NormalizeReasoningEffort(req.ReasoningEffort)

		// For models requiring thinking parameters (Claude 3.7 Thinking, Claude 4.6 Thinking)
		if strings.Contains(modelLower, "thinking") || strings.Contains(modelLower, "sonnet-3-7") || strings.Contains(modelLower, "claude-3-7") {
			budget := EffortToBudget(req.ReasoningEffort)
			req.Thinking = map[string]interface{}{
				"type":          "enabled",
				"budget_tokens": budget,
			}
			// When thinking is active on Anthropic models, temperature must be 1.0 or omitted
			req.Temperature = nil
		}

		// Reasoning models (o1, o3, o3-mini) reject custom temperature
		if strings.HasPrefix(modelLower, "o1") || strings.HasPrefix(modelLower, "o3") {
			req.Temperature = nil
		}
	}
}
