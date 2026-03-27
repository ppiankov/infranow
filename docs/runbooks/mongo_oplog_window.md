# MongoDB Oplog Window

## What it means

The oplog window has dropped below 2 hours. If a secondary goes offline for longer than the oplog window, it cannot catch up and requires a full resync — a slow, resource-intensive operation.

## Common causes

- High write volume filling the oplog faster than expected
- Oplog size too small for the current workload
- Large bulk write operations consuming oplog space rapidly
- Index builds or schema migrations generating high write amplification

## Diagnostic commands

```bash
# PromQL: oplog window in hours
mongodb_oplog_window_hours

# PromQL: oplog size and usage
mongodb_oplog_size_bytes

# MongoDB shell: check oplog window and size
db.getReplicationInfo()

# MongoDB shell: check current oplog size
use local
db.oplog.rs.stats().maxSize
```

## Resolution

- Increase oplog size with `replSetResizeOplog` (online, no restart required)
- Investigate and reduce write volume — batch large writes into smaller operations
- Avoid large bulk inserts/updates during peak hours
- Monitor oplog window trend to right-size before it becomes critical
- Plan for oplog size to cover at least 24 hours of writes for maintenance windows
