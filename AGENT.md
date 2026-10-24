# AGENT.md — Developer & AI Agent Handbook for GoRoute

> **For AI Agents & Contributors**: This document outlines the system architecture, code conventions, build toolchains, and operational rules for working on the GoRoute codebase.

---

## 1. Project Mission & Core Tenets

GoRoute is an ultra-fast, high-throughput AI LLM Gateway and multi-provider router engineered for AI coding agents (**Claude Code, OpenCode, Codex, Aider, OpenClaw, Cursor, Hermes, Cline**).

### Core Tenets
1. **Zero Node.js Runtime Overhead**: Headless Go + Rust engine running in `< 15 MB RAM` with sub-millisecond proxying (`0.86ms`).
2. **Zero Mock / Fake Data**: All models, quotas, agent fleet processes, and topology telemetry must reflect empirical system state. Never hardcode fake countdowns, mock worker nodes, or sample model arrays.
3. **Hybrid Architecture (Go Core + Rust SIMD Accelerator)**:
   - Go handles networking, Chi HTTP routing, concurrency pools, and SQLite WAL storage.
   - Rust (`crates/goroute-fast`) handles CPU-bound tasks: SIMD model & family resolution, PII sanitization, BPE token estimation, and prompt compression via dynamic C-ABI loading (`purego`).
4. **On-Demand Web Experience**: The gateway daemon runs 24/7 headlessly without UI bloat. The dashboard is pre-compiled and served directly from embedded memory via `goroute web`.

---

## 2. Directory & Architecture Map

```
goroute/
├── cmd/goroute/                  # Cobra CLI commands
│   ├── main.go                   # Root CLI entrypoint
│   ├── web.go                    # 'goroute web' / 'goroute ui' launcher
│   ├── commands.go               # 'serve', 'auth', 'combo', 'status', etc.
│   ├── setup.go                  # 1-click agent configuration (Claude, OpenCode, Codex)
│   └── launch.go                 # Agent process wrapper with injected environment
├── crates/goroute-fast/          # Rust Acceleration Engine (cdylib)
│   ├── Cargo.toml                # Rust crate configuration (regex, serde, libc)
│   └── src/lib.rs                # Model resolver, PII redactor, Caveman compressor
├── pkg/
│   ├── proxy/                    # HTTP Gateway & reverse proxy server
│   │   ├── server.go             # Chi HTTP mux, REST APIs, topology, quota endpoints
│   │   ├── anthropic.go          # Anthropic /v1/messages <-> OpenAI protocol translator
│   │   └── playground.go         # Web playground chat endpoint
│   ├── router/                   # Routing engine & combo cascades
│   │   └── router.go             # Model matching, failover execution, latency tracking
│   ├── fast/                     # Rust Purego dynamic linker bridge
│   │   └── rust.go               # purego.Dlopen and symbol resolution
│   ├── storage/                  # SQLite WAL database & persistence
│   │   ├── db.go                 # Database connection & migrations (~/.goroute/goroute.db)
│   │   ├── radar.go              # Live models catalog (radar_catalog table)
│   │   └── conductor.go          # Agent fleet tasks and store
│   ├── providers/                # Upstream AI Provider adapters
│   │   ├── antigravity/          # Google Cloud Code OAuth PKCE rotation
│   │   ├── kiro/                 # AWS Q / CodeWhisperer OAuth & social flow
│   │   ├── codex/                # OpenAI Codex CLI / ChatGPT backend session
│   │   ├── copilot/              # GitHub Copilot Chat adapter
│   │   ├── cursor/               # Cursor IDE proxy adapter
│   │   └── openai/               # Standard OpenAI-compatible API adapter
│   ├── pool/                     # Multi-account rotation & cooldown tracker
│   ├── quota/                    # Rolling window quota metrics (5h session, 7d weekly)
│   ├── keys/                     # Virtual API key auth & rate-limit lanes
│   └── web/                      # Embedded frontend dashboard assets
│       ├── embed.go              # //go:embed dist/* directive
│       └── dist/index.html       # Single-file zero-runtime responsive dashboard
└── assets/                       # Badges, SVG diagrams, and banners
```

---

## 3. Build & Test Commands

### Prerequisites
- Go 1.22+ or 1.24+
- Rust & Cargo (`rustc` 1.80+)
- `purego` (Linux x86_64 / arm64 compatible, no CGO required)

### Build Go Binary
```bash
go build -o ~/.local/bin/goroute ./cmd/goroute
```

### Build Rust Accelerator Library
Whenever `crates/goroute-fast/src/lib.rs` is modified:
```bash
cd crates/goroute-fast
cargo build --release
# Deploy dynamic library to discovery paths:
cp target/release/libgoroute_fast.so ~/.goroute/libgoroute_fast.so
cp target/release/libgoroute_fast.so ~/.local/lib/libgoroute_fast.so
cd ../..
```

### Run Tests
```bash
# Run all Go package tests:
go test ./pkg/...

# Run Rust accelerator tests:
cd crates/goroute-fast && cargo test && cd ../..
```

---

## 4. Runtime & Configuration Conventions

### File Locations
- **Configuration**: `~/.goroute/config.yaml`
- **Database**: `~/.goroute/goroute.db` (SQLite with WAL journal mode)
- **MITM Root CA**: `~/.goroute/certs/goroute-ca.crt` & `goroute-ca.key`
- **Rust Library**: `~/.goroute/libgoroute_fast.so`

### Endpoints
- `http://127.0.0.1:8128/` : Embedded Web Dashboard
- `http://127.0.0.1:8128/v1/chat/completions` : Universal OpenAI format
- `http://127.0.0.1:8128/v1/messages` : Anthropic Claude format (streaming SSE + tools)
- `http://127.0.0.1:8128/v1/models` : Active model catalog
- `http://127.0.0.1:8128/api/topology` : Live gateway, provider, and combo topology
- `http://127.0.0.1:8128/api/usage/quota` : 5h rolling session windows & 7d limits
- `http://127.0.0.1:8128/api/conductor/fleet` : Real running agent processes (`pgrep`)

---

## 5. Agent Rules & Quality Standards

1. **Verify Before Asserting**: Always execute `curl` tests against live endpoints and verify exit code 0 before concluding any fix or feature.
2. **Never Inject Mock Data**:
   - In frontend JavaScript: Do not use `const sampleModels = [...]` or hardcode fallback strings like `'4h 19m'`. Pull from `/api/radar/catalog` or `/api/usage/quota`.
   - In backend Go: Do not fabricate artificial nodes (`node-1 Local Worker`). Query host processes using `pgrep` or check SQLite tables.
3. **Preserve User Agent Setups**: Do not modify foreign agent configuration files (`~/.claude/settings.json`, `~/.codex/config.toml`, `~/.config/opencode/`) unless specifically commanded by the user.
4. **Git Hygiene**: Always run `git status`, stage only intended files, write atomic semantic commit messages (`feat:`, `fix:`, `refactor:`), and push to `origin/main`.
