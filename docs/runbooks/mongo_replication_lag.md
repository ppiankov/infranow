# MongoDB Replication Lag

## What it means

A secondary member is lagging 30 seconds or more behind the primary. Reads from secondaries return stale data, and if the primary fails, the secondary cannot be elected without data loss risk.

## Common causes

- High write volume on the primary exceeding secondary apply rate
- Slow disk I/O on the secondary (degraded storage, noisy neighbor)
- Network latency or bandwidth constraints between primary and secondary
- Large oplog entries from bulk writes or large document updates
- Secondary under heavy read load competing for disk I/O

## Diagnostic commands

```bash
# PromQL: replication lag in seconds
mongodb_replication_lag_seconds

# PromQL: oplog apply rate
mongodb_oplog_apply_ops_per_second

# MongoDB shell: check replication lag
rs.printSecondaryReplicationInfo()

# MongoDB shell: check replica set status
rs.status()

# MongoDB shell: check oplog size and window
db.getReplicationInfo()
```

## Resolution

- Check secondary disk I/O and CPU — ensure it is not resource-constrained
- Check network latency between primary and secondary members
- Increase oplog size if secondary falls off the oplog window during lag spikes
- Reduce read load on lagging secondary by routing reads elsewhere
- Investigate large write operations that may be generating oversized oplog entries
