# PostgreSQL Dead Tuple Ratio

## What it means

A table has more than 20% dead tuples. Dead tuples are rows deleted or updated but not yet reclaimed by vacuum. They waste disk space, bloat indexes, and slow sequential scans.

## Common causes

- Autovacuum not keeping up with update/delete rate
- Long-running transactions holding back the oldest visible transaction ID
- Autovacuum workers maxed out across all tables
- `autovacuum_vacuum_cost_delay` set too high, throttling vacuum
- Manual VACUUM never scheduled for large tables

## Diagnostic commands

```bash
# PromQL: dead tuple ratio per table
pg_dead_tuple_ratio

# PromQL: dead tuples count
pg_dead_tuples

# PromQL: last autovacuum
pg_last_autovacuum_seconds

# SQL: check vacuum stats
SELECT relname, n_dead_tup, n_live_tup, last_autovacuum, last_vacuum
FROM pg_stat_user_tables WHERE n_dead_tup > 0 ORDER BY n_dead_tup DESC;

# SQL: check for long transactions blocking vacuum
SELECT pid, xact_start, state, query FROM pg_stat_activity
WHERE xact_start < now() - interval '1 hour' ORDER BY xact_start;
```

## Resolution

- Kill long-running transactions that block vacuum: `SELECT pg_terminate_backend(pid);`
- Run manual vacuum on the affected table: `VACUUM (VERBOSE) table_name;`
- Tune autovacuum: lower `autovacuum_vacuum_cost_delay`, increase `autovacuum_max_workers`
- Set per-table autovacuum thresholds for high-churn tables
