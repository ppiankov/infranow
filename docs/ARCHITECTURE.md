# Architecture Documentation

This document describes the architecture and design decisions of infranow.

## Overview

infranow is built around a simple, focused architecture:

```
┌─────────────┐
│  CLI (Cobra)│
└──────┬──────┘
       │
       ├──► Monitor Command
       │     ├─► Prometheus Client
       │     ├─► Detector Registry
       │     ├─► Watcher (Orchestrator)
       │     └─► TUI (Bubbletea) / JSON Output
       │
       └──► Future Commands (analyze, etc.)
```

## Core Components

### 1. CLI Layer (`internal/cli/`)

**Responsibility**: Command-line interface and user interaction

- Uses Cobra for command structure
- Parses flags and validates input
- Wires up components (provider, registry, watcher, UI)
- Handles output formatting (TUI vs JSON)

**Files**:
- `root.go` - Root command with global flags
- `monitor.go` - Monitor command implementation

**Design Decisions**:
- Cobra chosen for consistency with ecosystem tools (kubectl, helm)
- Separate commands for different modes (monitor, analyze, etc.)
- Global flags for config file and verbosity

---

### 2. Metrics Layer (`internal/metrics/`)

**Responsibility**: Backend-agnostic metrics access

**Interface**:
```go
type MetricsProvider interface {
    QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Matrix, error)
    QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error)
    Health(ctx context.Context) error
}
```

**Implementations**:
- `PrometheusClient` - Prometheus backend
- `MockProvider` - Testing

**Files**:
- `interface.go` - MetricsProvider interface
- `prometheus.go` - Prometheus implementation
- `query.go` - PromQL query builder utilities
- `mock.go` - Mock for testing

**Design Decisions**:
- Interface allows future backends (Thanos, VictoriaMetrics, etc.)
- Query builder provides type-safe PromQL construction
- Context-aware for timeout and cancellation

---

### 3. Models Layer (`internal/models/`)

**Responsibility**: Core data structures

**Problem Model**:
```go
type Problem struct {
    // Identity
    ID         string
    Entity     string
    EntityType string
    Type       string

    // Classification
    Severity   Severity
    Title      string
    Message    string

    // Temporal
    FirstSeen  time.Time
    LastSeen   time.Time
    Count      int

    // Impact
    BlastRadius int
    Persistence float64
    Volatility  float64

    // Context
    Labels     map[string]string
    Metrics    map[string]float64
    Hint       string
}
```

**Scoring Algorithm**:
```
Score = SeverityWeight × BlastRadiusMultiplier × PersistenceMultiplier

SeverityWeight:
- FATAL: 100
- CRITICAL: 50
- WARNING: 10

BlastRadiusMultiplier = 1.0 + (BlastRadius × 0.1)
PersistenceMultiplier = 1.0 + (Persistence / 3600)  // Hours
```

**Design Decisions**:
- Severity levels match operational urgency
- Problem ID is deterministic (entity + type) for deduplication
- Score algorithm prioritizes severity, then blast radius, then persistence
- All timestamps in UTC for consistency

---

### 4. Detector Layer (`internal/detector/`)

**Responsibility**: Problem detection logic

**Interface**:
```go
type Detector interface {
    Name() string
    EntityTypes() []string
    Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error)
    Interval() time.Duration
}
```

**Registry**:
```go
type Registry struct {
    detectors map[string]Detector
}
```

**Implementations**:
- `kubernetes.go` - Kubernetes-specific detectors
- `generic.go` - Generic infrastructure detectors
- Future: `kafka.go`, `database.go`, etc.

**Design Decisions**:
- Each detector is independent and stateless
- Detectors run at their own intervals (30-60s typical)
- Registry pattern allows dynamic detector management
- Detectors return problems, don't store state
- Context-aware for graceful shutdown

---

### 5. Monitor Layer (`internal/monitor/`)

**Responsibility**: Real-time monitoring orchestration and UI

**Watcher**:
```go
type Watcher struct {
    provider   metrics.MetricsProvider
    registry   *detector.Registry
    problems   map[string]*models.Problem
    updateChan chan struct{}
}
```

**Responsibilities**:
- Run detectors concurrently at their intervals
- Deduplicate and merge problems by ID
- Track problem persistence (FirstSeen, LastSeen, Count)
- Prune stale problems (not seen in 2 minutes)
- Notify UI on state changes

**TUI (Bubbletea)**:
```go
type Model struct {
    watcher   *Watcher
    problems  []*models.Problem
    sortMode  SortMode
    paused    bool
    viewport  viewport.Model
}
```

**Responsibilities**:
- Display problems in interactive TUI
- Support keyboard navigation (scroll, sort, pause)
- Show empty state when healthy
- Update automatically on problem changes

**Design Decisions**:
- Watcher owns problem state, detectors are stateless
- Concurrent detector execution for performance
- Thread-safe problem map with RWMutex
- Channel-based UI notification (debounced)
- Bubbletea for terminal UI (Elm architecture)
- Viewport for scrolling large problem lists

---

### 6. Util Layer (`internal/util/`)

**Responsibility**: Shared utilities

**Exit Codes**:
- 0: Success
- 2: Invalid input (bad flags, invalid config)
- 3: Runtime error (connection failure, query error)

**Design Decisions**:
- Standard exit codes for shell integration
- Consistent with Spectre Tools conventions

---

## Data Flow

### Monitor Mode Startup

```
1. CLI parses flags and validates input
2. Create Prometheus client and health check
3. Create detector registry and register all detectors
4. Create watcher with provider and registry
5. Start watcher in background (goroutine)
6. Launch TUI or run JSON export
7. Wait for Ctrl+C or completion
8. Clean shutdown (cancel context, wait for goroutines)
```

### Detection Cycle

```
1. Watcher starts detector goroutines
2. Each detector runs at its interval:
   a. Execute PromQL query via provider
   b. Parse results and create Problem objects
   c. Return problems to watcher
3. Watcher updates problem state:
   a. Deduplicate by Problem.ID
   b. Update LastSeen and Count for existing
   c. Set FirstSeen for new problems
   d. Prune stale problems
   e. Notify UI via updateChan
4. TUI receives update and re-renders
```

### Problem Lifecycle

```
New Problem:
- FirstSeen = now
- LastSeen = now
- Count = 1

Seen Again:
- Count++
- LastSeen = now
- Persistence updated

Stale (not seen in 1 minute):
- Removed from problem map
- UI updated automatically
```

---

## Design Principles

### 1. Attention-First

**Principle**: Tools should be silent when everything works.

**Implementation**:
- Empty screen shows "✓ No problems detected"
- Problems appear automatically without user action
- Ranked by importance (Score algorithm)
- Stale problems auto-pruned

### 2. Deterministic Detection

**Principle**: Reproducible, explainable results.

**Implementation**:
- Rule-based detectors with fixed thresholds
- No ML/AI/anomaly detection
- PromQL queries are version-controlled
- Same metrics → same problems

### 3. Entity-Agnostic

**Principle**: Works across infrastructure types.

**Implementation**:
- Unified Problem abstraction
- Entity type is a label, not a constraint
- Detectors specify EntityTypes but share Problem model
- Consistent UX regardless of source

### 4. Bounded Context

**Principle**: One Prometheus source per instance.

**Implementation**:
- No multi-Prometheus aggregation in MVP
- Clear scope (one cluster, one environment)
- Compose with multiple terminals for multiple sources
- Simpler state management

### 5. Composable

**Principle**: Unix philosophy - do one thing well.

**Implementation**:
- JSON output for piping to other tools
- Exit codes for shell integration
- No built-in alerting (delegate to PagerDuty, etc.)
- Runs standalone, no daemon/server

---

## Concurrency Model

### Goroutine Hierarchy

```
main()
  └─► CLI Execute()
       └─► runMonitor()
            ├─► Watcher.Start() (goroutine)
            │    ├─► runDetector(OOMKill) (goroutine)
            │    ├─► runDetector(CrashLoop) (goroutine)
            │    ├─► runDetector(DiskSpace) (goroutine)
            │    └─► ... (one per detector)
            │
            └─► TUI Run() (main goroutine)
                 ├─► Update loop
                 └─► Render loop
```

### Synchronization

**Problem Map**:
- Protected by `sync.RWMutex`
- Write: detector results (updateProblems)
- Read: UI queries (GetProblems, GetSummary)

**Update Channel**:
- Buffered (size 1) for debouncing
- Non-blocking send (select with default)
- UI receives and re-fetches problems

### Shutdown

```
1. User presses Ctrl+C or 'q'
2. Cancel context propagates to all goroutines
3. Detectors stop on next iteration
4. Watcher waits for all detectors (WaitGroup)
5. TUI exits gracefully
6. Main returns with exit code 0
```

---

## Testing Strategy

### Unit Tests

**Detectors**:
- Test with MockProvider
- Verify problem creation logic
- Test edge cases (no data, partial data)
- Assert correct severity, labels, hints

**Models**:
- Test Score calculation
- Test Persistence updates
- Test severity ordering

**Watcher**:
- Test problem deduplication
- Test stale problem cleanup
- Test concurrent access

### Integration Tests

**Future**:
- Test against real Prometheus (testcontainers)
- End-to-end TUI tests
- JSON output validation

### Coverage

- Target: >80% line coverage
- Focus on core logic (detectors, watcher, models)
- Mock external dependencies (Prometheus)

---

## Performance Considerations

### Query Efficiency

- Use instant queries for current state
- Limit range queries to necessary windows (5m typical)
- Avoid expensive aggregations in PromQL
- Context timeouts prevent hanging queries

### Memory Usage

- Problem map bounded by detector count × entity count
- Stale problem cleanup prevents unbounded growth
- TUI viewport limits rendered content
- No historical data storage

### CPU Usage

- Detector intervals (30-60s) spread load
- Concurrent execution maximizes throughput
- No polling loops (event-driven UI updates)
- Efficient problem deduplication (map lookup)

---

## Future Architecture

### Post-MVP Extensions

**Historical Analysis**:
```
infranow analyze --start 1h --end now --prometheus-url ...
```
- Query metrics over time range
- Identify patterns and trends
- Generate reports

**Multi-Prometheus Aggregation**:
```
infranow aggregate --config multi-cluster.yaml
```
- Merge problems from multiple sources
- Unified view across clusters
- Requires state synchronization

**Alert Integration**:
```
infranow alert --pagerduty --config alerts.yaml
```
- Forward problems to PagerDuty, Slack, etc.
- Deduplication and grouping
- Acknowledgment tracking

**Detector Plugins**:
```
infranow monitor --plugin ./my-detector.so
```
- Load detectors at runtime
- Go plugin system or gRPC
- Community-contributed detectors

---

## Configuration (Future)

```yaml
# ~/.infranow.yaml
prometheus:
  url: http://localhost:9090
  timeout: 30s

detectors:
  kubernetes_oom_kills:
    enabled: true
    interval: 30s

  disk_space:
    enabled: true
    warning_threshold: 0.85
    critical_threshold: 0.95

filters:
  namespace: "prod-.*"
  min_severity: WARNING

ui:
  refresh_interval: 10s
  default_sort: severity
```

---

## Dependencies

### Core Libraries

- **github.com/spf13/cobra**: CLI framework
- **github.com/prometheus/client_golang**: Prometheus client
- **github.com/prometheus/common**: Prometheus data models
- **github.com/charmbracelet/bubbletea**: TUI framework
- **github.com/charmbracelet/lipgloss**: TUI styling
- **github.com/charmbracelet/bubbles**: TUI components (viewport)

### Build Tools

- Go 1.25+
- Make
- golangci-lint (optional)

---

## Operational Considerations

### Deployment

- Single binary, no runtime dependencies
- No persistent storage required
- No server/daemon mode (runs in foreground)
- Kubernetes: Run as kubectl plugin or local binary

### Monitoring infranow

- Exit codes indicate success/failure
- Verbose mode logs queries and errors
- No self-metrics (future: Prometheus exporter)

### Troubleshooting

**No problems detected**:
- Verify Prometheus connectivity
- Check detector queries in Prometheus UI
- Use `--verbose` to see query details

**High CPU usage**:
- Increase detector intervals
- Reduce number of active detectors
- Check Prometheus query performance

**Stale problems**:
- Check detector interval vs refresh interval
- Verify problem ID stability
- Review problem cleanup logic

---

## Security

### Prometheus Access

- No authentication in MVP
- Assumes trusted network or kubectl port-forward
- Future: Support for bearer tokens, mTLS

### Data Handling

- No persistent storage of metrics
- No PII in problem data (only metric labels)
- Prometheus labels may contain sensitive info (operator responsibility)

### Input Validation

- Flag validation before execution
- Context timeouts prevent resource exhaustion
- PromQL injection not possible (queries are hardcoded)

---

## Contributing to Architecture

When proposing architectural changes:

1. Start with a GitHub issue or discussion
2. Explain the problem being solved
3. Propose the design with diagrams
4. Consider backward compatibility
5. Update this document with approved changes

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.
