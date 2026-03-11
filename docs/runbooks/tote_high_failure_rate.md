# Tote High Failure Rate

## What it means

Most image pull failures detected by tote use tag-based references (e.g., `nginx:latest`) instead of digest references (e.g., `nginx@sha256:abc...`). Tote cannot salvage tag-based images because the tag may resolve to a different digest on different nodes. These failures have no automated recovery path.

## Common causes

- Deployments using mutable tags (`:latest`, `:stable`, `:v1`)
- Helm charts defaulting to tag-based image references
- CI/CD pipelines not pinning image digests after build
- Third-party charts or operators using unpinned images

## Diagnostic commands

```bash
# Find pods with image pull failures
kubectl get pods --all-namespaces --field-selector=status.phase!=Running | grep -E 'ImagePull|ErrImage'

# Check which images use tags vs digests
kubectl get pods --all-namespaces -o jsonpath='{range .items[*]}{.spec.containers[*].image}{"\n"}{end}' | sort -u

# PromQL: tag-based failures vs salvageable
increase(tote_not_actionable_total[10m])
increase(tote_salvageable_images_total[10m])
```

## Resolution

- Switch container images from tags to digests (`image@sha256:...`)
- Use tools like `crane digest` or `skopeo inspect` to resolve tags to digests
- Configure CI/CD to pin digests after image build and push
- For third-party charts: override image references with digest-pinned values
