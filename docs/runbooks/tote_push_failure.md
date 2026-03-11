# Tote Push Failure

## What it means

Tote successfully salvaged an image (node-to-node transfer worked) but failed to push a backup copy to the configured registry. This is non-critical — the pod is already running — but means the backup registry is missing images that may be needed if nodes are drained or replaced.

## Common causes

- Backup registry unreachable (DNS, network policy, firewall)
- Registry credentials expired or misconfigured
- Registry storage full (disk quota or object storage limit)
- Registry rate limiting applied to push operations
- TLS certificate mismatch on registry endpoint

## Diagnostic commands

```bash
# Check controller logs for push errors
kubectl logs -n tote-system -l app.kubernetes.io/component=controller --tail=100 | grep -i push

# Check registry secret
kubectl get secret -n tote-system tote-registry-credentials -o yaml

# PromQL: recent push failures
increase(tote_push_failures_total[10m])

# PromQL: push success rate
tote_push_successes_total / (tote_push_attempts_total + 1)
```

## Resolution

- Verify backup registry is reachable from the controller pod
- Rotate or update registry credentials if expired
- Check registry storage quotas and free space
- Review registry rate limits and adjust if needed
