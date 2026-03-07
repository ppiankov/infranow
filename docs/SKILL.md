---
name: infranow
description: Real-time infrastructure triage — deterministic problem detection for Kubernetes and Prometheus
user-invocable: false
metadata: {"requires":{"bins":["infranow"]}}
---

# infranow — Real-Time Infrastructure Triage

Consumes Prometheus metrics and deterministically identifies the most important infrastructure problems right now. 15 built-in detectors, ranked by severity and persistence.

## Install

```bash
brew install ppiankov/tap/infranow
```

## Commands

### infranow monitor

Real-time problem detection. Runs one cycle in non-TUI modes then exits, or loops in TUI mode.

**Flags:**
- `--prometheus-url` — Prometheus endpoint URL (required unless using --k8s-service)
- `--prometheus-timeout` — Prometheus query timeout (default: 30s)
- `--output` — output format: table, text, json, sarif (default: table, auto-detects piped stdout)
- `--once` — run one detection cycle and exit
- `--k8s-service` — Kubernetes service name for auto port-forward
- `--k8s-namespace` — Kubernetes namespace for service (default: monitoring)
- `--k8s-local-port` — local port for port-forward (default: 9090)
- `--k8s-remote-port` — remote port for port-forward (default: 9090)
- `--namespace` — filter by namespace pattern (regex)
- `--min-severity` — minimum severity: FATAL, CRITICAL, WARNING (default: WARNING)
- `--refresh-interval` — detection refresh rate (default: 10s)
- `--max-concurrency` — max concurrent detector executions (0 = unlimited)
- `--detector-timeout` — detector execution timeout (default: 30s)
- `--export-file` — export problems to file
- `--save-baseline` — save problems snapshot to file
- `--compare-baseline` — compare current problems to baseline file
- `--fail-on-drift` — exit 1 if new problems detected vs baseline
- `--fail-on` — exit with error if problems at/above severity
- `--include-namespaces` — comma-separated namespace patterns to include
- `--exclude-namespaces` — comma-separated namespace patterns to exclude
- `--history` — enable problem history tracking (local SQLite)
- `--history-db` — history database path (env: INFRANOW_HISTORY_DB)
- `--verbose` — enable verbose logging

**JSON output:**
```json
{
  "problems": [
    {
      "id": "oomkill::production::payment-api::payment-api",
      "type": "oomkill",
      "entity": "payment-api",
      "namespace": "production",
      "severity": "CRITICAL",
      "message": "Container payment-api OOMKilled 3 times in last 5 minutes",
      "hint": "Check memory limits and application memory usage",
      "count": 3,
      "score": 52.5
    }
  ],
  "summary": {"total": 4, "fatal": 0, "critical": 2, "warning": 2}
}
```

**Exit codes:**
- 0: no problems (or below --fail-on threshold)
- 1: warnings found
- 2: critical/fatal problems found
- 3: invalid input
- 4: runtime error

### infranow sweep

Scan all kubeconfig contexts for problems. Port-forwards to each cluster's Prometheus, runs one detection cycle, produces unified report.

**Flags:**
- `--k8s-service` — Kubernetes service name for Prometheus (required)
- `--k8s-namespace` — Kubernetes namespace for Prometheus service (default: monitoring)
- `--k8s-remote-port` — remote Prometheus port (default: 9090)
- `--contexts` — comma-separated glob patterns for context filtering (e.g. 'prod-*')
- `--parallel` — scan clusters concurrently
- `--output` — output format: text, json, sarif (default: text)
- `--fail-on` — exit with error if problems at/above severity
- `--include-namespaces` — comma-separated namespace patterns
- `--exclude-namespaces` — comma-separated namespace patterns to exclude

### infranow history

Manage the local SQLite database that tracks problem recurrence across sessions.

**Subcommands:**
- `history list` — list historical problems
  - `--since` — show problems seen since (default: 7d)
  - `--min-severity` — filter by severity (WARNING, CRITICAL, FATAL)
  - `--limit` — max records (default: 100)
  - `--output` — text or json (default: text)
- `history prune` — remove old entries
  - `--older-than` — age threshold (default: 90d)
  - `--dry-run` — show count without deleting

### infranow version

Print version in single-line format: `infranow 0.3.0 (commit: abc1234, built: 2026-03-03T12:00:00Z, go: go1.25.7)`

Use `--json` for machine-readable output.

## What this does NOT do

- Does not write to Prometheus or Kubernetes — read-only PromQL queries
- Does not install CRDs, agents, or controllers — zero cluster footprint
- Does not use ML or anomaly detection — deterministic thresholds only
- Does not store state by default — opt-in local SQLite for history tracking only

## Parsing examples

```bash
# One-shot JSON
infranow monitor --prometheus-url http://localhost:9090 --output json

# List critical+ problems
infranow monitor --prometheus-url "$PROM_URL" --output json | jq '.problems[] | select(.severity == "CRITICAL" or .severity == "FATAL")'

# Count by severity
infranow monitor --prometheus-url "$PROM_URL" --output json | jq '.summary'

# CI gate
infranow monitor --prometheus-url http://localhost:9090 --output json --fail-on CRITICAL

# Baseline drift
infranow monitor --prometheus-url "$PROM_URL" --output json --save-baseline baseline.json
infranow monitor --prometheus-url "$PROM_URL" --output json --compare-baseline baseline.json --fail-on-drift

# Sweep all clusters
infranow sweep --k8s-service prometheus-operated

# Sweep prod clusters, JSON output
infranow sweep --k8s-service prometheus-operated --contexts "prod-*" --output json
```
