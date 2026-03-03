# CrashLoopBackOff

## What it means

A container is repeatedly crashing and Kubernetes is backing off on restart attempts. The pod is not serving traffic. This is the most visible failure state — the application cannot start or stay running.

## Common causes

- Application startup failure (missing config, bad environment variables)
- Missing or incorrect secrets/configmaps
- Failed database or dependency connection on startup
- Incompatible container image (wrong architecture, missing libs)
- Liveness probe failing too aggressively

## Diagnostic commands

```bash
# Check pod status and restart count
kubectl get pods -n <namespace> | grep CrashLoopBackOff

# View container logs (current and previous)
kubectl logs <pod> -n <namespace>
kubectl logs <pod> -n <namespace> --previous

# Check pod events for scheduling/pull issues
kubectl describe pod <pod> -n <namespace>

# Check if configmaps/secrets exist
kubectl get configmaps -n <namespace>
kubectl get secrets -n <namespace>

# PromQL: pods in CrashLoopBackOff
kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"}
```

## Resolution

- Read the container logs (`--previous` for the crashed instance)
- Verify all required environment variables and secrets exist
- Check that dependent services (databases, APIs) are reachable
- If image-related, verify the image exists and matches the node architecture
- Relax liveness probe if the app needs more startup time (use startupProbe)
