package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/ppiankov/infranow/internal/correlator"
	"github.com/ppiankov/infranow/internal/detector"
	"github.com/ppiankov/infranow/internal/filter"
	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/ppiankov/infranow/internal/monitor"
	"github.com/ppiankov/infranow/internal/util"
)

var (
	sweepK8sService    string
	sweepK8sNamespace  string
	sweepK8sRemotePort string
	sweepContexts      string
	sweepParallel      bool
	sweepOutputFormat  string
	sweepFailOn        string
	sweepIncludeNS     string
	sweepExcludeNS     string
)

// ClusterResult holds the outcome of scanning a single cluster.
type ClusterResult struct {
	Context  string
	Problems []*models.Problem
	Error    error
}

// NewSweepCommand creates the sweep subcommand
func NewSweepCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Scan all kubeconfig clusters for problems",
		Long: `Sweep scans every kubeconfig context (or a filtered subset) for
infrastructure problems. Each cluster's Prometheus is accessed via port-forward.
Results are unified into a single output with cluster annotations.`,
		RunE: runSweep,
	}

	cmd.Flags().StringVar(&sweepK8sService, "k8s-service", "", "Kubernetes service name for Prometheus (required)")
	cmd.Flags().StringVar(&sweepK8sNamespace, "k8s-namespace", "monitoring", "Kubernetes namespace for Prometheus service")
	cmd.Flags().StringVar(&sweepK8sRemotePort, "k8s-remote-port", "9090", "Remote port for Prometheus")
	cmd.Flags().StringVar(&sweepContexts, "contexts", "", "Comma-separated glob patterns for context filtering (e.g. 'prod-*')")
	cmd.Flags().BoolVar(&sweepParallel, "parallel", false, "Scan clusters concurrently")
	cmd.Flags().StringVar(&sweepOutputFormat, "output", "text", "Output format (text, json, sarif)")
	cmd.Flags().StringVar(&sweepFailOn, "fail-on", "", "Exit with error if problems at/above severity (WARNING, CRITICAL, FATAL)")
	cmd.Flags().StringVar(&sweepIncludeNS, "include-namespaces", "", "Comma-separated namespace patterns (wildcards supported)")
	cmd.Flags().StringVar(&sweepExcludeNS, "exclude-namespaces", "", "Comma-separated namespace patterns to exclude")

	if err := cmd.MarkFlagRequired("k8s-service"); err != nil {
		panic(err)
	}

	return cmd
}

func runSweep(cmd *cobra.Command, args []string) error {
	if err := validatePort(sweepK8sRemotePort, "k8s-remote-port"); err != nil {
		return err
	}

	contexts, err := util.ListContexts("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}

	if sweepContexts != "" {
		contexts = util.MatchContexts(contexts, sweepContexts)
	}

	if len(contexts) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no kubeconfig contexts match filter %q\n", sweepContexts)
		util.Exit(util.ExitInvalidInput)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning %d clusters: %s\n", len(contexts), strings.Join(contexts, ", "))
	}

	var results []ClusterResult
	if sweepParallel {
		results = sweepParallelMode(contexts)
	} else {
		results = sweepSequentialMode(contexts)
	}

	allProblems, failures := mergeResults(results)
	allProblems = applySweepFilters(allProblems)
	allProblems = correlator.Correlate(allProblems)

	for i := range failures {
		fmt.Fprintf(os.Stderr, "Warning: cluster %q failed: %v\n", failures[i].Context, failures[i].Error)
	}

	switch sweepOutputFormat {
	case "json":
		return sweepOutputJSON(allProblems, contexts, failures)
	case "sarif":
		return sweepOutputSARIF(allProblems)
	default:
		return sweepOutputText(allProblems)
	}
}

func sweepSequentialMode(contexts []string) []ClusterResult {
	results := make([]ClusterResult, 0, len(contexts))
	for _, ctx := range contexts {
		results = append(results, scanCluster(ctx))
	}
	return results
}

func sweepParallelMode(contexts []string) []ClusterResult {
	results := make([]ClusterResult, len(contexts))
	var wg sync.WaitGroup

	for i, ctx := range contexts {
		wg.Add(1)
		go func(idx int, ctxName string) {
			defer wg.Done()
			results[idx] = scanCluster(ctxName)
		}(i, ctx)
	}

	wg.Wait()
	return results
}

func scanCluster(contextName string) ClusterResult {
	result := ClusterResult{Context: contextName}

	kctx, err := util.NewKubeContext("", contextName)
	if err != nil {
		result.Error = fmt.Errorf("kubeconfig: %w", err)
		return result
	}

	pf := util.NewPortForwardForContext(kctx, sweepK8sService, sweepK8sNamespace, "0", sweepK8sRemotePort)
	if startErr := pf.Start(); startErr != nil {
		result.Error = fmt.Errorf("port-forward: %w", startErr)
		return result
	}
	defer func() {
		_ = pf.Stop() // Best-effort cleanup
	}()

	actualPort := pf.ActualLocalPort()
	promURL := fmt.Sprintf("http://localhost:%s", actualPort)

	provider, err := metrics.NewPrometheusClient(promURL, prometheusTimeout)
	if err != nil {
		result.Error = fmt.Errorf("prometheus client: %w", err)
		return result
	}

	healthCtx, healthCancel := context.WithTimeout(context.Background(), prometheusTimeout)
	defer healthCancel()
	if err := provider.Health(healthCtx); err != nil {
		result.Error = fmt.Errorf("prometheus health: %w", err)
		return result
	}

	registry := detector.NewRegistry()
	registerDetectors(registry)

	watcher := monitor.NewWatcher(provider, registry, 0, detectorTimeout)

	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	go func() {
		_ = watcher.Start(watchCtx) // Best-effort
	}()

	select {
	case <-watcher.UpdateChan():
	case <-time.After(firstDetectionTimeout):
	}

	watchCancel()

	problems := watcher.GetProblems()
	for _, p := range problems {
		if p.Labels == nil {
			p.Labels = make(map[string]string)
		}
		p.Labels["cluster"] = contextName
		p.Entity = fmt.Sprintf("[%s] %s", contextName, p.Entity)
		p.ID = contextName + "/" + p.ID
	}

	result.Problems = problems

	if verbose {
		fmt.Fprintf(os.Stderr, "  %s: %d problems found\n", contextName, len(problems))
	}

	return result
}

func mergeResults(results []ClusterResult) ([]*models.Problem, []ClusterResult) {
	var allProblems []*models.Problem
	var failures []ClusterResult

	for i := range results {
		if results[i].Error != nil {
			failures = append(failures, results[i])
			continue
		}
		allProblems = append(allProblems, results[i].Problems...)
	}

	sort.Slice(allProblems, func(i, j int) bool {
		return allProblems[i].Score() > allProblems[j].Score()
	})

	return allProblems, failures
}

func applySweepFilters(problems []*models.Problem) []*models.Problem {
	if sweepIncludeNS == "" && sweepExcludeNS == "" {
		return problems
	}
	nsFilter := filter.NewNamespaceFilter(sweepIncludeNS, sweepExcludeNS)
	filtered := make([]*models.Problem, 0, len(problems))
	for _, p := range problems {
		ns := p.Labels["namespace"]
		if ns == "" || nsFilter.Matches(ns) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func sweepOutputText(problems []*models.Problem) error {
	fmt.Print(monitor.PlainText(problems, time.Now()))
	fmt.Fprintln(os.Stderr, monitor.PlainTextSummary(problems))
	return sweepExitCode(problems)
}

func sweepOutputJSON(problems []*models.Problem, contexts []string, failures []ClusterResult) error {
	failedContexts := make([]map[string]string, 0, len(failures))
	for i := range failures {
		failedContexts = append(failedContexts, map[string]string{
			"context": failures[i].Context,
			"error":   failures[i].Error.Error(),
		})
	}

	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"timestamp":        time.Now().Format(time.RFC3339),
			"contexts_scanned": len(contexts) - len(failures),
			"contexts_failed":  len(failures),
			"mode":             "sweep",
		},
		"summary": map[string]interface{}{
			"total_problems": len(problems),
			"fatal":          countBySeverity(problems, models.SeverityFatal),
			"critical":       countBySeverity(problems, models.SeverityCritical),
			"warning":        countBySeverity(problems, models.SeverityWarning),
		},
		"problems": problems,
	}

	if len(failedContexts) > 0 {
		output["failures"] = failedContexts
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return sweepExitCode(problems)
}

func sweepOutputSARIF(problems []*models.Problem) error {
	data, err := monitor.SARIF(problems, version)
	if err != nil {
		return fmt.Errorf("failed to render SARIF: %w", err)
	}
	if _, writeErr := fmt.Fprintln(os.Stdout, string(data)); writeErr != nil {
		return fmt.Errorf("failed to write SARIF output: %w", writeErr)
	}
	fmt.Fprintln(os.Stderr, monitor.FormatSARIFSummary(problems))
	return sweepExitCode(problems)
}

func sweepExitCode(problems []*models.Problem) error {
	if sweepFailOn != "" {
		threshold, err := models.ParseSeverity(sweepFailOn)
		if err != nil {
			return err
		}
		for _, p := range problems {
			if p.Severity.AtLeast(threshold) {
				util.Exit(util.ExitProblemsCritical)
			}
		}
		return nil
	}

	if len(problems) > 0 {
		switch monitor.HighestSeverity(problems) {
		case models.SeverityCritical, models.SeverityFatal:
			util.Exit(util.ExitProblemsCritical)
		default:
			util.Exit(util.ExitProblemsWarning)
		}
	}
	return nil
}

func countBySeverity(problems []*models.Problem, sev models.Severity) int {
	count := 0
	for _, p := range problems {
		if p.Severity == sev {
			count++
		}
	}
	return count
}
