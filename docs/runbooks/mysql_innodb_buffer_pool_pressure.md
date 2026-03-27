# MySQL InnoDB Buffer Pool Pressure

## What it means

The InnoDB buffer pool hit ratio has dropped below 95%. This means more than 5% of page reads are going to disk instead of being served from memory, causing significant I/O pressure and query latency increases.

## Common causes

- Buffer pool too small for the working set size
- Sudden workload change bringing new data pages into memory
- Large table scans evicting frequently accessed (hot) pages
- Buffer pool not warmed up after a restart
- Working set growth from new tables or increased data volume

## Diagnostic commands

```bash
# PromQL: buffer pool hit ratio
mysql_innodb_buffer_pool_hit_ratio

# PromQL: buffer pool pages free vs total
mysql_innodb_buffer_pool_pages_free

# SQL: check buffer pool stats
SHOW ENGINE INNODB STATUS\G -- look at BUFFER POOL AND MEMORY section

# SQL: check buffer pool size and usage
SHOW STATUS LIKE 'Innodb_buffer_pool%';

# SQL: check current buffer pool configuration
SHOW VARIABLES LIKE 'innodb_buffer_pool_size';
```

## Resolution

- Increase `innodb_buffer_pool_size` (dynamic in MySQL 8.0, change in chunks of `innodb_buffer_pool_chunk_size`)
- Investigate queries causing full table scans — add indexes to reduce I/O
- Use `innodb_old_blocks_time` to protect hot pages from scan eviction
- Monitor working set growth trend to plan capacity ahead of demand
- Enable buffer pool dump/load on restart (`innodb_buffer_pool_dump_at_shutdown`, `innodb_buffer_pool_load_at_startup`)
