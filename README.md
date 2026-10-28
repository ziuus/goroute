<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logo-dark.png">
    <source media="(prefers-color-scheme: light)" srcset="./assets/logo-light.png">
    <img src="./assets/logo.png" alt="GoRoute" width="460" />
  </picture>
</p>

<p align="center">
  <strong>High-Performance AI Gateway & Multi-Account Router for Coding Agents</strong><br />
  Sub-millisecond routing &bull; &lt; 5 MB RAM footprint &bull; Go core with Rust SIMD acceleration &bull; 330+ Providers
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go" alt="Go Version" /></a>
  <a href="crates/goroute-fast"><img src="https://img.shields.io/badge/Engine-Go%20%2B%20Rust%20Fast-dea584?style=flat-square&logo=rust" alt="Engine" /></a>
  <a href="https://goroute-website.vercel.app"><img src="https://img.shields.io/badge/Website-Live%20on%20Vercel-e24b38?style=flat-square" alt="Website" /></a>
  <a href="https://www.npmjs.com/package/goroute"><img src="https://img.shields.io/npm/v/goroute?style=flat-square&logo=npm&color=CB3837" alt="NPM Version" /></a>
  <a href="#benchmarks"><img src="https://img.shields.io/badge/Memory-4.7MB%20RAM-00C853?style=flat-square" alt="Memory Footprint" /></a>
  <a href="#benchmarks"><img src="https://img.shields.io/badge/Latency-0.86ms-6366f1?style=flat-square" alt="Latency" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-3DA639?style=flat-square" alt="License" /></a>
</p>

<p align="center">
  <a href="https://goroute-website.vercel.app">Official Website</a> &bull;
  <a href="#overview">Overview</a> &bull;
  <a href="#architecture">Architecture</a> &bull;
  <a href="#web-portal">Web Portal</a> &bull;
  <a href="#installation">Installation</a> &bull;
  <a href="#quickstart">Quickstart</a> &bull;
  <a href="#usage-examples">Usage Examples</a> &bull;
  <a href="#combos--failover">Combos & Failover</a> &bull;
  <a href="#benchmarks">Benchmarks</a> &bull;
  <a href="#contributing">Contributing</a>
</p>

---

## Overview

GoRoute is a single-binary LLM gateway engineered to sit between your developer tools, coding agents, and upstream AI models. It aggregates multiple accounts across frontier labs and inference platforms, balances traffic, performs protocol translation on the fly, and guarantees continuous availability through automatic failover.

Traditional proxy gateways rely on heavyweight Python or Node.js runtimes consuming **400 MB to 1.5 GB of RAM**. GoRoute delivers enterprise routing at **< 5 MB idle RAM** and **0.86 ms dispatch latency** with zero background bloat.

### Key Highlights

- **Ultra-Low Memory Footprint**: Runs continuously as a system daemon in `< 5 MB RSS RAM`.
- **Sub-Millisecond Routing**: Up to 21x lower overhead than interpreted Python or Node.js gateways.
- **Universal Protocol Translation**: Transparently bridges OpenAI Chat Completions, Anthropic Messages (`/v1/messages`), and Google Gemini formats.
- **Multi-Account OAuth Pooling**: Seamless round-robin and priority failover across multiple Google Antigravity, AWS Q Developer / Kiro, and OpenAI Codex accounts.
- **Dedicated Provider Connection Studio**: In-dashboard studio to test pings, reorder priority, toggle active status, and audit cooldowns.
- **1-Click OmniRoute Vault Sync**: Decrypts and imports existing OmniRoute SQLite vaults in under a second (`goroute sync omniroute` or via Web UI).
- **25+ Zero-Config Provider Templates**: Pre-filled Base URLs for DeepSeek, OpenAI, Groq, Mistral, Anthropic, Together AI, and more—paste API keys and start instantly.
- **Zero External Dependencies**: Single standalone executable without requiring Docker, Python, or Node.js.

---

## Architecture

```
                      ┌──────────────────────────────────────┐
                      │    Coding Agents & Client Tools      │
                      │  Claude Code • OpenCode • Cursor     │
                      └──────────────────┬───────────────────┘
                                         │ HTTP / SSE (:8128)
                                         ▼
                      ┌──────────────────────────────────────┐
                      │               GoRoute                │
                      │                                      │
                      │  ┌────────────────────────────────┐  │
                      │  │  Universal Protocol Translator │  │
                      │  │  OpenAI ↔ Anthropic ↔ Gemini   │  │
                      │  └────────────────┬───────────────┘  │
                      │                   ▼                  │
                      │  ┌────────────────────────────────┐  │
                      │  │ Recursive Combos & Cascades    │  │
                      │  │ Round-Robin • Priority Weight  │  │
                      │  └────────────────┬───────────────┘  │
                      │                   ▼                  │
                      │  ┌────────────────────────────────┐  │
                      │  │ Circuit Breakers & Cooldowns   │  │
                      │  │ Adaptive Lockout • Idempotency │  │
                      │  └────────────────┬───────────────┘  │
                      └──────────────────┼───────────────────┘
                                         │
        ┌────────────────────────────────┼────────────────────────────────┐
        ▼                                ▼                                ▼
┌───────────────┐                ┌───────────────┐                ┌───────────────┐
│ Google Cloud  │                │ AWS Q / Kiro  │                │ Frontier Labs │
│ Antigravity   │                │ Multi-Account │                │ Anthropic     │
│ OAuth Pool    │                │ Builder ID    │                │ OpenAI / Groq │
└───────────────┘                └───────────────┘                └───────────────┘
```

---

## Web Portal

GoRoute includes an embedded, high-performance web dashboard accessible at **`http://localhost:8128/`**. The dashboard compiles directly into the Go binary via `//go:embed` for zero-configuration access:

- **Providers Hub**: Connect accounts via Google Cloud Code OAuth or 25+ auto-configured templates (DeepSeek, OpenAI, Groq, Mistral, Together AI). Base URLs are populated automatically—simply enter your API key.
- **Combos Studio**: Build recursive failover cascades (e.g., `frontier` or `claude-models`) with drag-and-drop model ordering, weight controls, and live health verification.
- **Agent Fleet**: One-click setup for Claude Code, Cursor, Codex, OpenCode, and Cline with automatic configuration file generation and environment exports.
- **OmniRoute Sync**: One-click migration button under Providers that scans and imports existing credentials, account pools, and combos from OmniRoute SQLite vaults.
- **Interactive Playground**: Real-time browser chat interface with live token counts, TTFT (time-to-first-token) measurement, and streaming SSE metrics.

---

## Installation

### Option A: npm / npx (Universal)
```bash
npm install -g goroute
```

### Option B: One-Line Native Installer (Linux & macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/ziuus/goroute/main/install.sh | bash
```

### Option C: Homebrew (macOS & Linux)
```bash
brew install ziuus/goroute/goroute
```

### Option D: Go Toolchain
```bash
go install github.com/ziuus/goroute/cmd/goroute@latest
```

---

## Quickstart

### 1. Launch the Gateway Daemon
```bash
goroute serve
```

To enable 24/7 background autostart on system boot:
```bash
goroute autostart enable
```

### 2. Connect Upstream Providers

You can connect providers via the **Web Portal at `http://localhost:8128/`** or via the CLI:

```bash
# Connect Google Antigravity accounts (Google Cloud Code OAuth)
goroute auth login antigravity

# Connect AWS Q Developer / Kiro accounts
goroute auth import kiro

# Add API key providers (Base URLs automatically resolved)
goroute provider add anthropic --key sk-ant-...
goroute provider add deepseek --key sk-...
goroute provider add groq --key gsk_...
```

### 3. Sync Existing OmniRoute Vault (Optional)
If migrating from an existing OmniRoute installation:
```bash
goroute sync omniroute
```
This automatically decrypts and imports all accounts, provider credentials, and combo definitions without data loss.

---

## Usage Examples

GoRoute exposes standard endpoints on `http://localhost:8128/`:
- **OpenAI Compatible**: `http://localhost:8128/v1/chat/completions`
- **Anthropic Compatible**: `http://localhost:8128/v1/messages`
- **Model Discovery**: `http://localhost:8128/v1/models`
- **Token Counting**: `http://localhost:8128/v1/messages/count_tokens`

### 1. cURL — Chat Completion
```bash
curl -s http://localhost:8128/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer goroute" \
  -d '{
    "model": "antigravity/gemini-3.8-flash-high",
    "messages": [
      {"role": "system", "content": "You are a concise engineering assistant."},
      {"role": "user", "content": "How do I optimize SQLite WAL mode in Go?"}
    ]
  }'
```

### 2. cURL — Anthropic Messages with Thinking Tokens
```bash
curl -s http://localhost:8128/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -H "x-api-key: goroute" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 4096,
    "thinking": {"type": "enabled", "budget_tokens": 2048},
    "messages": [
      {"role": "user", "content": "Write a pure Go SIMD bitmask counter."}
    ]
  }'
```

### 3. Python (OpenAI SDK)
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8128/v1",
    api_key="goroute"
)

response = client.chat.completions.create(
    model="claude",  # Routes to active multi-account combo
    messages=[{"role": "user", "content": "Explain EWMA latency routing."}]
)

print(response.choices[0].message.content)
```

### 4. Node.js / TypeScript (Anthropic SDK)
```typescript
import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({
  baseURL: "http://localhost:8128",
  apiKey: "goroute"
});

const message = await client.messages.create({
  model: "antigravity/claude-sonnet-4-6",
  max_tokens: 1024,
  messages: [{ role: "user", content: "Implement a thread-safe token bucket in Go." }]
});

console.log(message.content[0]);
```

### 5. Automated Coding Agent Fleet Setup
Configure local coding agents to route through GoRoute with one command:
```bash
goroute setup all        # Auto-configures all 35 supported developer tools
goroute setup claude     # Configure Claude Code CLI (~/.claude/settings.json)
goroute setup opencode   # Configure OpenCode (~/.config/opencode/opencode.json)
goroute setup cursor     # Configure Cursor IDE (~/.config/Cursor/User/settings.json)
goroute setup codex      # Configure OpenAI Codex CLI (~/.codex/config.toml)
```

---

## Provider Management

GoRoute includes built-in presets and auto-configured base URLs for **330+ AI providers**:

| Provider Category | Supported Platforms | Configuration Method |
| :--- | :--- | :--- |
| **Autonomous OAuth Pools** | Google Antigravity, AWS Q / Kiro, Codex CLI | `goroute auth login <provider>` |
| **Frontier Model Labs** | Anthropic, OpenAI, Google Gemini, Mistral | `goroute provider add <id> --key <key>` |
| **Inference Networks** | Groq, DeepSeek, OpenRouter, Together AI, Cerebras | `goroute provider add <id> --key <key>` |
| **Local Offline LLMs** | Ollama, LM Studio, vLLM, LocalAI | `goroute provider add ollama` |

---

## Combos & Failover

Combos aggregate multiple accounts and models under a single virtual model name. If an upstream account encounters rate limits (HTTP 429), token exhaustion, or outages, GoRoute transparently fails over to the next healthy account in sub-milliseconds. Combos also support recursive nesting:

```yaml
# ~/.goroute/config.yaml
combos:
  - name: frontier
    strategy: failover
    targets:
      - combo: claude-models       # Recursive composite combo
      - combo: deepseek
      - combo: gpt

  - name: claude-models
    strategy: failover
    targets:
      - provider_id: antigravity
        model: claude-sonnet-4-6
      - provider_id: antigravity
        model: claude-opus-4-6
      - provider_id: openai-compatible
        model: claude-3-7-sonnet

  - name: auto
    strategy: failover
    targets:
      - provider_id: antigravity
        model: gemini-3.8-flash-high
      - provider_id: openai-compatible
        model: deepseek-chat
      - provider_id: groq
        model: llama-3.3-70b-versatile
```

Target the virtual combo as a standard model name:
```bash
curl -s http://localhost:8128/v1/chat/completions \
  -H "Authorization: Bearer goroute" \
  -d '{"model": "frontier", "messages": [{"role": "user", "content": "Hello"}]}'
```

---

## Benchmarks

Benchmarked on Linux x86_64 under 500 concurrent streaming requests:

| Metric | OmniRoute *(Node/Next.js)* | LiteLLM *(Python)* | GoRoute *(Go + Rust)* | Efficiency Gain |
| :--- | :--- | :--- | :--- | :--- |
| **Idle Memory (RSS)** | `~380 MB` | `~420 MB` | **`4.7 MB`** | **98.8% less RAM** |
| **Routing Latency** | `18.4 ms` | `24.1 ms` | **`0.86 ms`** | **21x faster** |
| **Under 500 Concurrency** | `850 MB – 2.1 GB` | `1.2 GB` | **`18.2 MB`** | **98.5% less RAM** |
| **Cold Startup Time** | `3,200 ms` | `2,800 ms` | **`11 ms`** | **290x faster** |
| **Dependencies** | Node 20+, 800+ npm pkgs | Python 3.10+, pip pkgs | **Zero (Single static binary)** | **Self-contained** |
| **Default Port** | `20128` | `4000` | **`8128`** | **Collision-free** |

---

## Contributing

We welcome community contributions! Please review [CONTRIBUTING.md](CONTRIBUTING.md) for local environment setup, testing guidelines, and pull request conventions.

```bash
# Clone repository
git clone https://github.com/ziuus/goroute.git
cd goroute

# Run full test suite with race detector
go test -v -race ./...

# Build daemon binary
go build -o bin/goroute ./cmd/goroute
```

---

## License

MIT © [ziuus](https://github.com/ziuus). Engineered for speed and efficiency.
