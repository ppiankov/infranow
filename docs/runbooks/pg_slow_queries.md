# PostgreSQL Slow Queries

## What it means

More than 5 queries are concurrently running beyond the slow query threshold (default 5s). This indicates the database is under stress — queries that should be fast are taking too long.

## Common causes

- Missing indexes on frequently queried columns
- Query planner choosing sequential scans on large tables
- Lock contention causing queries to wait
- Resource exhaustion (CPU, memory, I/O) on the database host
- N+1 query patterns or unoptimized joins

## Diagnostic commands

```bash
# PromQL: slow query count
pg_slow_queries

# PromQL: longest running query
pg_longest_query_seconds

# PromQL: waiting queries
pg_waiting_queries

# SQL: find slow queries
SELECT pid, now() - query_start AS duration, state, query
FROM pg_stat_activity WHERE state = 'active'
  AND now() - query_start > interval '5 seconds'
ORDER BY duration DESC;

# SQL: check for missing indexes
SELECT relname, seq_scan, idx_scan
FROM pg_stat_user_tables WHERE seq_scan > idx_scan ORDER BY seq_scan DESC LIMIT 20;
```

## Resolution

- Identify the slow queries and add missing indexes
- Use `EXPLAIN ANALYZE` to diagnose query plans
- Set `statement_timeout` to prevent runaway queries
- Check `pg_stat_statements` for query regression (mean time increase)
- Kill specific slow queries: `SELECT pg_terminate_backend(pid);`
