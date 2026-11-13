# E2E Test Infra: GoRoute Zero-Mock Parity

## Test Philosophy
- **Opaque-box, requirement-driven**: Tests interact with GoRoute strictly via CLI execution, HTTP endpoints, WebSocket/SSE streams, and SQLite persistence. No internal module hooks or non-public interfaces are used.
- **Methodology**: 4-Tier verification combining Category-Partition, Boundary Value Analysis (BVA), Pairwise Combinatorial Testing, and Real-World Agent Workload Scenarios.
- **Parity verification**: Matches OmniRoute reference behavior across telemetry, proxying, provider protocols, web dashboard SPA delivery, and zero mock/dummy data tolerance.

---

## Feature Inventory

| # | Feature Domain | Source | Tier 1 (Min 5) | Tier 2 (Min 5) | Tier 3 (Pairwise) |
|---|----------------|--------|:--------------:|:--------------:|:-----------------:|
| F1 | CLI Lifecycle & Subcommands | ORIGINAL_REQUEST §R1, PROJECT #1, #2, #31 | 5 | 5 | ✓ |
| F2 | Daemon Health & Graceful Shutdown | ORIGINAL_REQUEST §R1, PROJECT #2, #3 | 5 | 5 | ✓ |
| F3 | Authentication & Virtual Keys | ORIGINAL_REQUEST §R1, PROJECT #4, #8 | 5 | 5 | ✓ |
| F4 | Router & Model Resolution | ORIGINAL_REQUEST §R1, PROJECT #9, #13 | 5 | 5 | ✓ |
| F5 | Provider Adapters & Protocols | ORIGINAL_REQUEST §R1, PROJECT #16, #17, #18, #19, #21 | 5 | 5 | ✓ |
| F6 | Streaming, TTFT & SSE Telemetry | ORIGINAL_REQUEST §R1, R3, PROJECT #14, #15, #30 | 5 | 5 | ✓ |
| F7 | Fallback Cascades & Error Handling | ORIGINAL_REQUEST §R1, PROJECT #13, #24 | 5 | 5 | ✓ |
| F8 | Quotas, Ledger & Fair-Share Pooling | ORIGINAL_REQUEST §R1, PROJECT #10, #11, #12 | 5 | 5 | ✓ |
| F9 | Webhooks & Batch Processing | ORIGINAL_REQUEST §R1, PROJECT #6, #7, #23 | 5 | 5 | ✓ |
| F10 | Chaos & Dynamic Radar Resilience | ORIGINAL_REQUEST §R1, PROJECT #5, #24, #30 | 5 | 5 | ✓ |
| F11 | Web UI Dashboard & Endpoints | ORIGINAL_REQUEST §R2, PROJECT #32, #33, #35, #36 | 5 | 5 | ✓ |

---

## Test Architecture

### 1. Test Harness & Runner
- **Test Runner Location**: `tests/e2e/runner_test.go` and `tests/e2e/run.sh`
- **Invocation**:
  ```bash
  go test -v -tags=e2e ./tests/e2e/...
  # OR
  bash tests/e2e/run.sh
  ```
- **Pass/Fail Semantics**: Standard Go test exit codes (0 = all pass, non-zero = failure). Each tier is subdivided into clean subtests with detailed failure assertions.
- **Mock Upstream Engine**: `tests/e2e/infra/mock_upstreams.go` provides lightweight HTTP servers simulating upstream providers (OpenAI, Anthropic with header inspection, Codex OAuth, Cursor Protobuf, Copilot) without external internet or API key requirements.

### 2. Directory Layout
```
tests/e2e/
├── run.sh                          # Master E2E execution script
├── runner_test.go                  # Go test entrypoint executing all test suites
├── infra/                          # Shared opaque-box testing harness
│   ├── harness.go                  # Daemon process manager, port allocator, temp SQLite DB
│   ├── mock_upstreams.go           # Opaque HTTP servers simulating OpenAI/Anthropic/Codex/Cursor
│   ├── client.go                   # HTTP/SSE client helpers with auth and headers
│   └── sqlite_verifier.go          # Direct SQLite table & WAL inspector
├── tier1/                          # Tier 1: Feature Coverage Suites
│   ├── f1_cli_test.go
│   ├── f2_daemon_test.go
│   ├── f3_auth_keys_test.go
│   ├── f4_router_test.go
│   ├── f5_providers_test.go
│   ├── f6_streaming_test.go
│   ├── f7_fallbacks_test.go
│   ├── f8_quota_ledger_test.go
│   ├── f9_webhooks_batch_test.go
│   ├── f10_chaos_radar_test.go
│   └── f11_webui_test.go
├── tier2/                          # Tier 2: Boundary & Corner Cases Suites
│   ├── b1_payload_limits_test.go   # Empty bodies, 10MB bodies, invalid JSON
│   ├── b2_auth_boundaries_test.go  # Expired keys, dummy bypass rejection, malformed headers
│   ├── b3_concurrency_test.go      # Parallel requests hitting connection & rate limits
│   ├── b4_timeout_abort_test.go    # Client aborts, slow upstream timeouts, disconnects
│   └── b5_data_corruption_test.go  # SQLite locked/recovered, invalid configs
├── tier3/                          # Tier 3: Cross-Feature Interactions Suites
│   ├── combo_fallback_ledger_test.go # Multi-provider fallback updating ledger & call logs
│   ├── streaming_quota_pool_test.go  # SSE streaming accurately deducting quota & rotating accounts
│   ├── chaos_under_load_test.go      # Dynamic latency/chaos injection during concurrent queries
│   └── webhook_batch_audit_test.go   # Batch completion triggering signed HMAC webhooks
└── tier4/                          # Tier 4: Real-World Agent Workload Scenarios
    ├── claude_code_workflow_test.go # Multi-turn tool calling, thinking tags, streaming SSE
    ├── opencode_workflow_test.go    # Parallel completions, model routing, token tracking
    ├── codex_responses_test.go      # Codex OAuth backend responses and completions
    └── cursor_connectrpc_test.go    # Protobuf-framed agent service simulation
```

---

## Real-World Application Scenarios (Tier 4)

| # | Scenario | Features Exercised | Complexity |
|---|----------|--------------------|------------|
| S1 | Claude Code Interactive Multi-Turn Session | F3 (Keys), F4 (Router), F5 (Anthropic protocol), F6 (Streaming SSE), F8 (Ledger) | High |
| S2 | OpenCode High-Concurrency Parallel Edits | F1 (CLI), F4 (Router), F7 (Fallbacks), F8 (Account Pool), F10 (Radar) | High |
| S3 | Codex OAuth Agent Coding Pipeline | F3 (Auth), F5 (Codex Backend), F6 (TTFT Tracking), F8 (Ledger) | Medium |
| S4 | Cursor ConnectRPC Protobuf Simulation | F4 (Router), F5 (Cursor Protobuf Framing), F6 (Stream), F8 (Pool) | High |
| S5 | Enterprise Webhook & Audit Pipeline | F3 (Keys), F6 (Call Logs), F8 (Quota Limits), F9 (HMAC Webhooks) | Medium |
| S6 | Chaos & Cascade Fault Recovery | F4 (Combos), F7 (Failover on 429/500), F10 (Chaos Injected), F2 (Health) | High |

---

## Coverage Thresholds
- **Tier 1**: ≥5 per feature domain (11 domains × 5 = 55 tests minimum)
- **Tier 2**: ≥5 per boundary category (5 categories × 5 = 25 tests minimum)
- **Tier 3**: ≥10 cross-feature combination tests covering pairwise interactions
- **Tier 4**: ≥6 realistic multi-turn agent workload scenarios
- **Total Suite**: ≥100+ automated test cases
