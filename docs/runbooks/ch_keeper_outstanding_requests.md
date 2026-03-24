# ClickHouse Keeper Outstanding Requests

## What it means

ZooKeeper/ClickHouse Keeper has more than 100 outstanding (pending) requests. This means the Keeper is overloaded and cannot process requests fast enough. The backlog will cause replication stalls, DDL timeouts, and potential cluster-wide degradation.

## Common causes

- Burst of replication activity (many tables syncing simultaneously)
- Keeper leader election in progress (temporary spike)
- Keeper node disk I/O saturated during snapshot
- Too many ClickHouse nodes competing for Keeper resources
- Network partition causing request retries

## Diagnostic commands

```bash
# PromQL: outstanding requests
clickhouse_keeper_outstanding_requests

# PromQL: is this node the leader?
clickhouse_keeper_is_leader

# PromQL: Keeper latency (correlated signal)
clickhouse_keeper_latency_seconds

# Keeper CLI: detailed stats
echo mntr | nc keeper-host 2181

# SQL: check replica queue sizes (downstream effect)
SELECT database, table, queue_size FROM system.replicas WHERE queue_size > 0;
```

## Resolution

- If during leader election, wait for it to complete (1-2 minutes)
- Scale Keeper resources (CPU, memory, fast disk)
- Reduce number of replicated tables if excessive
- Add more Keeper nodes to distribute load (odd number: 3 or 5)
- Check for ClickHouse nodes in a reconnect loop flooding Keeper
