# Airflow Task Queue Backlog

## What it means

More than 100 tasks are queued and waiting for execution. A growing backlog means tasks are being scheduled faster than workers can execute them, leading to pipeline delays and potential SLA violations.

## Common causes

- Insufficient worker capacity for the current workload
- Executor bottleneck (Celery broker overloaded, Kubernetes pod limits reached)
- Slow tasks blocking worker slots for extended periods
- Pool limits set too low for the number of tasks being scheduled
- Worker pods failing to start (image pull errors, resource quotas)

## Diagnostic commands

```bash
# PromQL: queued task count
airflow_queued_tasks

# PromQL: running task count
airflow_running_tasks

# CLI: check task states
airflow tasks list <dag_id> --tree

# CLI: check pool usage
airflow pools list

# Celery: check worker status (if using CeleryExecutor)
celery -A airflow.executors.celery_executor inspect active
celery -A airflow.executors.celery_executor inspect reserved
```

## Resolution

- Increase worker count or parallelism (`parallelism`, `dag_concurrency` settings)
- Adjust pool sizes to match workload requirements
- Investigate and optimize slow tasks that are blocking worker slots
- Check executor-specific limits (Celery concurrency, Kubernetes pod quotas)
- Scale out workers horizontally if vertical scaling is insufficient
