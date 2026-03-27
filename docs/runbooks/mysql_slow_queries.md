# MySQL Slow Queries

## What it means

More than 10 slow queries are running concurrently. A high count of active slow queries indicates systemic performance problems — queries are piling up, consuming connections, and degrading overall database responsiveness.

## Common causes

- Missing indexes causing full table scans
- Unoptimized queries with unnecessary joins or subqueries
- Lock contention causing queries to wait for row or table locks
- Disk I/O bottleneck (InnoDB buffer pool too small for working set)
- Large result sets being sorted or grouped without appropriate indexes

## Diagnostic commands

```bash
# PromQL: active slow queries
mysql_slow_queries_active

# PromQL: slow query rate
rate(mysql_slow_queries_total[5m])

# SQL: check currently running queries
SHOW PROCESSLIST;

# SQL: find queries running longer than 10 seconds
SELECT * FROM information_schema.processlist WHERE TIME > 10 AND COMMAND != 'Sleep';

# SQL: check slow query log status
SHOW VARIABLES LIKE 'slow_query_log%';
```

## Resolution

- Identify the slowest queries from `SHOW PROCESSLIST` or the slow query log
- Run `EXPLAIN` on slow queries to find missing indexes or inefficient plans
- Add indexes based on `EXPLAIN` analysis (covering indexes where possible)
- Check `innodb_buffer_pool_size` — should be 70-80% of available memory for dedicated DB hosts
- Kill long-running queries as an immediate fix: `KILL <process_id>`
