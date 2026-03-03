# Istio Component Crash

## What it means

An Istio component pod in the istio-system namespace is in CrashLoopBackOff. Sidecar injection, policy enforcement, or telemetry collection may be affected.

## Common causes

- Incompatible Istio version after upgrade
- Certificate chain broken or expired
- Insufficient resources for istio-system pods
- Webhook misconfiguration causing injection failures
- Conflicting Istio configurations (VirtualService, DestinationRule)

## Diagnostic commands

```bash
# Check which component is crashing
kubectl get pods -n istio-system | grep CrashLoopBackOff

# View crash logs
kubectl logs <pod> -n istio-system --previous
kubectl logs <pod> -n istio-system

# Check events
kubectl describe pod <pod> -n istio-system

# Run Istio diagnostics
istioctl analyze
istioctl proxy-status

# PromQL: crashing istio pods
kube_pod_container_status_waiting_reason{namespace="istio-system", reason="CrashLoopBackOff"}
```

## Resolution

- Read the crash logs to identify the specific failure
- If version mismatch: ensure all components run the same Istio version
- If cert-related: check certificate chain with `istioctl proxy-config secret`
- If webhook-related: check `kubectl get mutatingwebhookconfigurations`
- Restart the failing component: `kubectl rollout restart deployment/<component> -n istio-system`
