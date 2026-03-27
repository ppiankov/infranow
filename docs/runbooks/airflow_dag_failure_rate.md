# Airflow DAG Failure Rate

## What it means

The DAG failure rate has exceeded 10%. A sustained high failure rate means pipelines are not delivering data reliably, which can cascade into downstream data quality issues and SLA breaches.

## Common causes

- Upstream dependency failures (APIs, databases, external services unavailable)
- Code bugs in custom operators or task callables
- Resource exhaustion on worker nodes (OOM, disk full, CPU saturation)
- Connection pool timeouts to external systems (databases, cloud APIs)
- Environment drift between development and production

## Diagnostic commands

```bash
# PromQL: DAG failure ratio
airflow_dag_failed_runs_ratio

# PromQL: failed task instances rate
rate(airflow_task_instance_failures_total[1h])

# CLI: list recent DAG runs with status
airflow dags list-runs --state failed

# CLI: check task logs for a failed run
airflow tasks logs <dag_id> <task_id> <execution_date>

# CLI: list DAGs with recent failures
airflow dags list-runs -o table
```

## Resolution

- Check task logs for the specific error causing failures
- Fix operator code bugs or update dependencies causing import errors
- Add retries with exponential backoff (`retries`, `retry_delay`, `retry_exponential_backoff`)
- Check worker resource utilization — increase worker memory or CPU if exhausted
- Verify external service connectivity and credentials
