# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-03-11

### Added

- 3 tote image salvage detectors (total: 18 detectors)
  - ToteSalvageFailure: detects failing node-to-node image transfers (CRITICAL)
  - TotePushFailure: detects failing backup registry pushes (WARNING)
  - ToteHighFailureRate: detects when most image pull failures use tags instead of digests (WARNING)
- Runbook docs for all 3 tote detectors
- CLAUDE.md removed from git tracking (was committed since v0.1.0)

## [0.3.1] - 2026-03-11

### Fixed

- Port-forward errors now surface immediately instead of waiting 10s and returning generic "timeout" message

## [0.3.0] - 2026-03-03

### Added

- SARIF 2.1.0 output for GitHub Code Scanning (`--output sarif`)
- Cross-detector correlation with 6 built-in rules (deployment failure, memory pressure, mesh cascades)
- Runbook links for all 15 detectors with `?` key in TUI and HelpURI in SARIF
- Problem history with local SQLite backend (`--history` flag, `history list/prune` subcommands)
- Recurring problem badges in TUI, plain text, and SARIF output
- Multi-cluster sweep command (`infranow sweep`) — scans all kubeconfig contexts via port-forward
- Sweep parallel mode (`--parallel`), context glob filtering (`--contexts`), namespace filtering
- Sweep output in text, JSON, and SARIF formats with cluster annotations
- Table-based TUI with selectable rows, detail panel, copy/yank, and digit jump keys

### Changed

- TUI rebuilt with bubbles/table replacing viewport-based renderer
- Detail panel expanded to 7 lines (added history badge and runbook link)
- Named constants replace magic numbers throughout codebase

## [0.2.0] - 2026-03-02

### Added

- Plain text output mode (`--output text`) for piped and CI usage
- `--once` flag for single detection cycle then exit
- Auto-detection of piped stdout — falls back to text mode when not a TTY
- `--json` flag on `version` subcommand for machine-readable version output
- Tiered exit codes: 0=clean, 1=warnings, 2=critical/fatal
- GoReleaser config for automated builds, checksums, and Homebrew tap updates
- SKILL.md badge in README linking to docs/SKILL.md

### Changed

- Version output unified to single-line format: `infranow 0.2.0 (commit: X, built: Y, go: Z)`
- Release workflow replaced with GoReleaser action (from custom shell script)
- SKILL.md moved from repo root to docs/SKILL.md per ANCC convention
- Makefile LDFLAGS now include `-s -w` for smaller binaries
- Exit codes renumbered: invalid input=3, runtime error=4

### Fixed

- README detector table missing trustwatch detectors (now shows all 15)
- README architecture section showed 7 detectors instead of 15
- ARCHITECTURE.md stale problem pruning time said 2 minutes (actual: 1 minute)

## [0.1.2] - 2026-02-21

### Added

- Trustwatch certificate and probe failure detectors (2 new detectors, total: 15)
  - TrustwatchCertExpiry: tiered alerts for trustwatch-monitored certificate expiry
  - TrustwatchProbeFailure: detects trustwatch endpoint probe failures
- Safety Model section in README documenting zero-footprint guarantees
- golangci-lint config (.golangci.yml) with gocritic, gocyclo, revive enabled
- Trivy security scanning in CI pipeline
- GPG signing for release checksums

### Security

- SHA-pinned all GitHub Actions to commit hashes (supply chain hardening)
- Added `go mod verify` to release workflow (dependency integrity)
- Added `-trimpath` to release builds (path leak prevention)
- Scoped release workflow permissions to job level
- Added context timeouts for Kubernetes API calls in port-forward
- Added HTTP client timeout for SPDY port-forward transport
- Enhanced Prometheus URL validation with link-local SSRF rejection
- Fixed export file permissions: 0666 → 0600
- Fixed TUI signal handling: graceful shutdown via tea.Quit instead of os.Exit
- Capped problem map at 10,000 entries to prevent unbounded memory growth
- Sanitized Prometheus URL in TUI header display
- Fixed LDFLAGS to use VERSION_NUM (no v prefix)

### Fixed

- Octal literal style in baseline file permissions (0600 → 0o600)
- Import ordering with goimports local-prefix grouping

## [0.1.1] - 2026-02-11

### Added

- 6 service mesh detectors for linkerd and istio
  - LinkerdControlPlane: detects linkerd deployments with zero replicas (FATAL)
  - LinkerdProxyInjection: detects linkerd pods in CrashLoopBackOff (CRITICAL)
  - IstioControlPlane: detects istiod with zero replicas (FATAL)
  - IstioSidecarInjection: detects istio-system pods in CrashLoopBackOff (CRITICAL)
  - LinkerdCertExpiry: tiered alerts for identity cert expiry (<7d WARNING, <48h CRITICAL, <24h FATAL)
  - IstioCertExpiry: tiered alerts for root cert expiry (<7d WARNING, <48h CRITICAL, <24h FATAL)

### Changed

- Total detector count: 7 → 13
- CLAUDE.md synced with global project standards
- CONTRIBUTING.md commit message format aligned with conventional commits
- ARCHITECTURE.md Go version corrected to 1.25+, stale timing fixed to 1 minute

### Fixed

- Duplicate `.PHONY: deps` in Makefile
- Stale problem timing documented as 2 minutes but implemented as 1 minute

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

[Unreleased]: https://github.com/ppiankov/infranow/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/ppiankov/infranow/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/ppiankov/infranow/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/ppiankov/infranow/compare/v0.2.0...v0.3.0
[0.1.2]: https://github.com/ppiankov/infranow/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/ppiankov/infranow/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ppiankov/infranow/releases/tag/v0.1.0
