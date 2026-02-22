---
name: infranow
description: Real-time infrastructure triage — deterministic problem detection for Kubernetes and Prometheus
user-invocable: false
metadata: {"requires":{"bins":["infranow"]}}
---

# infranow — Real-Time Infrastructure Triage

You have access to `infranow`, a tool that consumes Prometheus metrics and deterministically identifies the most important infrastructure problems right now. It runs 15 built-in detectors on a loop, ranks problems by severity and persistence, and presents them in a TUI or as structured JSON.

## Install

```bash
brew install ppiankov/tap/infranow
```

## Commands

| Command | What it does |
|---------|-------------|
| `infranow monitor --prometheus-url <url>` | Real-time problem detection (TUI or JSON) |
| `infranow version` | Print version |

## Key Flags

| Flag | Description |
|------|-------------|
| `--prometheus-url` | Prometheus endpoint URL (required unless using --k8s-service) |
| `--prometheus-timeout` | Prometheus query timeout (default 30s) |
| `--k8s-service` | Kubernetes service name for auto port-forward |
| `--k8s-namespace` | Kubernetes namespace for service (default "monitoring") |
| `--k8s-local-port` | Local port for port-forward (default "9090") |
| `--k8s-remote-port` | Remote port for port-forward (default "9090") |
| `--namespace` | Filter by namespace pattern (regex) |
| `--entity-type` | Filter by entity type |
| `--min-severity` | Minimum severity: FATAL, CRITICAL, WARNING (default "WARNING") |
| `--refresh-interval` | Detection refresh rate (default 10s) |
| `--max-concurrency` | Max concurrent detector executions (0 = unlimited) |
| `--detector-timeout` | Detector execution timeout (default 30s) |
| `--output` | Output format: table, json (default "table") |
| `--export-file` | Export problems to file |
| `--save-baseline` | Save problems snapshot to file |
| `--compare-baseline` | Compare current problems to baseline file |
| `--fail-on-drift` | Exit 1 if new problems detected vs baseline |
| `--fail-on` | Exit 1 if problems at/above severity (WARNING, CRITICAL, FATAL) |
| `--include-namespaces` | Comma-separated namespace patterns to include |
| `--exclude-namespaces` | Comma-separated namespace patterns to exclude |
| `-v, --verbose` | Enable verbose logging |

## Detectors

15 built-in detectors, each a PromQL query with an explicit threshold:

| Detector | Severity | What it detects |
|----------|----------|-----------------|
| OOMKill | CRITICAL | Container OOM kills in last 5 minutes |
| CrashLoopBackOff | FATAL | Pod stuck in CrashLoopBackOff |
| ImagePullBackOff | CRITICAL | Pod stuck in ImagePullBackOff/ErrImagePull |
| PodPending | CRITICAL | Pod pending for > 5 minutes |
| HighErrorRate | CRITICAL | HTTP 5xx error rate > 5% |
| DiskSpace | WARNING/CRITICAL | Disk usage >= 90% / >= 95% |
| HighMemoryPressure | CRITICAL | Node memory usage > 90% |
| LinkerdControlPlane | FATAL | Linkerd deployment has 0 replicas |
| LinkerdProxyInjection | CRITICAL | Linkerd proxy CrashLoopBackOff |
| IstioControlPlane | FATAL | Istio deployment has 0 replicas |
| IstioSidecarInjection | CRITICAL | Istio sidecar CrashLoopBackOff |
| LinkerdCertExpiry | WARNING-FATAL | Linkerd certificate expiring (7d/48h/24h) |
| IstioCertExpiry | WARNING-FATAL | Istio certificate expiring (7d/48h/24h) |
| TrustwatchCerts | WARNING-FATAL | Trustwatch-managed certificate expiring |
| TrustwatchProbes | CRITICAL | Trustwatch probe endpoint failures |

## Agent Usage Pattern

```bash
# One-shot JSON output for CI/CD — runs one cycle then exits
infranow monitor --prometheus-url http://localhost:9090 --output json

# CI gate — fail if critical problems exist
infranow monitor --prometheus-url http://localhost:9090 --output json --fail-on CRITICAL

# Baseline comparison — detect new problems since last snapshot
infranow monitor --prometheus-url http://localhost:9090 --output json \
  --compare-baseline baseline.json --fail-on-drift

# Kubernetes port-forward — auto-discover Prometheus
infranow monitor --k8s-service prometheus-operated --k8s-namespace monitoring --output json

# Namespace filtering
infranow monitor --prometheus-url http://localhost:9090 --output json \
  --include-namespaces "production,staging" --min-severity CRITICAL
```

### JSON Output Structure

```json
{
  "problems": [
    {
      "id": "oomkill::production::payment-api::payment-api",
      "type": "oomkill",
      "entity": "payment-api",
      "entity_type": "container",
      "namespace": "production",
      "severity": "CRITICAL",
      "message": "Container payment-api OOMKilled 3 times in last 5 minutes",
      "hint": "Check memory limits and application memory usage",
      "first_seen": "2026-02-22T10:00:00Z",
      "last_seen": "2026-02-22T10:05:00Z",
      "count": 3,
      "score": 52.5
    }
  ],
  "summary": {
    "total": 4,
    "fatal": 0,
    "critical": 2,
    "warning": 2
  }
}
```

### Parsing Examples

```bash
# List all critical+ problems
infranow monitor --prometheus-url "$PROM_URL" --output json | \
  jq '.problems[] | select(.severity == "CRITICAL" or .severity == "FATAL")'

# Get unique namespaces with problems
infranow monitor --prometheus-url "$PROM_URL" --output json | \
  jq -r '[.problems[].namespace] | unique[]'

# Count problems by severity
infranow monitor --prometheus-url "$PROM_URL" --output json | \
  jq '.summary'

# Save baseline, then detect drift
infranow monitor --prometheus-url "$PROM_URL" --output json --save-baseline baseline.json
# ... later ...
infranow monitor --prometheus-url "$PROM_URL" --output json \
  --compare-baseline baseline.json --fail-on-drift
```

## Problem Scoring

Problems are ranked by: `severity_weight * (1 + blast_radius * 0.1) * (1 + persistence / 3600)`

Severity weights: WARNING=10, CRITICAL=50, FATAL=100. Higher score = more important.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No problems (or below --fail-on threshold) |
| `1` | Problems at/above --fail-on severity, or --fail-on-drift detected new problems |

## What infranow Does NOT Do

- Does not write to Prometheus or Kubernetes — read-only PromQL queries
- Does not install CRDs, agents, or controllers — zero cluster footprint
- Does not use ML or anomaly detection — deterministic thresholds only
- Does not store state — all in-memory, exits clean
- Does not open network ports — no servers started
