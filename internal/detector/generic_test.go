package detector

import (
	"context"
	"testing"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/prometheus/common/model"
)

func TestHighErrorRateDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"service": "payment-api",
					},
					Value: 0.08, // 8% error rate
				},
			}, nil
		},
	}

	detector := NewHighErrorRateDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

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

	if p.Type != "high_error_rate" {
		t.Errorf("expected type 'high_error_rate', got '%s'", p.Type)
	}

	if p.Metrics["error_rate"] < 5.0 {
		t.Errorf("expected error rate > 5%%, got %.2f%%", p.Metrics["error_rate"])
	}
}

func TestDiskSpaceDetector(t *testing.T) {
	tests := []struct {
		name             string
		usageValue       float64
		expectedSeverity models.Severity
	}{
		{"warning level", 0.92, models.SeverityWarning},
		{"critical level", 0.97, models.SeverityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &metrics.MockProvider{
				QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
					return model.Vector{
						&model.Sample{
							Metric: model.Metric{
								"instance":   "node-1",
								"mountpoint": "/var/lib",
								"device":     "/dev/sda1",
							},
							Value: model.SampleValue(tt.usageValue),
						},
					}, nil
				},
			}

			detector := NewDiskSpaceDetector()
			problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(problems) != 1 {
				t.Fatalf("expected 1 problem, got %d", len(problems))
			}

			p := problems[0]
			if p.Severity != tt.expectedSeverity {
				t.Errorf("expected %v severity, got %v", tt.expectedSeverity, p.Severity)
			}

			if p.Type != "disk_full" {
				t.Errorf("expected type 'disk_full', got '%s'", p.Type)
			}
		})
	}
}

func TestHighMemoryPressureDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"instance": "node-2",
					},
					Value: 0.93, // 93% memory usage
				},
			}, nil
		},
	}

	detector := NewHighMemoryPressureDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

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

	if p.Type != "high_memory" {
		t.Errorf("expected type 'high_memory', got '%s'", p.Type)
	}

	if p.BlastRadius < 5 {
		t.Errorf("expected blast radius >= 5 for node problems, got %d", p.BlastRadius)
	}
}

func TestHighMemoryPressureDetector_NoProblems(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil // Healthy system
		},
	}

	detector := NewHighMemoryPressureDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}
