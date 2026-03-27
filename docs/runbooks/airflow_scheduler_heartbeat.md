# Airflow Scheduler Heartbeat

## What it means

The scheduler's last heartbeat was more than 30 seconds ago. The scheduler is the core component that triggers DAG runs and schedules task instances. If it stops, no new work is scheduled and the entire pipeline system stalls.

## Common causes

- Scheduler process crashed or was OOM-killed
- Metadata database connection lost or overloaded
- Scheduler host resource exhaustion (CPU, memory, disk)
- Database lock contention from too many concurrent schedulers
- Python import errors in DAG files causing scheduler loop to hang

## Diagnostic commands

```bash
# PromQL: scheduler heartbeat age
airflow_scheduler_heartbeat_seconds

# PromQL: scheduler running status
airflow_scheduler_running

# CLI: check scheduler job status
airflow jobs check --job-type SchedulerJob

# CLI: check scheduler health
airflow db check

# Host: check scheduler process
ps aux | grep "airflow scheduler"
systemctl status airflow-scheduler
```

## Resolution

- Restart the scheduler process immediately to restore pipeline execution
- Check scheduler logs for crash reason (OOM, database errors, import failures)
- Verify metadata database connectivity and responsiveness
- Increase scheduler host resources if OOM-killed (check `dmesg` or `journalctl`)
- Fix DAG file syntax errors that may cause the scheduler parsing loop to hang
