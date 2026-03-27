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

func TestMongoConnectionExhaustionDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.92, Metric: model.Metric{"instance": "mongo-primary:27017"}},
			}, nil
		},
	}

	d := NewMongoConnectionExhaustionDetector()
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
	if p.Type != "mongo_connection_exhaustion" {
		t.Errorf("expected type 'mongo_connection_exhaustion', got '%s'", p.Type)
	}
	if p.Metrics["used_ratio_percent"] != 92 {
		t.Errorf("expected 92%%, got %v", p.Metrics["used_ratio_percent"])
	}
	if p.BlastRadius != blastRadiusMongoDB {
		t.Errorf("expected blast radius %d, got %d", blastRadiusMongoDB, p.BlastRadius)
	}
}

func TestMongoConnectionExhaustionDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMongoConnectionExhaustionDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMongoConnectionExhaustionDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewMongoConnectionExhaustionDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestMongoReplicationLagDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 45, Metric: model.Metric{"instance": "mongo-primary:27017", "member": "secondary-0"}},
			}, nil
		},
	}

	d := NewMongoReplicationLagDetector()
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
	if p.Metrics["lag_seconds"] != 45 {
		t.Errorf("expected 45s lag, got %v", p.Metrics["lag_seconds"])
	}
}

func TestMongoReplicationLagDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMongoReplicationLagDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMongoOplogWindowDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 1.2, Metric: model.Metric{"instance": "mongo-primary:27017"}},
			}, nil
		},
	}

	d := NewMongoOplogWindowDetector()
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
	if p.Metrics["oplog_window_hours"] != 1.2 {
		t.Errorf("expected 1.2h, got %v", p.Metrics["oplog_window_hours"])
	}
}

func TestMongoOplogWindowDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMongoOplogWindowDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMongoLockPercentageDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 0.65, Metric: model.Metric{"instance": "mongo-primary:27017"}},
			}, nil
		},
	}

	d := NewMongoLockPercentageDetector()
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
	if p.BlastRadius != blastRadiusMongoLock {
		t.Errorf("expected blast radius %d, got %d", blastRadiusMongoLock, p.BlastRadius)
	}
}

func TestMongoLockPercentageDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMongoLockPercentageDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMongoCursorTimeoutDetector_Detected(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{Value: 25, Metric: model.Metric{"instance": "mongo-primary:27017"}},
			}, nil
		},
	}

	d := NewMongoCursorTimeoutDetector()
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
	if p.Metrics["cursors_timed_out"] != 25 {
		t.Errorf("expected 25 cursors, got %v", p.Metrics["cursors_timed_out"])
	}
	if p.BlastRadius != blastRadiusMongoCursor {
		t.Errorf("expected blast radius %d, got %d", blastRadiusMongoCursor, p.BlastRadius)
	}
}

func TestMongoCursorTimeoutDetector_NoIssues(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewMongoCursorTimeoutDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestMongopulseDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name           string
		detector       Detector
		expectedName   string
		expectedEntity string
	}{
		{"MongoConnectionExhaustion", NewMongoConnectionExhaustionDetector(), "mongo_connection_exhaustion", "mongodb"},
		{"MongoReplicationLag", NewMongoReplicationLagDetector(), "mongo_replication_lag", "mongodb"},
		{"MongoOplogWindow", NewMongoOplogWindowDetector(), "mongo_oplog_window", "mongodb"},
		{"MongoLockPercentage", NewMongoLockPercentageDetector(), "mongo_lock_percentage", "mongodb"},
		{"MongoCursorTimeout", NewMongoCursorTimeoutDetector(), "mongo_cursor_timeout", "mongodb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != mongopulseCheckInterval {
				t.Errorf("expected %v interval, got %v", mongopulseCheckInterval, tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != tt.expectedEntity {
				t.Errorf("expected entity type '%s', got %v", tt.expectedEntity, entityTypes)
			}
		})
	}
}
