# HighMemoryPressure

## What it means

A node's memory usage exceeds 90%. The Linux kernel may start OOM-killing pods. System stability is at risk.

## Common causes

- Too many pods scheduled on the node (overcommitted)
- Memory leak in one or more applications
- Memory requests set too low (pods use more than requested)
- System processes consuming unexpected memory
- Large in-memory workloads (caches, databases)

## Diagnostic commands

```bash
# Check node memory
kubectl top nodes
kubectl describe node <node> | grep -A10 "Allocated resources"

# Check which pods use the most memory
kubectl top pods --all-namespaces --sort-by=memory | head -20

# Check node-level memory breakdown
ssh <node> free -h
ssh <node> ps aux --sort=-%mem | head -20

# PromQL: memory usage by node
1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)

# PromQL: top memory consumers
container_memory_usage_bytes{node="<node>"} / 1024 / 1024
```

## Resolution

- Identify and fix memory-leaking pods (increase limits or fix the leak)
- Cordon the node and drain workloads to other nodes
- Add more nodes to distribute the load
- Set proper memory requests so the scheduler avoids overcommitting
- Consider using memory-based HPA to auto-scale before pressure builds
