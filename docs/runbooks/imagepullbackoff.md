# ImagePullBackOff

## What it means

Kubernetes cannot pull the container image from the registry. The pod will not start until the image becomes available. This blocks deployments and rollbacks.

## Common causes

- Image tag does not exist (typo, deleted, not yet pushed)
- Private registry requires authentication (missing imagePullSecret)
- Registry is unreachable (network issue, DNS failure, rate limit)
- Image was deleted or garbage collected from the registry
- Wrong registry URL in the image reference

## Diagnostic commands

```bash
# Check pod events for pull error details
kubectl describe pod <pod> -n <namespace> | grep -A10 "Events"

# Verify image exists (from your machine)
docker manifest inspect <image>:<tag>

# Check imagePullSecrets on the pod
kubectl get pod <pod> -n <namespace> -o jsonpath='{.spec.imagePullSecrets}'

# Check if the secret exists and is valid
kubectl get secret <secret-name> -n <namespace> -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d

# PromQL: pods with image pull failures
kube_pod_container_status_waiting_reason{reason=~"ImagePullBackOff|ErrImagePull"}
```

## Resolution

- Verify the image tag exists in the registry
- Create or update imagePullSecret for private registries
- Check network connectivity from cluster nodes to the registry
- If rate-limited (Docker Hub), use a registry mirror or authenticate
- Pin to digest (`image@sha256:...`) instead of mutable tags
