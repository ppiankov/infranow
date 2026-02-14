# Work Orders — infranow

## WO-01: Trustwatch Metrics Detector

**Goal:** Add detector that consumes trustwatch Prometheus metrics for unified cert expiry alerting.

### Context
trustwatch exposes these metrics when deployed with its Helm chart + ServiceMonitor:
- `trustwatch_cert_expires_in_seconds{source, namespace, name}` — seconds until cert expires
- `trustwatch_probe_success{source, namespace, name}` — 1=ok, 0=probe failed
- `trustwatch_findings_total{severity}` — total findings count

infranow already has Linkerd and Istio cert detectors in `internal/detector/servicemesh_certs.go`. The trustwatch detector follows the same pattern but covers a broader trust surface (webhooks, API aggregation, external TLS, mesh issuers).

### Steps
1. Create `internal/detector/trustwatch_certs.go`
2. Create `internal/detector/trustwatch_certs_test.go`
3. Register in `internal/detector/registry.go`
4. Follow existing pattern from `servicemesh_certs.go`:
   - Stateless: PromQL query in, `[]*Problem` out
   - Tiered severity: WARNING (7d), CRITICAL (48h), FATAL (24h)
   - Same threshold constants already defined in `servicemesh_certs.go`

### PromQL Queries
```promql
# Certs expiring within 7 days
trustwatch_cert_expires_in_seconds < 604800

# Probe failures (unreachable TLS endpoints)
trustwatch_probe_success == 0
```

### Problem Fields
- Entity: `trustwatch/{source}/{namespace}/{name}` (e.g. `trustwatch/webhook/kube-system/cert-manager-webhook`)
- Title: "Certificate expiring in {remaining}" or "TLS probe failed"
- Hint: "Run: trustwatch now --context {ctx}" for probe failures
- Type: `trustwatch_cert_expiry` or `trustwatch_probe_failure`

### Acceptance
- `make test` passes with -race
- Detector appears in `docs/DETECTORS.md`
- Works when trustwatch is not installed (query returns empty = no problems)

---

## WO-02: TUI Copy Support

**Goal:** Allow copying problem details from TUI to clipboard.

### Context
trustwatch TUI uses `charmbracelet/bubbles/table` with a detail panel below the table. Users can select rows and see full details. infranow uses `charmbracelet/bubbles/viewport` for scrolling rendered text — no row selection, no detail panel, no way to copy.

### What trustwatch has that infranow doesn't
- **Table widget** (`bubbles/table`) with selectable rows and focused cursor
- **Detail panel** below table showing full finding details for selected row
- **Number keys 1-9** for jumping to specific row
- **Row-based navigation** — cursor highlights a single problem
- **PlainText() function** for piped/non-TTY output

### Steps
1. Replace viewport-based problem rendering with `bubbles/table`
2. Add detail panel below table (entity, message, hint, first seen, count)
3. Add `c` key to copy selected problem details to clipboard (`golang.design/x/clipboard` or exec `pbcopy`)
4. Add `y` key to yank entity name only (for quick paste into kubectl)
5. Keep existing: search (`/`), sort (`s`), pause (`p`), scroll (`j/k/g/G`)
6. Add number keys `1-9` for row jump (trustwatch pattern)

### Acceptance
- `c` copies full problem detail to system clipboard
- `y` copies entity name to clipboard
- Detail panel shows selected problem
- `make test` passes with -race

---

## WO-03: Code Style Alignment Audit

**Goal:** Audit infranow codebase for alignment with current coding conventions.

### Scope
Review against CLAUDE.md conventions and trustwatch patterns:

1. **Error handling** — verify all errors are wrapped with `fmt.Errorf("context: %w", err)`
2. **Constants** — check for magic numbers (thresholds, timeouts, intervals)
3. **Test coverage** — verify >85% on detector, models, baseline packages
4. **Test patterns** — ensure table-driven tests, no flaky timing-dependent tests
5. **Package API** — ensure clean interfaces (detector.Detector, metrics.MetricsProvider)
6. **Struct tags** — consistent json tags on all exported structs
7. **Comment style** — "why" not "what", no decorative comments, no stale TODOs
8. **Early returns** — replace nested if/else with guard clauses
9. **golangci-lint** — run with trustwatch's linter config, fix any new warnings

### Steps
1. Run `make lint` and fix warnings
2. Run `go vet ./...` and fix issues
3. Review each detector for consistent pattern adherence
4. Review TUI code for consistency with trustwatch patterns
5. Check test coverage: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
6. Fix gaps, one commit per category

### Acceptance
- `make lint` clean
- `go vet ./...` clean
- Coverage >85% on detector, models, baseline packages
- No magic numbers in detector logic

---

## WO-04: PlainText Output Mode

**Goal:** Add non-interactive output for piped/CI usage.

### Context
trustwatch has `PlainText()` that outputs a formatted table when stdout is not a TTY. infranow always launches the TUI.

### Steps
1. Detect `!isatty(stdout)` → print plain text instead of launching TUI
2. Add `--format text|json` flag to `monitor` command
3. Plain text: tabular output matching TUI columns (severity, entity, title, age, count)
4. JSON: array of Problem structs
5. Exit code: 0 = no problems, 1 = warnings only, 2 = critical/fatal found

### Acceptance
- `infranow monitor --prom-url http://... | head` prints table
- `infranow monitor --prom-url http://... --format json | jq .` works
- Exit codes reflect severity
- `make test` passes

---

## Non-Goals

- No web UI or dashboard
- No metric collection or storage (infranow reads, never writes)
- No ML or anomaly detection
- No alerting or notification system (infranow shows, Prometheus alerts)
