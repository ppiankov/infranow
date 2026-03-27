# MySQL Replication Lag

## What it means

A replica is lagging 30 seconds or more behind the primary. Reads from the replica return stale data, and failover to a lagging replica risks data inconsistency.

## Common causes

- Single-threaded SQL applier bottleneck (default before MySQL 8.0.27)
- Large transactions that take a long time to replay on the replica
- Slow disk I/O on the replica (degraded storage, competing workloads)
- Heavy read load on the replica competing with replication for resources
- Network latency between primary and replica

## Diagnostic commands

```bash
# PromQL: replication lag in seconds
mysql_replication_lag_seconds

# PromQL: replication thread status
mysql_replica_sql_running

# SQL: check replication status
SHOW REPLICA STATUS\G

# SQL: check seconds behind primary
SHOW REPLICA STATUS\G -- look at Seconds_Behind_Source

# SQL: check relay log space
SHOW STATUS LIKE 'Relay_log%';
```

## Resolution

- Enable multi-threaded replication (`replica_parallel_workers`, `replica_parallel_type=LOGICAL_CLOCK`)
- Break large transactions into smaller batches to reduce replay time
- Check replica disk I/O — ensure storage is not a bottleneck
- Reduce read load on the lagging replica by routing reads elsewhere
- Check network bandwidth and latency between primary and replica
