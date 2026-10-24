# AGENT.md — Developer & AI Agent Handbook for GoRoute

> **For AI Agents & Contributors**: This document outlines the system architecture, code conventions, build toolchains, and operational rules for working on the GoRoute codebase.

---

## 1. Project Mission & Core Tenets

GoRoute is an ultra-fast, high-throughput AI LLM Gateway and multi-provider router engineered for autonomous AI coding agents (**Claude Code, OpenCode, Codex, Aider, OpenClaw, Cursor, Hermes, Cline, Zed**).

### Core Tenets
1. **Zero Node.js Runtime Overhead**: Headless Go + Rust engine running in `< 5 MB RSS RAM` with sub-millisecond proxying (`0.86ms`).
2. **Zero Mock / Fake Data**: All models, quotas, agent fleet processes, and topology telemetry must reflect empirical system state. Never hardcode fake countdowns, mock worker nodes, or sample model arrays.
3. **Hybrid Architecture (Go Core + Rust SIMD Accelerator)**:
   - Go handles networking, Chi HTTP routing, concurrency pools, and SQLite WAL storage.
   - Rust (`crates/goroute-fast`) handles CPU-bound tasks: SIMD model & family resolution, PII sanitization, BPE token estimation, and prompt compression via dynamic C-ABI loading (`purego`).
4. **On-Demand Web Experience**: The gateway daemon runs 24/7 headlessly without UI bloat. The dashboard is pre-compiled via Next.js 15 Tailwind v4 into static HTML/CSS/JS and served directly from embedded memory via `//go:embed dist` at `http://localhost:8128/`.
5. **1-Click Vault Migration**: Native pure Go AES-256-GCM + `scrypt` engine decrypts OmniRoute vaults in `< 0.6s` via `goroute sync omniroute`.

---

## 2. Directory & Architecture Map

```
goroute/
├── cmd/goroute/                  # Cobra CLI commands
│   ├── main.go                   # Root CLI entrypoint (subcommands registered)
│   ├── web.go                    # 'goroute web' / 'goroute ui' browser launcher
│   ├── commands.go               # 'serve', 'auth', 'combo', 'provider', 'sync', 'status'
│   ├── setup.go                  # 1-click agent configuration (Claude, OpenCode, Codex, etc.)
│   └── launch.go                 # Agent process wrapper with injected proxy environment
├── crates/goroute-fast/          # Rust Acceleration Engine (cdylib)
│   ├── Cargo.toml                # Rust crate configuration (regex, serde, libc)
│   └── src/lib.rs                # Model resolver, PII redactor, Caveman compressor
├── web-ui/                       # Source Next.js 15 Tailwind v4 Dashboard
│   ├── app/page.tsx              # Liquid glass obsidian dashboard interface
│   ├── app/globals.css           # Tailwind v4 styles, aurora mesh, noise overlay
│   ├── postcss.config.mjs        # @tailwindcss/postcss plugin configuration
│   └── package.json              # export-to-goroute build script
├── pkg/
│   ├── proxy/                    # HTTP Gateway & reverse proxy server
│   │   ├── server.go             # Chi HTTP mux, REST APIs, topology, quota endpoints
│   │   ├── anthropic.go          # Anthropic /v1/messages <-> OpenAI protocol translator
│   │   └── playground.go         # Web playground chat endpoint
│   ├── router/                   # Routing engine & combo cascades
│   │   └── router.go             # Model matching, failover execution, EWMA latency tracking
│   ├── sync/                     # Native OmniRoute SQLite vault migration
│   │   ├── omnisync.go           # Pure Go AES-256-GCM + scrypt credential decryptor
│   │   └── omnisync_test.go      # Vault decryption & combo import tests
│   ├── fast/                     # Rust Purego dynamic linker bridge
│   │   └── rust.go               # purego.Dlopen and symbol resolution
│   ├── storage/                  # SQLite WAL database & persistence
│   │   ├── aliases.go            # Model alias resolution (gpt-4 -> gpt-4o, etc.)
│   │   ├── sqlite.go             # Database connection & schema (~/.goroute/goroute.db)
│   │   ├── radar.go              # Live models catalog (radar_catalog table)
│   │   └── conductor.go          # Agent fleet tasks and store
│   ├── tokens/                   # Multi-account background health scheduler
│   │   └── scheduler.go          # Proactive 3-min token refresh loop for Antigravity & Kiro
│   ├── providers/                # Upstream AI Provider adapters
│   │   ├── antigravity/          # Google Cloud Code OAuth PKCE multi-account pool
│   │   ├── kiro/                 # AWS Q / CodeWhisperer OIDC Device Flow multi-account pool
│   │   ├── codex/                # OpenAI Codex CLI / ChatGPT backend session
│   │   ├── copilot/              # GitHub Copilot Chat adapter
│   │   ├── cursor/               # Cursor IDE proxy adapter
│   │   └── openai/               # Standard & custom OpenAI-compatible API adapter
│   ├── pool/                     # Multi-account rotation & cooldown tracker
│   ├── quota/                    # Rolling window quota metrics (5h session, 7d weekly)
│   ├── catalog/                  # 338+ provider presets & OmniRoute registry catalog
│   └── web/                      # Embedded frontend dashboard assets
│       ├── embed.go              # //go:embed dist directive
│       └── dist/                 # Static Next.js export bundle
└── assets/                       # Badges, SVG diagrams, and banners
```

---

## 3. Build & Test Commands

### Prerequisites
- Go 1.22+ or 1.24+
- Rust & Cargo (`rustc` 1.80+)
- Node.js 20+ (only required when modifying `web-ui/`)
- `purego` (Linux x86_64 / arm64 compatible, zero CGO required)

### Build Go Binary
```bash
go build -o ~/.local/bin/goroute ./cmd/goroute
```

### Recompile Web Dashboard (When editing `web-ui/`)
```bash
cd web-ui
npm run export-to-goroute
cd ..
go build -o ~/.local/bin/goroute ./cmd/goroute
```

### Build Rust Accelerator Library
Whenever `crates/goroute-fast/src/lib.rs` is modified:
```bash
cd crates/goroute-fast
cargo build --release
cp target/release/libgoroute_fast.so ~/.goroute/libgoroute_fast.so
cp target/release/libgoroute_fast.so ~/.local/lib/libgoroute_fast.so
cd ../..
```

### Run Tests
```bash
# Run all Go package tests:
go test ./pkg/sync ./pkg/router ./pkg/proxy ./pkg/tokens

# Run Rust accelerator tests:
cd crates/goroute-fast && cargo test && cd ../..
```

---

## 4. Runtime & Configuration Conventions

### File Locations
- **Configuration**: `~/.goroute/config.yaml`
- **Database**: `~/.goroute/goroute.db` (SQLite with WAL journal mode)
- **Rust Library**: `~/.goroute/libgoroute_fast.so`
- **Systemd Unit**: `~/.config/systemd/user/goroute.service`

### Endpoints
- `http://127.0.0.1:8128/` : Embedded Web Dashboard
- `http://127.0.0.1:8128/v1/chat/completions` : Universal OpenAI format
- `http://127.0.0.1:8128/v1/messages` : Anthropic Claude format (streaming SSE + tools)
- `http://127.0.0.1:8128/v1/models` : Active model catalog
- `http://127.0.0.1:8128/api/topology` : Live gateway, provider, and account topology
- `http://127.0.0.1:8128/api/usage/quota` : 5h rolling session windows & 7d limits
- `http://127.0.0.1:8128/api/conductor/fleet` : Real running host agent processes (`pgrep`)

---

## 5. Agent Rules & Quality Standards

1. **Verify Before Asserting**: Always execute `curl` tests against live endpoints and verify exit code 0 before concluding any fix or feature.
2. **Never Inject Mock Data**:
   - In frontend React: Pull from `/api/radar/catalog`, `/api/topology`, or `/api/usage/quota`. Do not fabricate static dummy objects.
   - In backend Go: Query host processes using `pgrep` or check SQLite tables.
3. **Preserve User Agent Setups**: Do not modify foreign agent configuration files (`~/.claude/settings.json`, `~/.codex/config.toml`, `~/.config/opencode/`) unless explicitly requested.
4. **Git Hygiene**: Always run `git status`, stage only intended files, write atomic semantic commit messages (`feat:`, `fix:`, `refactor:`), and push to `origin/main`.
