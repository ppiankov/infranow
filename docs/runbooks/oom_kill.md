# OOMKill

## What it means

A container was terminated by the Linux kernel OOM killer because it exceeded its memory limit. The pod restarts but will likely be killed again if the root cause is not addressed.

## Common causes

- Container memory limit set too low for the workload
- Memory leak in the application (heap growth over time)
- Sudden traffic spike causing increased memory allocation
- Large in-memory caches or buffers without eviction
- JVM-based apps with heap size exceeding container limit

## Diagnostic commands

```bash
# Check pod restarts and reason
kubectl get pods -n <namespace> -o wide
kubectl describe pod <pod> -n <namespace> | grep -A5 "Last State"

# Check current memory usage vs limits
kubectl top pods -n <namespace>
kubectl get pod <pod> -n <namespace> -o jsonpath='{.spec.containers[*].resources}'

# Check OOM events in node dmesg
kubectl get events -n <namespace> --field-selector reason=OOMKilling

# PromQL: recent OOM restarts
kube_pod_container_status_restarts_total{reason="OOMKilled", namespace="<namespace>"}
```

## Resolution

- Increase container memory limits if the workload legitimately needs more memory
- Profile the application for memory leaks (heap dumps, pprof)
- Add memory-aware health checks to detect leaks before OOM
- For JVM apps, set `-Xmx` to 75% of the container memory limit
- Consider horizontal scaling instead of larger memory limits
