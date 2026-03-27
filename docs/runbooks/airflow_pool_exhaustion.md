# Airflow Pool Exhaustion

## What it means

A pool is using more than 90% of its allocated slots. Pools control task concurrency for shared resources. When a pool is exhausted, new tasks assigned to that pool queue indefinitely until slots free up, blocking entire DAG pipelines.

## Common causes

- Too many tasks assigned to a single pool without adequate slot allocation
- Long-running tasks holding pool slots for extended periods
- Pool size not adjusted after workload growth
- Tasks stuck in running state due to worker failures (zombie tasks consuming slots)
- Default pool too small for the overall task volume

## Diagnostic commands

```bash
# PromQL: pool usage ratio
airflow_pool_used_ratio

# PromQL: pool queued tasks
airflow_pool_queued_slots

# CLI: list all pools with usage
airflow pools list

# CLI: check tasks using a specific pool
airflow tasks list <dag_id> --tree

# Airflow UI: Pools page shows open/used/queued per pool
```

## Resolution

- Increase pool slot count to match current workload: `airflow pools set <pool> <new_size> "<description>"`
- Redistribute tasks across multiple pools to balance load on shared resources
- Investigate long-running tasks holding slots — optimize or set timeouts (`execution_timeout`)
- Check for zombie tasks consuming slots without making progress
- Review pool assignments in DAG definitions — ensure tasks use the correct pool
