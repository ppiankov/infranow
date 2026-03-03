# Linkerd Certificate Expiry

## What it means

The Linkerd identity certificate is approaching expiry. Severity depends on remaining time: WARNING (< 7 days), CRITICAL (< 48 hours), FATAL (< 24 hours). When the certificate expires, mTLS between proxies fails and the mesh stops working.

## Common causes

- Certificate rotation not configured or failing
- Linkerd installed with short-lived trust anchor
- cert-manager integration broken
- Manual certificate management without renewal plan

## Diagnostic commands

```bash
# Check certificate expiry
linkerd check --proxy
linkerd identity

# View certificate details
kubectl get secret linkerd-identity-issuer -n linkerd -o jsonpath='{.data.crt\.pem}' | base64 -d | openssl x509 -text -noout

# Check trust anchor expiry
kubectl get configmap linkerd-identity-trust-roots -n linkerd -o jsonpath='{.data.ca-bundle\.crt}' | openssl x509 -text -noout | grep "Not After"

# PromQL: time until expiry
identity_cert_expiry_timestamp - time()
```

## Resolution

- Rotate the identity issuer certificate: `linkerd upgrade | kubectl apply -f -`
- If trust anchor is expiring, rotate it following the Linkerd trust anchor rotation docs
- Set up cert-manager for automatic certificate rotation
- Monitor with alerting well before the 7-day WARNING threshold
