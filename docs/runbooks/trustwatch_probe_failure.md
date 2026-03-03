# Trustwatch Probe Failure

## What it means

A trustwatch endpoint probe is returning failure (success = 0). The monitored endpoint is unreachable or returning an invalid TLS certificate. This may indicate a service outage or certificate misconfiguration.

## Common causes

- Endpoint is down or unreachable
- TLS certificate is invalid, expired, or mismatched
- DNS resolution failure for the endpoint
- Network connectivity issue (firewall, security group)
- Certificate chain incomplete (missing intermediate CA)

## Diagnostic commands

```bash
# Check trustwatch probe status
trustwatch now

# Test TLS connection manually
openssl s_client -connect <host>:443 -servername <host> </dev/null 2>&1 | head -20

# Check DNS resolution
dig <host>
nslookup <host>

# Check connectivity
curl -vI https://<host> 2>&1 | head -30

# PromQL: probe status
trustwatch_probe_success
```

## Resolution

- Run `trustwatch now` to identify which endpoint is failing
- If endpoint is down: check the service hosting the endpoint
- If cert-related: check certificate validity with openssl
- If DNS-related: verify DNS records and propagation
- If network-related: check firewalls, security groups, and routing
