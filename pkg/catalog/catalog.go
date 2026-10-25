package catalog

import (
	"strings"
)

type CatalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`        // "openai", "antigravity", "kiro"
	Category    string   `json:"category"`    // "frontier", "inference", "gateway", "cloud", "asia", "local", "oauth"
	BaseURL     string   `json:"base_url"`
	AuthType    string   `json:"auth_type"`   // "apikey", "oauth", "refresh_token", "none"
	Description string   `json:"description"`
	Models      []string `json:"models,omitempty"`
}

var Registry = []CatalogEntry{
	// ─── Frontier Labs ────────────────────────────────────────────────────────
	{
		ID:          "openai",
		Name:        "OpenAI",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.openai.com/v1",
		AuthType:    "apikey",
		Description: "Official OpenAI GPT-4o, o1, o3, o3-mini models",
		Models:      []string{"gpt-4o", "gpt-4o-mini", "o1", "o3-mini", "gpt-4-turbo"},
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic Direct",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.anthropic.com/v1",
		AuthType:    "apikey",
		Description: "Official Anthropic Claude 3.7 Sonnet, 3.5 Sonnet, 3.5 Haiku",
		Models:      []string{"claude-3-7-sonnet-20250219", "claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"},
	},
	{
		ID:          "gemini",
		Name:        "Google AI Studio (Gemini)",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta/openai",
		AuthType:    "apikey",
		Description: "Google Gemini 2.5 Pro / Flash OpenAI-compatible endpoint",
		Models:      []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash-exp"},
	},
	{
		ID:          "xai",
		Name:        "xAI (Grok)",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.x.ai/v1",
		AuthType:    "apikey",
		Description: "Grok 2 / Grok 3 reasoning and vision models",
		Models:      []string{"grok-2-1212", "grok-2-vision-1212", "grok-beta"},
	},
	{
		ID:          "deepseek",
		Name:        "DeepSeek",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.deepseek.com/v1",
		AuthType:    "apikey",
		Description: "DeepSeek V3 and DeepSeek R1 open-weights reasoning",
		Models:      []string{"deepseek-chat", "deepseek-reasoner"},
	},
	{
		ID:          "mistral",
		Name:        "Mistral AI",
		Type:        "mistral",
		Category:    "frontier",
		BaseURL:     "https://api.mistral.ai/v1",
		AuthType:    "apikey",
		Description: "Mistral Large, Codestral 2501, Pixtral Large, Ministral",
		Models:      []string{"mistral-large-latest", "codestral-latest", "pixtral-large-latest", "ministral-8b-latest"},
	},
	{
		ID:          "cohere",
		Name:        "Cohere",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.cohere.com/v2",
		AuthType:    "apikey",
		Description: "Command R+ and Command R enterprise models",
		Models:      []string{"command-r-plus-08-2024", "command-r-08-2024"},
	},
	{
		ID:          "perplexity",
		Name:        "Perplexity AI",
		Type:        "openai",
		Category:    "frontier",
		BaseURL:     "https://api.perplexity.ai",
		AuthType:    "apikey",
		Description: "Sonar Reasoning, Sonar Pro search-grounded LLMs",
		Models:      []string{"sonar-reasoning", "sonar-pro", "sonar"},
	},

	// ─── Free & Subscription Multi-Account Clouds ─────────────────────────────
	{
		ID:          "kiro",
		Name:        "AWS Kiro / CodeWhisperer",
		Type:        "kiro",
		Category:    "oauth",
		BaseURL:     "https://codewhisperer.us-east-1.amazonaws.com",
		AuthType:    "refresh_token",
		Description: "Claude Sonnet 4.5, Haiku 4.5 & DeepSeek via AWS Builder ID OIDC with 1-click import",
		Models:      []string{"claude-sonnet-4.5", "claude-haiku-4.5", "deepseek-3.2", "qwen3-coder-next"},
	},
	{
		ID:          "antigravity",
		Name:        "Google Antigravity (Cloud Code)",
		Type:        "antigravity",
		Category:    "oauth",
		BaseURL:     "https://cloudcode-pa.googleapis.com",
		AuthType:    "oauth",
		Description: "Multi-account Google Cloud Code Assist OAuth PKCE pool with quota failover",
		Models:      []string{"gemini-pro-agent", "gemini-2.5-flash", "gemini-3.8-flash-high", "gpt-oss-120b-medium"},
	},

	{
		ID:          "codex",
		Name:        "OpenAI Codex CLI",
		Type:        "codex",
		Category:    "oauth",
		BaseURL:     "https://chatgpt.com/backend-anon/sentinel/chat-requirements",
		AuthType:    "oauth",
		Description: "OpenAI Codex CLI OAuth authentication with 1-click import from ~/.codex/auth.json",
		Models:      []string{"gpt-5.6-sol", "gpt-5.6-sol-ultra", "gpt-5.6-sol-high", "gpt-5.6-sol-medium", "gpt-5.5-sol", "o3-mini", "o1", "gpt-4o"},
	},
	{
		ID:          "copilot",
		Name:        "GitHub Copilot",
		Type:        "copilot",
		Category:    "oauth",
		BaseURL:     "https://api.githubcopilot.com",
		AuthType:    "oauth",
		Description: "GitHub Copilot subscription OAuth integration with 1-click import from ~/.config/github-copilot",
		Models:      []string{"gpt-4o", "claude-3.5-sonnet", "o1-preview", "o1-mini"},
	},
	{
		ID:          "cursor",
		Name:        "Cursor Pro / IDE",
		Type:        "cursor",
		Category:    "oauth",
		BaseURL:     "https://api2.cursor.sh",
		AuthType:    "oauth",
		Description: "Cursor Pro subscription OAuth credentials with 1-click import from ~/.config/cursor/auth.json",
		Models:      []string{"auto", "composer-2", "claude-4.6-sonnet-medium", "claude-4.6-opus-high-thinking-fast", "gpt-5.4-low-fast", "gpt-5.3-codex", "gemini-3.1-pro", "grok-4.6-medium"},
	},
	{
		ID:          "grok-cli",
		Name:        "xAI Grok CLI",
		Type:        "openai",
		Category:    "oauth",
		BaseURL:     "https://cli-chat-proxy.grok.com/v1",
		AuthType:    "oauth",
		Description: "xAI Grok CLI OAuth client with Grok 4.6, 4.5, and Composer 2.5",
		Models:      []string{"grok-4.6", "grok-4.5", "grok-composer-2.5-fast"},
	},
	{
		ID:          "kimi",
		Name:        "Moonshot Kimi AI",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.moonshot.cn/v1",
		AuthType:    "apikey",
		Description: "Moonshot AI Kimi K3, K2.7 Code, and long-context 128k models",
		Models:      []string{"kimi-k3", "kimi-k2.7-code", "kimi-k2.5", "moonshot-v1-128k", "moonshot-v1-32k", "moonshot-v1-8k"},
	},

	// ─── Ultra-Fast LPUs & High-Speed Inference ──────────────────────────────
	{
		ID:          "groq",
		Name:        "Groq LPU",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.groq.com/openai/v1",
		AuthType:    "apikey",
		Description: "Ultra-low latency LPU inference (500+ tok/s) for Llama 3.3, DeepSeek, Gemma",
		Models:      []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "deepseek-r1-distill-llama-70b"},
	},
	{
		ID:          "cerebras",
		Name:        "Cerebras",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.cerebras.ai/v1",
		AuthType:    "apikey",
		Description: "Wafer-scale high-throughput inference (2,000+ tokens/sec)",
		Models:      []string{"llama3.3-70b", "llama3.1-8b"},
	},
	{
		ID:          "sambanova",
		Name:        "SambaNova",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.sambanova.ai/v1",
		AuthType:    "apikey",
		Description: "High-speed SN40L full-precision open models",
		Models:      []string{"Meta-Llama-3.3-70B-Instruct", "DeepSeek-R1-Distill-Llama-70B"},
	},
	{
		ID:          "together",
		Name:        "Together AI",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.together.xyz/v1",
		AuthType:    "apikey",
		Description: "Fast serverless inference engine for leading open-source models",
		Models:      []string{"meta-llama/Llama-3.3-70B-Instruct-Turbo", "deepseek-ai/DeepSeek-V3", "deepseek-ai/DeepSeek-R1"},
	},
	{
		ID:          "fireworks",
		Name:        "Fireworks AI",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.fireworks.ai/inference/v1",
		AuthType:    "apikey",
		Description: "Production-speed compound AI inference platform",
		Models:      []string{"accounts/fireworks/models/llama-v3p3-70b-instruct", "accounts/fireworks/models/deepseek-v3"},
	},
	{
		ID:          "nvidia",
		Name:        "NVIDIA NIM",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://integrate.api.nvidia.com/v1",
		AuthType:    "apikey",
		Description: "NVIDIA NIM microservices & hosted GPU models (Llama, DeepSeek, Nemotron)",
		Models:      []string{"meta/llama-3.3-70b-instruct", "deepseek-ai/deepseek-r1", "nvidia/llama-3.1-nemotron-70b-instruct"},
	},
	{
		ID:          "deepinfra",
		Name:        "DeepInfra",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.deepinfra.com/v1/openai",
		AuthType:    "apikey",
		Description: "Pay-as-you-go serverless inference for LLMs and embeddings",
		Models:      []string{"meta-llama/Meta-Llama-3.3-70B-Instruct", "deepseek-ai/DeepSeek-V3"},
	},
	{
		ID:          "hyperbolic",
		Name:        "Hyperbolic",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.hyperbolic.xyz/v1",
		AuthType:    "apikey",
		Description: "Decentralized GPU cluster open model inference",
		Models:      []string{"deepseek-ai/DeepSeek-V3", "meta-llama/Llama-3.3-70B-Instruct"},
	},
	{
		ID:          "nebius",
		Name:        "Nebius AI Studio",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.studio.nebius.ai/v1",
		AuthType:    "apikey",
		Description: "NVIDIA H100/H200 cloud inference with ultra-fast TTFT",
		Models:      []string{"meta-llama/Meta-Llama-3.3-70B-Instruct", "deepseek-ai/DeepSeek-R1"},
	},
	{
		ID:          "cloudflare",
		Name:        "Cloudflare Workers AI",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
		AuthType:    "apikey",
		Description: "Serverless edge GPU inference on Cloudflare global network",
		Models:      []string{"@cf/meta/llama-3.3-70b-instruct", "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b"},
	},
	{
		ID:          "replicate",
		Name:        "Replicate",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.replicate.com/v1",
		AuthType:    "apikey",
		Description: "Serverless cloud execution for open-source AI models",
		Models:      []string{"meta/meta-llama-3-70b-instruct"},
	},
	{
		ID:          "lambda",
		Name:        "Lambda Labs",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.lambdalabs.com/v1",
		AuthType:    "apikey",
		Description: "Low-cost high-performance GPU cloud inference",
		Models:      []string{"hermes-3-llama-3.1-405b"},
	},
	{
		ID:          "runpod",
		Name:        "RunPod Serverless",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.runpod.ai/v2/{endpoint_id}/openai/v1",
		AuthType:    "apikey",
		Description: "Custom serverless vLLM endpoints on RunPod GPUs",
		Models:      []string{"default"},
	},

	// ─── Multi-Model Gateways & Routers ──────────────────────────────────────
	{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://openrouter.ai/api/v1",
		AuthType:    "apikey",
		Description: "Universal gateway aggregating 250+ models with smart fallbacks",
		Models:      []string{"anthropic/claude-3.7-sonnet", "openai/gpt-4o", "deepseek/deepseek-r1"},
	},
	{
		ID:          "tokenrouter",
		Name:        "TokenRouter",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://api.tokenrouter.io/v1",
		AuthType:    "apikey",
		Description: "High-performance AI model routing gateway",
		Models:      []string{"gpt-4o", "claude-3-5-sonnet", "deepseek-chat"},
	},
	{
		ID:          "agentrouter",
		Name:        "AgentRouter",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://api.agentrouter.org/v1",
		AuthType:    "apikey",
		Description: "Dedicated proxy for Claude Code, Codex, and coding CLI agents",
		Models:      []string{"gpt-5.6-sol", "gpt-6-astra", "claude-3-7-sonnet"},
	},
	{
		ID:          "apinexbond",
		Name:        "ApinexBond",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://api.apinex.bond/v1",
		AuthType:    "apikey",
		Description: "Multi-provider AI endpoint bridge",
		Models:      []string{"gpt-4o", "claude-3-5-sonnet"},
	},
	{
		ID:          "portkey",
		Name:        "Portkey AI Gateway",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://api.portkey.ai/v1",
		AuthType:    "apikey",
		Description: "Enterprise AI gateway with rate-limiting, retries, and logging",
		Models:      []string{"gpt-4o"},
	},
	{
		ID:          "helicone",
		Name:        "Helicone Gateway",
		Type:        "openai",
		Category:    "gateway",
		BaseURL:     "https://oai.hconeai.com/v1",
		AuthType:    "apikey",
		Description: "Open-source LLM observability proxy and gateway",
		Models:      []string{"gpt-4o"},
	},

	// ─── Asia & Global Open Weights ──────────────────────────────────────────
	{
		ID:          "siliconflow",
		Name:        "SiliconFlow",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.siliconflow.cn/v1",
		AuthType:    "apikey",
		Description: "High-concurrency platform for Qwen, DeepSeek, and Llama",
		Models:      []string{"deepseek-ai/DeepSeek-V3", "deepseek-ai/DeepSeek-R1", "Qwen/Qwen2.5-72B-Instruct"},
	},
	{
		ID:          "dashscope",
		Name:        "Alibaba DashScope (Qwen)",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		AuthType:    "apikey",
		Description: "Official Alibaba Cloud Qwen 2.5, Qwen Coder, and QwQ reasoning models",
		Models:      []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen2.5-coder-32b-instruct", "qwq-32b-preview"},
	},
	{
		ID:          "zhipu",
		Name:        "Zhipu AI (GLM)",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
		AuthType:    "apikey",
		Description: "GLM-4 Plus and GLM-4 Flash frontier series",
		Models:      []string{"glm-4-plus", "glm-4-flash", "glm-4-long"},
	},
	{
		ID:          "moonshot",
		Name:        "Moonshot AI (Kimi)",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.moonshot.cn/v1",
		AuthType:    "apikey",
		Description: "Kimi long-context 128k/200k chat models",
		Models:      []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
	},
	{
		ID:          "yi",
		Name:        "01.AI (Yi)",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.lingyiwanwu.com/v1",
		AuthType:    "apikey",
		Description: "Yi-Lightning, Yi-Large, and Yi-Medium by Kai-Fu Lee's 01.AI",
		Models:      []string{"yi-lightning", "yi-large", "yi-medium"},
	},
	{
		ID:          "minimax",
		Name:        "MiniMax",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.minimax.chat/v1",
		AuthType:    "apikey",
		Description: "ABAB 6.5s and MiniMax Text-01 models",
		Models:      []string{"abab6.5s-chat", "MiniMax-Text-01"},
	},
	{
		ID:          "stepfun",
		Name:        "StepFun (Step)",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.stepfun.com/v1",
		AuthType:    "apikey",
		Description: "Step-2 and Step-1V large multimodal models",
		Models:      []string{"step-2-16k", "step-1-8k"},
	},
	{
		ID:          "novita",
		Name:        "Novita AI",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.novita.ai/v3/openai",
		AuthType:    "apikey",
		Description: "High-speed API hosting DeepSeek, Llama, and Qwen",
		Models:      []string{"deepseek/deepseek-r1", "meta-llama/llama-3.3-70b-instruct"},
	},
	{
		ID:          "upstage",
		Name:        "Upstage AI",
		Type:        "openai",
		Category:    "asia",
		BaseURL:     "https://api.upstage.ai/v1/solar",
		AuthType:    "apikey",
		Description: "Solar Mini and Solar Pro specialized models",
		Models:      []string{"solar-pro", "solar-mini"},
	},

	// ─── Local & Self-Hosted Engines ─────────────────────────────────────────
	{
		ID:          "ollama",
		Name:        "Ollama (Local)",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:11434/v1",
		AuthType:    "none",
		Description: "Zero-config local model runner on port 11434",
		Models:      []string{"llama3.3", "deepseek-r1", "qwen2.5-coder", "mistral"},
	},
	{
		ID:          "lmstudio",
		Name:        "LM Studio (Local)",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:1234/v1",
		AuthType:    "none",
		Description: "Desktop GUI with embedded OpenAI-compatible server on port 1234",
		Models:      []string{"loaded-model"},
	},
	{
		ID:          "vllm",
		Name:        "vLLM Engine",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:8000/v1",
		AuthType:    "none",
		Description: "Production PagedAttention high-throughput serving engine",
		Models:      []string{"default"},
	},
	{
		ID:          "llamacpp",
		Name:        "llama.cpp Server",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:8080/v1",
		AuthType:    "none",
		Description: "Ultra-lean native C++ GGUF inference server",
		Models:      []string{"default"},
	},
	{
		ID:          "localai",
		Name:        "LocalAI",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:8080/v1",
		AuthType:    "none",
		Description: "Drop-in self-hosted OpenAI alternative for CPU and GPU",
		Models:      []string{"default"},
	},
	{
		ID:          "jan",
		Name:        "Jan Desktop",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:1337/v1",
		AuthType:    "none",
		Description: "Open-source desktop LLM runner with local API on port 1337",
		Models:      []string{"default"},
	},
	{
		ID:          "sglang",
		Name:        "SGLang Server",
		Type:        "openai",
		Category:    "local",
		BaseURL:     "http://localhost:30000/v1",
		AuthType:    "none",
		Description: "Fast serving framework for complex LLM programs and reasoning models",
		Models:      []string{"default"},
	},

	// ─── Embeddings & Search Specialists ─────────────────────────────────────
	{
		ID:          "voyage",
		Name:        "Voyage AI",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.voyageai.com/v1",
		AuthType:    "apikey",
		Description: "State-of-the-art embedding and reranker models for RAG",
		Models:      []string{"voyage-3", "voyage-3-lite", "voyage-code-3"},
	},
	{
		ID:          "jina",
		Name:        "Jina AI",
		Type:        "openai",
		Category:    "inference",
		BaseURL:     "https://api.jina.ai/v1",
		AuthType:    "apikey",
		Description: "Search foundation models, embeddings, and rerankers",
		Models:      []string{"jina-embeddings-v3", "jina-reranker-v2-base-multilingual"},
	},
}

// FindEntry looks up a provider preset by ID or name
func FindEntry(query string) *CatalogEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, entry := range Registry {
		if strings.ToLower(entry.ID) == q || strings.ToLower(entry.Name) == q {
			return &entry
		}
	}
	for _, entry := range OmniRouteRegistry {
		if strings.ToLower(entry.ID) == q || strings.ToLower(entry.Name) == q {
			return &entry
		}
	}
	return nil
}

// Search returns entries matching a keyword
func Search(keyword string) []CatalogEntry {
	all := append(Registry, OmniRouteRegistry...)
	if keyword == "" {
		return all
	}
	q := strings.ToLower(keyword)
	var matches []CatalogEntry
	seen := make(map[string]bool)
	for _, entry := range all {
		if seen[entry.ID] {
			continue
		}
		if strings.Contains(strings.ToLower(entry.ID), q) ||
			strings.Contains(strings.ToLower(entry.Name), q) ||
			strings.Contains(strings.ToLower(entry.Description), q) ||
			strings.Contains(strings.ToLower(entry.Category), q) {
			seen[entry.ID] = true
			matches = append(matches, entry)
		}
	}
	return matches
}
