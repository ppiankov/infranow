package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	airflowpulseCheckInterval = 30 * time.Second

	// Thresholds
	airflowDAGFailureRateThreshold     = 0.10 // 10% failure rate
	airflowSchedulerHeartbeatThreshold = 30.0 // 30 seconds
	airflowQueuedTasksThreshold        = 100  // queued tasks
	airflowPoolUsedRatioThreshold      = 0.90 // 90% pool usage
	airflowZombieTasksThreshold        = 0    // any zombie tasks

	// Blast radius
	blastRadiusAirflowDAG       = 5
	blastRadiusAirflowScheduler = 10
	blastRadiusAirflowExecutor  = 5
	blastRadiusAirflowPool      = 8
	blastRadiusAirflowTask      = 3
)

// AirflowDAGFailureRateDetector detects when DAG failure rate exceeds threshold
type AirflowDAGFailureRateDetector struct {
	interval  time.Duration
	threshold float64
}

func NewAirflowDAGFailureRateDetector() *AirflowDAGFailureRateDetector {
	return &AirflowDAGFailureRateDetector{interval: airflowpulseCheckInterval, threshold: airflowDAGFailureRateThreshold}
}

func (d *AirflowDAGFailureRateDetector) Name() string            { return "airflow_dag_failure_rate" }
func (d *AirflowDAGFailureRateDetector) EntityTypes() []string   { return []string{"airflow_dag"} }
func (d *AirflowDAGFailureRateDetector) Interval() time.Duration { return d.interval }

func (d *AirflowDAGFailureRateDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`airflow_dag_failed_runs_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("airflow DAG failure rate query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		dag := string(sample.Metric["dag_id"])
		if dag == "" {
			dag = "unknown"
		}
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "airflow"
		}

		ratio := float64(sample.Value) * 100
		entity := fmt.Sprintf("%s/%s", instance, dag)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/airflow_dag_failure_rate", entity),
			Entity:      entity,
			EntityType:  "airflow_dag",
			Type:        "airflow_dag_failure_rate",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("DAG %s failing at %.0f%%", dag, ratio),
			Message:     fmt.Sprintf("airflowpulse: DAG %s has %.0f%% failure rate — pipeline reliability degraded", dag, ratio),
			Labels:      map[string]string{"instance": instance, "dag_id": dag},
			Metrics:     map[string]float64{"failure_rate_percent": ratio},
			Hint:        fmt.Sprintf("Failure rate above %.0f%% — check task logs and upstream dependencies", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "airflow_dag_failure_rate.md",
			BlastRadius: blastRadiusAirflowDAG,
		})
	}
	return problems, nil
}

// AirflowSchedulerHeartbeatDetector detects when the Airflow scheduler is unresponsive
type AirflowSchedulerHeartbeatDetector struct {
	interval  time.Duration
	threshold float64
}

func NewAirflowSchedulerHeartbeatDetector() *AirflowSchedulerHeartbeatDetector {
	return &AirflowSchedulerHeartbeatDetector{interval: airflowpulseCheckInterval, threshold: airflowSchedulerHeartbeatThreshold}
}

func (d *AirflowSchedulerHeartbeatDetector) Name() string { return "airflow_scheduler_heartbeat" }
func (d *AirflowSchedulerHeartbeatDetector) EntityTypes() []string {
	return []string{"airflow_scheduler"}
}
func (d *AirflowSchedulerHeartbeatDetector) Interval() time.Duration { return d.interval }

func (d *AirflowSchedulerHeartbeatDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`airflow_scheduler_heartbeat_seconds > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("airflow scheduler heartbeat query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "airflow-scheduler"
		}

		seconds := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/airflow_scheduler_heartbeat", instance),
			Entity:      instance,
			EntityType:  "airflow_scheduler",
			Type:        "airflow_scheduler_heartbeat",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("Scheduler heartbeat %.0fs ago on %s", seconds, instance),
			Message:     fmt.Sprintf("airflowpulse: scheduler %s last heartbeat %.0fs ago — no new tasks being scheduled", instance, seconds),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"heartbeat_seconds": seconds},
			Hint:        fmt.Sprintf("Scheduler heartbeat older than %.0fs — check scheduler process and database connectivity", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "airflow_scheduler_heartbeat.md",
			BlastRadius: blastRadiusAirflowScheduler,
		})
	}
	return problems, nil
}

// AirflowTaskQueueBacklogDetector detects when the task queue has too many pending tasks
type AirflowTaskQueueBacklogDetector struct {
	interval  time.Duration
	threshold int
}

func NewAirflowTaskQueueBacklogDetector() *AirflowTaskQueueBacklogDetector {
	return &AirflowTaskQueueBacklogDetector{interval: airflowpulseCheckInterval, threshold: airflowQueuedTasksThreshold}
}

func (d *AirflowTaskQueueBacklogDetector) Name() string            { return "airflow_task_queue_backlog" }
func (d *AirflowTaskQueueBacklogDetector) EntityTypes() []string   { return []string{"airflow_executor"} }
func (d *AirflowTaskQueueBacklogDetector) Interval() time.Duration { return d.interval }

func (d *AirflowTaskQueueBacklogDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`airflow_queued_tasks > %d`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("airflow task queue backlog query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "airflow"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/airflow_task_queue_backlog", instance),
			Entity:      instance,
			EntityType:  "airflow_executor",
			Type:        "airflow_task_queue_backlog",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d tasks queued on %s", int(count), instance),
			Message:     fmt.Sprintf("airflowpulse: %d tasks queued on %s — executor cannot keep up", int(count), instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"queued_tasks": count},
			Hint:        fmt.Sprintf("More than %d queued tasks — increase executor parallelism or worker count", d.threshold),
			RunbookURL:  models.RunbookBaseURL + "airflow_task_queue_backlog.md",
			BlastRadius: blastRadiusAirflowExecutor,
		})
	}
	return problems, nil
}

// AirflowPoolExhaustionDetector detects when Airflow pools are near capacity
type AirflowPoolExhaustionDetector struct {
	interval  time.Duration
	threshold float64
}

func NewAirflowPoolExhaustionDetector() *AirflowPoolExhaustionDetector {
	return &AirflowPoolExhaustionDetector{interval: airflowpulseCheckInterval, threshold: airflowPoolUsedRatioThreshold}
}

func (d *AirflowPoolExhaustionDetector) Name() string            { return "airflow_pool_exhaustion" }
func (d *AirflowPoolExhaustionDetector) EntityTypes() []string   { return []string{"airflow_pool"} }
func (d *AirflowPoolExhaustionDetector) Interval() time.Duration { return d.interval }

func (d *AirflowPoolExhaustionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := fmt.Sprintf(`airflow_pool_used_ratio > %f`, d.threshold)
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("airflow pool exhaustion query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		pool := string(sample.Metric["pool"])
		if pool == "" {
			pool = "default_pool"
		}
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "airflow"
		}

		ratio := float64(sample.Value) * 100
		entity := fmt.Sprintf("%s/%s", instance, pool)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/airflow_pool_exhaustion", entity),
			Entity:      entity,
			EntityType:  "airflow_pool",
			Type:        "airflow_pool_exhaustion",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("Pool %s at %.0f%%", pool, ratio),
			Message:     fmt.Sprintf("airflowpulse: pool %s at %.0f%% capacity — tasks stuck in queued state", pool, ratio),
			Labels:      map[string]string{"instance": instance, "pool": pool},
			Metrics:     map[string]float64{"pool_used_percent": ratio},
			Hint:        fmt.Sprintf("Pool usage above %.0f%% — increase pool slots or redistribute tasks across pools", d.threshold*100),
			RunbookURL:  models.RunbookBaseURL + "airflow_pool_exhaustion.md",
			BlastRadius: blastRadiusAirflowPool,
		})
	}
	return problems, nil
}

// AirflowZombieTasksDetector detects orphaned tasks that are consuming resources
type AirflowZombieTasksDetector struct {
	interval time.Duration
}

func NewAirflowZombieTasksDetector() *AirflowZombieTasksDetector {
	return &AirflowZombieTasksDetector{interval: airflowpulseCheckInterval}
}

func (d *AirflowZombieTasksDetector) Name() string            { return "airflow_zombie_tasks" }
func (d *AirflowZombieTasksDetector) EntityTypes() []string   { return []string{"airflow_task"} }
func (d *AirflowZombieTasksDetector) Interval() time.Duration { return d.interval }

func (d *AirflowZombieTasksDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := `airflow_zombie_tasks > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("airflow zombie tasks query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		instance := string(sample.Metric["instance"])
		if instance == "" {
			instance = "airflow"
		}

		count := float64(sample.Value)

		problems = append(problems, &models.Problem{
			ID:          fmt.Sprintf("%s/airflow_zombie_tasks", instance),
			Entity:      instance,
			EntityType:  "airflow_task",
			Type:        "airflow_zombie_tasks",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("%d zombie tasks on %s", int(count), instance),
			Message:     fmt.Sprintf("airflowpulse: %d zombie tasks on %s — orphaned tasks consuming resources", int(count), instance),
			Labels:      map[string]string{"instance": instance},
			Metrics:     map[string]float64{"zombie_tasks": count},
			Hint:        "Zombie tasks detected — check for worker crashes or executor instability",
			RunbookURL:  models.RunbookBaseURL + "airflow_zombie_tasks.md",
			BlastRadius: blastRadiusAirflowTask,
		})
	}
	return problems, nil
}
