# Project: infranow

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
  - detector — Rule-based problem detection (7 detectors, each a PromQL query + threshold)
  - metrics — MetricsProvider interface + PrometheusClient implementation
  - models — Problem struct, Severity type, scoring logic
  - monitor — Watcher orchestration (concurrent detector loop) + Bubble Tea TUI
  - baseline — Save/load problem snapshots, diff comparison for drift detection
  - filter — Namespace include/exclude glob pattern filtering
  - util — Exit codes, native Kubernetes port-forward via client-go

## Conventions
- Minimal main.go — single Execute() call
- Internal packages: short single-word names
- Struct-based domain models with json tags
- Standard Go formatting (gofmt/goimports)
- Version injected via LDFLAGS at build time
- Detectors are stateless: PromQL query in, []*Problem out

## Anti-Patterns
- NEVER add ML, anomaly detection, or probabilistic approaches — all detection must be deterministic
- NEVER add dashboard, graphing, or visualization features — infranow shows failures, not metrics
- NEVER use time.Sleep for synchronization — use channels, sync.WaitGroup, or context
- NEVER skip error handling — always check returned errors
- NEVER use init() functions unless absolutely necessary
- NEVER use global mutable state
- All detectors MUST use explicit PromQL queries with defined thresholds

## Verification
- Run `make test` after code changes (includes -race)
- Run `make lint` before marking complete
- Run `go vet ./...` for suspicious constructs
