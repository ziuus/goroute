type schemaEdgeCaseProvider struct {
	id         string
	shouldFail bool
	failCode   int
	callCount  int
	lastReq    *providers.ChatRequest
}

func (m *schemaEdgeCaseProvider) ID() string   { return m.id }
func (m *schemaEdgeCaseProvider) Type() string { return "mock" }
func (m *schemaEdgeCaseProvider) IsHealthy() bool { return true }
func (m *schemaEdgeCaseProvider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "claude-3-7-sonnet", Object: "model"}}, nil
}
func (m *schemaEdgeCaseProvider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return nil, nil
}
func (m *schemaEdgeCaseProvider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	m.callCount++
	m.lastReq = req

	if m.shouldFail {
		return io.NopCloser(bytes.NewReader([]byte("rate limited"))), false, m.failCode, nil
	}

	// Simulate returning valid JSON but invalid schema data for email
	resp := map[string]interface{}{
		"id": "chatcmpl-test",
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role": "assistant",
					"tool_calls": []map[string]interface{}{
						{
							"id":   "call_abc123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "subscribe",
								"arguments": `{"email":"not-an-email"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}
	b, _ := json.Marshal(resp)
	return io.NopCloser(bytes.NewReader(b)), false, http.StatusOK, nil
}

func TestRouterFailoverSchemaValidationEdgeCase(t *testing.T) {
	primary := &schemaEdgeCaseProvider{id: "primaryEdge", shouldFail: true, failCode: http.StatusTooManyRequests} // 429
	backup := &schemaEdgeCaseProvider{id: "backupEdge", shouldFail: false}

	r := NewRouter()
	r.RegisterProvider(primary)
	r.RegisterProvider(backup)

	r.RegisterCombo("smart-model-edge", "failover", []Target{
		{ProviderID: "primaryEdge", Model: "claude-3-7-sonnet"},
		{ProviderID: "backupEdge", Model: "claude-3-7-sonnet"},
	})

	req := &providers.ChatRequest{
		Model: "smart-model-edge",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "subscribe me"},
		},
		Tools: []interface{}{
			map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name": "subscribe",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"email": map[string]interface{}{
								"type":   "string",
								"format": "email",
							},
						},
						"required": []string{"email"},
					},
				},
			},
		},
	}

	body, _, status, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	// Verify failover happened
	if primary.callCount != 1 || backup.callCount != 1 {
		t.Errorf("expected both providers called, primary=%d, backup=%d", primary.callCount, backup.callCount)
	}

	// Verify schema was preserved
	if backup.lastReq == nil || len(backup.lastReq.Tools) == 0 {
		t.Fatalf("expected tools to be passed to backup provider")
	}

	toolMap, ok := backup.lastReq.Tools[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tool to be a map")
	}
	funcMap, ok := toolMap["function"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected function to be a map")
	}
	paramsMap, ok := funcMap["parameters"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters to be a map")
	}
	propsMap, ok := paramsMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties to be a map")
	}
	emailProp, ok := propsMap["email"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected email property to be a map")
	}
	if emailProp["format"] != "email" {
		t.Errorf("expected email format to be preserved, got %v", emailProp["format"])
	}

	// Verify response from backup containing invalid email was successfully passed back
	resp, _ := io.ReadAll(body)
	respStr := string(resp)
	if !strings.Contains(respStr, `"not-an-email"`) && !strings.Contains(respStr, `\"not-an-email\"`) {
		t.Errorf("expected response to contain invalid email, got %s", respStr)
	}
}
