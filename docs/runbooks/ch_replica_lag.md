# ClickHouse Replica Lag

## What it means

A ClickHouse replica is more than 30 seconds behind the primary. Queries hitting this replica return stale data. In a cluster with replicated tables, this can cause inconsistent results depending on which replica serves the query.

## Common causes

- ZooKeeper/Keeper latency or instability
- Network issues between replicas
- Replica overloaded with queries, slowing fetch from ZooKeeper
- Large INSERT batches creating replication queue backlog
- Replica recovering after restart (catching up on backlog)

## Diagnostic commands

```bash
# PromQL: replica lag
clickhouse_replica_lag_seconds

# PromQL: replication queue size
clickhouse_replica_queue_size

# PromQL: read-only replicas
clickhouse_replica_readonly

# SQL: check replication status
SELECT database, table, is_readonly, absolute_delay, queue_size, inserts_in_queue
FROM system.replicas WHERE absolute_delay > 0 ORDER BY absolute_delay DESC;

# SQL: check ZooKeeper health
SELECT * FROM system.zookeeper WHERE path = '/clickhouse';
```

## Resolution

- Check ZooKeeper/Keeper health — high latency causes replication delays
- Reduce query load on the lagging replica
- If replica is read-only, check `is_readonly` reason in `system.replicas`
- Restart replication for stuck tables: `SYSTEM RESTART REPLICA db.table`
- Verify network connectivity between replica nodes
