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

func TestToteSalvageFailureDetector_Failures(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 3},
			}, nil
		},
	}

	d := NewToteSalvageFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityCritical {
		t.Errorf("expected CRITICAL, got %v", p.Severity)
	}
	if p.Type != "tote_salvage_failure" {
		t.Errorf("expected type 'tote_salvage_failure', got '%s'", p.Type)
	}
	if p.Metrics["failures_5m"] != 3 {
		t.Errorf("expected 3 failures, got %v", p.Metrics["failures_5m"])
	}
	if p.BlastRadius != blastRadiusSalvage {
		t.Errorf("expected blast radius %d, got %d", blastRadiusSalvage, p.BlastRadius)
	}
}

func TestToteSalvageFailureDetector_NoFailures(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewToteSalvageFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestToteSalvageFailureDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewToteSalvageFailureDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestTotePushFailureDetector_Failures(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 2},
			}, nil
		},
	}

	d := NewTotePushFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityWarning {
		t.Errorf("expected WARNING, got %v", p.Severity)
	}
	if p.Type != "tote_push_failure" {
		t.Errorf("expected type 'tote_push_failure', got '%s'", p.Type)
	}
}

func TestTotePushFailureDetector_NoFailures(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewTotePushFailureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestToteHighFailureRateDetector_HighRate(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 8},
			}, nil
		},
	}

	d := NewToteHighFailureRateDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityWarning {
		t.Errorf("expected WARNING, got %v", p.Severity)
	}
	if p.Type != "tote_high_failure_rate" {
		t.Errorf("expected type 'tote_high_failure_rate', got '%s'", p.Type)
	}
}

func TestToteHighFailureRateDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewToteHighFailureRateDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestToteDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name           string
		detector       Detector
		expectedName   string
		expectedEntity string
	}{
		{"ToteSalvageFailure", NewToteSalvageFailureDetector(), "tote_salvage_failure", "tote_salvage"},
		{"TotePushFailure", NewTotePushFailureDetector(), "tote_push_failure", "tote_push"},
		{"ToteHighFailureRate", NewToteHighFailureRateDetector(), "tote_high_failure_rate", "tote_detection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != toteCheckInterval {
				t.Errorf("expected %v interval, got %v", toteCheckInterval, tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != tt.expectedEntity {
				t.Errorf("expected entity type '%s', got %v", tt.expectedEntity, entityTypes)
			}
		})
	}
}
