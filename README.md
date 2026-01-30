# infranow

> Attention-first monitoring and analysis of infrastructure metrics — focused on what matters right now.

## What is infranow?

infranow is a CLI/TUI tool that consumes Prometheus metrics and deterministically identifies the most important infrastructure problems right now. It prioritizes silence when systems are healthy and surfaces only ranked, actionable problems when intervention is required.

**Not**: dashboards, alerting, anomaly detection, or ML/AI-powered insights.
**Is**: a focused, real-time triage tool for operators under pressure.

## Features

- **Attention-first design**: Empty screen when healthy, problems appear automatically
- **Entity-agnostic**: Works across Kubernetes, Kafka, databases, generic services
- **Deterministic**: Rule-based detection, no ML/AI, reproducible results
- **Real-time TUI**: Interactive terminal UI with scrolling, sorting, pause/resume
- **JSON export**: Structured output for automation and scripting
- **Bounded context**: One Prometheus source per instance
- **Composable**: Run multiple terminals for multiple sources

## Installation

### From Source

```bash
go install github.com/ppiankov/infranow/cmd/infranow@latest
```

### Binary Download

Download pre-built binaries from the [releases page](https://github.com/ppiankov/infranow/releases).

### Build from Source

```bash
git clone https://github.com/ppiankov/infranow.git
cd infranow
make build
./bin/infranow --version
```

## Quick Start

### Monitor Everything

```bash
infranow monitor --prometheus-url http://localhost:9090
```

### Focus on Specific Namespace

```bash
infranow monitor --prometheus-url http://prom:9090 --namespace "prod-.*"
```

### Only Show Critical+ Problems

```bash
infranow monitor --prometheus-url http://prom:9090 --min-severity CRITICAL
```

### Export JSON Snapshot

```bash
infranow monitor --prometheus-url http://prom:9090 --output json > report.json
```

### Faster Refresh Rate

```bash
infranow monitor --prometheus-url http://prom:9090 --refresh-interval 5s
```

## Detectors

infranow MVP includes 7 built-in detectors:

### Kubernetes Detectors

- **OOMKillDetector**: Container OOM kills
- **CrashLoopBackOffDetector**: Pod startup failures
- **ImagePullBackOffDetector**: Image pull failures
- **PodPendingDetector**: Unschedulable pods stuck in Pending state

### Generic Detectors

- **HighErrorRateDetector**: HTTP 5xx error rates above 5%
- **DiskSpaceDetector**: Filesystem usage above 90% (WARNING) or 95% (CRITICAL)
- **HighMemoryPressureDetector**: Node memory usage above 90%

See [docs/DETECTORS.md](docs/DETECTORS.md) for detailed detector documentation.

## Usage

### Command Line Interface

```
infranow monitor [flags]

Flags:
  --prometheus-url URL             Prometheus endpoint (required)
  --prometheus-timeout DURATION    Query timeout (default: 30s)
  --namespace REGEX                Filter by namespace pattern
  --entity-type TYPE               Filter by entity type
  --min-severity SEVERITY          Minimum severity (FATAL, CRITICAL, WARNING)
  --refresh-interval DURATION      Detection refresh rate (default: 10s)
  --output FORMAT                  Output format: table (TUI) or json
  --export-file PATH               Export problems to file

Global Flags:
  --config FILE                    Config file (default: $HOME/.infranow.yaml)
  -v, --verbose                    Enable verbose logging
```

### Interactive TUI

The TUI displays problems automatically as they're detected, ranked by importance.

**Keyboard Shortcuts**:
- `q` or `Ctrl+C` - Quit
- `p` or `Space` - Pause/Resume detection
- `s` - Cycle sort mode (severity → recency → count)
- `↑`/`↓` or `j`/`k` - Scroll up/down
- `Home`/`g` - Jump to top
- `End`/`G` - Jump to bottom
- `PgUp`/`PgDown` - Page up/down

**Empty State**: When no problems are detected, the screen shows:
```
✓ No problems detected
```

**Problem Display**: Problems are shown with:
- Severity icon and level (🔴 FATAL, 🟠 CRITICAL, 🟡 WARNING)
- Problem title and entity identifier
- First seen timestamp and occurrence count
- Actionable hint for remediation

### JSON Output

Use `--output json` for machine-readable output:

```json
{
  "metadata": {
    "prometheus_url": "http://prometheus:9090",
    "timestamp": "2026-01-30T12:34:56Z",
    "refresh_interval": "10s"
  },
  "summary": {
    "total_problems": 3,
    "fatal": 1,
    "critical": 1,
    "warning": 1
  },
  "problems": [
    {
      "id": "prod/payment-api/worker-7d8f9/crashloop",
      "entity": "prod/payment-api/worker-7d8f9",
      "entity_type": "kubernetes_pod",
      "type": "crashloopbackoff",
      "severity": "FATAL",
      "title": "Pod CrashLoopBackOff",
      "message": "Pod prod/payment-api is in CrashLoopBackOff state",
      "first_seen": "2026-01-30T12:29:56Z",
      "last_seen": "2026-01-30T12:34:56Z",
      "count": 12,
      "hint": "Application startup failure or fatal runtime error"
    }
  ]
}
```

## Philosophy

infranow follows attention-first design principles:

1. **Tools should stay quiet when everything works** - Empty screen is success
2. **Problems should appear automatically** - No manual investigation required
3. **Ranked by importance** - Most critical issues surface first
4. **Evidence over recommendations** - Show data, provide hints, not prescriptions
5. **Deterministic over ML/AI** - Reproducible, explainable results
6. **Composable over integrated** - Run multiple instances for multiple sources

## Architecture

```
infranow/
├── cmd/infranow/          # CLI entry point
├── internal/
│   ├── cli/               # Cobra command implementations
│   ├── detector/          # Problem detection logic
│   ├── metrics/           # Prometheus client
│   ├── models/            # Core data structures
│   ├── monitor/           # Watcher and TUI
│   └── util/              # Utilities
└── docs/                  # Documentation
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Requirements

- Go 1.23+ (for building from source)
- Prometheus server with metrics exposure
- Terminal with Unicode support (for TUI icons)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development

```bash
# Clone repository
git clone https://github.com/ppiankov/infranow.git
cd infranow

# Install dependencies
make deps

# Build binary
make build

# Run tests
make test

# Run linters
make lint

# Format code
make fmt
```

## Roadmap

### MVP (v0.1.0) ✅
- [x] Monitor mode with TUI
- [x] Kubernetes detectors (OOMKill, CrashLoop, ImagePull, Pending)
- [x] Generic detectors (ErrorRate, DiskSpace, Memory)
- [x] JSON export
- [x] Problem scoring and ranking

### Future (Post-MVP)
- [ ] Historical analysis commands
- [ ] Additional detectors (Kafka, databases, custom)
- [ ] Multi-Prometheus aggregation
- [ ] Alert integration (PagerDuty, Slack)
- [ ] Config file-based detector customization
- [ ] Detector plugin system
- [ ] Web UI

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/ppiankov/infranow/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ppiankov/infranow/discussions)

---

**Remember**: The best monitoring tool is the one that's silent when everything works. 🎯
