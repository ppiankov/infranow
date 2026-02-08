package detector

import (
	"context"
	"testing"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/prometheus/common/model"
)

func TestOOMKillDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "prod",
						"pod":       "worker-123",
						"container": "app",
					},
					Value: 3,
				},
			}, nil
		},
	}

	detector := NewOOMKillDetector()
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

	if p.Entity != "prod/worker-123/app" {
		t.Errorf("expected entity 'prod/worker-123/app', got '%s'", p.Entity)
	}

	if p.Type != "oom_kill" {
		t.Errorf("expected type 'oom_kill', got '%s'", p.Type)
	}
}

func TestCrashLoopBackOffDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "staging",
						"pod":       "api-456",
						"container": "main",
					},
					Value: 1,
				},
			}, nil
		},
	}

	detector := NewCrashLoopBackOffDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity, got %v", p.Severity)
	}

	if p.Type != "crashloopbackoff" {
		t.Errorf("expected type 'crashloopbackoff', got '%s'", p.Type)
	}
}

func TestImagePullBackOffDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil // No problems
		},
	}

	detector := NewImagePullBackOffDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestPodPendingDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "default",
						"pod":       "pending-pod",
					},
					Value: 1,
				},
			}, nil
		},
	}

	detector := NewPodPendingDetector()
	problems, err := detector.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Type != "pending" {
		t.Errorf("expected type 'pending', got '%s'", p.Type)
	}
}
