# ClickHouse Merge Pressure

## What it means

ClickHouse has more than 10 active background merges running concurrently. Merge pressure indicates the system cannot merge parts fast enough to keep up with insert rate. If unchecked, this leads to part count explosion and "too many parts" errors.

## Common causes

- High insert rate creating too many small parts
- Inserts not batched (many small INSERT statements instead of bulk)
- Insufficient background pool threads for the write workload
- Disk I/O saturated, slowing merge throughput
- Wide tables or complex ORDER BY keys making merges expensive

## Diagnostic commands

```bash
# PromQL: active merges
clickhouse_merges_active

# PromQL: merge throughput
clickhouse_merge_bytes_per_second

# PromQL: parts being merged
clickhouse_merge_parts_count

# SQL: check active merges
SELECT database, table, elapsed, progress, num_parts, result_part_name
FROM system.merges ORDER BY elapsed DESC;

# SQL: check background pool utilization
SELECT * FROM system.metrics WHERE metric LIKE '%BackgroundMerges%';
```

## Resolution

- Batch inserts: aim for 1 insert per second per table, not per row
- Increase `background_pool_size` to allow more concurrent merges
- Check disk I/O — consider faster storage for merge-heavy workloads
- Use `Buffer` engine or async inserts to aggregate small writes
- Monitor `clickhouse_parts_per_partition` to catch escalation early
