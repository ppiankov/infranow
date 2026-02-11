# infranow

[![CI](https://github.com/ppiankov/infranow/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/infranow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.25-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Real-time infrastructure triage -- deterministic problem detection for Kubernetes and Prometheus.

## What it is

infranow is a CLI/TUI tool that consumes Prometheus metrics and deterministically identifies the most important infrastructure problems right now. It runs 13 built-in detectors on a loop, ranks problems by severity and persistence, and presents them in an interactive terminal UI or as structured JSON.

When systems are healthy, the screen is empty. When something breaks, it appears immediately, ranked by importance. No dashboards. No graphs. No exploration.

## What it is NOT

- Not a dashboard or visualization tool
- Not a metric collector or storage engine
- Not an alerting system (no PagerDuty, Slack, webhooks)
- Not an anomaly detection or ML/AI system
- Not a replacement for Grafana, Datadog, or Prometheus itself
- Not a historical analysis tool

infranow shows what is failing right now. Nothing else.

## Philosophy

**Principiis obsta** -- resist the beginnings.

infranow is designed to surface active failures before damage spreads, then go silent when the problem resolves. The core principles:

- **Silence is the success condition.** Empty output means healthy systems.
- **Deterministic detection only.** Every detector is a PromQL query with an explicit threshold. No statistical models, no learning, no probability. The same metrics always produce the same result.
- **Evidence over recommendations.** Show the data, provide a hint, let the operator decide.
- **Attention is finite.** Problems are ranked by a score that combines severity, blast radius, and persistence. The most important problem is always first.
- **Bounded scope.** One Prometheus source per instance. Run multiple instances for multiple sources. No aggregation, no federation.

## Quick start

```bash
# Install from source
go install github.com/ppiankov/infranow/cmd/infranow@latest

# Or build locally
git clone https://github.com/ppiankov/infranow.git
cd infranow
make build

# Run against a Prometheus instance
./bin/infranow monitor --prometheus-url http://localhost:9090
```

## Usage

### Monitor mode (TUI)

```bash
infranow monitor --prometheus-url http://localhost:9090
```

The interactive TUI displays problems ranked by importance. Keyboard controls:

| Key | Action |
|-----|--------|
| `q`, `Ctrl+C` | Quit |
| `p`, `Space` | Pause/resume detection |
| `s` | Cycle sort: severity, recency, count |
| `j`/`k`, Up/Down | Scroll |
| `g`/`G` | Jump to top/bottom |
| `/` | Search/filter |
| `Esc` | Clear filter |

### JSON mode

```bash
infranow monitor --prometheus-url http://localhost:9090 --output json
```

Waits for the first detection cycle, then outputs all problems as JSON to stdout and exits. Suitable for CI/CD pipelines and scripting.

### Baseline compare

```bash
# Save a baseline snapshot
infranow monitor --prometheus-url http://prom:9090 --output json --save-baseline baseline.json

# Compare against baseline, fail if new problems appear
infranow monitor --prometheus-url http://prom:9090 --output json \
  --compare-baseline baseline.json --fail-on-drift
```

### Kubernetes port-forward

```bash
# Automatic port-forward management
infranow monitor --k8s-service prometheus-operated --k8s-namespace monitoring

# Custom ports
infranow monitor --k8s-service prometheus-operated \
  --k8s-namespace monitoring \
  --k8s-local-port 9091 --k8s-remote-port 9090
```

### CI/CD gate

```bash
# Exit 1 if any CRITICAL or FATAL problems exist
infranow monitor --prometheus-url http://prom:9090 --output json --fail-on CRITICAL
```

### All flags

```
infranow monitor [flags]

Connection:
  --prometheus-url string       Prometheus endpoint URL (required unless using --k8s-service)
  --prometheus-timeout duration Prometheus query timeout (default 30s)
  --k8s-service string          Kubernetes service name for port-forward
  --k8s-namespace string        Kubernetes namespace for service (default "monitoring")
  --k8s-local-port string       Local port for port-forward (default "9090")
  --k8s-remote-port string      Remote port for port-forward (default "9090")

Detection:
  --namespace string            Filter by namespace pattern (regex)
  --entity-type string          Filter by entity type
  --min-severity string         Minimum severity: FATAL, CRITICAL, WARNING (default "WARNING")
  --refresh-interval duration   Detection refresh rate (default 10s)
  --max-concurrency int         Max concurrent detector executions (0 = unlimited)
  --detector-timeout duration   Detector execution timeout (default 30s)

Output:
  --output string               Output format: table, json (default "table")
  --export-file string          Export problems to file

Baseline:
  --save-baseline string        Save problems snapshot to file
  --compare-baseline string     Compare current problems to baseline file
  --fail-on-drift               Exit 1 if new problems detected vs baseline

CI/CD:
  --fail-on string              Exit 1 if problems at/above severity (WARNING, CRITICAL, FATAL)
  --include-namespaces string   Comma-separated namespace patterns to include
  --exclude-namespaces string   Comma-separated namespace patterns to exclude

Global:
  --config string               Config file (default $HOME/.infranow.yaml)
  -v, --verbose                 Enable verbose logging
```

## Detectors

13 built-in detectors ship with infranow. Each runs independently at its own interval.

| Detector | Metric | Severity | Threshold | Interval |
|----------|--------|----------|-----------|----------|
| OOMKill | `kube_pod_container_status_restarts_total{reason="OOMKilled"}` | CRITICAL | > 0 restarts in 5m window | 30s |
| CrashLoopBackOff | `kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"}` | FATAL | Pod in state | 30s |
| ImagePullBackOff | `kube_pod_container_status_waiting_reason{reason=~"ImagePullBackOff\|ErrImagePull"}` | CRITICAL | Pod in state | 30s |
| PodPending | `kube_pod_status_phase{phase="Pending"}` | CRITICAL | Pending > 5 minutes | 30s |
| HighErrorRate | `rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])` | CRITICAL | > 5% error rate | 30s |
| DiskSpace | `1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)` | WARNING / CRITICAL | >= 90% / >= 95% | 60s |
| HighMemoryPressure | `1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)` | CRITICAL | > 90% usage | 30s |
| LinkerdControlPlane | `kube_deployment_status_replicas_available{namespace="linkerd"}` | FATAL | == 0 replicas | 30s |
| LinkerdProxyInjection | `kube_pod_container_status_waiting_reason{namespace="linkerd"}` | CRITICAL | CrashLoopBackOff | 30s |
| IstioControlPlane | `kube_deployment_status_replicas_available{namespace="istio-system"}` | FATAL | == 0 replicas | 30s |
| IstioSidecarInjection | `kube_pod_container_status_waiting_reason{namespace="istio-system"}` | CRITICAL | CrashLoopBackOff | 30s |
| LinkerdCertExpiry | `identity_cert_expiry_timestamp - time()` | WARNING / CRITICAL / FATAL | < 7d / < 48h / < 24h | 60s |
| IstioCertExpiry | `citadel_server_root_cert_expiry_timestamp - time()` | WARNING / CRITICAL / FATAL | < 7d / < 48h / < 24h | 60s |

See [docs/DETECTORS.md](docs/DETECTORS.md) for detailed documentation.

## Architecture

```
cmd/infranow/          Entry point. Minimal main.go, delegates to internal/cli.
internal/
  cli/                 Cobra commands. Root command + monitor subcommand.
  metrics/             MetricsProvider interface + PrometheusClient implementation.
  detector/            Detector interface + Registry + 7 concrete detectors.
  models/              Problem struct, Severity type, scoring logic.
  monitor/             Watcher (detection orchestrator) + Bubble Tea TUI.
  filter/              Post-detection namespace filtering (include/exclude globs).
  baseline/            Snapshot save/load and diff comparison.
  util/                Exit codes + Kubernetes port-forward via client-go.
```

Data flow:

```
Prometheus --> MetricsProvider --> Detectors --> Watcher --> TUI or JSON
                                                  |
                                          Filter + Baseline
                                                  |
                                           Exit code (CI/CD)
```

The Watcher runs each detector in its own goroutine at the detector's configured interval. Results are merged into a shared problem map (deduplicated by ID, count incremented on re-detection, pruned after 1 minute of staleness). The TUI subscribes to change notifications via a channel. JSON mode waits for the first detection cycle, then dumps and exits.

Problem score formula: `severity_weight * (1 + blast_radius * 0.1) * (1 + persistence / 3600)`. Severity weights: WARNING=10, CRITICAL=50, FATAL=100.

## Known limitations

- **No integration tests.** Unit test coverage is >80% but there are no integration tests against a live Prometheus instance.
- **No config file support.** The `--config` flag is accepted but not wired to anything yet.
- **Single Prometheus source.** No federation, no multi-source aggregation. By design, but worth noting.
- **No custom detectors.** Detector set is compiled in. No plugin system or config-driven detection yet.
- **Stale problem pruning is time-based.** Problems disappear after 1 minute without re-detection, regardless of whether the underlying issue resolved.

## Roadmap

### v0.1.1 (current)

- ~~Increase test coverage to >80% across all packages~~ (done)
- ~~Service mesh detectors for linkerd and istio~~ (done: 6 detectors)
- ~~Certificate expiry detection with tiered severity~~ (done)
- Integration tests with docker-compose + Prometheus
- Config file support (YAML)
- Custom detector thresholds via config

### v0.1.2

- SARIF output for GitHub Code Scanning integration
- Prometheus self-metrics endpoint
- Detector plugin system

### Future

- Additional detectors (Kafka, databases, custom services)
- Multi-Prometheus aggregation mode
- Web UI for remote access

## License

MIT License. See [LICENSE](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and the detector authoring guide.
