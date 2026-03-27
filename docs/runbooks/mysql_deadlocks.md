# MySQL Deadlocks

## What it means

More than 5 deadlocks per minute are occurring. InnoDB automatically detects deadlocks and rolls back one transaction, but a high rate indicates a systemic locking problem that degrades throughput and causes application errors.

## Common causes

- Conflicting row-level lock acquisition order across transactions
- Large transactions holding many row locks for extended periods
- Missing indexes causing InnoDB to lock entire index ranges instead of specific rows
- Gap locks in REPEATABLE READ isolation causing unexpected contention
- Hot rows updated by many concurrent transactions

## Diagnostic commands

```bash
# PromQL: deadlock rate
rate(mysql_deadlocks_total[5m])

# PromQL: lock wait time
mysql_innodb_lock_wait_seconds

# SQL: check latest deadlock details
SHOW ENGINE INNODB STATUS\G -- look at LATEST DETECTED DEADLOCK section

# SQL: check current lock waits
SELECT * FROM performance_schema.data_lock_waits;

# SQL: check transactions holding locks
SELECT * FROM information_schema.innodb_trx;
```

## Resolution

- Reorder SQL statements so all transactions acquire locks in the same order
- Add indexes to reduce the scope of row locks (fewer gap locks)
- Break large transactions into smaller ones that hold locks briefly
- Consider lowering isolation level to READ COMMITTED to eliminate gap locks
- Retry deadlocked transactions in application code with backoff
