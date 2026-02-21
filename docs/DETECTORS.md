# Detector Documentation

This document describes all detectors included in infranow MVP and how they work.

## Overview

Detectors are pluggable components that query Prometheus metrics and identify infrastructure problems. Each detector:

- Runs at a configurable interval (default: 30-60 seconds)
- Queries Prometheus for specific metrics
- Applies deterministic rules to identify problems
- Returns a list of `Problem` objects with severity, entity, and hints

## Kubernetes Detectors

### OOMKillDetector

**Purpose**: Detects containers that have been killed due to Out Of Memory (OOM) errors.

**Entity Type**: `kubernetes_pod`

**Query**:
```promql
increase(kube_pod_container_status_restarts_total{reason="OOMKilled"}[5m]) > 0
```

**Severity**: `CRITICAL`

**Hint**: "Container memory limit too low or memory leak detected"

**Detection Logic**:
- Checks for any container restarts with reason "OOMKilled" in the last 5 minutes
- Creates a problem for each affected container
- Blast radius: 1 (single container)

**Remediation**:
1. Check container memory limits: `kubectl describe pod <pod-name>`
2. Review application memory usage patterns
3. Increase memory limits if necessary
4. Investigate for memory leaks

---

### CrashLoopBackOffDetector

**Purpose**: Detects pods stuck in CrashLoopBackOff state.

**Entity Type**: `kubernetes_pod`

**Query**:
```promql
kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"} > 0
```

**Severity**: `FATAL`

**Hint**: "Application startup failure or fatal runtime error"

**Detection Logic**:
- Identifies containers waiting with reason "CrashLoopBackOff"
- Indicates repeated application crashes
- Blast radius: 1 (single container)

**Remediation**:
1. Check pod logs: `kubectl logs <pod-name> --previous`
2. Describe pod for events: `kubectl describe pod <pod-name>`
3. Review application startup code and dependencies
4. Check environment variables and configuration

---

### ImagePullBackOffDetector

**Purpose**: Detects pods unable to pull container images.

**Entity Type**: `kubernetes_pod`

**Query**:
```promql
kube_pod_container_status_waiting_reason{reason=~"ImagePullBackOff|ErrImagePull"} > 0
```

**Severity**: `CRITICAL`

**Hint**: "Image not found or registry authentication failure"

**Detection Logic**:
- Checks for containers waiting due to image pull failures
- Matches both "ImagePullBackOff" and "ErrImagePull" reasons
- Blast radius: 1 (single container)

**Remediation**:
1. Verify image name and tag: `kubectl describe pod <pod-name>`
2. Check registry authentication: `kubectl get secrets`
3. Verify network connectivity to registry
4. Ensure image exists in the registry

---

### PodPendingDetector

**Purpose**: Detects pods stuck in Pending state for more than 5 minutes.

**Entity Type**: `kubernetes_pod`

**Query**:
```promql
kube_pod_status_phase{phase="Pending"} * on(namespace, pod) group_left() (time() - kube_pod_created > 300)
```

**Severity**: `CRITICAL`

**Hint**: "Insufficient cluster resources or scheduling constraints"

**Detection Logic**:
- Identifies pods in Pending phase for >5 minutes
- Combines pod phase and creation time metrics
- Blast radius: 1 (single pod)

**Remediation**:
1. Check pod events: `kubectl describe pod <pod-name>`
2. Verify node resources: `kubectl top nodes`
3. Check for taints and tolerations
4. Review node selectors and affinity rules
5. Consider adding nodes if cluster is under-provisioned

---

## Generic Detectors

### HighErrorRateDetector

**Purpose**: Detects services with high HTTP 5xx error rates.

**Entity Type**: `service`

**Query**:
```promql
(rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])) > 0.05
```

**Severity**: `CRITICAL`

**Hint**: "5xx error rate above 5% threshold"

**Detection Logic**:
- Calculates 5xx error rate over 5-minute window
- Threshold: 5% (configurable in detector)
- Blast radius: 5 (assumes service affects multiple entities)

**Remediation**:
1. Check service logs for error details
2. Review recent deployments or changes
3. Verify backend service health
4. Check database connection pools
5. Review application error handling

**Requirements**:
- Requires `http_requests_total` metric with `status` label
- Common in services instrumented with Prometheus client libraries

---

### DiskSpaceDetector

**Purpose**: Detects filesystems with low available space.

**Entity Type**: `filesystem`

**Query**:
```promql
(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) > 0.90
```

**Severity**:
- `WARNING`: Usage > 90%
- `CRITICAL`: Usage > 95%

**Hint**: "Disk usage above 90%"

**Detection Logic**:
- Calculates filesystem usage percentage
- WARNING threshold: 90%
- CRITICAL threshold: 95%
- Blast radius: 3 (could affect multiple services on node)

**Remediation**:
1. Identify large files: `du -h /path | sort -rh | head -20`
2. Clean up old logs or temporary files
3. Rotate or compress logs
4. Consider increasing disk size
5. Review log retention policies

**Requirements**:
- Requires `node_filesystem_avail_bytes` and `node_filesystem_size_bytes` metrics
- Typically provided by node_exporter

---

### HighMemoryPressureDetector

**Purpose**: Detects nodes with high memory pressure.

**Entity Type**: `node`

**Query**:
```promql
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.90
```

**Severity**: `CRITICAL`

**Hint**: "Memory pressure above 90%"

**Detection Logic**:
- Calculates memory usage as percentage
- Threshold: 90% of total memory
- Uses MemAvailable (accounts for caches/buffers)
- Blast radius: 10 (high impact on node)

**Remediation**:
1. Check memory usage by process: `top` or `htop`
2. Identify memory-hungry pods: `kubectl top pods --all-namespaces`
3. Review pod memory limits
4. Consider evicting or scaling down pods
5. Add memory to node or scale cluster

**Requirements**:
- Requires `node_memory_MemAvailable_bytes` and `node_memory_MemTotal_bytes` metrics
- Typically provided by node_exporter

---

## Service Mesh Detectors

### LinkerdControlPlaneDetector

**Purpose**: Detects linkerd control plane deployments with zero available replicas. When the control plane is down, proxy injection fails, mTLS breaks, and traffic routing stops for all meshed services.

**Entity Type**: `service_mesh_control_plane`

**Query**:
```promql
kube_deployment_status_replicas_available{namespace="linkerd"} == 0
```

**Severity**: `FATAL`

**Blast Radius**: 15 (affects all meshed services)

**Hint**: "Check pod status: kubectl get pods -n linkerd"

---

### LinkerdProxyInjectionDetector

**Purpose**: Detects linkerd pods in CrashLoopBackOff. Catches proxy-injector, identity, and destination service failures.

**Entity Type**: `service_mesh_control_plane`

**Query**:
```promql
kube_pod_container_status_waiting_reason{namespace="linkerd",reason="CrashLoopBackOff"} > 0
```

**Severity**: `CRITICAL`

**Blast Radius**: 10

**Hint**: "Proxy injector or identity service failure"

---

### IstioControlPlaneDetector

**Purpose**: Detects istiod with zero available replicas. When istiod is down, xDS config distribution stops and new deployments break.

**Entity Type**: `service_mesh_control_plane`

**Query**:
```promql
kube_deployment_status_replicas_available{namespace="istio-system",deployment="istiod"} == 0
```

**Severity**: `FATAL`

**Blast Radius**: 15

**Hint**: "Check pod status: kubectl get pods -n istio-system"

---

### IstioSidecarInjectionDetector

**Purpose**: Detects istio-system pods in CrashLoopBackOff. Catches istiod, gateway, and pilot failures.

**Entity Type**: `service_mesh_control_plane`

**Query**:
```promql
kube_pod_container_status_waiting_reason{namespace="istio-system",reason="CrashLoopBackOff"} > 0
```

**Severity**: `CRITICAL`

**Blast Radius**: 10

**Hint**: "Sidecar injector or pilot failure"

---

## Service Mesh Certificate Detectors

### LinkerdCertExpiryDetector

**Purpose**: Detects linkerd identity certificates approaching expiry. Certificate expiry is the silent killer of service meshes — mTLS fails across all meshed services without warning.

**Entity Type**: `service_mesh_certificate`

**Query**:
```promql
(identity_cert_expiry_timestamp - time()) < 604800
```

**Severity** (tiered):
- `WARNING`: < 7 days remaining
- `CRITICAL`: < 48 hours remaining
- `FATAL`: < 24 hours remaining or expired

**Blast Radius**: 20 (highest — cert expiry kills the entire mesh)

**Interval**: 60s

**Hint**: "Rotate certs: linkerd check --proxy; Renew: linkerd upgrade | kubectl apply -f -"

---

### IstioCertExpiryDetector

**Purpose**: Detects istio root certificate approaching expiry. When the Citadel root cert expires, all workload certificates become invalid.

**Entity Type**: `service_mesh_certificate`

**Query**:
```promql
(citadel_server_root_cert_expiry_timestamp - time()) < 604800
```

**Severity** (tiered):
- `WARNING`: < 7 days remaining
- `CRITICAL`: < 48 hours remaining
- `FATAL`: < 24 hours remaining or expired

**Blast Radius**: 20

**Interval**: 60s

**Hint**: "Check status: istioctl proxy-status; Rotate: istioctl create-remote-secret"

---

## Trustwatch Detectors

### TrustwatchCertExpiryDetector

**Purpose**: Detects certificates nearing expiry via trustwatch Prometheus metrics. Covers a broader trust surface than mesh-specific detectors: webhooks, API aggregation, external TLS, and mesh issuers.

**Entity Type**: `trustwatch_certificate`

**Query**:
```promql
trustwatch_cert_expires_in_seconds < 604800
```

**Severity** (tiered):
- `WARNING`: < 7 days remaining
- `CRITICAL`: < 48 hours remaining
- `FATAL`: < 24 hours remaining or expired

**Blast Radius**: 10

**Interval**: 60s

**Entity Format**: `trustwatch/{source}/{namespace}/{name}` (e.g. `trustwatch/webhook/kube-system/cert-manager-webhook`)

**Hint**: "Run: trustwatch now"

**Graceful absence**: When trustwatch is not installed, the query returns empty results and no problems are reported.

---

### TrustwatchProbeFailureDetector

**Purpose**: Detects TLS endpoints that trustwatch cannot reach. A probe failure means the endpoint is unreachable or returning invalid TLS.

**Entity Type**: `trustwatch_certificate`

**Query**:
```promql
trustwatch_probe_success == 0
```

**Severity**: `CRITICAL`

**Blast Radius**: 5

**Interval**: 60s

**Entity Format**: `trustwatch/{source}/{namespace}/{name}`

**Hint**: "Run: trustwatch now"

**Graceful absence**: When trustwatch is not installed, the query returns empty results and no problems are reported.

---

### Trustwatch Metric Requirements

| Metric | Source | Detector |
|--------|--------|----------|
| `trustwatch_cert_expires_in_seconds` | trustwatch Helm chart + ServiceMonitor | Cert expiry |
| `trustwatch_probe_success` | trustwatch Helm chart + ServiceMonitor | Probe failure |
| `trustwatch_findings_total` | trustwatch Helm chart + ServiceMonitor | (future use) |

---

### Certificate Monitoring Setup

Service mesh certificate metrics are **often missing from Prometheus**. This is the most common reason cert expiry goes undetected.

**Why cert metrics are missing**:
- Linkerd identity service not in Prometheus scrape targets
- Istiod metrics endpoint not scraped
- cert-manager not exporting metrics
- ServiceMonitor or PodMonitor CRDs missing

**How to verify**:
```bash
# Check if linkerd metrics are being scraped
curl -s http://prometheus:9090/api/v1/targets | \
  jq '.data.activeTargets[] | select(.labels.job | contains("linkerd"))'

# Verify cert metric exists
curl -s http://prometheus:9090/api/v1/query?query=identity_cert_expiry_timestamp
curl -s http://prometheus:9090/api/v1/query?query=citadel_server_root_cert_expiry_timestamp
```

**Required scrape targets**:
- **Linkerd**: `linkerd-identity` (port 9990), `linkerd-proxy-injector` (port 9995)
- **Istio**: `istiod` (port 15014)
- **cert-manager** (optional): `certmanager_certificate_expiration_timestamp_seconds`

---

### Service Mesh Metric Requirements

| Metric | Source | Detector |
|--------|--------|----------|
| `kube_deployment_status_replicas_available` | kube-state-metrics | Control plane health |
| `kube_pod_container_status_waiting_reason` | kube-state-metrics | Component crashes |
| `identity_cert_expiry_timestamp` | linkerd-identity | Linkerd cert expiry |
| `citadel_server_root_cert_expiry_timestamp` | istiod | Istio cert expiry |

---

## Adding Custom Detectors

To add a custom detector:

1. **Create detector file** in `internal/detector/`:

```go
package detector

type MyDetector struct {
    interval time.Duration
}

func NewMyDetector() *MyDetector {
    return &MyDetector{
        interval: 30 * time.Second,
    }
}

func (d *MyDetector) Name() string {
    return "my_custom_detector"
}

func (d *MyDetector) EntityTypes() []string {
    return []string{"my_entity_type"}
}

func (d *MyDetector) Interval() time.Duration {
    return d.interval
}

func (d *MyDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
    query := "my_metric > threshold"
    result, err := provider.QueryInstant(ctx, query, time.Now())
    if err != nil {
        return nil, err
    }

    problems := make([]*models.Problem, 0)
    for _, sample := range result {
        // Extract labels and create problem
        problem := &models.Problem{
            ID:         "unique-id",
            Entity:     "entity-identifier",
            EntityType: "my_entity_type",
            Type:       "problem_type",
            Severity:   models.SeverityCritical,
            Title:      "Problem Title",
            Message:    "Detailed message",
            Hint:       "Actionable hint",
            BlastRadius: 1,
            Labels:     map[string]string{},
            Metrics:    map[string]float64{},
        }
        problems = append(problems, problem)
    }

    return problems, nil
}
```

2. **Add tests** in `internal/detector/my_test.go`

3. **Register detector** in `internal/cli/monitor.go`:

```go
func registerDetectors(registry *detector.Registry) {
    // ... existing
    registry.Register(detector.NewMyDetector())
}
```

4. **Document detector** in this file

## Detector Best Practices

### Query Design

- Use instant queries for current state
- Use range queries for rate calculations
- Keep queries simple and efficient
- Test queries in Prometheus UI first

### Severity Selection

- **FATAL**: Service down, data loss imminent
- **CRITICAL**: Degraded performance, risk of failure
- **WARNING**: Anomaly detected, no immediate impact

### Blast Radius

Estimate affected entities:
- 1: Single container/pod
- 3-5: Multiple pods/services
- 10+: Node-level or cluster-wide impact

### Hints

Provide actionable, specific hints:
- ✅ "Container memory limit too low or memory leak detected"
- ❌ "There's a problem with memory"

### Testing

Test with:
- No data (empty metrics)
- Partial data (some labels missing)
- Edge cases (exactly at threshold)
- Multiple problems simultaneously

## Metric Requirements

infranow detectors expect standard Prometheus metrics:

### Kubernetes Metrics
- `kube_pod_container_status_restarts_total`
- `kube_pod_container_status_waiting_reason`
- `kube_pod_status_phase`
- `kube_pod_created`

**Provided by**: [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics)

### Node Metrics
- `node_filesystem_avail_bytes`
- `node_filesystem_size_bytes`
- `node_memory_MemAvailable_bytes`
- `node_memory_MemTotal_bytes`

**Provided by**: [node_exporter](https://github.com/prometheus/node_exporter)

### Application Metrics
- `http_requests_total` (with `status` label)

**Provided by**: Application instrumentation (Prometheus client libraries)

## Future Detector Ideas

Post-MVP detector candidates:

- **DatabaseReplicationLagDetector**: PostgreSQL/MySQL replication lag
- **KafkaUnderReplicatedPartitionsDetector**: Kafka partition health
- **RedisMemoryPressureDetector**: Redis memory usage
- ~~**CertificateExpirationDetector**: TLS certificate expiration~~ (implemented in v0.1.1 as LinkerdCertExpiry + IstioCertExpiry)
- **PVCFullDetector**: PersistentVolumeClaim usage
- **NodeNotReadyDetector**: Kubernetes node health
- **DeploymentReplicaMismatchDetector**: Desired vs actual replicas
