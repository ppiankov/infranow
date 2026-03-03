# Istio Certificate Expiry

## What it means

The Istio root certificate (Citadel) is approaching expiry. Severity depends on remaining time: WARNING (< 7 days), CRITICAL (< 48 hours), FATAL (< 24 hours). When the certificate expires, mTLS between sidecars fails and service-to-service communication breaks.

## Common causes

- Default Istio root certificate has a 10-year lifetime but was never rotated
- Custom CA certificate approaching end of life
- cert-manager integration not configured
- Failed certificate rotation attempt

## Diagnostic commands

```bash
# Check certificate status
istioctl proxy-status
istioctl proxy-config secret <pod>.<namespace>

# View root cert expiry
kubectl get secret istio-ca-secret -n istio-system -o jsonpath='{.data.ca-cert\.pem}' | base64 -d | openssl x509 -text -noout | grep "Not After"

# Check istiod certificate logs
kubectl logs -n istio-system -l app=istiod | grep -i cert

# PromQL: time until expiry
citadel_server_root_cert_expiry_timestamp - time()
```

## Resolution

- Rotate the root certificate following Istio's root cert rotation guide
- If using cert-manager, verify the Certificate resource and issuer
- For custom CA: generate a new root cert and update the istio-ca-secret
- After rotation, restart istiod and verify with `istioctl proxy-status`
