# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-01-30

### Added

- Initial MVP release
- `monitor` command for real-time infrastructure monitoring
- Interactive TUI with keyboard navigation (scroll, sort, pause/resume)
- JSON export mode for automation and scripting
- Kubernetes detectors:
  - OOMKillDetector - Detects container OOM kills
  - CrashLoopBackOffDetector - Detects pod startup failures
  - ImagePullBackOffDetector - Detects image pull failures
  - PodPendingDetector - Detects unschedulable pods
- Generic detectors:
  - HighErrorRateDetector - Detects HTTP 5xx error rates above 5%
  - DiskSpaceDetector - Detects low disk space (WARNING at 90%, CRITICAL at 95%)
  - HighMemoryPressureDetector - Detects high memory usage above 90%
- Problem scoring and ranking system
- Prometheus metrics provider with health checking
- Detector registry for pluggable detectors
- Watcher for concurrent problem detection
- Empty state display when no problems detected
- Problem persistence tracking and stale problem cleanup
- Comprehensive test suite with >80% coverage
- Multi-platform builds (Linux, macOS, Windows × amd64/arm64)
- CI/CD pipeline with GitHub Actions
- Documentation: README, CONTRIBUTING, ARCHITECTURE, DETECTORS

### Philosophy

- Attention-first design: silence when healthy
- Deterministic detection: no ML/AI
- Entity-agnostic: works across infrastructure types
- Bounded context: one Prometheus per instance
- Composable: run multiple terminals for multiple sources

[Unreleased]: https://github.com/ppiankov/infranow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ppiankov/infranow/releases/tag/v0.1.0
