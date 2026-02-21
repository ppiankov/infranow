package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ppiankov/infranow/internal/baseline"
	"github.com/ppiankov/infranow/internal/detector"
	"github.com/ppiankov/infranow/internal/filter"
	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/ppiankov/infranow/internal/monitor"
	"github.com/ppiankov/infranow/internal/util"
)

var (
	prometheusURL     string
	prometheusTimeout time.Duration
	namespaceFilter   string
	entityTypeFilter  string
	minSeverity       string
	refreshInterval   time.Duration
	outputFormat      string
	exportFile        string

	// Kubernetes port-forward options
	k8sService    string
	k8sNamespace  string
	k8sLocalPort  string
	k8sRemotePort string

	// v0.1.2 features
	failOnSeverity    string // Feature 2: --fail-on
	includeNamespaces string // Feature 3: namespace filters
	excludeNamespaces string // Feature 3: namespace filters
	saveBaseline      string // Feature 1: baseline mode
	compareBaseline   string // Feature 1: baseline mode
	failOnDrift       bool   // Feature 1: baseline mode
	maxConcurrency    int    // Feature 4: concurrency controls
	detectorTimeout   time.Duration
)

// NewMonitorCommand creates the monitor subcommand
func NewMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor infrastructure in real-time",
		Long: `Monitor command polls Prometheus metrics and displays infrastructure problems
in real-time. The display stays empty when systems are healthy and automatically
surfaces problems ranked by importance.`,
		RunE: runMonitor,
	}

	// Flags
	cmd.Flags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus endpoint URL (required unless using --k8s-service)")
	cmd.Flags().DurationVar(&prometheusTimeout, "prometheus-timeout", 30*time.Second, "Prometheus query timeout")
	cmd.Flags().StringVar(&namespaceFilter, "namespace", "", "Filter by namespace pattern (regex)")
	cmd.Flags().StringVar(&entityTypeFilter, "entity-type", "", "Filter by entity type")
	cmd.Flags().StringVar(&minSeverity, "min-severity", "WARNING", "Minimum severity (FATAL, CRITICAL, WARNING)")
	cmd.Flags().DurationVar(&refreshInterval, "refresh-interval", 10*time.Second, "Detection refresh rate")
	cmd.Flags().StringVar(&outputFormat, "output", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&exportFile, "export-file", "", "Export problems to file")

	// Kubernetes port-forward flags
	cmd.Flags().StringVar(&k8sService, "k8s-service", "", "Kubernetes service name for port-forward (e.g., 'prometheus-operated')")
	cmd.Flags().StringVar(&k8sNamespace, "k8s-namespace", "monitoring", "Kubernetes namespace for service")
	cmd.Flags().StringVar(&k8sLocalPort, "k8s-local-port", "9090", "Local port for port-forward")
	cmd.Flags().StringVar(&k8sRemotePort, "k8s-remote-port", "9090", "Remote port for port-forward")

	// v0.1.2 feature flags
	cmd.Flags().StringVar(&failOnSeverity, "fail-on", "", "Exit 1 if problems at/above this severity (WARNING, CRITICAL, FATAL)")
	cmd.Flags().StringVar(&includeNamespaces, "include-namespaces", "", "Comma-separated namespace patterns (wildcards supported)")
	cmd.Flags().StringVar(&excludeNamespaces, "exclude-namespaces", "", "Comma-separated namespace patterns to exclude")
	cmd.Flags().StringVar(&saveBaseline, "save-baseline", "", "Save problems snapshot to file")
	cmd.Flags().StringVar(&compareBaseline, "compare-baseline", "", "Compare current problems to baseline file")
	cmd.Flags().BoolVar(&failOnDrift, "fail-on-drift", false, "Exit 1 if new problems detected vs baseline")
	cmd.Flags().IntVar(&maxConcurrency, "max-concurrency", 0, "Max concurrent detector executions (0 = unlimited)")
	cmd.Flags().DurationVar(&detectorTimeout, "detector-timeout", 30*time.Second, "Detector execution timeout")
	return cmd
}

func runMonitor(cmd *cobra.Command, args []string) error {
	// Validate port numbers before use
	if k8sService != "" {
		if err := validatePort(k8sLocalPort, "k8s-local-port"); err != nil {
			return err
		}
		if err := validatePort(k8sRemotePort, "k8s-remote-port"); err != nil {
			return err
		}
	}

	// Setup kubectl port-forward if k8s-service is specified
	var portForward *util.PortForward
	if k8sService != "" {
		if verbose {
			fmt.Printf("Setting up native port-forward to %s/%s...\n", k8sNamespace, k8sService)
		}

		var err error
		portForward, err = util.NewPortForward(k8sService, k8sNamespace, k8sLocalPort, k8sRemotePort)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create port-forward: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Make sure you have access to the Kubernetes cluster (check ~/.kube/config)\n")
			util.Exit(util.ExitRuntimeError)
		}

		if err := portForward.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to start port-forward: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Check that service '%s' exists in namespace '%s'\n", k8sService, k8sNamespace)
			util.Exit(util.ExitRuntimeError)
		}

		// Set prometheus URL to local port-forward
		prometheusURL = fmt.Sprintf("http://localhost:%s", k8sLocalPort)

		if verbose {
			fmt.Printf("Port-forward established: %s\n", sanitizeURL(prometheusURL))
		}

		// Ensure cleanup on exit
		defer func() {
			if verbose {
				fmt.Println("Stopping port-forward...")
			}
			if err := portForward.Stop(); err != nil && verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to stop port-forward: %v\n", err)
			}
		}()
	}

	// Validate Prometheus URL
	if prometheusURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --prometheus-url or --k8s-service is required\n")
		util.Exit(util.ExitInvalidInput)
	}
	if err := validatePrometheusURL(prometheusURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitInvalidInput)
	}

	// Create Prometheus client
	provider, err := metrics.NewPrometheusClient(prometheusURL, prometheusTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Prometheus client: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), prometheusTimeout)
	defer cancel()

	if err := provider.Health(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Prometheus health check failed: %v\n", err)
		if portForward != nil {
			fmt.Fprintf(os.Stderr, "Hint: Port-forward may still be initializing, try waiting a moment\n")
		}
		util.Exit(util.ExitRuntimeError)
	}

	// Create detector registry and register all detectors
	registry := detector.NewRegistry()
	registerDetectors(registry)

	if verbose {
		fmt.Printf("Connected to Prometheus: %s\n", sanitizeURL(prometheusURL))
		fmt.Printf("Registered %d detectors\n", registry.Count())
		fmt.Printf("Refresh interval: %s\n", refreshInterval)
		fmt.Printf("Output format: %s\n", outputFormat)
	}

	// Create watcher with concurrency controls
	watcher := monitor.NewWatcher(provider, registry, maxConcurrency, detectorTimeout)

	// Setup signal handling
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	defer monitorCancel()

	// Start watcher in background
	go func() {
		if err := watcher.Start(monitorCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}()

	// JSON output mode - run once and exit
	if outputFormat == "json" {
		return runJSONMode(monitorCtx, watcher)
	}

	// TUI mode - interactive
	return runTUIMode(monitorCtx, watcher, prometheusURL, refreshInterval, portForward)
}

func registerDetectors(registry *detector.Registry) {
	// Kubernetes detectors
	registry.Register(detector.NewOOMKillDetector())
	registry.Register(detector.NewCrashLoopBackOffDetector())
	registry.Register(detector.NewImagePullBackOffDetector())
	registry.Register(detector.NewPodPendingDetector())

	// Generic detectors
	registry.Register(detector.NewHighErrorRateDetector())
	registry.Register(detector.NewDiskSpaceDetector())
	registry.Register(detector.NewHighMemoryPressureDetector())

	// Service mesh control plane detectors
	registry.Register(detector.NewLinkerdControlPlaneDetector())
	registry.Register(detector.NewLinkerdProxyInjectionDetector())
	registry.Register(detector.NewIstioControlPlaneDetector())
	registry.Register(detector.NewIstioSidecarInjectionDetector())

	// Service mesh certificate expiry detectors
	registry.Register(detector.NewLinkerdCertExpiryDetector())
	registry.Register(detector.NewIstioCertExpiryDetector())

	// Trustwatch certificate detectors
	registry.Register(detector.NewTrustwatchCertExpiryDetector())
	registry.Register(detector.NewTrustwatchProbeFailureDetector())
}

func runJSONMode(ctx context.Context, watcher *monitor.Watcher) error {
	// Wait for first detection cycle to complete
	select {
	case <-watcher.UpdateChan():
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
	}

	problems := watcher.GetProblems()

	// Apply namespace filter (v0.1.2 Feature 3)
	problems = applyFilters(problems)

	// Save baseline if requested (v0.1.2 Feature 1)
	if saveBaseline != "" {
		metadata := map[string]string{
			"prometheus_url": prometheusURL,
			"version":        version,
		}
		if err := baseline.SaveBaseline(problems, saveBaseline, metadata); err != nil {
			return fmt.Errorf("failed to save baseline: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Baseline saved to: %s\n", saveBaseline)
		}
	}

	// Compare to baseline if requested (v0.1.2 Feature 1)
	if compareBaseline != "" {
		b, err := baseline.LoadBaseline(compareBaseline)
		if err != nil {
			return fmt.Errorf("failed to load baseline: %w", err)
		}
		comparison := baseline.Compare(problems, b)

		// Output comparison instead of raw problems
		output := map[string]interface{}{
			"metadata": map[string]interface{}{
				"prometheus_url": prometheusURL,
				"timestamp":      time.Now().Format(time.RFC3339),
				"baseline_time":  b.Timestamp.Format(time.RFC3339),
			},
			"comparison": comparison,
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

		// Fail if new problems detected (v0.1.2 Feature 1)
		if failOnDrift && len(comparison.New) > 0 {
			util.Exit(1)
		}

		return nil
	}

	// Normal JSON output
	summary := watcher.GetSummary()
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"prometheus_url":   prometheusURL,
			"timestamp":        time.Now().Format(time.RFC3339),
			"refresh_interval": refreshInterval.String(),
		},
		"summary": map[string]interface{}{
			"total_problems": len(problems),
			"fatal":          summary[models.SeverityFatal],
			"critical":       summary[models.SeverityCritical],
			"warning":        summary[models.SeverityWarning],
		},
		"problems": problems,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Export to file if specified
	if exportFile != "" {
		file, err := os.OpenFile(exportFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("failed to create export file: %w", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "[infranow] warning: failed to close export file: %v\n", err)
			}
		}()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to write export file: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Exported to: %s\n", exportFile)
		}
	}

	// Check fail-on severity threshold (v0.1.2 Feature 2)
	if failOnSeverity != "" {
		threshold, err := models.ParseSeverity(failOnSeverity)
		if err != nil {
			return err
		}

		for _, p := range problems {
			if p.Severity.AtLeast(threshold) {
				util.Exit(1) // Fail CI/CD
			}
		}
	}

	return nil
}

func runTUIMode(ctx context.Context, watcher *monitor.Watcher, prometheusURL string, refreshInterval time.Duration, portForward *util.PortForward) error {
	// Create TUI model
	model := monitor.NewModel(watcher, prometheusURL, refreshInterval, portForward)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		<-sigChan
		p.Send(tea.Quit())
	}()
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// applyFilters applies namespace filtering to problems (v0.1.2 Feature 3)
func applyFilters(problems []*models.Problem) []*models.Problem {
	// Apply namespace filter if specified
	if includeNamespaces != "" || excludeNamespaces != "" {
		nsFilter := filter.NewNamespaceFilter(includeNamespaces, excludeNamespaces)
		problems = nsFilter.Apply(problems)
	}

	return problems
}

// sanitizeURL redacts userinfo (credentials) from a URL for safe logging
func sanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid URL]"
	}
	if u.User != nil {
		u.User = url.UserPassword("REDACTED", "REDACTED")
	}
	return u.String()
}

// validatePort checks that a port string is numeric and in range 1-65535
func validatePort(portStr, name string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("--%s must be a number: %q", name, portStr)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("--%s must be between 1 and 65535, got %d", name, port)
	}
	return nil
}

// validatePrometheusURL checks that the URL has a valid http or https scheme
// and does not point to link-local addresses (SSRF prevention).
func validatePrometheusURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid --prometheus-url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("--prometheus-url must use http:// or https:// scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("--prometheus-url must include a host")
	}

	// Reject link-local addresses (SSRF prevention)
	hostname := u.Hostname()
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return nil // DNS failure is not a validation error
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
			return fmt.Errorf("prometheus URL resolves to link-local address %s (possible SSRF)", ipStr)
		}
	}

	return nil
}
