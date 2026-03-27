# MongoDB Connection Exhaustion

## What it means

MongoDB is using more than 85% of `maxIncomingConnections`. When connections reach the limit, new clients get "connection refused" errors and the database becomes effectively unavailable.

## Common causes

- No connection pooling in application code (each request opens a new connection)
- Connection leaks (connections opened but never returned to pool or closed)
- Too many application instances each maintaining their own pool
- High thread count from driver misconfiguration
- Burst traffic exceeding connection limits

## Diagnostic commands

```bash
# PromQL: current connection ratio
mongodb_connections_used_ratio

# PromQL: current connections by state
mongodb_connections_current

# MongoDB shell: check connection counts
db.serverStatus().connections

# MongoDB shell: check current operations
db.currentOp({"active": true})

# MongoDB shell: check max connections setting
db.adminCommand({getParameter: 1, maxIncomingConnections: 1})
```

## Resolution

- Identify the client application consuming the most connections via `db.currentOp()`
- Check for connection leaks — connections opened but never closed or returned to pool
- Add or tune connection pooling in the application driver (`maxPoolSize`, `minPoolSize`)
- Increase `maxIncomingConnections` as a short-term fix (requires restart)
- Set `net.maxIdleTimeMS` to close idle connections automatically
