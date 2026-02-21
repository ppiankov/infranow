package detector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/common/model"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

func TestTrustwatchCertExpiryDetector_Warning(t *testing.T) {
	remainingSeconds := 5 * 24 * 3600.0 // 5 days
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "webhook",
						"namespace": "kube-system",
						"name":      "cert-manager-webhook",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityWarning {
		t.Errorf("expected WARNING severity for 5 days remaining, got %v", p.Severity)
	}
	if p.Entity != "trustwatch/webhook/kube-system/cert-manager-webhook" {
		t.Errorf("unexpected entity: %s", p.Entity)
	}
	if p.Type != "trustwatch_cert_expiry" {
		t.Errorf("expected type 'trustwatch_cert_expiry', got '%s'", p.Type)
	}
	if p.Labels["source"] != "webhook" {
		t.Errorf("expected source label 'webhook', got '%s'", p.Labels["source"])
	}
	if p.BlastRadius != 10 {
		t.Errorf("expected blast radius 10, got %d", p.BlastRadius)
	}
}

func TestTrustwatchCertExpiryDetector_Critical(t *testing.T) {
	remainingSeconds := 36 * 3600.0 // 36 hours
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "apiservice",
						"namespace": "default",
						"name":      "v1.apps",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	if problems[0].Severity != models.SeverityCritical {
		t.Errorf("expected CRITICAL severity for 36h remaining, got %v", problems[0].Severity)
	}
}

func TestTrustwatchCertExpiryDetector_Fatal(t *testing.T) {
	remainingSeconds := 12 * 3600.0 // 12 hours
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "mesh-issuer",
						"namespace": "linkerd",
						"name":      "identity-issuer",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	if problems[0].Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity for 12h remaining, got %v", problems[0].Severity)
	}
}

func TestTrustwatchCertExpiryDetector_Expired(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "webhook",
						"namespace": "kube-system",
						"name":      "expired-webhook",
					},
					Value: -3600, // expired 1 hour ago
				},
			}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	if problems[0].Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity for expired cert, got %v", problems[0].Severity)
	}
}

func TestTrustwatchCertExpiryDetector_NoMetrics(t *testing.T) {
	// Empty result — trustwatch not installed or no certs expiring
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems when trustwatch absent, got %d", len(problems))
	}
}

func TestTrustwatchCertExpiryDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestTrustwatchCertExpiryDetector_MultipleCerts(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "webhook",
						"namespace": "kube-system",
						"name":      "cert-manager-webhook",
					},
					Value: model.SampleValue(5 * 24 * 3600), // WARNING
				},
				&model.Sample{
					Metric: model.Metric{
						"source":    "apiservice",
						"namespace": "default",
						"name":      "v1beta1.metrics.k8s.io",
					},
					Value: model.SampleValue(12 * 3600), // FATAL
				},
			}, nil
		},
	}

	d := NewTrustwatchCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(problems))
	}

	if problems[0].Severity != models.SeverityWarning {
		t.Errorf("expected WARNING for first cert, got %v", problems[0].Severity)
	}
	if problems[1].Severity != models.SeverityFatal {
		t.Errorf("expected FATAL for second cert, got %v", problems[1].Severity)
	}
}

func TestTrustwatchProbeFailureDetector_Failure(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"source":    "webhook",
						"namespace": "kube-system",
						"name":      "cert-manager-webhook",
					},
					Value: 0,
				},
			}, nil
		},
	}

	d := NewTrustwatchProbeFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityCritical {
		t.Errorf("expected CRITICAL severity, got %v", p.Severity)
	}
	if p.Type != "trustwatch_probe_failure" {
		t.Errorf("expected type 'trustwatch_probe_failure', got '%s'", p.Type)
	}
	if p.Entity != "trustwatch/webhook/kube-system/cert-manager-webhook" {
		t.Errorf("unexpected entity: %s", p.Entity)
	}
	if p.BlastRadius != 5 {
		t.Errorf("expected blast radius 5, got %d", p.BlastRadius)
	}
}

func TestTrustwatchProbeFailureDetector_NoFailures(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewTrustwatchProbeFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems when no probe failures, got %d", len(problems))
	}
}

func TestTrustwatchProbeFailureDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("timeout")
		},
	}

	d := NewTrustwatchProbeFailureDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestTrustwatchDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name         string
		detector     Detector
		expectedName string
	}{
		{"TrustwatchCertExpiry", NewTrustwatchCertExpiryDetector(), "trustwatch_cert_expiry"},
		{"TrustwatchProbeFailure", NewTrustwatchProbeFailureDetector(), "trustwatch_probe_failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != 60*time.Second {
				t.Errorf("expected 60s interval, got %v", tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != "trustwatch_certificate" {
				t.Errorf("expected entity type 'trustwatch_certificate', got %v", entityTypes)
			}
		})
	}
}
