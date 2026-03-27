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

func TestMySQLConnectionExhaustionDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.90, Metric: model.Metric{"instance": "mysql-primary:3306"}},
			}, nil
		},
	}

	d := NewMySQLConnectionExhaustionDetector()
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
	if p.Type != "mysql_connection_exhaustion" {
		t.Errorf("expected type 'mysql_connection_exhaustion', got '%s'", p.Type)
	}
	if p.Metrics["used_ratio_percent"] != 90 {
		t.Errorf("expected 90%%, got %v", p.Metrics["used_ratio_percent"])
	}
	if p.BlastRadius != blastRadiusMySQLDatabase {
		t.Errorf("expected blast radius %d, got %d", blastRadiusMySQLDatabase, p.BlastRadius)
	}
}

func TestMySQLConnectionExhaustionDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMySQLConnectionExhaustionDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMySQLConnectionExhaustionDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewMySQLConnectionExhaustionDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestMySQLReplicationLagDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 60, Metric: model.Metric{"instance": "mysql-replica:3306"}},
			}, nil
		},
	}

	d := NewMySQLReplicationLagDetector()
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
	if p.Metrics["lag_seconds"] != 60 {
		t.Errorf("expected 60s lag, got %v", p.Metrics["lag_seconds"])
	}
}

func TestMySQLReplicationLagDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMySQLReplicationLagDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMySQLDeadlocksDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 8, Metric: model.Metric{"instance": "mysql-primary:3306"}},
			}, nil
		},
	}

	d := NewMySQLDeadlocksDetector()
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
	if p.Type != "mysql_deadlocks" {
		t.Errorf("expected type 'mysql_deadlocks', got '%s'", p.Type)
	}
}

func TestMySQLDeadlocksDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMySQLDeadlocksDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMySQLSlowQueriesDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 15, Metric: model.Metric{"instance": "mysql-primary:3306"}},
			}, nil
		},
	}

	d := NewMySQLSlowQueriesDetector()
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
	if p.Metrics["slow_query_count"] != 15 {
		t.Errorf("expected 15 slow queries, got %v", p.Metrics["slow_query_count"])
	}
}

func TestMySQLSlowQueriesDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMySQLSlowQueriesDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMySQLInnoDBBufferPoolPressureDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.88, Metric: model.Metric{"instance": "mysql-primary:3306"}},
			}, nil
		},
	}

	d := NewMySQLInnoDBBufferPoolPressureDetector()
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
	if p.Type != "mysql_innodb_buffer_pool_pressure" {
		t.Errorf("expected type 'mysql_innodb_buffer_pool_pressure', got '%s'", p.Type)
	}
	if p.BlastRadius != blastRadiusMySQLDatabase {
		t.Errorf("expected blast radius %d, got %d", blastRadiusMySQLDatabase, p.BlastRadius)
	}
}

func TestMySQLInnoDBBufferPoolPressureDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMySQLInnoDBBufferPoolPressureDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMysqlpulseDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name           string
		detector       Detector
		expectedName   string
		expectedEntity string
	}{
		{"MySQLConnectionExhaustion", NewMySQLConnectionExhaustionDetector(), "mysql_connection_exhaustion", "mysql"},
		{"MySQLReplicationLag", NewMySQLReplicationLagDetector(), "mysql_replication_lag", "mysql"},
		{"MySQLDeadlocks", NewMySQLDeadlocksDetector(), "mysql_deadlocks", "mysql"},
		{"MySQLSlowQueries", NewMySQLSlowQueriesDetector(), "mysql_slow_queries", "mysql"},
		{"MySQLInnoDBBufferPoolPressure", NewMySQLInnoDBBufferPoolPressureDetector(), "mysql_innodb_buffer_pool_pressure", "mysql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != mysqlpulseCheckInterval {
				t.Errorf("expected %v interval, got %v", mysqlpulseCheckInterval, tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != tt.expectedEntity {
				t.Errorf("expected entity type '%s', got %v", tt.expectedEntity, entityTypes)
			}
		})
	}
}
