# Airflow Zombie Tasks

## What it means

Zombie tasks have been detected. These are task instances that the metadata database shows as running, but no worker process is actually executing them. They consume pool and worker slots without making progress, blocking other tasks from running.

## Common causes

- Worker process killed mid-execution (OOM kill, SIGKILL, node eviction)
- Executor lost connection to the worker (network partition, broker failure)
- Worker host crashed or restarted without graceful task shutdown
- Kubernetes pod evicted due to resource limits or node pressure
- Celery broker (Redis/RabbitMQ) dropped the task acknowledgment

## Diagnostic commands

```bash
# PromQL: zombie task count
airflow_zombie_tasks

# PromQL: task instance state distribution
airflow_task_instance_running

# CLI: list task instances in running state
airflow tasks list <dag_id> --tree

# Host: check for OOM kills on worker nodes
dmesg | grep -i "oom"
journalctl -k | grep -i "killed process"

# Kubernetes: check pod evictions
kubectl get events --field-selector reason=Evicted
```

## Resolution

- Let the scheduler's built-in zombie detection handle cleanup (runs every `scheduler_zombie_task_threshold` seconds)
- Investigate worker stability — check for OOM kills on worker hosts or pods
- Increase worker memory or CPU limits if tasks are being OOM-killed
- Set `execution_timeout` on tasks to prevent indefinite hangs
- Check Celery broker health if using CeleryExecutor (Redis/RabbitMQ connectivity and memory)
