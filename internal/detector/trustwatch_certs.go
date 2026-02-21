package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// TrustwatchCertExpiryDetector detects certificates nearing expiry via trustwatch metrics
type TrustwatchCertExpiryDetector struct {
	interval time.Duration
}

func NewTrustwatchCertExpiryDetector() *TrustwatchCertExpiryDetector {
	return &TrustwatchCertExpiryDetector{
		interval: certCheckInterval,
	}
}

func (d *TrustwatchCertExpiryDetector) Name() string {
	return "trustwatch_cert_expiry"
}

func (d *TrustwatchCertExpiryDetector) EntityTypes() []string {
	return []string{"trustwatch_certificate"}
}

func (d *TrustwatchCertExpiryDetector) Interval() time.Duration {
	return d.interval
}

func (d *TrustwatchCertExpiryDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`trustwatch_cert_expires_in_seconds < %d`, certWarningThreshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("trustwatch cert expiry query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		remainingSeconds := float64(sample.Value)
		severity := certSeverity(remainingSeconds)

		source := string(sample.Metric["source"])
		namespace := string(sample.Metric["namespace"])
		name := string(sample.Metric["name"])

		entity := fmt.Sprintf("trustwatch/%s/%s/%s", source, namespace, name)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/trustwatch_cert_expiry", entity),
			Entity:     entity,
			EntityType: "trustwatch_certificate",
			Type:       "trustwatch_cert_expiry",
			Severity:   severity,
			Title:      fmt.Sprintf("Certificate expiring in %s", formatDuration(remainingSeconds)),
			Message:    fmt.Sprintf("trustwatch: %s/%s cert expires in %s", namespace, name, formatDuration(remainingSeconds)),
			Labels: map[string]string{
				"source":    source,
				"namespace": namespace,
				"name":      name,
			},
			Metrics: map[string]float64{
				"remaining_seconds": remainingSeconds,
			},
			Hint:        "Run: trustwatch now",
			BlastRadius: blastRadiusMeshComponent,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// TrustwatchProbeFailureDetector detects TLS endpoints that trustwatch cannot reach
type TrustwatchProbeFailureDetector struct {
	interval time.Duration
}

func NewTrustwatchProbeFailureDetector() *TrustwatchProbeFailureDetector {
	return &TrustwatchProbeFailureDetector{
		interval: certCheckInterval,
	}
}

func (d *TrustwatchProbeFailureDetector) Name() string {
	return "trustwatch_probe_failure"
}

func (d *TrustwatchProbeFailureDetector) EntityTypes() []string {
	return []string{"trustwatch_certificate"}
}

func (d *TrustwatchProbeFailureDetector) Interval() time.Duration {
	return d.interval
}

func (d *TrustwatchProbeFailureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `trustwatch_probe_success == 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("trustwatch probe failure query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		source := string(sample.Metric["source"])
		namespace := string(sample.Metric["namespace"])
		name := string(sample.Metric["name"])

		entity := fmt.Sprintf("trustwatch/%s/%s/%s", source, namespace, name)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/trustwatch_probe_failure", entity),
			Entity:     entity,
			EntityType: "trustwatch_certificate",
			Type:       "trustwatch_probe_failure",
			Severity:   models.SeverityCritical,
			Title:      "TLS probe failed",
			Message:    fmt.Sprintf("trustwatch: TLS probe failed for %s/%s (source: %s)", namespace, name, source),
			Labels: map[string]string{
				"source":    source,
				"namespace": namespace,
				"name":      name,
			},
			Metrics:     map[string]float64{},
			Hint:        "Run: trustwatch now",
			BlastRadius: blastRadiusService,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}
