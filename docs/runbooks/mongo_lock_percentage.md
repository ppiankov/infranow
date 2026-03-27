# MongoDB Global Lock Percentage

## What it means

The global lock ratio has exceeded 50%. High lock contention degrades throughput and increases latency for all operations, as reads and writes queue behind locked resources.

## Common causes

- Collection-level locks from long-running write operations
- Unindexed queries causing full collection scans that hold locks
- Schema migrations or large updates locking entire collections
- WiredTiger ticket exhaustion under heavy concurrent load
- Map-reduce or aggregation operations holding locks for extended periods

## Diagnostic commands

```bash
# PromQL: global lock ratio
mongodb_global_lock_ratio

# PromQL: lock queue depth
mongodb_global_lock_queue_total

# MongoDB shell: check global lock stats
db.serverStatus().globalLock

# MongoDB shell: check current operations holding locks
db.currentOp({"waitingForLock": true})

# MongoDB shell: check WiredTiger ticket availability
db.serverStatus().wiredTiger.concurrentTransactions
```

## Resolution

- Identify and add missing indexes for queries causing full collection scans
- Optimize long-running write operations — break large updates into smaller batches
- Ensure WiredTiger storage engine is in use (document-level locking vs collection-level)
- Check WiredTiger read/write ticket settings (`wiredTigerConcurrentReadTransactions`)
- Schedule heavy operations (migrations, bulk writes) during low-traffic periods
