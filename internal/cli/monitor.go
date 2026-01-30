package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ppiankov/infranow/internal/detector"
	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/ppiankov/infranow/internal/monitor"
	"github.com/ppiankov/infranow/internal/util"
	"github.com/spf13/cobra"
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
	cmd.Flags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus endpoint URL (required)")
	cmd.Flags().DurationVar(&prometheusTimeout, "prometheus-timeout", 30*time.Second, "Prometheus query timeout")
	cmd.Flags().StringVar(&namespaceFilter, "namespace", "", "Filter by namespace pattern (regex)")
	cmd.Flags().StringVar(&entityTypeFilter, "entity-type", "", "Filter by entity type")
	cmd.Flags().StringVar(&minSeverity, "min-severity", "WARNING", "Minimum severity (FATAL, CRITICAL, WARNING)")
	cmd.Flags().DurationVar(&refreshInterval, "refresh-interval", 10*time.Second, "Detection refresh rate")
	cmd.Flags().StringVar(&outputFormat, "output", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&exportFile, "export-file", "", "Export problems to file")

	// Mark required flags
	cmd.MarkFlagRequired("prometheus-url")

	return cmd
}

func runMonitor(cmd *cobra.Command, args []string) error {
	// Validate flags
	if prometheusURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --prometheus-url is required\n")
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
		util.Exit(util.ExitRuntimeError)
	}

	// Create detector registry and register all detectors
	registry := detector.NewRegistry()
	registerDetectors(registry)

	if verbose {
		fmt.Printf("Connected to Prometheus: %s\n", prometheusURL)
		fmt.Printf("Registered %d detectors\n", registry.Count())
		fmt.Printf("Refresh interval: %s\n", refreshInterval)
		fmt.Printf("Output format: %s\n", outputFormat)
	}

	// Create watcher
	watcher := monitor.NewWatcher(provider, registry)

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
	return runTUIMode(monitorCtx, watcher, prometheusURL, refreshInterval)
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
}

func runJSONMode(ctx context.Context, watcher *monitor.Watcher) error {
	// Wait a bit for initial detection cycle
	time.Sleep(5 * time.Second)

	problems := watcher.GetProblems()
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
		file, err := os.Create(exportFile)
		if err != nil {
			return fmt.Errorf("failed to create export file: %w", err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to write export file: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Exported to: %s\n", exportFile)
		}
	}

	return nil
}

func runTUIMode(ctx context.Context, watcher *monitor.Watcher, prometheusURL string, refreshInterval time.Duration) error {
	// Create TUI model
	model := monitor.NewModel(watcher, prometheusURL, refreshInterval)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		// Send quit message to the TUI
		os.Exit(0)
	}()

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
