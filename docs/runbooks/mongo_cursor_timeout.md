# MongoDB Cursor Timeouts

## What it means

More than 10 cursors have timed out. Cursor timeouts indicate that clients opened queries but did not consume results before the server-side timeout (default 10 minutes), wasting server resources.

## Common causes

- Slow queries that take longer to iterate than the cursor timeout
- Missing indexes causing full collection scans on large collections
- Network timeouts between application and MongoDB mid-iteration
- Large result sets fetched without limits or pagination
- Application errors abandoning cursors without closing them

## Diagnostic commands

```bash
# PromQL: timed out cursors
mongodb_cursors_timed_out

# PromQL: open cursors count
mongodb_cursors_open

# MongoDB shell: check cursor metrics
db.serverStatus().metrics.cursor

# MongoDB shell: check currently open cursors
db.currentOp({"op": "getmore"})

# MongoDB shell: list active cursors
db.aggregate([{$currentOp: {allUsers: true, idleCursors: true}}])
```

## Resolution

- Add indexes for queries causing slow full collection scans
- Paginate large result sets using `limit()` and `skip()` or range-based pagination
- Use `noCursorTimeout` only when explicitly needed and ensure manual cursor cleanup
- Check application code for abandoned cursors — ensure `close()` is called in finally blocks
- Increase `cursorTimeoutMillis` only as a last resort after fixing root causes
