package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	clickpulseCheckInterval = 30 * time.Second

	// Thresholds
	chMergesActiveThreshold          = 10   // concurrent merges
	chPartsPerPartitionThreshold     = 300  // parts per partition
	chReplicaLagThreshold            = 30.0 // seconds
	chKeeperLatencyThreshold         = 0.5  // seconds
	chKeeperOutstandingReqsThreshold = 100  // pending requests

	// Blast radius
	blastRadiusChCluster = 8
	blastRadiusChTable   = 5
	blastRadiusChKeeper  = 10
)

// ChMergePressureDetector detects when ClickHouse has too many active merges
type ChMergePressureDetector struct {
	interval  time.Duration
	threshold int
}

func NewChMergePressureDetector() *ChMergePressureDetector {
	return &ChMergePressureDetector{interval: clickpulseCheckInterval, threshold: chMergesActiveThreshold}
}

func (d *ChMergePressureDetector) Name() string            { return "ch_merge_pressure" }
func (d *ChMergePressureDetector) EntityTypes() []string   { return []string{"clickhouse"} }
func (d *ChMergePressureDetector) Interval() time.Duration { return d.interval }

func (d *ChMergePressureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`clickhouse_merges_active > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse merge pressure query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		node := string(sample.Metric["node"])
		if node == "" {
			node = string(sample.Metric["instance"])
		}
		if node == "" {
			node = "clickhouse"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_merge_pressure", node),
			Entity:      node,
			EntityType:  "clickhouse",
			Type:        "ch_merge_pressure",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d active merges on %s", int(count), node),
			Message:     fmt.Sprintf("clickpulse: %d concurrent merges on %s — inserts may back up", int(count), node),
			Labels:      map[string]string{"node": node},
			Metrics:     map[string]float64{"active_merges": count},
			Hint:        fmt.Sprintf("More than %d active merges — check insert rate and part count", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "ch_merge_pressure.md",
			BlastRadius: blastRadiusChCluster,
		})
	}
	return problems, nil
}

// ChStuckMutationsDetector detects mutations that appear stuck in ClickHouse
type ChStuckMutationsDetector struct {
	interval time.Duration
}

func NewChStuckMutationsDetector() *ChStuckMutationsDetector {
	return &ChStuckMutationsDetector{interval: clickpulseCheckInterval}
}

func (d *ChStuckMutationsDetector) Name() string            { return "ch_stuck_mutations" }
func (d *ChStuckMutationsDetector) EntityTypes() []string   { return []string{"clickhouse"} }
func (d *ChStuckMutationsDetector) Interval() time.Duration { return d.interval }

func (d *ChStuckMutationsDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := `clickhouse_mutations_stuck > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse stuck mutations query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		node := string(sample.Metric["node"])
		if node == "" {
			node = string(sample.Metric["instance"])
		}
		if node == "" {
			node = "clickhouse"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_stuck_mutations", node),
			Entity:      node,
			EntityType:  "clickhouse",
			Type:        "ch_stuck_mutations",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("%d stuck mutations on %s", int(count), node),
			Message:     fmt.Sprintf("clickpulse: %d mutations stuck on %s — data may be inconsistent", int(count), node),
			Labels:      map[string]string{"node": node},
			Metrics:     map[string]float64{"stuck_mutations": count},
			Hint:        "Check system.mutations for stuck entries — may need KILL MUTATION",
			RunbookURL:  models.RunbookBaseURL + "ch_stuck_mutations.md",
			BlastRadius: blastRadiusChCluster,
		})
	}
	return problems, nil
}

// ChReplicaLagDetector detects high replication lag in ClickHouse replicated tables
type ChReplicaLagDetector struct {
	interval  time.Duration
	threshold float64
}

func NewChReplicaLagDetector() *ChReplicaLagDetector {
	return &ChReplicaLagDetector{interval: clickpulseCheckInterval, threshold: chReplicaLagThreshold}
}

func (d *ChReplicaLagDetector) Name() string            { return "ch_replica_lag" }
func (d *ChReplicaLagDetector) EntityTypes() []string   { return []string{"clickhouse"} }
func (d *ChReplicaLagDetector) Interval() time.Duration { return d.interval }

func (d *ChReplicaLagDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`clickhouse_replica_lag_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse replica lag query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		node := string(sample.Metric["node"])
		if node == "" {
			node = string(sample.Metric["instance"])
		}
		if node == "" {
			node = "clickhouse"
		}

		lagSeconds := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_replica_lag", node),
			Entity:      node,
			EntityType:  "clickhouse",
			Type:        "ch_replica_lag",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Replica lag %.0fs on %s", lagSeconds, node),
			Message:     fmt.Sprintf("clickpulse: replica %s lagging %.0f seconds", node, lagSeconds),
			Labels:      map[string]string{"node": node},
			Metrics:     map[string]float64{"lag_seconds": lagSeconds},
			Hint:        fmt.Sprintf("Replication lag exceeds %.0fs — check ZooKeeper/Keeper health and network", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "ch_replica_lag.md",
			BlastRadius: blastRadiusChCluster,
		})
	}
	return problems, nil
}

// ChPartCountExplosionDetector detects when a partition has too many parts (too-many-parts error risk)
type ChPartCountExplosionDetector struct {
	interval  time.Duration
	threshold int
}

func NewChPartCountExplosionDetector() *ChPartCountExplosionDetector {
	return &ChPartCountExplosionDetector{interval: clickpulseCheckInterval, threshold: chPartsPerPartitionThreshold}
}

func (d *ChPartCountExplosionDetector) Name() string            { return "ch_part_count_explosion" }
func (d *ChPartCountExplosionDetector) EntityTypes() []string   { return []string{"clickhouse_table"} }
func (d *ChPartCountExplosionDetector) Interval() time.Duration { return d.interval }

func (d *ChPartCountExplosionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`clickhouse_parts_per_partition > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse part count query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		node := string(sample.Metric["node"])
		if node == "" {
			node = string(sample.Metric["instance"])
		}
		if node == "" {
			node = "clickhouse"
		}
		database := string(sample.Metric["database"])
		table := string(sample.Metric["table"])
		partition := string(sample.Metric["partition"])

		parts := float64(sample.Value)
		entity := fmt.Sprintf("%s/%s.%s/%s", node, database, table, partition)

		labels := map[string]string{"node": node}
		if database != "" {
			labels["database"] = database
		}
		if table != "" {
			labels["table"] = table
		}
		if partition != "" {
			labels["partition"] = partition
		}

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_part_count", entity),
			Entity:      entity,
			EntityType:  "clickhouse_table",
			Type:        "ch_part_count_explosion",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("%d parts in %s.%s partition %s", int(parts), database, table, partition),
			Message:     fmt.Sprintf("clickpulse: partition %s of %s.%s has %d parts — too-many-parts error imminent", partition, database, table, int(parts)),
			Labels:      labels,
			Metrics:     map[string]float64{"parts_per_partition": parts},
			Hint:        fmt.Sprintf("Parts per partition above %d — reduce insert frequency or optimize partition key", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "ch_part_count_explosion.md",
			BlastRadius: blastRadiusChTable,
		})
	}
	return problems, nil
}

// ChDDLQueueStuckDetector detects stuck distributed DDL operations
type ChDDLQueueStuckDetector struct {
	interval time.Duration
}

func NewChDDLQueueStuckDetector() *ChDDLQueueStuckDetector {
	return &ChDDLQueueStuckDetector{interval: clickpulseCheckInterval}
}

func (d *ChDDLQueueStuckDetector) Name() string            { return "ch_ddl_queue_stuck" }
func (d *ChDDLQueueStuckDetector) EntityTypes() []string   { return []string{"clickhouse"} }
func (d *ChDDLQueueStuckDetector) Interval() time.Duration { return d.interval }

func (d *ChDDLQueueStuckDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := `clickhouse_ddl_queue_stuck > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse DDL queue stuck query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		node := string(sample.Metric["node"])
		if node == "" {
			node = string(sample.Metric["instance"])
		}
		if node == "" {
			node = "clickhouse"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_ddl_queue_stuck", node),
			Entity:      node,
			EntityType:  "clickhouse",
			Type:        "ch_ddl_queue_stuck",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d stuck DDL entries on %s", int(count), node),
			Message:     fmt.Sprintf("clickpulse: %d distributed DDL entries stuck on %s", int(count), node),
			Labels:      map[string]string{"node": node},
			Metrics:     map[string]float64{"stuck_ddl_entries": count},
			Hint:        "Check system.distributed_ddl_queue for stuck entries — may indicate ZooKeeper issues",
			RunbookURL:  models.RunbookBaseURL + "ch_ddl_queue_stuck.md",
			BlastRadius: blastRadiusChCluster,
		})
	}
	return problems, nil
}

// ChKeeperHighLatencyDetector detects when ZooKeeper/Keeper latency is too high
type ChKeeperHighLatencyDetector struct {
	interval  time.Duration
	threshold float64
}

func NewChKeeperHighLatencyDetector() *ChKeeperHighLatencyDetector {
	return &ChKeeperHighLatencyDetector{interval: clickpulseCheckInterval, threshold: chKeeperLatencyThreshold}
}

func (d *ChKeeperHighLatencyDetector) Name() string            { return "ch_keeper_high_latency" }
func (d *ChKeeperHighLatencyDetector) EntityTypes() []string   { return []string{"clickhouse_keeper"} }
func (d *ChKeeperHighLatencyDetector) Interval() time.Duration { return d.interval }

func (d *ChKeeperHighLatencyDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`clickhouse_keeper_latency_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse keeper latency query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		keeper := string(sample.Metric["keeper"])
		if keeper == "" {
			keeper = string(sample.Metric["instance"])
		}
		if keeper == "" {
			keeper = "keeper"
		}

		latencyMs := float64(sample.Value) * 1000

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_keeper_high_latency", keeper),
			Entity:      keeper,
			EntityType:  "clickhouse_keeper",
			Type:        "ch_keeper_high_latency",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Keeper latency %.0fms on %s", latencyMs, keeper),
			Message:     fmt.Sprintf("clickpulse: Keeper %s latency at %.0fms — replication and DDL ops affected", keeper, latencyMs),
			Labels:      map[string]string{"keeper": keeper},
			Metrics:     map[string]float64{"latency_ms": latencyMs},
			Hint:        fmt.Sprintf("Keeper latency above %.0fms — check Keeper node resources and network", d.threshold*1000),
			RunbookURL:  models.RunbookBaseURL + "ch_keeper_high_latency.md",
			BlastRadius: blastRadiusChKeeper,
		})
	}
	return problems, nil
}

// ChKeeperOutstandingRequestsDetector detects when Keeper has a large request backlog
type ChKeeperOutstandingRequestsDetector struct {
	interval  time.Duration
	threshold int
}

func NewChKeeperOutstandingRequestsDetector() *ChKeeperOutstandingRequestsDetector {
	return &ChKeeperOutstandingRequestsDetector{interval: clickpulseCheckInterval, threshold: chKeeperOutstandingReqsThreshold}
}

func (d *ChKeeperOutstandingRequestsDetector) Name() string {
	return "ch_keeper_outstanding_requests"
}
func (d *ChKeeperOutstandingRequestsDetector) EntityTypes() []string {
	return []string{"clickhouse_keeper"}
}
func (d *ChKeeperOutstandingRequestsDetector) Interval() time.Duration { return d.interval }

func (d *ChKeeperOutstandingRequestsDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`clickhouse_keeper_outstanding_requests > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("clickhouse keeper outstanding requests query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		keeper := string(sample.Metric["keeper"])
		if keeper == "" {
			keeper = string(sample.Metric["instance"])
		}
		if keeper == "" {
			keeper = "keeper"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/ch_keeper_outstanding_requests", keeper),
			Entity:      keeper,
			EntityType:  "clickhouse_keeper",
			Type:        "ch_keeper_outstanding_requests",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("%d outstanding Keeper requests on %s", int(count), keeper),
			Message:     fmt.Sprintf("clickpulse: Keeper %s has %d outstanding requests — overloaded", keeper, int(count)),
			Labels:      map[string]string{"keeper": keeper},
			Metrics:     map[string]float64{"outstanding_requests": count},
			Hint:        fmt.Sprintf("Outstanding requests above %d — Keeper cannot keep up with cluster demand", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "ch_keeper_outstanding_requests.md",
			BlastRadius: blastRadiusChKeeper,
		})
	}
	return problems, nil
}
