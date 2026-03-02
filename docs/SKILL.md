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

Real-time problem detection. Runs one cycle in JSON mode then exits, or loops in TUI mode.

**Flags:**
- `--prometheus-url` — Prometheus endpoint URL (required unless using --k8s-service)
- `--format json` — output format: table, json (default: table)
- `--k8s-service` — Kubernetes service name for auto port-forward
- `--k8s-namespace` — Kubernetes namespace for service (default: monitoring)
- `--namespace` — filter by namespace pattern (regex)
- `--min-severity` — minimum severity: FATAL, CRITICAL, WARNING (default: WARNING)
- `--refresh-interval` — detection refresh rate (default: 10s)
- `--export-file` — export problems to file
- `--save-baseline` — save problems snapshot to file
- `--compare-baseline` — compare current problems to baseline file
- `--fail-on-drift` — exit 1 if new problems detected vs baseline
- `--fail-on` — exit 1 if problems at/above severity
- `--include-namespaces` — comma-separated namespace patterns to include
- `--exclude-namespaces` — comma-separated namespace patterns to exclude
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
- 1: problems at/above --fail-on severity, or drift detected

### infranow version

Print version in single-line format: `infranow 0.1.2 (commit: abc1234, built: 2026-02-21T12:34:56Z, go: go1.25.7)`

Use `--json` for machine-readable output.

### infranow init

Not implemented. No config file required.

## What this does NOT do

- Does not write to Prometheus or Kubernetes — read-only PromQL queries
- Does not install CRDs, agents, or controllers — zero cluster footprint
- Does not use ML or anomaly detection — deterministic thresholds only
- Does not store state — all in-memory, exits clean

## Parsing examples

```bash
# One-shot JSON
infranow monitor --prometheus-url http://localhost:9090 --format json

# List critical+ problems
infranow monitor --prometheus-url "$PROM_URL" --format json | jq '.problems[] | select(.severity == "CRITICAL" or .severity == "FATAL")'

# Count by severity
infranow monitor --prometheus-url "$PROM_URL" --format json | jq '.summary'

# CI gate
infranow monitor --prometheus-url http://localhost:9090 --format json --fail-on CRITICAL

# Baseline drift
infranow monitor --prometheus-url "$PROM_URL" --format json --save-baseline baseline.json
infranow monitor --prometheus-url "$PROM_URL" --format json --compare-baseline baseline.json --fail-on-drift
```
