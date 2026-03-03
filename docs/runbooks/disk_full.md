# DiskSpace

## What it means

A filesystem on a node has usage above 90% (WARNING) or 95% (CRITICAL). At 100%, writes fail, causing application crashes, log loss, and potential data corruption.

## Common causes

- Log files growing without rotation
- Container images accumulating (unused layers)
- Application writing temporary files without cleanup
- PersistentVolume filling up (database WAL, backups)
- Prometheus TSDB or etcd data growth

## Diagnostic commands

```bash
# Check disk usage on the node
ssh <node> df -h
ssh <node> du -sh /var/log/* | sort -rh | head -20

# Check from Kubernetes
kubectl debug node/<node> -it --image=busybox -- df -h

# Find large files
kubectl debug node/<node> -it --image=busybox -- find / -size +100M -type f 2>/dev/null

# PromQL: disk usage by node and mountpoint
1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)

# PromQL: disk usage prediction (will it fill in 4 hours?)
predict_linear(node_filesystem_avail_bytes[1h], 4*3600) < 0
```

## Resolution

- Clean up old log files and enable log rotation
- Run `docker system prune` or `crictl rmi --prune` on nodes
- Delete unused container images and dangling volumes
- Expand the filesystem or PersistentVolume
- Move data to larger storage or add retention policies
