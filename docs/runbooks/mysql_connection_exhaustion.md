# MySQL Connection Exhaustion

## What it means

MySQL is using more than 85% of `max_connections`. When connections reach the limit, new clients get "Too many connections" errors and the database becomes effectively unavailable.

## Common causes

- Connection pool misconfiguration (pool size too large or no pooling at all)
- Connection leaks in application code (connections opened but never closed)
- Too many application instances each maintaining their own connection pool
- Burst traffic exceeding pool capacity
- No connection pooler (ProxySQL, MySQL Router) in front of MySQL

## Diagnostic commands

```bash
# PromQL: current connection ratio
mysql_connections_used_ratio

# PromQL: connections over time
mysql_threads_connected

# SQL: check current connections
SHOW STATUS LIKE 'Threads_connected';

# SQL: check max_connections setting
SHOW VARIABLES LIKE 'max_connections';

# SQL: list active connections
SHOW PROCESSLIST;
```

## Resolution

- Identify the user/host consuming the most connections via `SHOW PROCESSLIST`
- Check for connection leaks — sleeping connections that never close
- Add or tune a connection pooler (ProxySQL, MySQL Router)
- Increase `max_connections` as a short-term fix (dynamic, no restart required)
- Set `wait_timeout` and `interactive_timeout` to close idle connections automatically
