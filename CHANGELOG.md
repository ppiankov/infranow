# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-08

### Added

- 7 deterministic detectors with explicit PromQL thresholds
  - Kubernetes: OOMKill, CrashLoopBackOff, ImagePullBackOff, PodPending
  - Generic: HighErrorRate (>5% 5xx), DiskSpace (90%/95%), HighMemoryPressure (>90%)
- Interactive TUI with real-time problem ranking by severity, recency, or count
- JSON output mode for CI/CD pipelines (waits for first detection cycle, then exits)
- Baseline save/compare with `--fail-on-drift` for regression detection
- Severity gate via `--fail-on` for CI/CD exit code control
- Native Kubernetes port-forwarding via client-go (no kubectl dependency)
- Namespace include/exclude filtering with glob patterns
- Configurable polling intervals, concurrency limits, and detector timeouts
- Problem scoring based on severity weight, blast radius, and persistence
- Stale problem pruning (removed after 1 minute without re-detection)
- TUI keyboard navigation: scroll, sort, search/filter, pause/resume
- Prometheus health monitoring with connection status in TUI header
- Multi-platform builds via Makefile (Linux, macOS, Windows)

[Unreleased]: https://github.com/ppiankov/infranow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ppiankov/infranow/releases/tag/v0.1.0
