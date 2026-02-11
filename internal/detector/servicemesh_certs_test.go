package detector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/prometheus/common/model"
)

func TestLinkerdCertExpiryDetector_Warning(t *testing.T) {
	// 5 days remaining — should be WARNING
	remainingSeconds := 5 * 24 * 3600.0
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "linkerd",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewLinkerdCertExpiryDetector()
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
	if p.BlastRadius != 20 {
		t.Errorf("expected blast radius 20, got %d", p.BlastRadius)
	}
}

func TestLinkerdCertExpiryDetector_Critical(t *testing.T) {
	// 36 hours remaining — should be CRITICAL
	remainingSeconds := 36 * 3600.0
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "linkerd",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewLinkerdCertExpiryDetector()
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

func TestLinkerdCertExpiryDetector_Fatal(t *testing.T) {
	// 12 hours remaining — should be FATAL
	remainingSeconds := 12 * 3600.0
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "linkerd",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewLinkerdCertExpiryDetector()
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

func TestLinkerdCertExpiryDetector_Expired(t *testing.T) {
	// Negative value — already expired
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "linkerd",
					},
					Value: -3600, // expired 1 hour ago
				},
			}, nil
		},
	}

	d := NewLinkerdCertExpiryDetector()
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

func TestLinkerdCertExpiryDetector_NoCertMetric(t *testing.T) {
	// Empty result — cert metric not exposed (no problem, absence != expiry)
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewLinkerdCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems when cert metric absent, got %d", len(problems))
	}
}

func TestLinkerdCertExpiryDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewLinkerdCertExpiryDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestIstioCertExpiryDetector_Warning(t *testing.T) {
	remainingSeconds := 5 * 24 * 3600.0
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "istio-system",
					},
					Value: model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewIstioCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityWarning {
		t.Errorf("expected WARNING severity, got %v", p.Severity)
	}
	if p.Labels["mesh"] != "istio" {
		t.Errorf("expected mesh label 'istio', got '%s'", p.Labels["mesh"])
	}
	if p.Type != "istio_cert_expiry" {
		t.Errorf("expected type 'istio_cert_expiry', got '%s'", p.Type)
	}
}

func TestIstioCertExpiryDetector_Fatal(t *testing.T) {
	remainingSeconds := 6 * 3600.0 // 6 hours
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{},
					Value:  model.SampleValue(remainingSeconds),
				},
			}, nil
		},
	}

	d := NewIstioCertExpiryDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	if problems[0].Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity for 6h remaining, got %v", problems[0].Severity)
	}
}

func TestIstioCertExpiryDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("timeout")
		},
	}

	d := NewIstioCertExpiryDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestCertSeverity(t *testing.T) {
	tests := []struct {
		name             string
		remainingSeconds float64
		expected         models.Severity
	}{
		{"expired", -3600, models.SeverityFatal},
		{"just expired", 0, models.SeverityFatal},
		{"12 hours", 12 * 3600, models.SeverityFatal},
		{"23 hours", 23 * 3600, models.SeverityFatal},
		{"25 hours", 25 * 3600, models.SeverityCritical},
		{"47 hours", 47 * 3600, models.SeverityCritical},
		{"3 days", 3 * 24 * 3600, models.SeverityWarning},
		{"6 days", 6 * 24 * 3600, models.SeverityWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := certSeverity(tt.remainingSeconds)
			if got != tt.expected {
				t.Errorf("certSeverity(%v) = %v, want %v", tt.remainingSeconds, got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{-100, "EXPIRED"},
		{0, "EXPIRED"},
		{3600, "1h 0m"},
		{7200, "2h 0m"},
		{90000, "1d 1h"},
		{259200, "3d 0h"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.seconds, got, tt.expected)
		}
	}
}

func TestCertDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name         string
		detector     Detector
		expectedName string
	}{
		{"LinkerdCertExpiry", NewLinkerdCertExpiryDetector(), "servicemesh_linkerd_cert_expiry"},
		{"IstioCertExpiry", NewIstioCertExpiryDetector(), "servicemesh_istio_cert_expiry"},
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
			if len(entityTypes) != 1 || entityTypes[0] != "service_mesh_certificate" {
				t.Errorf("expected entity type 'service_mesh_certificate', got %v", entityTypes)
			}
		})
	}
}
