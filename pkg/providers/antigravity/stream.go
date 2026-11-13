package antigravity

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type geminiStreamChunk struct {
	Response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text             string              `json:"text"`
					ReasoningContent string              `json:"thought"`
					FunctionCall     *geminiFunctionCall `json:"functionCall,omitempty"`
					ThoughtSignature string              `json:"thoughtSignature,omitempty"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	} `json:"response"`
}

type openAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content          string `json:"content,omitempty"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// TranslateGeminiSSEToOpenAI wraps a raw Gemini SSE stream and emits standard OpenAI SSE lines
func TranslateGeminiSSEToOpenAI(rawStream io.ReadCloser, modelName string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer rawStream.Close()
		defer pw.Close()

		scanner := bufio.NewScanner(rawStream)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 16*1024*1024)

		now := time.Now().Unix()
		toolCallIndex := 0
		seenCallIDs := make(map[string]int)
		var lastGeminiFinishReason string

		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			if !strings.HasPrefix(trimmed, "data:") {
				continue
			}

			payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}

			var gChunk geminiStreamChunk
			if err := json.Unmarshal([]byte(payload), &gChunk); err != nil {
				continue
			}

			if len(gChunk.Response.Candidates) == 0 {
				continue
			}

			candidate := gChunk.Response.Candidates[0]
			if candidate.FinishReason != "" {
				lastGeminiFinishReason = candidate.FinishReason
			}
			var latestSig string
			for _, part := range candidate.Content.Parts {
				if part.ThoughtSignature != "" {
					latestSig = part.ThoughtSignature
				}
				if part.FunctionCall != nil {
					callID := part.FunctionCall.ID
					if callID == "" {
						callID = fmt.Sprintf("toolu_%d", time.Now().UnixNano())
					}
					sig := part.ThoughtSignature
					if sig == "" {
						sig = latestSig
					}
					if sig != "" {
						StoreThoughtSignature(callID, sig)
					}

					curIdx, exists := seenCallIDs[callID]
					if !exists {
						curIdx = toolCallIndex
						seenCallIDs[callID] = curIdx
						toolCallIndex++
					}

					argsBytes, _ := json.Marshal(part.FunctionCall.Args)
					toolCallChunk := map[string]interface{}{
						"id":      "chatcmpl-goroute",
						"object":  "chat.completion.chunk",
						"created": now,
						"model":   modelName,
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"delta": map[string]interface{}{
									"tool_calls": []map[string]interface{}{
										{
											"index": curIdx,
											"id":    callID,
											"type":  "function",
											"function": map[string]string{
												"name":      part.FunctionCall.Name,
												"arguments": string(argsBytes),
											},
										},
									},
								},
								"finish_reason": "tool_calls",
							},
						},
					}
					data, _ := json.Marshal(toolCallChunk)
					fmt.Fprintf(pw, "data: %s\n\n", data)
					continue
				}

				if part.Text == "" && part.ReasoningContent == "" {
					continue
				}

				chunk := openAIStreamChunk{
					ID:      "chatcmpl-goroute",
					Object:  "chat.completion.chunk",
					Created: now,
					Model:   modelName,
					Choices: make([]struct {
						Index int `json:"index"`
						Delta struct {
							Content          string `json:"content,omitempty"`
							ReasoningContent string `json:"reasoning_content,omitempty"`
						} `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					}, 1),
				}
				chunk.Choices[0].Index = 0
				chunk.Choices[0].Delta.Content = part.Text
				chunk.Choices[0].Delta.ReasoningContent = part.ReasoningContent

				data, _ := json.Marshal(chunk)
				fmt.Fprintf(pw, "data: %s\n\n", data)
			}
		}

		// Emit final finish_reason chunk required by OpenAI spec before [DONE]
		stopReason := "stop"
		if toolCallIndex > 0 {
			stopReason = "tool_calls"
		} else if lastGeminiFinishReason != "" {
			switch strings.ToUpper(lastGeminiFinishReason) {
			case "MAX_TOKENS":
				stopReason = "length"
			case "SAFETY":
				stopReason = "content_filter"
			case "STOP":
				stopReason = "stop"
			}
		}
		finalChunk := openAIStreamChunk{
			ID:      "chatcmpl-goroute",
			Object:  "chat.completion.chunk",
			Created: now,
			Model:   modelName,
			Choices: make([]struct {
				Index int `json:"index"`
				Delta struct {
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}, 1),
		}
		finalChunk.Choices[0].Index = 0
		finalChunk.Choices[0].FinishReason = &stopReason
		if data, err := json.Marshal(finalChunk); err == nil {
			fmt.Fprintf(pw, "data: %s\n\n", data)
		}

		fmt.Fprint(pw, "data: [DONE]\n\n")
	}()

	return pr
}

// ConvertGeminiSSEToNonStreaming aggregates a Gemini SSE stream into a standard OpenAI non-streaming JSON
func ConvertGeminiSSEToNonStreaming(rawStream io.ReadCloser, modelName string) (io.ReadCloser, error) {
	defer rawStream.Close()

	scanner := bufio.NewScanner(rawStream)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 16*1024*1024)

	var contentBuilder strings.Builder
	var toolCalls []map[string]interface{}
	finishReason := "stop"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var gChunk geminiStreamChunk
		if err := json.Unmarshal([]byte(payload), &gChunk); err != nil {
			continue
		}

		if len(gChunk.Response.Candidates) == 0 {
			continue
		}

		candidate := gChunk.Response.Candidates[0]
		if candidate.FinishReason != "" {
			switch strings.ToUpper(candidate.FinishReason) {
			case "MAX_TOKENS":
				if finishReason != "tool_calls" {
					finishReason = "length"
				}
			case "SAFETY":
				if finishReason != "tool_calls" {
					finishReason = "content_filter"
				}
			}
		}
		var latestSig string
		for _, part := range candidate.Content.Parts {
			if part.ThoughtSignature != "" {
				latestSig = part.ThoughtSignature
			}
			if part.Text != "" {
				contentBuilder.WriteString(part.Text)
			}
			if part.FunctionCall != nil {
				finishReason = "tool_calls"
				callID := part.FunctionCall.ID
				if callID == "" {
					callID = fmt.Sprintf("toolu_%d", time.Now().UnixNano())
				}
				sig := part.ThoughtSignature
				if sig == "" {
					sig = latestSig
				}
				if sig != "" {
					StoreThoughtSignature(callID, sig)
				}
				argsBytes, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   callID,
					"type": "function",
					"function": map[string]string{
						"name":      part.FunctionCall.Name,
						"arguments": string(argsBytes),
					},
				})
			}
		}
	}

	fullText := contentBuilder.String()
	tokens := len(fullText) / 4
	if tokens == 0 && (len(fullText) > 0 || len(toolCalls) > 0) {
		tokens = 1
	}

	msgObj := map[string]interface{}{
		"role":    "assistant",
		"content": fullText,
	}
	if len(toolCalls) > 0 {
		msgObj["tool_calls"] = toolCalls
	}

	resp := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-goroute-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   modelName,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       msgObj,
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": tokens,
			"total_tokens":      10 + tokens,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}
