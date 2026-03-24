# ClickHouse Part Count Explosion

## What it means

A single partition has more than 300 active parts. ClickHouse will reject inserts with "too many parts" error (default limit 300) when this threshold is reached. This is a critical condition that can make the table unwritable.

## Common causes

- Too many small inserts (one row per INSERT instead of batching)
- Partition key too granular (e.g., partitioning by minute instead of month)
- Merges falling behind insert rate (see merge pressure)
- Table using `ReplacingMergeTree` or `CollapsingMergeTree` with deferred merges
- Recovery from replica sync generating many small parts

## Diagnostic commands

```bash
# PromQL: parts per partition
clickhouse_parts_per_partition

# PromQL: total parts per table
clickhouse_parts_total

# PromQL: active merges
clickhouse_merges_active

# SQL: check parts per partition
SELECT database, table, partition, count() AS parts
FROM system.parts WHERE active GROUP BY database, table, partition
ORDER BY parts DESC LIMIT 20;

# SQL: check insert rate
SELECT event_time, InsertedRows FROM system.events WHERE event = 'InsertedRows';
```

## Resolution

- Batch inserts immediately — this is the most common fix
- Use `Buffer` engine or async inserts to aggregate writes
- Optimize partition key — use monthly or daily, not hourly
- Force merge: `OPTIMIZE TABLE db.table PARTITION 'partition' FINAL`
- Increase `parts_to_throw_insert` temporarily while fixing the root cause
