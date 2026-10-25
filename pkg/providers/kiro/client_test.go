package kiro

import (
	"testing"

	"github.com/ziuus/goroute/pkg/providers"
)

func TestBuildNormalizedTurns_ToolBijection(t *testing.T) {
	messages := []providers.ChatMessage{
		{
			Role:    "user",
			Content: "List files in directory",
		},
		{
			Role:    "assistant",
			Content: "Let me check the files.",
			ToolCalls: []interface{}{
				map[string]interface{}{
					"id":   "toolu_vrtx_01SUaDSYbt4ThG4gfwFRnwqS",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "Bash",
						"arguments": `{"command":"ls"}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			ToolCallID: "toolu_vrtx_01SUaDSYbt4ThG4gfwFRnwqS",
			Content:    "file1.txt\nfile2.txt",
		},
	}

	normalized := buildNormalizedTurns(messages)

	if len(normalized) != 3 {
		t.Fatalf("expected 3 normalized turns, got %d", len(normalized))
	}

	// Turn 0: User
	if normalized[0].Role != "user" || normalized[0].Text != "List files in directory" {
		t.Errorf("unexpected turn 0: %+v", normalized[0])
	}

	// Turn 1: Assistant with exactly 1 toolUse
	if normalized[1].Role != "assistant" {
		t.Errorf("unexpected turn 1 role: %s", normalized[1].Role)
	}
	if len(normalized[1].ToolUses) != 1 {
		t.Fatalf("expected 1 toolUse in assistant turn, got %d", len(normalized[1].ToolUses))
	}
	if normalized[1].ToolUses[0].ToolUseId != "toolu_vrtx_01SUaDSYbt4ThG4gfwFRnwqS" {
		t.Errorf("unexpected toolUseId: %s", normalized[1].ToolUses[0].ToolUseId)
	}

	// Turn 2: User with exactly 1 matching toolResult
	if normalized[2].Role != "user" {
		t.Errorf("unexpected turn 2 role: %s", normalized[2].Role)
	}
	if len(normalized[2].ToolResults) != 1 {
		t.Fatalf("expected 1 toolResult in user turn, got %d", len(normalized[2].ToolResults))
	}
	if normalized[2].ToolResults[0].ToolUseId != "toolu_vrtx_01SUaDSYbt4ThG4gfwFRnwqS" {
		t.Errorf("unexpected toolResult toolUseId: %s", normalized[2].ToolResults[0].ToolUseId)
	}
}

func TestBuildNormalizedTurns_UnansweredToolUseDemoted(t *testing.T) {
	// If assistant called a tool, but user interrupted or failed to return result,
	// the toolUse MUST be demoted to text so Bedrock doesn't fail with "Expected toolResult blocks".
	messages := []providers.ChatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Calling tool...",
			ToolCalls: []interface{}{
				map[string]interface{}{
					"id": "unanswered_tool_call_123",
					"function": map[string]interface{}{
						"name":      "Bash",
						"arguments": `{"command":"pwd"}`,
					},
				},
			},
		},
		{
			Role:    "user",
			Content: "Never mind, cancel that",
		},
	}

	normalized := buildNormalizedTurns(messages)

	if len(normalized) != 3 {
		t.Fatalf("expected 3 turns, got %d", len(normalized))
	}

	// Assistant turn must have 0 ToolUses, and text must include the demoted call
	assistTurn := normalized[1]
	if len(assistTurn.ToolUses) != 0 {
		t.Errorf("expected 0 toolUses for unanswered call, got %d", len(assistTurn.ToolUses))
	}
	if assistTurn.Text == "" {
		t.Errorf("expected assistant text to contain demoted tool context")
	}

	// User turn must have 0 ToolResults
	userTurn := normalized[2]
	if len(userTurn.ToolResults) != 0 {
		t.Errorf("expected 0 toolResults, got %d", len(userTurn.ToolResults))
	}
}

func TestBuildNormalizedTurns_OrphanToolResultDemoted(t *testing.T) {
	// If user sends a tool result that was never called by assistant,
	// it must be demoted to text so Bedrock doesn't fail with "number of toolResult blocks exceeds toolUse blocks".
	messages := []providers.ChatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Just talking, no tools called.",
		},
		{
			Role:       "tool",
			ToolCallID: "ghost_tool_999",
			Content:    "output from unknown tool",
		},
	}

	normalized := buildNormalizedTurns(messages)

	// User turn must have 0 ToolResults, and output demoted to text
	lastTurn := normalized[len(normalized)-1]
	if lastTurn.Role != "user" {
		t.Errorf("expected last turn to be user, got %s", lastTurn.Role)
	}
	if len(lastTurn.ToolResults) != 0 {
		t.Errorf("expected 0 toolResults for ghost tool, got %d", len(lastTurn.ToolResults))
	}
}

func TestBuildNormalizedTurns_ConsecutiveToolResultsCoalesced(t *testing.T) {
	messages := []providers.ChatMessage{
		{
			Role:    "user",
			Content: "Run tests and lint",
		},
		{
			Role:    "assistant",
			Content: "Running both in parallel.",
			ToolCalls: []interface{}{
				map[string]interface{}{
					"id": "tool_1",
					"function": map[string]interface{}{
						"name": "test",
					},
				},
				map[string]interface{}{
					"id": "tool_2",
					"function": map[string]interface{}{
						"name": "lint",
					},
				},
			},
		},
		{
			Role:       "tool",
			ToolCallID: "tool_1",
			Content:    "tests passed",
		},
		{
			Role:       "tool",
			ToolCallID: "tool_2",
			Content:    "lint clean",
		},
	}

	normalized := buildNormalizedTurns(messages)

	if len(normalized) != 3 {
		t.Fatalf("expected 3 turns (consecutive tools coalesced), got %d", len(normalized))
	}

	assistTurn := normalized[1]
	userTurn := normalized[2]

	if len(assistTurn.ToolUses) != 2 {
		t.Fatalf("expected 2 toolUses, got %d", len(assistTurn.ToolUses))
	}
	if len(userTurn.ToolResults) != 2 {
		t.Fatalf("expected 2 toolResults, got %d", len(userTurn.ToolResults))
	}
	if userTurn.ToolResults[0].ToolUseId != "tool_1" || userTurn.ToolResults[1].ToolUseId != "tool_2" {
		t.Errorf("mismatched tool result IDs in coalesced turn")
	}
}
