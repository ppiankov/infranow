# Tote Salvage Failure

## What it means

Tote detected image pull failures and found the image digest on another cluster node, but the node-to-node transfer failed. Pods remain stuck in ImagePullBackOff because the image cannot be recovered from the cluster.

## Common causes

- Tote agent pod not running on the source or target node
- Agent-to-agent gRPC connectivity blocked by network policy
- Source node disk pressure causing containerd export failure
- Target node out of disk space for image import
- Containerd socket not mounted into the agent DaemonSet

## Diagnostic commands

```bash
# Check tote controller and agent pods
kubectl get pods -n tote-system

# Check agent logs for transfer errors
kubectl logs -n tote-system -l app.kubernetes.io/component=agent --tail=100

# Check controller logs for salvage orchestration
kubectl logs -n tote-system -l app.kubernetes.io/component=controller --tail=100

# PromQL: recent salvage failures
increase(tote_salvage_failures_total[5m])

# PromQL: salvage success rate
tote_salvage_successes_total / (tote_salvage_attempts_total + 1)
```

## Resolution

- Verify tote agent DaemonSet is running on all nodes: `kubectl get ds -n tote-system`
- Check network policies allow agent-to-agent traffic on the gRPC port
- Check node disk usage on both source and target nodes
- Restart the agent pod on the affected node if containerd socket is stale
