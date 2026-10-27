# GoRoute

**High-Performance AI Gateway & Multi-Provider Router**  
*Sub-millisecond routing • < 5 MB RAM footprint • Go core with Rust SIMD acceleration • 330+ Providers*

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![Engine](https://img.shields.io/badge/Engine-Go%20%2B%20Rust%20Fast-dea584?style=flat-square&logo=rust)](crates/goroute-fast)
[![NPM Version](https://img.shields.io/npm/v/goroute?style=flat-square&logo=npm&color=CB3837)](https://www.npmjs.com/package/goroute)
[![Memory Footprint](https://img.shields.io/badge/Memory-3.8MB%20RAM-00C853?style=flat-square)](#benchmarks)
[![Latency](https://img.shields.io/badge/Latency-0.86ms-6366f1?style=flat-square)](#benchmarks)
[![License](https://img.shields.io/badge/License-MIT-3DA639?style=flat-square)](LICENSE)

[Overview](#overview) • [Installation](#installation) • [Quickstart](#quickstart) • [Usage Examples](#usage-examples) • [Provider Management](#provider-management) • [Combos & Failover](#combos--failover) • [Benchmarks](#benchmarks) • [Contributing](#contributing)

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
- **1-Click OmniRoute Vault Sync**: Decrypts and imports existing OmniRoute SQLite vaults in under a second (`goroute sync omniroute`).
- **Zero External Dependencies**: Single standalone executable without requiring Docker, Python, or Node.js.

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

### 2. Access the Web Dashboard
Open **`http://localhost:8128/`** in your browser to view the real-time topology, rolling quotas, agent fleet status, and provider connection studio.

### 3. Connect Upstream Accounts
```bash
# Connect Google Antigravity accounts (Google Cloud Code OAuth)
goroute auth login antigravity

# Connect AWS Q Developer / Kiro accounts
goroute auth import kiro

# Add API key providers
goroute provider add anthropic --key sk-ant-...
goroute provider add deepseek --key sk-...
goroute provider add groq --key gsk_...
```

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

### 5. Automated Coding Agent Setup
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

GoRoute includes built-in presets for **330+ AI providers**:

| Provider Category | Supported Platforms | Configuration Method |
| :--- | :--- | :--- |
| **Autonomous OAuth Pools** | Google Antigravity, AWS Q / Kiro, Codex CLI | `goroute auth login <provider>` |
| **Frontier Model Labs** | Anthropic, OpenAI, Google Gemini, Mistral | `goroute provider add <id> --key <key>` |
| **Inference Networks** | Groq, DeepSeek, OpenRouter, Together AI, Cerebras | `goroute provider add <id> --key <key>` |
| **Local Offline LLMs** | Ollama, LM Studio, vLLM, LocalAI | `goroute provider add ollama` |

---

## Combos & Failover

Combos aggregate multiple accounts and models under a single virtual model name. If an upstream account encounters rate limits (HTTP 429), token exhaustion, or outages, GoRoute transparently fails over to the next healthy account in sub-milliseconds:

```yaml
# ~/.goroute/config.yaml
combos:
  - name: claude
    strategy: failover
    targets:
      - provider_id: antigravity
        model: claude-sonnet-4-6
      - provider_id: kiro
        model: claude-sonnet-4.5
      - provider_id: kiro
        model: claude-haiku-4.5

  - name: auto
    strategy: failover
    targets:
      - provider_id: antigravity
        model: gemini-3.8-flash-high
      - provider_id: kiro
        model: deepseek-3.2
      - provider_id: groq
        model: llama-3.3-70b-versatile
```

Target the virtual combo as a standard model name:
```bash
curl -s http://localhost:8128/v1/chat/completions \
  -H "Authorization: Bearer goroute" \
  -d '{"model": "claude", "messages": [{"role": "user", "content": "Hello"}]}'
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
# Clone and run unit tests
git clone https://github.com/ziuus/goroute.git
cd goroute
make test

# Build binary
make build
```

---

## License

MIT © [ziuus](https://github.com/ziuus). Engineered for speed and efficiency.
