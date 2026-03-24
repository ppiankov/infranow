# ClickHouse Stuck Mutations

## What it means

One or more ALTER TABLE mutations (UPDATE, DELETE, MATERIALIZE) are stuck and not making progress. Stuck mutations hold resources and can block subsequent mutations on the same table.

## Common causes

- Mutation blocked by a running merge on the same partition
- Out of disk space for mutation temporary files
- ClickHouse server restart interrupted the mutation
- Bug in mutation logic for specific data types
- Table locked by another DDL operation

## Diagnostic commands

```bash
# PromQL: stuck mutation count
clickhouse_mutations_stuck

# PromQL: parts remaining per mutation
clickhouse_mutation_parts_remaining

# SQL: check mutation status
SELECT database, table, mutation_id, command, create_time, parts_to_do, is_done, latest_fail_reason
FROM system.mutations WHERE is_done = 0 ORDER BY create_time;

# SQL: check if merges are blocking
SELECT database, table, elapsed, progress FROM system.merges;
```

## Resolution

- Wait for blocking merges to complete, then check if mutation resumes
- Kill stuck mutations: `KILL MUTATION WHERE mutation_id = 'id'`
- Check `latest_fail_reason` in `system.mutations` for the root cause
- Ensure sufficient disk space for mutation temporary data
- Re-issue the mutation after killing the stuck one
