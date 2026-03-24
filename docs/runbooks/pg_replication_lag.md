# PostgreSQL Replication Lag

## What it means

A streaming replica is more than 30 seconds behind the primary. Reads from this replica return stale data, and failover to it would lose recent writes.

## Common causes

- Replica under heavy read load, slowing WAL replay
- Network bottleneck between primary and replica
- WAL sender process killed or disconnected
- Replica disk I/O saturated
- Large transactions or DDL operations on primary generating burst WAL

## Diagnostic commands

```bash
# PromQL: replication lag per replica
pg_replication_lag_seconds

# PromQL: replication lag in bytes
pg_replication_lag_bytes

# PromQL: connected replicas
pg_replication_connected_replicas

# SQL (on primary): check replication status
SELECT client_addr, state, sent_lsn, replay_lsn, replay_lag FROM pg_stat_replication;
```

## Resolution

- Check replica CPU and I/O — reduce read load if saturated
- Verify network connectivity and bandwidth between primary and replica
- Check `max_wal_senders` and `wal_keep_size` on primary
- Restart replication slot if stuck: `SELECT pg_drop_replication_slot('slot_name');`
- Consider `recovery_min_apply_delay` if intentionally delayed
