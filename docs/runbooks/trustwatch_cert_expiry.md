# Trustwatch Certificate Expiry

## What it means

A certificate monitored by trustwatch is approaching expiry. Severity depends on remaining time: WARNING (< 7 days), CRITICAL (< 48 hours), FATAL (< 24 hours). When the certificate expires, TLS connections to the affected endpoint will fail.

## Common causes

- Certificate renewal automation not configured
- ACME/Let's Encrypt renewal job failing silently
- cert-manager Certificate resource misconfigured
- DNS validation failing for certificate renewal
- Manual certificate management without tracking

## Diagnostic commands

```bash
# Check trustwatch status
trustwatch now

# Check specific certificate
trustwatch check <endpoint>

# If using cert-manager, check Certificate status
kubectl get certificates --all-namespaces
kubectl describe certificate <name> -n <namespace>

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager --tail=100

# PromQL: time until expiry
trustwatch_cert_expires_in_seconds
```

## Resolution

- Run `trustwatch now` to see current certificate status
- If using Let's Encrypt: check ACME challenge completion and DNS
- If using cert-manager: verify Certificate and Issuer resources
- For manual certs: renew and deploy the new certificate
- Set up automated renewal if not already configured
