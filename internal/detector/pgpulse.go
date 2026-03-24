package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	pgpulseCheckInterval = 30 * time.Second

	// Thresholds
	pgConnUsedRatioThreshold  = 0.85 // 85% connection usage
	pgReplicationLagThreshold = 30.0 // 30 seconds
	pgDeadTupleRatioThreshold = 0.20 // 20% dead tuples
	pgLockChainDepthThreshold = 3    // lock chain depth
	pgSlowQueriesThreshold    = 5    // concurrent slow queries

	// Blast radius
	blastRadiusPgDatabase = 8
	blastRadiusPgTable    = 3
	blastRadiusPgQuery    = 4
)

// PgConnectionExhaustionDetector detects when PostgreSQL connections are near max_connections
type PgConnectionExhaustionDetector struct {
	interval  time.Duration
	threshold float64
}

func NewPgConnectionExhaustionDetector() *PgConnectionExhaustionDetector {
	return &PgConnectionExhaustionDetector{interval: pgpulseCheckInterval, threshold: pgConnUsedRatioThreshold}
}

func (d *PgConnectionExhaustionDetector) Name() string            { return "pg_connection_exhaustion" }
func (d *PgConnectionExhaustionDetector) EntityTypes() []string   { return []string{"postgresql"} }
func (d *PgConnectionExhaustionDetector) Interval() time.Duration { return d.interval }

func (d *PgConnectionExhaustionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`pg_connections_used_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pg connection exhaustion query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "postgresql"
		}
		ratio := float64(sample.Value) * 100

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/pg_connection_exhaustion", instance),
			Entity:      instance,
			EntityType:  "postgresql",
			Type:        "pg_connection_exhaustion",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("PostgreSQL connections at %.0f%%", ratio),
			Message:     fmt.Sprintf("pgpulse: %s using %.0f%% of max_connections", instance, ratio),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"used_ratio_percent": ratio},
			Hint:        fmt.Sprintf("Connection usage above %.0f%% — check for leaked connections or increase max_connections", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "pg_connection_exhaustion.md",
			BlastRadius: blastRadiusPgDatabase,
		})
	}
	return problems, nil
}

// PgReplicationLagDetector detects high replication lag between primary and replicas
type PgReplicationLagDetector struct {
	interval  time.Duration
	threshold float64
}

func NewPgReplicationLagDetector() *PgReplicationLagDetector {
	return &PgReplicationLagDetector{interval: pgpulseCheckInterval, threshold: pgReplicationLagThreshold}
}

func (d *PgReplicationLagDetector) Name() string            { return "pg_replication_lag" }
func (d *PgReplicationLagDetector) EntityTypes() []string   { return []string{"postgresql"} }
func (d *PgReplicationLagDetector) Interval() time.Duration { return d.interval }

func (d *PgReplicationLagDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`pg_replication_lag_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pg replication lag query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		slot := string(sample.Metric["slot"])
		clientAddr := string(sample.Metric["client_addr"])
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "postgresql"
		}

		lagSeconds := float64(sample.Value)
		entity := fmt.Sprintf("%s/%s", instance, slot)

		labels := map[string]string{"instance": instance}
		if slot != "" {
			labels["slot"] = slot
		}
		if clientAddr != "" {
			labels["client_addr"] = clientAddr
		}

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/pg_replication_lag", entity),
			Entity:      entity,
			EntityType:  "postgresql",
			Type:        "pg_replication_lag",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Replication lag %.0fs on slot %s", lagSeconds, slot),
			Message:     fmt.Sprintf("pgpulse: replica %s lagging %.0f seconds behind primary", clientAddr, lagSeconds),
			Labels:      labels,
			Metrics:     map[string]float64{"lag_seconds": lagSeconds},
			Hint:        fmt.Sprintf("Replication lag exceeds %.0fs — check replica load, network, or WAL sender", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "pg_replication_lag.md",
			BlastRadius: blastRadiusPgDatabase,
		})
	}
	return problems, nil
}

// PgDeadTupleRatioDetector detects tables with excessive dead tuples needing vacuum
type PgDeadTupleRatioDetector struct {
	interval  time.Duration
	threshold float64
}

func NewPgDeadTupleRatioDetector() *PgDeadTupleRatioDetector {
	return &PgDeadTupleRatioDetector{interval: pgpulseCheckInterval, threshold: pgDeadTupleRatioThreshold}
}

func (d *PgDeadTupleRatioDetector) Name() string            { return "pg_dead_tuple_ratio" }
func (d *PgDeadTupleRatioDetector) EntityTypes() []string   { return []string{"postgresql_table"} }
func (d *PgDeadTupleRatioDetector) Interval() time.Duration { return d.interval }

func (d *PgDeadTupleRatioDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`pg_dead_tuple_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pg dead tuple ratio query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		table := string(sample.Metric["table"])
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "postgresql"
		}
		if table == "" {
			table = "unknown"
		}

		ratio := float64(sample.Value) * 100
		entity := fmt.Sprintf("%s/%s", instance, table)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/pg_dead_tuples", entity),
			Entity:      entity,
			EntityType:  "postgresql_table",
			Type:        "pg_dead_tuple_ratio",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Dead tuples at %.0f%% on %s", ratio, table),
			Message:     fmt.Sprintf("pgpulse: table %s has %.0f%% dead tuples — vacuum may be blocked or lagging", table, ratio),
			Labels:      map[string]string{"instance": instance, "table": table},
			Metrics:     map[string]float64{"dead_tuple_ratio_percent": ratio},
			Hint:        fmt.Sprintf("Dead tuple ratio above %.0f%% — check autovacuum status and long-running transactions", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "pg_dead_tuple_ratio.md",
			BlastRadius: blastRadiusPgTable,
		})
	}
	return problems, nil
}

// PgLockChainDepthDetector detects deep lock wait chains indicating contention
type PgLockChainDepthDetector struct {
	interval  time.Duration
	threshold int
}

func NewPgLockChainDepthDetector() *PgLockChainDepthDetector {
	return &PgLockChainDepthDetector{interval: pgpulseCheckInterval, threshold: pgLockChainDepthThreshold}
}

func (d *PgLockChainDepthDetector) Name() string            { return "pg_lock_chain_depth" }
func (d *PgLockChainDepthDetector) EntityTypes() []string   { return []string{"postgresql"} }
func (d *PgLockChainDepthDetector) Interval() time.Duration { return d.interval }

func (d *PgLockChainDepthDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`pg_lock_chain_max_depth > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pg lock chain depth query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "postgresql"
		}

		depth := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/pg_lock_chain_depth", instance),
			Entity:      instance,
			EntityType:  "postgresql",
			Type:        "pg_lock_chain_depth",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Lock chain depth %d on %s", int(depth), instance),
			Message:     fmt.Sprintf("pgpulse: lock wait chain depth %d — queries are blocking each other", int(depth)),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"chain_depth": depth},
			Hint:        fmt.Sprintf("Lock chain deeper than %d — identify and terminate the blocking query", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "pg_lock_chain_depth.md",
			BlastRadius: blastRadiusPgQuery,
		})
	}
	return problems, nil
}

// PgSlowQueriesDetector detects when many slow queries are running concurrently
type PgSlowQueriesDetector struct {
	interval  time.Duration
	threshold int
}

func NewPgSlowQueriesDetector() *PgSlowQueriesDetector {
	return &PgSlowQueriesDetector{interval: pgpulseCheckInterval, threshold: pgSlowQueriesThreshold}
}

func (d *PgSlowQueriesDetector) Name() string            { return "pg_slow_queries" }
func (d *PgSlowQueriesDetector) EntityTypes() []string   { return []string{"postgresql"} }
func (d *PgSlowQueriesDetector) Interval() time.Duration { return d.interval }

func (d *PgSlowQueriesDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`pg_slow_queries > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pg slow queries query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "postgresql"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/pg_slow_queries", instance),
			Entity:      instance,
			EntityType:  "postgresql",
			Type:        "pg_slow_queries",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d slow queries on %s", int(count), instance),
			Message:     fmt.Sprintf("pgpulse: %d concurrent slow queries running on %s", int(count), instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"slow_query_count": count},
			Hint:        fmt.Sprintf("More than %d slow queries — check pg_stat_activity for long-running statements", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "pg_slow_queries.md",
			BlastRadius: blastRadiusPgQuery,
		})
	}
	return problems, nil
}
