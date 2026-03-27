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

func TestAirflowDAGFailureRateDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.25, Metric: model.Metric{"instance": "airflow-web:8080", "dag_id": "etl_pipeline"}},
			}, nil
		},
	}

	d := NewAirflowDAGFailureRateDetector()
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
	if p.Type != "airflow_dag_failure_rate" {
		t.Errorf("expected type 'airflow_dag_failure_rate', got '%s'", p.Type)
	}
	if p.Metrics["failure_rate_percent"] != 25 {
		t.Errorf("expected 25%%, got %v", p.Metrics["failure_rate_percent"])
	}
	if p.BlastRadius != blastRadiusAirflowDAG {
		t.Errorf("expected blast radius %d, got %d", blastRadiusAirflowDAG, p.BlastRadius)
	}
}

func TestAirflowDAGFailureRateDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewAirflowDAGFailureRateDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestAirflowDAGFailureRateDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewAirflowDAGFailureRateDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestAirflowSchedulerHeartbeatDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 120, Metric: model.Metric{"instance": "airflow-scheduler:8793"}},
			}, nil
		},
	}

	d := NewAirflowSchedulerHeartbeatDetector()
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
	if p.Metrics["heartbeat_seconds"] != 120 {
		t.Errorf("expected 120s, got %v", p.Metrics["heartbeat_seconds"])
	}
}

func TestAirflowSchedulerHeartbeatDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewAirflowSchedulerHeartbeatDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestAirflowTaskQueueBacklogDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 250, Metric: model.Metric{"instance": "airflow:8080"}},
			}, nil
		},
	}

	d := NewAirflowTaskQueueBacklogDetector()
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
	if p.Metrics["queued_tasks"] != 250 {
		t.Errorf("expected 250 queued, got %v", p.Metrics["queued_tasks"])
	}
}

func TestAirflowTaskQueueBacklogDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewAirflowTaskQueueBacklogDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestAirflowPoolExhaustionDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.95, Metric: model.Metric{"instance": "airflow:8080", "pool": "default_pool"}},
			}, nil
		},
	}

	d := NewAirflowPoolExhaustionDetector()
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
	if p.Type != "airflow_pool_exhaustion" {
		t.Errorf("expected type 'airflow_pool_exhaustion', got '%s'", p.Type)
	}
	if p.BlastRadius != blastRadiusAirflowPool {
		t.Errorf("expected blast radius %d, got %d", blastRadiusAirflowPool, p.BlastRadius)
	}
}

func TestAirflowPoolExhaustionDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewAirflowPoolExhaustionDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestAirflowZombieTasksDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 3, Metric: model.Metric{"instance": "airflow:8080"}},
			}, nil
		},
	}

	d := NewAirflowZombieTasksDetector()
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
	if p.Metrics["zombie_tasks"] != 3 {
		t.Errorf("expected 3 zombie tasks, got %v", p.Metrics["zombie_tasks"])
	}
	if p.BlastRadius != blastRadiusAirflowTask {
		t.Errorf("expected blast radius %d, got %d", blastRadiusAirflowTask, p.BlastRadius)
	}
}

func TestAirflowZombieTasksDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewAirflowZombieTasksDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestAirflowpulseDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name           string
		detector       Detector
		expectedName   string
		expectedEntity string
	}{
		{"AirflowDAGFailureRate", NewAirflowDAGFailureRateDetector(), "airflow_dag_failure_rate", "airflow_dag"},
		{"AirflowSchedulerHeartbeat", NewAirflowSchedulerHeartbeatDetector(), "airflow_scheduler_heartbeat", "airflow_scheduler"},
		{"AirflowTaskQueueBacklog", NewAirflowTaskQueueBacklogDetector(), "airflow_task_queue_backlog", "airflow_executor"},
		{"AirflowPoolExhaustion", NewAirflowPoolExhaustionDetector(), "airflow_pool_exhaustion", "airflow_pool"},
		{"AirflowZombieTasks", NewAirflowZombieTasksDetector(), "airflow_zombie_tasks", "airflow_task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != airflowpulseCheckInterval {
				t.Errorf("expected %v interval, got %v", airflowpulseCheckInterval, tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != tt.expectedEntity {
				t.Errorf("expected entity type '%s', got %v", tt.expectedEntity, entityTypes)
			}
		})
	}
}
