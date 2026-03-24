# ClickHouse Keeper High Latency

## What it means

ZooKeeper/ClickHouse Keeper average request latency exceeds 500ms. Keeper coordinates replication, DDL distribution, and leader election — high latency slows all of these, causing replica lag, stuck DDL operations, and degraded cluster performance.

## Common causes

- Keeper node under resource pressure (CPU, memory, disk I/O)
- Too many znodes causing slow tree traversal
- Network latency between ClickHouse nodes and Keeper
- Keeper log/snapshot disk is slow or full
- Too many ephemeral nodes from large number of replicated tables

## Diagnostic commands

```bash
# PromQL: Keeper latency
clickhouse_keeper_latency_seconds

# PromQL: outstanding requests (queue depth)
clickhouse_keeper_outstanding_requests

# PromQL: znode count
clickhouse_keeper_znode_count

# SQL: check ZooKeeper health from ClickHouse
SELECT * FROM system.zookeeper WHERE path = '/';

# Keeper CLI: check stats
echo stat | nc keeper-host 2181
```

## Resolution

- Check Keeper node CPU and memory — scale up if saturated
- Move Keeper data directory to fast storage (SSD/NVMe)
- Reduce znode count by cleaning up unused replicated tables
- Increase Keeper `tickTime` and `syncLimit` if network latency is high
- Consider dedicated Keeper nodes (not co-located with ClickHouse)
