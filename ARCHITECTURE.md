# ARCHITECTURE.md — GoRoute Engine Design

## High-Level System Architecture

GoRoute is split into two co-operative layers designed for maximal throughput, minimal memory usage, and zero runtime bloat:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        AGENT CLIENTS & TOOLS                           │
│     Claude Code CLI • OpenCode • Codex CLI • Aider • OpenClaw • Cursor │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ HTTP / SSE
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                      GOROUTE GO GATEWAY (PORT 8128)                    │
│                                                                        │
│  ┌─────────────────────────┐            ┌───────────────────────────┐  │
│  │   /v1/chat/completions  │            │        /v1/messages       │  │
│  │   (OpenAI Protocol)     │            │   (Anthropic Protocol)    │  │
│  └────────────┬────────────┘            └─────────────┬─────────────┘  │
│               │                                       │                │
│               └───────────────────┬───────────────────┘                │
│                                   │                                    │
│  ┌────────────────────────────────▼─────────────────────────────────┐  │
│  │                     ROUTER & COMBO ENGINE                        │  │
│  │   • Exact Combo Match (/home/zius/.goroute/config.yaml)          │  │
│  │   • EWMA Latency Tracker & Cooldown Circuit                      │  │
│  │   • SQLite WAL Audit Logging (pkg/storage/db.go)                 │  │
│  └────────────────────────────────┬─────────────────────────────────┘  │
└───────────────────────────────────┼────────────────────────────────────┘
                                    │ Purego C-ABI (No CGO)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                     RUST ACCELERATOR (goroute-fast)                    │
│                                                                        │
│  ┌────────────────────────┐  ┌────────────────┐  ┌──────────────────┐  │
│  │ SIMD Model Resolver    │  │ PII Redactor   │  │ Caveman Engine   │  │
│  │ (Fuzzy Family Match)   │  │ (Secret Mask)  │  │ (Token Compress) │  │
│  └────────────────────────┘  └────────────────┘  └──────────────────┘  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
       ┌────────────────────────────┼────────────────────────────┐
       ▼                            ▼                            ▼
┌──────────────┐             ┌──────────────┐             ┌──────────────┐
│ Antigravity  │             │  AWS Q Kiro  │             │ OpenRouter / │
│ Google OAuth │             │ Social OAuth │             │ Custom APIs  │
└──────────────┘             └──────────────┘             └──────────────┘
```

---

## Key Subsystems

### 1. Dual Protocol Normalization (`pkg/proxy/anthropic.go`)
- **OpenAI -> Anthropic Stream Transformation**: Incoming Anthropic `/v1/messages` requests are mapped to universal `providers.ChatRequest`. Upstream OpenAI-format SSE streams are translated on the fly into Anthropic SSE events (`message_start`, `content_block_start`, `content_block_delta`, `message_delta`, `message_stop`).
- **Tool Calling AST Mapping**: Bidirectionally translates Anthropic `<tool_use>` / `<tool_result>` blocks to OpenAI `tool_calls` schema and handles JSON argument deltas seamlessly.

### 2. Rust Acceleration Bridge (`pkg/fast/rust.go`)
- Uses `github.com/ebitengine/purego` for dynamic loading of `libgoroute_fast.so` at runtime.
- **Zero-CGO compilation**: The Go project builds instantly without `gcc` or `CGO_ENABLED=1`.
- **Functions exported from Rust**:
  - `goroute_rust_version`: Version and capability signature.
  - `goroute_rust_resolve_model`: High-speed SIMD regex matching that classifies models into families (`claude`, `openai`, `gemini`, `deepseek`) and binds them to active providers.
  - `goroute_rust_sanitize_pii`: Regex scanner that redacts leaked API tokens and OAuth credentials.
  - `goroute_rust_compress`: Multi-tier Caveman prompt compressor protecting code blocks and inline code literals.
  - `goroute_rust_count_tokens`: Calibrated sub-microsecond token estimator.

### 3. Resilience & Cascade Strategy (`pkg/router/router.go`)
- **Strategies Supported**:
  - `failover`: Sequential cascade upon HTTP 429, 503, or connection timeouts.
  - `lowest_latency`: EWMA-weighted routing favoring the fastest responsive upstream.
  - `quota_weighted`: Account rotation respecting 5-hour rolling session quotas.
- **Dynamic Cooldown**: When an account returns 429 or auth failure, it enters exponential cooldown and triggers proactive token refresh.

### 4. Storage Architecture (`pkg/storage/`)
- SQLite database configured with `WAL` (Write-Ahead Logging), `busy_timeout=5000`, and `synchronous=NORMAL`.
- Stores:
  - `call_logs`: Full request/response telemetry, latencies, tokens, and errors.
  - `combos`: Model combo configurations.
  - `radar_catalog`: Synced model catalog across all upstream providers.
  - `conductor_tasks`: Agent task queue and orchestration history.
