# Project: infranow

## Philosophy: RootOps

Principiis obsta — resist the beginnings. Address root causes, not symptoms. Control over observability. Determinism over ML. Restraint over speed.

- Systems MUST prevent dangerous conditions structurally, not detect them after the fact
- If extraordinary effort is needed to maintain safety, the architecture is broken
- Tools present evidence and let users decide — mirrors, not oracles
- Actions should be reversible; when they cannot be, require explicit consent and structural safeguards
- Every project MUST define what it is NOT — unbounded scope leads to architectural decay

## Commands

- `make build` — Build binary to bin/infranow
- `make test` — Run tests with -race -timeout 30s
- `make lint` — Run golangci-lint
- `make fmt` — Format with gofmt
- `make vet` — Run go vet
- `make check` — Run fmt + vet + test
- `make clean` — Clean build artifacts

## Architecture

- Entry: cmd/infranow/main.go (minimal, delegates to internal/cli)
- CLI framework: Cobra (spf13/cobra)
- Internal packages:
  - cli — Cobra command definitions (root + monitor subcommand)
  - detector — Rule-based problem detection (each a PromQL query + threshold)
  - metrics — MetricsProvider interface + PrometheusClient implementation
  - models — Problem struct, Severity type, scoring logic
  - monitor — Watcher orchestration (concurrent detector loop) + Bubble Tea TUI
  - baseline — Save/load problem snapshots, diff comparison for drift detection
  - filter — Namespace include/exclude glob pattern filtering
  - util — Exit codes, native Kubernetes port-forward via client-go

## Code Style

- Go: minimal main.go delegating to internal/, Cobra for CLIs, golangci-lint, race detection in tests
- Comments explain "why" not "what". No decorative comments
- No magic numbers — name and document constants
- Defensive coding: null checks, graceful degradation, fallback to defaults

## Naming

- Go files: snake_case.go
- Go packages: short single-word (cli, detector, metrics, models, monitor, filter, baseline, util)
- Conventional commits: feat:, fix:, docs:, test:, refactor:, chore:

## Conventions

- Minimal main.go — single Execute() call
- Internal packages: short single-word names
- Struct-based domain models with json tags
- Standard Go formatting (gofmt/goimports)
- Version injected via LDFLAGS at build time
- Detectors are stateless: PromQL query in, []*Problem out

## Testing

- Tests are mandatory for all new code. Coverage target: >85%
- Deterministic tests only — no flaky/probabilistic tests
- Go: -race flag always
- Test files alongside source (Go convention)
- TDD preferred: write tests first, then implement to make them pass

## Verification — IMPORTANT

YOU MUST verify your work:
- Run `make test` after code changes (includes -race)
- Run `make lint` before marking complete
- Run `go vet ./...` for suspicious constructs
- Never mark a task complete if tests fail or implementation is partial

## Workflow

- Start complex tasks in Plan mode (Shift+Tab)
- Explore first, plan second, implement third, commit fourth
- Use /clear between unrelated tasks
- Use /compact when context grows — preserve test output and code changes
- Use subagents for investigation to keep main context clean

## Git Safety — CRITICAL

- NEVER force push, reset --hard, or skip hooks (--no-verify) unless explicitly told
- NEVER commit secrets, binaries, backups, or generated files
- NEVER include Co-Authored-By lines in commits — the pre-commit hook blocks them
- NEVER add "Generated with Claude Code" or emoji watermarks to PRs, commits, or code
- Small, focused commits over large monolithic ones

## Commit Messages — IMPORTANT

Format: `type: concise imperative statement` (lowercase after colon, no period)
Types: feat, fix, docs, test, refactor, chore, perf, ci, build
- ONE line. Max 72 chars. Say WHAT changed, not every detail of HOW
- NEVER write changelog-style commit messages

## Anti-Patterns — NEVER Do These

- NEVER add ML, anomaly detection, or probabilistic approaches — all detection must be deterministic
- NEVER add dashboard, graphing, or visualization features — infranow shows failures, not metrics
- NEVER use time.Sleep for synchronization — use channels, sync.WaitGroup, or context
- NEVER skip error handling — always check returned errors
- NEVER use init() functions unless absolutely necessary
- NEVER use global mutable state
- All detectors MUST use explicit PromQL queries with defined thresholds
- NEVER add features, refactor code, or make improvements beyond what was asked
- NEVER add docstrings/comments/types to code you did not change
- NEVER create helpers or abstractions for one-time operations
- NEVER design for hypothetical future requirements
- NEVER create documentation files unless explicitly requested
- NEVER suppress errors or bypass safety checks as shortcuts
- NEVER remove existing CI jobs (security scans, coverage, build matrix) when updating workflows — only add or modify

## Token Efficiency

- Keep CLAUDE.md lean — every line must earn its place
- Use /clear between tasks. Stale context wastes tokens on every message
- Point to docs/ files instead of inlining documentation
- Use CLI tools (gh, aws) over MCP servers when possible
- Delegate verbose operations (test runs, log processing) to subagents

## Compact Instructions

When compacting, preserve: list of modified files, test commands used, current task state, any error messages being debugged. Discard: exploration of rejected approaches, file contents already committed.
