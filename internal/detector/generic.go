package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// HighErrorRateDetector detects high HTTP 5xx error rates
type HighErrorRateDetector struct {
	interval  time.Duration
	threshold float64 // Error rate threshold (0.05 = 5%)
}

func NewHighErrorRateDetector() *HighErrorRateDetector {
	return &HighErrorRateDetector{
		interval:  30 * time.Second,
		threshold: 0.05, // 5%
	}
}

func (d *HighErrorRateDetector) Name() string {
	return "generic_high_error_rate"
}

func (d *HighErrorRateDetector) EntityTypes() []string {
	return []string{"service", "http_endpoint"}
}

func (d *HighErrorRateDetector) Interval() time.Duration {
	return d.interval
}

func (d *HighErrorRateDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`(rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])) > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("error rate query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		service := string(sample.Metric["service"])
		if service == "" {
			service = string(sample.Metric["job"])
		}
		if service == "" {
			service = "unknown"
		}

		errorRate := float64(sample.Value) * 100 // Convert to percentage

		entity := service
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/high_error_rate", entity),
			Entity:     entity,
			EntityType: "service",
			Type:       "high_error_rate",
			Severity:   models.SeverityCritical,
			Title:      "High Error Rate",
			Message:    fmt.Sprintf("Service %s has %.2f%% 5xx error rate", service, errorRate),
			Labels: map[string]string{
				"service": service,
			},
			Metrics: map[string]float64{
				"error_rate": errorRate,
			},
			Hint:        fmt.Sprintf("5xx error rate above %.0f%% threshold", d.threshold*100),
			BlastRadius: 5, // Assume service affects multiple entities
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// DiskSpaceDetector detects low disk space on nodes
type DiskSpaceDetector struct {
	interval          time.Duration
	warningThreshold  float64 // Percentage used (0.9 = 90%)
	criticalThreshold float64 // Percentage used (0.95 = 95%)
}

func NewDiskSpaceDetector() *DiskSpaceDetector {
	return &DiskSpaceDetector{
		interval:          60 * time.Second,
		warningThreshold:  0.90, // 90%
		criticalThreshold: 0.95, // 95%
	}
}

func (d *DiskSpaceDetector) Name() string {
	return "generic_disk_space"
}

func (d *DiskSpaceDetector) EntityTypes() []string {
	return []string{"node", "filesystem"}
}

func (d *DiskSpaceDetector) Interval() time.Duration {
	return d.interval
}

func (d *DiskSpaceDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	// Check for filesystems with low available space
	query := fmt.Sprintf(`(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) > %f`, d.warningThreshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("disk space query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		node := string(sample.Metric["instance"])
		mountpoint := string(sample.Metric["mountpoint"])
		device := string(sample.Metric["device"])

		if node == "" {
			node = "unknown"
		}

		usagePercent := float64(sample.Value) * 100

		// Determine severity
		severity := models.SeverityWarning
		if float64(sample.Value) >= d.criticalThreshold {
			severity = models.SeverityCritical
		}

		entity := fmt.Sprintf("%s:%s", node, mountpoint)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/disk_space", entity),
			Entity:     entity,
			EntityType: "filesystem",
			Type:       "disk_full",
			Severity:   severity,
			Title:      "Low Disk Space",
			Message:    fmt.Sprintf("Filesystem %s on %s is %.1f%% full", mountpoint, node, usagePercent),
			Labels: map[string]string{
				"node":       node,
				"mountpoint": mountpoint,
				"device":     device,
			},
			Metrics: map[string]float64{
				"usage_percent": usagePercent,
			},
			Hint:        fmt.Sprintf("Disk usage above %.0f%%", d.warningThreshold*100),
			BlastRadius: 3, // Could affect multiple services on the node
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// HighMemoryPressureDetector detects high memory pressure on nodes
type HighMemoryPressureDetector struct {
	interval  time.Duration
	threshold float64 // Memory usage threshold (0.9 = 90%)
}

func NewHighMemoryPressureDetector() *HighMemoryPressureDetector {
	return &HighMemoryPressureDetector{
		interval:  30 * time.Second,
		threshold: 0.90, // 90%
	}
}

func (d *HighMemoryPressureDetector) Name() string {
	return "generic_memory_pressure"
}

func (d *HighMemoryPressureDetector) EntityTypes() []string {
	return []string{"node"}
}

func (d *HighMemoryPressureDetector) Interval() time.Duration {
	return d.interval
}

func (d *HighMemoryPressureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("memory pressure query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		node := string(sample.Metric["instance"])
		if node == "" {
			node = "unknown"
		}

		usagePercent := float64(sample.Value) * 100

		entity := node
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/memory_pressure", entity),
			Entity:     entity,
			EntityType: "node",
			Type:       "high_memory",
			Severity:   models.SeverityCritical,
			Title:      "High Memory Pressure",
			Message:    fmt.Sprintf("Node %s has %.1f%% memory usage", node, usagePercent),
			Labels: map[string]string{
				"node": node,
			},
			Metrics: map[string]float64{
				"memory_usage_percent": usagePercent,
			},
			Hint:        fmt.Sprintf("Memory pressure above %.0f%%", d.threshold*100),
			BlastRadius: 10, // Could affect many pods on the node
		}
		problems = append(problems, problem)
	}

	return problems, nil
}
