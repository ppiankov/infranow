# PostgreSQL Lock Chain Depth

## What it means

The lock wait chain depth exceeds 3 levels — queries are blocking other queries, which in turn block more queries. This cascading contention can freeze the database for all clients.

## Common causes

- Long-running transaction holding an exclusive lock (ALTER TABLE, VACUUM FULL)
- Application-level deadlock patterns with inconsistent lock ordering
- Bulk updates conflicting with concurrent transactions
- Missing indexes causing full-table locks on UPDATE/DELETE

## Diagnostic commands

```bash
# PromQL: lock chain depth
pg_lock_chain_max_depth

# PromQL: blocked queries
pg_lock_blocked_queries

# PromQL: locks by type
pg_lock_by_type

# SQL: find blocking queries
SELECT blocked_locks.pid AS blocked_pid,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.query AS blocking_query
FROM pg_locks blocked_locks
JOIN pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
  AND blocking_locks.granted AND NOT blocked_locks.granted
JOIN pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid;
```

## Resolution

- Identify the root blocking query and terminate it: `SELECT pg_terminate_backend(pid);`
- Use `lock_timeout` setting to prevent unbounded lock waits
- Review application lock ordering for deadlock patterns
- Schedule exclusive-lock DDL during low-traffic windows
