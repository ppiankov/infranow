package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	mysqlpulseCheckInterval = 30 * time.Second

	// Thresholds
	mysqlConnUsedRatioThreshold  = 0.85 // 85% connection usage
	mysqlReplicationLagThreshold = 30.0 // 30 seconds
	mysqlDeadlocksRateThreshold  = 5    // deadlocks per minute
	mysqlSlowQueriesThreshold    = 10   // concurrent slow queries
	mysqlBufferPoolHitThreshold  = 0.95 // 95% hit ratio

	// Blast radius
	blastRadiusMySQLDatabase = 8
	blastRadiusMySQLQuery    = 4
)

// MySQLConnectionExhaustionDetector detects when MySQL connections are near max_connections
type MySQLConnectionExhaustionDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMySQLConnectionExhaustionDetector() *MySQLConnectionExhaustionDetector {
	return &MySQLConnectionExhaustionDetector{interval: mysqlpulseCheckInterval, threshold: mysqlConnUsedRatioThreshold}
}

func (d *MySQLConnectionExhaustionDetector) Name() string            { return "mysql_connection_exhaustion" }
func (d *MySQLConnectionExhaustionDetector) EntityTypes() []string   { return []string{"mysql"} }
func (d *MySQLConnectionExhaustionDetector) Interval() time.Duration { return d.interval }

func (d *MySQLConnectionExhaustionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mysql_connections_used_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mysql connection exhaustion query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mysql"
		}
		ratio := float64(sample.Value) * 100

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mysql_connection_exhaustion", instance),
			Entity:      instance,
			EntityType:  "mysql",
			Type:        "mysql_connection_exhaustion",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("MySQL connections at %.0f%%", ratio),
			Message:     fmt.Sprintf("mysqlpulse: %s using %.0f%% of max_connections", instance, ratio),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"used_ratio_percent": ratio},
			Hint:        fmt.Sprintf("Connection usage above %.0f%% — check for leaked connections or increase max_connections", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "mysql_connection_exhaustion.md",
			BlastRadius: blastRadiusMySQLDatabase,
		})
	}
	return problems, nil
}

// MySQLReplicationLagDetector detects high replication lag between primary and replicas
type MySQLReplicationLagDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMySQLReplicationLagDetector() *MySQLReplicationLagDetector {
	return &MySQLReplicationLagDetector{interval: mysqlpulseCheckInterval, threshold: mysqlReplicationLagThreshold}
}

func (d *MySQLReplicationLagDetector) Name() string            { return "mysql_replication_lag" }
func (d *MySQLReplicationLagDetector) EntityTypes() []string   { return []string{"mysql"} }
func (d *MySQLReplicationLagDetector) Interval() time.Duration { return d.interval }

func (d *MySQLReplicationLagDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mysql_replication_lag_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mysql replication lag query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mysql"
		}
		channel := string(sample.Metric["channel"])

		lagSeconds := float64(sample.Value)
		entity := instance
		if channel != "" {
			entity = fmt.Sprintf("%s/%s", instance, channel)
		}

		labels := map[string]string{"instance": instance}
		if channel != "" {
			labels["channel"] = channel
		}

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mysql_replication_lag", entity),
			Entity:      entity,
			EntityType:  "mysql",
			Type:        "mysql_replication_lag",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Replication lag %.0fs on %s", lagSeconds, instance),
			Message:     fmt.Sprintf("mysqlpulse: replica %s lagging %.0f seconds behind primary", instance, lagSeconds),
			Labels:      labels,
			Metrics:     map[string]float64{"lag_seconds": lagSeconds},
			Hint:        fmt.Sprintf("Replication lag exceeds %.0fs — check replica load, network, or binlog throughput", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mysql_replication_lag.md",
			BlastRadius: blastRadiusMySQLDatabase,
		})
	}
	return problems, nil
}

// MySQLDeadlocksDetector detects high deadlock rates
type MySQLDeadlocksDetector struct {
	interval  time.Duration
	threshold int
}

func NewMySQLDeadlocksDetector() *MySQLDeadlocksDetector {
	return &MySQLDeadlocksDetector{interval: mysqlpulseCheckInterval, threshold: mysqlDeadlocksRateThreshold}
}

func (d *MySQLDeadlocksDetector) Name() string            { return "mysql_deadlocks" }
func (d *MySQLDeadlocksDetector) EntityTypes() []string   { return []string{"mysql"} }
func (d *MySQLDeadlocksDetector) Interval() time.Duration { return d.interval }

func (d *MySQLDeadlocksDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`rate(mysql_deadlocks_total[5m]) * 60 > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mysql deadlocks query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mysql"
		}

		ratePerMin := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mysql_deadlocks", instance),
			Entity:      instance,
			EntityType:  "mysql",
			Type:        "mysql_deadlocks",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%.1f deadlocks/min on %s", ratePerMin, instance),
			Message:     fmt.Sprintf("mysqlpulse: %.1f deadlocks per minute on %s — transactions are rolling back", ratePerMin, instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"deadlocks_per_min": ratePerMin},
			Hint:        fmt.Sprintf("More than %d deadlocks/min — check SHOW ENGINE INNODB STATUS for lock contention patterns", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mysql_deadlocks.md",
			BlastRadius: blastRadiusMySQLQuery,
		})
	}
	return problems, nil
}

// MySQLSlowQueriesDetector detects when many slow queries are running concurrently
type MySQLSlowQueriesDetector struct {
	interval  time.Duration
	threshold int
}

func NewMySQLSlowQueriesDetector() *MySQLSlowQueriesDetector {
	return &MySQLSlowQueriesDetector{interval: mysqlpulseCheckInterval, threshold: mysqlSlowQueriesThreshold}
}

func (d *MySQLSlowQueriesDetector) Name() string            { return "mysql_slow_queries" }
func (d *MySQLSlowQueriesDetector) EntityTypes() []string   { return []string{"mysql"} }
func (d *MySQLSlowQueriesDetector) Interval() time.Duration { return d.interval }

func (d *MySQLSlowQueriesDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mysql_slow_queries_active > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mysql slow queries query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mysql"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mysql_slow_queries", instance),
			Entity:      instance,
			EntityType:  "mysql",
			Type:        "mysql_slow_queries",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d slow queries on %s", int(count), instance),
			Message:     fmt.Sprintf("mysqlpulse: %d concurrent slow queries running on %s", int(count), instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"slow_query_count": count},
			Hint:        fmt.Sprintf("More than %d slow queries — check SHOW PROCESSLIST for long-running statements", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "mysql_slow_queries.md",
			BlastRadius: blastRadiusMySQLQuery,
		})
	}
	return problems, nil
}

// MySQLInnoDBBufferPoolPressureDetector detects low InnoDB buffer pool hit ratio
type MySQLInnoDBBufferPoolPressureDetector struct {
	interval  time.Duration
	threshold float64
}

func NewMySQLInnoDBBufferPoolPressureDetector() *MySQLInnoDBBufferPoolPressureDetector {
	return &MySQLInnoDBBufferPoolPressureDetector{interval: mysqlpulseCheckInterval, threshold: mysqlBufferPoolHitThreshold}
}

func (d *MySQLInnoDBBufferPoolPressureDetector) Name() string {
	return "mysql_innodb_buffer_pool_pressure"
}
func (d *MySQLInnoDBBufferPoolPressureDetector) EntityTypes() []string   { return []string{"mysql"} }
func (d *MySQLInnoDBBufferPoolPressureDetector) Interval() time.Duration { return d.interval }

func (d *MySQLInnoDBBufferPoolPressureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`mysql_innodb_buffer_pool_hit_ratio < %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("mysql innodb buffer pool query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "mysql"
		}

		hitRatio := float64(sample.Value) * 100

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/mysql_innodb_buffer_pool_pressure", instance),
			Entity:      instance,
			EntityType:  "mysql",
			Type:        "mysql_innodb_buffer_pool_pressure",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("InnoDB buffer pool hit ratio %.1f%% on %s", hitRatio, instance),
			Message:     fmt.Sprintf("mysqlpulse: InnoDB buffer pool hit ratio at %.1f%% on %s — excessive disk I/O", hitRatio, instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"hit_ratio_percent": hitRatio},
			Hint:        fmt.Sprintf("Buffer pool hit ratio below %.0f%% — increase innodb_buffer_pool_size or investigate working set growth", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "mysql_innodb_buffer_pool_pressure.md",
			BlastRadius: blastRadiusMySQLDatabase,
		})
	}
	return problems, nil
}
