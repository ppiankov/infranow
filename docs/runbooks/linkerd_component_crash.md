# Linkerd Component Crash

## What it means

A Linkerd component pod in the linkerd namespace is in CrashLoopBackOff. The mesh control plane is partially broken. Proxy injection, certificate rotation, or traffic routing may be affected.

## Common causes

- Control plane upgrade left incompatible components
- Certificate expiry preventing mTLS handshake
- Insufficient resources for proxy injector or identity service
- Configuration error in Linkerd ConfigMap
- Node-level issues affecting linkerd namespace pods

## Diagnostic commands

```bash
# Check which component is crashing
kubectl get pods -n linkerd | grep CrashLoopBackOff

# View crash logs
kubectl logs <pod> -n linkerd --previous
kubectl logs <pod> -n linkerd

# Check events
kubectl describe pod <pod> -n linkerd

# Run Linkerd diagnostics
linkerd check

# PromQL: crashing linkerd pods
kube_pod_container_status_waiting_reason{namespace="linkerd", reason="CrashLoopBackOff"}
```

## Resolution

- Read the crash logs to identify the specific failure
- If identity service: check certificate chain with `linkerd check --proxy`
- If proxy injector: check webhook configuration
- If upgrade-related: ensure all components are at the same version
- Restart the failing component: `kubectl rollout restart deployment/<component> -n linkerd`
