# Linkerd Control Plane Down

## What it means

A Linkerd control plane deployment has zero available replicas. The service mesh cannot inject proxies, issue certificates, or route traffic for new connections. Existing connections may continue briefly.

## Common causes

- Control plane pod crashed (OOM, config error)
- Node failure hosting control plane pods
- Failed Linkerd upgrade leaving broken state
- Resource limits too low for control plane components
- Certificate expiry preventing component startup

## Diagnostic commands

```bash
# Check control plane pod status
kubectl get pods -n linkerd
kubectl describe deployment <deployment> -n linkerd

# Check control plane logs
kubectl logs -n linkerd -l app=<deployment> --tail=100

# Run Linkerd diagnostics
linkerd check
linkerd check --proxy

# Check certificate validity
linkerd identity

# PromQL: control plane replicas
kube_deployment_status_replicas_available{namespace="linkerd"}
```

## Resolution

- Check pod logs for crash reason and fix the root cause
- If cert-related, rotate certificates: `linkerd upgrade | kubectl apply -f -`
- If resource-related, increase memory/CPU limits on control plane deployments
- If upgrade-related, check `linkerd check` output and follow remediation steps
- As last resort, reinstall: `linkerd install | kubectl apply -f -`
