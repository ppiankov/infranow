# Istio Control Plane Down

## What it means

An Istio control plane deployment (istiod) has zero available replicas. The mesh cannot issue certificates, configure proxies, or enforce policies. Existing connections continue until certificates expire.

## Common causes

- istiod pod crashed (OOM, startup failure)
- Root certificate expired
- Failed Istio upgrade
- Resource limits too low
- Webhook configuration error preventing pod creation

## Diagnostic commands

```bash
# Check istiod status
kubectl get pods -n istio-system
kubectl describe deployment istiod -n istio-system

# Check istiod logs
kubectl logs -n istio-system -l app=istiod --tail=100

# Run Istio diagnostics
istioctl analyze
istioctl proxy-status

# Check certificate status
istioctl proxy-config secret <pod>.<namespace>

# PromQL: control plane replicas
kube_deployment_status_replicas_available{namespace="istio-system"}
```

## Resolution

- Check istiod logs for the crash reason
- If cert-related: `istioctl create-remote-secret` or rotate root cert
- If resource-related: increase istiod memory/CPU limits
- If upgrade-related: check version compatibility and run `istioctl upgrade`
- Verify mutating webhook is correctly configured
