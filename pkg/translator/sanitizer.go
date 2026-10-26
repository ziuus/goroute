package translator

import (
	"regexp"
	"strings"

	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/providers/antigravity"
)

var (
	gpt5NoTempRegex      = regexp.MustCompile(`(?i)^gpt-(?:5\.(?:[4-9]|[1-9]\d+)|6(?:\.\d+)?)(?:-|$)`)
	reasoningModelRegex  = regexp.MustCompile(`(?i)^(o1|o3|o3-mini|o4|gpt-5)`)
	claudeOpusDeprecated = regexp.MustCompile(`(?i)claude-opus-4`)
)

// SanitizeChatRequest applies model- and provider-specific parameter filters,
// preventing upstream HTTP 400 Bad Request errors when clients like Claude Code,
// Codex, or Cursor send parameters unsupported by the target provider.
func SanitizeChatRequest(providerID, providerType string, req *providers.ChatRequest) {
	if req == nil {
		return
	}

	// Translate reasoning_effort and thinking parameters (OmniRoute parity: providers/xai/thinking.ts)
	TranslateThinking(providerID, providerType, req)

	modelLower := strings.ToLower(req.Model)

	// 1. Temperature & Sampling rules
	if reasoningModelRegex.MatchString(modelLower) || providerID == "codex" || gpt5NoTempRegex.MatchString(modelLower) {
		req.Temperature = nil
	} else if claudeOpusDeprecated.MatchString(modelLower) {
		req.Temperature = nil
	} else if providerID == "copilot" && gpt5NoTempRegex.MatchString(modelLower) {
		req.Temperature = nil
	} else if req.Temperature != nil {
		// Clamp temperature between 0.0 and 2.0
		if *req.Temperature < 0.0 {
			zero := 0.0
			req.Temperature = &zero
		} else if *req.Temperature > 2.0 {
			two := 2.0
			req.Temperature = &two
		}
	}

	// 2. MaxTokens sanity checks
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		req.MaxTokens = nil
	}

	// 3. Tools sanitation
	if len(req.Tools) == 0 {
		req.Tools = nil
		req.ToolChoice = nil
	} else {
		// Dedup tool names and sanitize schemas
		var sanitizedTools []interface{}
		seenNames := make(map[string]bool)

		for _, tool := range req.Tools {
			m, ok := tool.(map[string]interface{})
			if !ok {
				sanitizedTools = append(sanitizedTools, tool)
				continue
			}

			// Handle OpenAI standard {"type": "function", "function": {"name": ..., "parameters": ...}}
			if fn, ok := m["function"].(map[string]interface{}); ok {
				name, _ := fn["name"].(string)
				cleanName := strings.TrimSpace(name)
				if cleanName == "" || seenNames[cleanName] {
					continue
				}
				seenNames[cleanName] = true

				if providerType == "antigravity" || strings.Contains(modelLower, "gemini") {
					if schema, exists := fn["parameters"]; exists {
						fn["parameters"] = antigravity.CleanJSONSchemaForGemini(schema)
					}
					fn["name"] = antigravity.SanitizeGeminiToolName(cleanName)
				}
				sanitizedTools = append(sanitizedTools, m)
				continue
			}

			// Handle Anthropic / Direct {"name": ..., "input_schema": ...}
			name, _ := m["name"].(string)
			cleanName := strings.TrimSpace(name)
			if cleanName == "" || seenNames[cleanName] {
				continue
			}
			seenNames[cleanName] = true

			if providerType == "antigravity" || strings.Contains(modelLower, "gemini") {
				if schema, exists := m["input_schema"]; exists {
					m["input_schema"] = antigravity.CleanJSONSchemaForGemini(schema)
				}
				m["name"] = antigravity.SanitizeGeminiToolName(cleanName)
			}
			sanitizedTools = append(sanitizedTools, m)
		}

		if len(sanitizedTools) > 0 {
			req.Tools = sanitizedTools
		} else {
			req.Tools = nil
			req.ToolChoice = nil
		}
	}
}
