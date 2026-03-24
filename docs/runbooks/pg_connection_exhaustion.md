# PostgreSQL Connection Exhaustion

## What it means

PostgreSQL is using more than 85% of `max_connections`. When connections reach the limit, new clients get "too many connections" errors and the database becomes effectively unavailable.

## Common causes

- Connection pool misconfiguration (too many workers, no pool limit)
- Connection leaks in application code (connections opened but never closed)
- Idle connections accumulating from long-lived services
- Burst traffic exceeding pool capacity
- PgBouncer or connection pooler not in use

## Diagnostic commands

```bash
# PromQL: current connection ratio
pg_connections_used_ratio

# PromQL: connections by user
pg_connections_by_user

# PromQL: idle connections
pg_idle_connections

# SQL: check active connections
SELECT usename, state, count(*) FROM pg_stat_activity GROUP BY usename, state ORDER BY count DESC;

# SQL: check max_connections
SHOW max_connections;
```

## Resolution

- Identify the user/application consuming the most connections
- Check for connection leaks — idle connections that never close
- Add or tune a connection pooler (PgBouncer, pgpool-II)
- Increase `max_connections` as a short-term fix (requires restart)
- Set `idle_in_transaction_session_timeout` to kill abandoned transactions
