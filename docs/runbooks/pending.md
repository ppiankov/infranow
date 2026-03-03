# PodPending

## What it means

A pod has been in Pending state for more than 5 minutes. Kubernetes cannot schedule it onto a node. The workload is not running.

## Common causes

- Insufficient CPU or memory resources on all nodes
- Node selector or affinity rules cannot be satisfied
- PersistentVolumeClaim cannot be bound (no matching PV)
- Taints on all nodes with no matching tolerations
- Resource quotas exceeded in the namespace

## Diagnostic commands

```bash
# Check why the pod is pending
kubectl describe pod <pod> -n <namespace> | grep -A10 "Events"

# Check node resources
kubectl top nodes
kubectl describe nodes | grep -A5 "Allocated resources"

# Check PVC status if volumes are involved
kubectl get pvc -n <namespace>

# Check resource quotas
kubectl get resourcequotas -n <namespace>

# Check node taints
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints

# PromQL: pending pods
kube_pod_status_phase{phase="Pending"}
```

## Resolution

- Scale up the cluster or add nodes with the required resources
- Adjust resource requests to fit available capacity
- Check and fix PVC bindings (provision storage, fix storageClass)
- Add tolerations for node taints or remove unnecessary taints
- Increase namespace resource quotas if applicable
