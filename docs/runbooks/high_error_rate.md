# HighErrorRate

## What it means

More than 5% of HTTP requests are returning 5xx status codes over a 5-minute window. Users are experiencing errors. The service is degraded.

## Common causes

- Backend service crash or unavailability
- Database connection pool exhaustion
- Upstream dependency failure (timeout, connection refused)
- Deployment of buggy code (bad release)
- Resource exhaustion (CPU throttling, memory pressure)

## Diagnostic commands

```bash
# Check pod health
kubectl get pods -n <namespace> -l app=<service>
kubectl logs -n <namespace> -l app=<service> --tail=100

# PromQL: error rate by service
rate(http_requests_total{status=~"5..", job="<service>"}[5m])
  / rate(http_requests_total{job="<service>"}[5m])

# PromQL: error rate over time
rate(http_requests_total{status=~"5.."}[5m])

# Check recent deployments
kubectl rollout history deployment/<service> -n <namespace>

# Check resource usage
kubectl top pods -n <namespace> -l app=<service>
```

## Resolution

- Check application logs for the root cause of 5xx errors
- If caused by a bad deployment, rollback: `kubectl rollout undo deployment/<service>`
- Verify upstream dependencies are healthy
- Check for resource exhaustion (CPU throttling, OOM)
- Scale up if the service is overloaded
