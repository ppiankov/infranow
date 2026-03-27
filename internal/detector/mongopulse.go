package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	mongopulseCheckInterval = 30 * time.Second

	// Thresholds
	mongoConnUsedRatioThreshold   = 0.85 // 85% connection usage
	mongoReplicationLagThreshold  = 30.0 // 30 seconds
	mongoOplogWindowThreshold     = 2.0  // 2 hours minimum
	mongoGlobalLockRatioThreshold = 0.50 // 50% lock ratio
	mongoCursorsTimedOutThreshold = 10   // cursors timed out per interval

	// Blast radius
	blastRadiusMongoDB     = 8
	blastRadiusMongoLock   = 4
	blastRadiusMongoCursor = 3
)

// MongoConnectionExhaustionDetector detects when MongoDB connections are near the limit
type MongoConnectionExhaustionDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMongoConnectionExhaustionDetector() *MongoConnectionExhaustionDetector {
	return &MongoConnectionExhaustionDetector{interval: mongopulseCheckInterval, threshold: mongoConnUsedRatioThreshold}
}

func (d *MongoConnectionExhaustionDetector) Name() string            { return "mongo_connection_exhaustion" }
func (d *MongoConnectionExhaustionDetector) EntityTypes() []string   { return []string{"mongodb"} }
func (d *MongoConnectionExhaustionDetector) Interval() time.Duration { return d.interval }

func (d *MongoConnectionExhaustionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mongodb_connections_used_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mongo connection exhaustion query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mongodb"
		}
		ratio := float64(sample.Value) * 100

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mongo_connection_exhaustion", instance),
			Entity:      instance,
			EntityType:  "mongodb",
			Type:        "mongo_connection_exhaustion",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("MongoDB connections at %.0f%%", ratio),
			Message:     fmt.Sprintf("mongopulse: %s using %.0f%% of available connections", instance, ratio),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"used_ratio_percent": ratio},
			Hint:        fmt.Sprintf("Connection usage above %.0f%% — check for leaked connections or increase maxIncomingConnections", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "mongo_connection_exhaustion.md",
			BlastRadius: blastRadiusMongoDB,
		})
	}
	return problems, nil
}

// MongoReplicationLagDetector detects high replication lag between primary and secondaries
type MongoReplicationLagDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMongoReplicationLagDetector() *MongoReplicationLagDetector {
	return &MongoReplicationLagDetector{interval: mongopulseCheckInterval, threshold: mongoReplicationLagThreshold}
}

func (d *MongoReplicationLagDetector) Name() string            { return "mongo_replication_lag" }
func (d *MongoReplicationLagDetector) EntityTypes() []string   { return []string{"mongodb"} }
func (d *MongoReplicationLagDetector) Interval() time.Duration { return d.interval }

func (d *MongoReplicationLagDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mongodb_replication_lag_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mongo replication lag query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mongodb"
		}
		member := string(sample.Metric["member"])

		lagSeconds := float64(sample.Value)
		entity := fmt.Sprintf("%s/%s", instance, member)

		labels := map[string]string{"instance": instance}
		if member != "" {
			labels["member"] = member
		}

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mongo_replication_lag", entity),
			Entity:      entity,
			EntityType:  "mongodb",
			Type:        "mongo_replication_lag",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Replication lag %.0fs on %s", lagSeconds, member),
			Message:     fmt.Sprintf("mongopulse: secondary %s lagging %.0f seconds behind primary", member, lagSeconds),
			Labels:      labels,
			Metrics:     map[string]float64{"lag_seconds": lagSeconds},
			Hint:        fmt.Sprintf("Replication lag exceeds %.0fs — check secondary load, network, or oplog size", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mongo_replication_lag.md",
			BlastRadius: blastRadiusMongoDB,
		})
	}
	return problems, nil
}

// MongoOplogWindowDetector detects when the oplog window is dangerously small
type MongoOplogWindowDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMongoOplogWindowDetector() *MongoOplogWindowDetector {
	return &MongoOplogWindowDetector{interval: mongopulseCheckInterval, threshold: mongoOplogWindowThreshold}
}

func (d *MongoOplogWindowDetector) Name() string            { return "mongo_oplog_window" }
func (d *MongoOplogWindowDetector) EntityTypes() []string   { return []string{"mongodb"} }
func (d *MongoOplogWindowDetector) Interval() time.Duration { return d.interval }

func (d *MongoOplogWindowDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mongodb_oplog_window_hours < %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mongo oplog window query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mongodb"
		}

		windowHours := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mongo_oplog_window", instance),
			Entity:      instance,
			EntityType:  "mongodb",
			Type:        "mongo_oplog_window",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("Oplog window %.1fh on %s", windowHours, instance),
			Message:     fmt.Sprintf("mongopulse: oplog window on %s is %.1f hours — secondaries may not recover from maintenance", instance, windowHours),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"oplog_window_hours": windowHours},
			Hint:        fmt.Sprintf("Oplog window below %.0fh — increase oplog size or reduce write volume", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mongo_oplog_window.md",
			BlastRadius: blastRadiusMongoDB,
		})
	}
	return problems, nil
}

// MongoLockPercentageDetector detects high global lock percentage
type MongoLockPercentageDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMongoLockPercentageDetector() *MongoLockPercentageDetector {
	return &MongoLockPercentageDetector{interval: mongopulseCheckInterval, threshold: mongoGlobalLockRatioThreshold}
}

func (d *MongoLockPercentageDetector) Name() string            { return "mongo_lock_percentage" }
func (d *MongoLockPercentageDetector) EntityTypes() []string   { return []string{"mongodb"} }
func (d *MongoLockPercentageDetector) Interval() time.Duration { return d.interval }

func (d *MongoLockPercentageDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mongodb_global_lock_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mongo lock percentage query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mongodb"
		}

		ratio := float64(sample.Value) * 100

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mongo_lock_percentage", instance),
			Entity:      instance,
			EntityType:  "mongodb",
			Type:        "mongo_lock_percentage",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Global lock at %.0f%% on %s", ratio, instance),
			Message:     fmt.Sprintf("mongopulse: global lock ratio at %.0f%% on %s — write throughput may collapse", ratio, instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"lock_ratio_percent": ratio},
			Hint:        fmt.Sprintf("Global lock above %.0f%% — check for collection-level locks and long write operations", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "mongo_lock_percentage.md",
			BlastRadius: blastRadiusMongoLock,
		})
	}
	return problems, nil
}

// MongoCursorTimeoutDetector detects excessive cursor timeouts
type MongoCursorTimeoutDetector struct {
	interval  time.Duration
	threshold int
}

func NewMongoCursorTimeoutDetector() *MongoCursorTimeoutDetector {
	return &MongoCursorTimeoutDetector{interval: mongopulseCheckInterval, threshold: mongoCursorsTimedOutThreshold}
}

func (d *MongoCursorTimeoutDetector) Name() string            { return "mongo_cursor_timeout" }
func (d *MongoCursorTimeoutDetector) EntityTypes() []string   { return []string{"mongodb"} }
func (d *MongoCursorTimeoutDetector) Interval() time.Duration { return d.interval }

func (d *MongoCursorTimeoutDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mongodb_cursors_timed_out > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mongo cursor timeout query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mongodb"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mongo_cursor_timeout", instance),
			Entity:      instance,
			EntityType:  "mongodb",
			Type:        "mongo_cursor_timeout",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d cursors timed out on %s", int(count), instance),
			Message:     fmt.Sprintf("mongopulse: %d cursors timed out on %s — clients may see query failures", int(count), instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"cursors_timed_out": count},
			Hint:        fmt.Sprintf("More than %d cursor timeouts — check for slow queries or missing indexes", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mongo_cursor_timeout.md",
			BlastRadius: blastRadiusMongoCursor,
		})
	}
	return problems, nil
}
