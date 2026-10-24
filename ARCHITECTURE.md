# ARCHITECTURE.md — GoRoute Engine Design

## High-Level System Architecture

GoRoute is split into two co-operative layers designed for maximal throughput, minimal memory usage (< 5MB RSS), and zero runtime bloat:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        AGENT CLIENTS & TOOLS                           │
│   Claude Code CLI • OpenCode • Codex CLI • Aider • OpenClaw • Cursor   │
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
│  │   • Dynamic Model Aliases (storage.ResolveModelAlias)            │  │
│  │   • Combos Waterfall (failover, round_robin, lowest_latency)     │  │
│  │   • Pre-emptive Quota Fallback (< 5% skip)                       │  │
│  │   • Cooldown Circuit Breaker (10m for 429/500, 30s for timeout) │  │
│  │   • Proactive Token Health Loop (pkg/tokens/scheduler.go)        │  │
│  │   • SQLite WAL Audit Logging (~/.goroute/goroute.db)             │  │
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
                                    │ Upstream API Calls
       ┌────────────────────────────┼────────────────────────────┐
       ▼                            ▼                            ▼
┌──────────────┐             ┌──────────────┐             ┌──────────────┐
│ Antigravity  │             │  AWS Q Kiro  │             │ OpenRouter / │
│ Multi-Acct   │             │  Multi-Acct  │             │ Groq / NIM / │
│ OAuth Pool   │             │  OIDC Pool   │             │ Frontier API │
└──────────────┘             └──────────────┘             └──────────────┘
```

---

## 1. Request Lifecycle & Routing Engine

When an agent sends a request (`POST /v1/chat/completions` or `POST /v1/messages`):

1. **Protocol Normalization**:
   - OpenAI format requests pass directly into `providers.ChatRequest`.
   - Anthropic `/v1/messages` requests are parsed into `AnthropicMessageRequest`, system prompt arrays are flattened, and parameters are adapted.
2. **Dynamic Alias Resolution**:
   - `storage.ResolveModelAlias(req.Model)` looks up custom and built-in aliases (`claude-3-5-sonnet-latest` $\rightarrow$ `claude-sonnet-4-6`, `gpt-4` $\rightarrow$ `gpt-4o`).
3. **Combo & Cascade Evaluation**:
   - If the model matches a configured combo (e.g. `claude`, `deepseek`, `auto`), targets are ordered according to the chosen strategy:
     - `failover`: Sequential order.
     - `round_robin`: Atomic counter offset.
     - `lowest_latency`: Exponentially weighted moving average (EWMA) latency sorting.
     - `quota_weighted`: Remaining quota percentage sorting.
4. **Resilience & Pre-Emptive Guards**:
   - If a target model is in cooldown, it is skipped.
   - If remaining quota is $< 5\%$, it is skipped pre-emptively to avoid upstream 429 rate limits.
   - Parameters are sanitized for provider-specific constraints via `translator.SanitizeChatRequest`.
5. **Execution & Circuit Breaker**:
   - The selected provider executes the request.
   - On success ($200 \le \text{status} < 400$), latency telemetry is recorded.
   - On failure ($429, 402, 5xx$), the model enters a 10-minute cooldown and GoRoute immediately tries the next target in the combo cascade without breaking the client connection.

---

## 2. Multi-Account Pooling & Background Token Health

For providers with multiple pooled accounts (Google Antigravity, AWS Kiro):
* Accounts are pooled under `pool1` with priority or round-robin strategies.
* `pkg/tokens/scheduler.go` runs a proactive background loop every **3 minutes**:
  - Tests connectivity and latency for each account independently.
  - Automatically refreshes expired OAuth access tokens using stored refresh tokens.
  - Inactive or standby accounts can be toggled on/off via `POST /api/providers/{id}/accounts/{accId}/toggle`.

---

## 3. Native OmniRoute Vault Migration (`pkg/sync`)

GoRoute includes a pure Go decryption engine for OmniRoute's SQLite database (`~/.omniroute/storage.sqlite`):
* Reads `STORAGE_ENCRYPTION_KEY` from `~/.omniroute/.env`.
* Derives the AES key using `scrypt` ($N=32768, r=8, p=1$).
* Decrypts AES-256-GCM encrypted tokens (handling 16-byte IVs).
* Dynamically imports all configured accounts, keys, and combos into `~/.goroute/config.yaml`.
* Validates combo targets, filtering out references to unconfigured providers.
