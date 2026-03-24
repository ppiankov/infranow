# ClickHouse DDL Queue Stuck

## What it means

Distributed DDL operations are stuck in the queue and not completing. This means ALTER TABLE, CREATE, or DROP commands issued via `ON CLUSTER` are not being applied to all nodes, leaving the cluster in an inconsistent schema state.

## Common causes

- ZooKeeper/Keeper unavailable or unreachable from some nodes
- A cluster node is down and cannot execute the DDL
- Previous DDL failed on a node, blocking the queue
- ZooKeeper session expired during DDL execution
- Network partition between cluster nodes

## Diagnostic commands

```bash
# PromQL: stuck DDL entries
clickhouse_ddl_queue_stuck

# PromQL: DDL queue size
clickhouse_ddl_queue_size

# PromQL: oldest DDL entry age
clickhouse_ddl_oldest_entry_seconds

# SQL: check DDL queue
SELECT entry, host_name, host_address, query, initiator_host, state
FROM system.distributed_ddl_queue ORDER BY entry DESC LIMIT 20;

# SQL: check ZooKeeper connectivity
SELECT * FROM system.zookeeper WHERE path = '/clickhouse/task_queue/ddl';
```

## Resolution

- Check all cluster nodes are online and healthy
- Verify ZooKeeper/Keeper connectivity from all nodes
- Manually execute the stuck DDL on the failing node
- Clean up stuck entries: remove the ZooKeeper node for the stuck task
- Check `system.distributed_ddl_queue` for the specific failure reason
