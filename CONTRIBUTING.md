# Contributing to infranow

Thank you for your interest in contributing to infranow! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- Make
- (Optional) golangci-lint for linting

### Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/infranow.git
cd infranow
```

3. Add the upstream repository:

```bash
git remote add upstream https://github.com/ppiankov/infranow.git
```

4. Install dependencies:

```bash
make deps
```

5. Build the project:

```bash
make build
```

6. Run tests:

```bash
make test
```

## Development Workflow

### Making Changes

1. Create a new branch for your feature or fix:

```bash
git checkout -b feature/my-new-feature
```

2. Make your changes, following the coding standards below

3. Write or update tests as needed

4. Run the test suite:

```bash
make test
```

5. Format your code:

```bash
make fmt
```

6. Run linters (if golangci-lint is installed):

```bash
make lint
```

7. Commit your changes:

```bash
git add .
git commit -m "feat: add my new feature"
```

### Submitting Changes

1. Push your branch to your fork:

```bash
git push origin feature/my-new-feature
```

2. Open a Pull Request on GitHub

3. Ensure all CI checks pass

4. Wait for review and address any feedback

## Coding Standards

### Go Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable and function names
- Keep functions focused and reasonably sized
- Add comments for exported functions and types
- Use early returns to reduce nesting

### Project Structure

- `cmd/` - Command-line entry points
- `internal/` - Internal packages (not importable by external projects)
  - `cli/` - CLI command implementations
  - `detector/` - Problem detection logic
  - `metrics/` - Metrics provider implementations
  - `models/` - Core data structures
  - `monitor/` - Monitoring orchestration and TUI
  - `util/` - Shared utilities

### Adding New Detectors

To add a new detector:

1. Create a new detector struct in `internal/detector/` (e.g., `kafka.go`)

2. Implement the `Detector` interface:

```go
type MyDetector struct {
    interval time.Duration
}

func NewMyDetector() *MyDetector {
    return &MyDetector{
        interval: 30 * time.Second,
    }
}

func (d *MyDetector) Name() string {
    return "my_detector"
}

func (d *MyDetector) EntityTypes() []string {
    return []string{"my_entity_type"}
}

func (d *MyDetector) Interval() time.Duration {
    return d.interval
}

func (d *MyDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
    // Implementation
}
```

3. Add tests in `internal/detector/my_test.go`

4. Register the detector in `internal/cli/monitor.go`:

```go
func registerDetectors(registry *detector.Registry) {
    // ... existing detectors
    registry.Register(detector.NewMyDetector())
}
```

5. Document the detector in `docs/DETECTORS.md`

### Testing

- Write unit tests for all new code
- Use table-driven tests where appropriate
- Mock external dependencies (Prometheus, etc.)
- Aim for >80% code coverage
- Test edge cases and error conditions

Example test structure:

```go
func TestMyDetector(t *testing.T) {
    mockProvider := &metrics.MockProvider{
        QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
            // Return mock data
        },
    }

    detector := NewMyDetector()
    problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assertions
}
```

## Commit Messages

Follow conventional commits format: `type: concise imperative statement`

Lowercase after colon, no period, max 72 characters. Say WHAT changed, not every detail of HOW.

**Types**: feat, fix, docs, test, refactor, chore, perf, ci, build

**Examples**:
```
feat: add Kafka broker detector
fix: correct problem count display
docs: update detector documentation
test: add watcher orchestration tests
```

Optional body (separated by blank line) for WHY, not WHAT.

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] Documentation is updated if needed
- [ ] CHANGELOG.md is updated (if applicable)

### PR Description

Include in your PR description:
1. What problem does this solve?
2. How does it solve it?
3. Are there any breaking changes?
4. Screenshots (for UI changes)
5. Related issues (e.g., "Closes #123")

### Review Process

1. Maintainers will review your PR
2. Address feedback and update your PR
3. Once approved, a maintainer will merge

## Documentation

- Update README.md for user-facing changes
- Update ARCHITECTURE.md for architectural changes
- Update DETECTORS.md when adding new detectors
- Add inline code comments for complex logic

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Reach out to maintainers

## Code of Conduct

Be respectful, inclusive, and constructive. We're all here to build better tools together.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
