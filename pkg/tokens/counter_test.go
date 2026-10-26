package tokens

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	req := &CountTokensRequest{
		Model: "claude-3-7-sonnet",
		System: "You are a senior systems engineer.",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Please inspect this repository and find syntax bugs.",
			},
			map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Scanning repository files now.",
					},
					map[string]interface{}{
						"type": "tool_use",
						"id":   "toolu_01",
						"name": "bash",
						"input": map[string]interface{}{
							"command": "git status",
						},
					},
				},
			},
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_01",
						"content":     "On branch main\nnothing to commit, working tree clean",
					},
				},
			},
		},
		Tools: []interface{}{
			map[string]interface{}{
				"name":        "bash",
				"description": "Run shell commands",
			},
		},
	}

	toks := EstimateTokens(req)
	if toks < 40 || toks > 200 {
		t.Fatalf("expected estimated tokens between 40 and 200, got %d", toks)
	}
}
