# Contributing to GoRoute

Thank you for contributing to GoRoute! We welcome bug fixes, performance optimizations, new provider integrations, and documentation improvements.

---

## 1. Core Principles

1. **Ultra-Low Memory Footprint**: Core daemon RSS memory must remain `< 5 MB RAM` at idle and `< 20 MB` under 500 concurrent connections.
2. **Sub-Millisecond Routing**: Routing pipeline latency must stay `< 1 ms`. Avoid heavyweight middleware, reflection loops, or unbuffered I/O.
3. **Evidence Before Assertions**: Every code change must be accompanied by passing unit tests and benchmark verification.
4. **Zero CGO Dependency**: The Go binary builds with `CGO_ENABLED=0`. CPU-intensive SIMD operations are dynamically linked via `ebitengine/purego`.

---

## 2. Local Development Setup

```bash
# Clone the repository
git clone https://github.com/ziuus/goroute.git
cd goroute

# Run the test suite
make test

# Build the standalone binary
make build

# Start the gateway locally
./bin/goroute serve
```

### Dashboard Development (Next.js)

```bash
cd web-ui
bun install
bun run dev          # Live reload on http://localhost:3000

# Export static bundle into Go embedded assets
bun run export-to-goroute
```

---

## 3. Commit Message Conventions

We adhere to the [Conventional Commits](https://www.conventionalcommits.org/) specification:

| Prefix | Usage | Example |
| :--- | :--- | :--- |
| `feat:` | New feature or provider capability | `feat(provider): add Mistral Large adapter` |
| `fix:` | Bug fix or error resolution | `fix(auth): allow loopback agent requests without token rejection` |
| `perf:` | Performance optimization | `perf(router): reduce mutex contention in EWMA latency picker` |
| `docs:` | Documentation update | `docs(readme): add curl streaming examples` |
| `test:` | Adding or fixing test coverage | `test(proxy): add table tests for Anthropic count_tokens` |
| `refactor:` | Code restructuring without behavior change | `refactor(keys): simplify key validation logic` |
| `chore:` | Build script, tooling, or dependency updates | `chore(deps): update purego to v0.8.2` |

---

## 4. Pull Request Checklist

Before submitting a PR, verify all of the following:

- [ ] **Tests Pass**: `go test -v -race ./...` runs without failures.
- [ ] **Memory Verification**: `go test -v -run=TestMemoryAndConcurrency ./tests` confirms `< 5 MB` idle footprint.
- [ ] **No Direct Edits to Generated Code**: Did not manually edit `web/dist/*` or `go.sum`.
- [ ] **Clean Git Hygiene**: Branch is rebased onto `main` without merge commits.
- [ ] **Documentation Updated**: If CLI flags, endpoints, or conventions changed, updated `README.md` and `AGENTS.md` in the same PR.
- [ ] **Security Checked**: Any outbound URL calls use `ValidateBaseURL` to prevent SSRF against link-local metadata services.
