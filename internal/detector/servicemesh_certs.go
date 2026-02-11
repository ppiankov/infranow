package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	// Cert expiry thresholds in seconds
	certFatalThreshold    = 86400  // 24 hours
	certCriticalThreshold = 172800 // 48 hours
	certWarningThreshold  = 604800 // 7 days
	certCheckInterval     = 60     // 60 seconds between checks
)

// LinkerdCertExpiryDetector detects linkerd identity certificates nearing expiry
type LinkerdCertExpiryDetector struct {
	interval time.Duration
}

func NewLinkerdCertExpiryDetector() *LinkerdCertExpiryDetector {
	return &LinkerdCertExpiryDetector{
		interval: certCheckInterval * time.Second,
	}
}

func (d *LinkerdCertExpiryDetector) Name() string {
	return "servicemesh_linkerd_cert_expiry"
}

func (d *LinkerdCertExpiryDetector) EntityTypes() []string {
	return []string{"service_mesh_certificate"}
}

func (d *LinkerdCertExpiryDetector) Interval() time.Duration {
	return d.interval
}

func (d *LinkerdCertExpiryDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	// Query linkerd identity cert expiry timestamp
	// identity_cert_expiry_timestamp is exposed by linkerd-identity when scraped
	query := fmt.Sprintf(`(identity_cert_expiry_timestamp - time()) < %d`, certWarningThreshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("linkerd cert expiry query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		remainingSeconds := float64(sample.Value)
		severity := certSeverity(remainingSeconds)

		namespace := string(sample.Metric["namespace"])
		if namespace == "" {
			namespace = "linkerd"
		}

		entity := fmt.Sprintf("%s/identity-cert", namespace)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/linkerd_cert_expiry", entity),
			Entity:     entity,
			EntityType: "service_mesh_certificate",
			Type:       "linkerd_cert_expiry",
			Severity:   severity,
			Title:      "Linkerd Certificate Expiring",
			Message:    fmt.Sprintf("Linkerd identity certificate expires in %s", formatDuration(remainingSeconds)),
			Labels: map[string]string{
				"mesh":      "linkerd",
				"namespace": namespace,
				"type":      "identity_cert",
			},
			Metrics: map[string]float64{
				"remaining_seconds": remainingSeconds,
			},
			Hint:        "Rotate certs: linkerd check --proxy; Renew: linkerd upgrade | kubectl apply -f -",
			BlastRadius: 20,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// IstioCertExpiryDetector detects istio root/workload certificates nearing expiry
type IstioCertExpiryDetector struct {
	interval time.Duration
}

func NewIstioCertExpiryDetector() *IstioCertExpiryDetector {
	return &IstioCertExpiryDetector{
		interval: certCheckInterval * time.Second,
	}
}

func (d *IstioCertExpiryDetector) Name() string {
	return "servicemesh_istio_cert_expiry"
}

func (d *IstioCertExpiryDetector) EntityTypes() []string {
	return []string{"service_mesh_certificate"}
}

func (d *IstioCertExpiryDetector) Interval() time.Duration {
	return d.interval
}

func (d *IstioCertExpiryDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	// citadel_server_root_cert_expiry_timestamp is exposed by istiod
	// istio_agent_cert_expiry_seconds is exposed by sidecar proxies
	query := fmt.Sprintf(`(citadel_server_root_cert_expiry_timestamp - time()) < %d`, certWarningThreshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("istio cert expiry query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		remainingSeconds := float64(sample.Value)
		severity := certSeverity(remainingSeconds)

		namespace := string(sample.Metric["namespace"])
		if namespace == "" {
			namespace = "istio-system"
		}

		entity := fmt.Sprintf("%s/root-cert", namespace)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/istio_cert_expiry", entity),
			Entity:     entity,
			EntityType: "service_mesh_certificate",
			Type:       "istio_cert_expiry",
			Severity:   severity,
			Title:      "Istio Root Certificate Expiring",
			Message:    fmt.Sprintf("Istio root certificate expires in %s", formatDuration(remainingSeconds)),
			Labels: map[string]string{
				"mesh":      "istio",
				"namespace": namespace,
				"type":      "root_cert",
			},
			Metrics: map[string]float64{
				"remaining_seconds": remainingSeconds,
			},
			Hint:        "Check status: istioctl proxy-status; Rotate: istioctl create-remote-secret",
			BlastRadius: 20,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// certSeverity returns the appropriate severity based on remaining time
func certSeverity(remainingSeconds float64) models.Severity {
	switch {
	case remainingSeconds <= 0:
		return models.SeverityFatal
	case remainingSeconds < certFatalThreshold:
		return models.SeverityFatal
	case remainingSeconds < certCriticalThreshold:
		return models.SeverityCritical
	default:
		return models.SeverityWarning
	}
}

// formatDuration converts seconds to a human-readable duration
func formatDuration(seconds float64) string {
	if seconds <= 0 {
		return "EXPIRED"
	}
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh %dm", hours, int(d.Minutes())%60)
}
