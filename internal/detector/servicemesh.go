package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// LinkerdControlPlaneDetector detects linkerd control plane components with zero available replicas
type LinkerdControlPlaneDetector struct {
	interval time.Duration
}

func NewLinkerdControlPlaneDetector() *LinkerdControlPlaneDetector {
	return &LinkerdControlPlaneDetector{
		interval: 30 * time.Second,
	}
}

func (d *LinkerdControlPlaneDetector) Name() string {
	return "servicemesh_linkerd_controlplane"
}

func (d *LinkerdControlPlaneDetector) EntityTypes() []string {
	return []string{"service_mesh_control_plane"}
}

func (d *LinkerdControlPlaneDetector) Interval() time.Duration {
	return d.interval
}

func (d *LinkerdControlPlaneDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_deployment_status_replicas_available{namespace="linkerd"} == 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("linkerd control plane query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		deployment := string(sample.Metric["deployment"])

		entity := fmt.Sprintf("%s/%s", namespace, deployment)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/linkerd_cp_down", entity),
			Entity:     entity,
			EntityType: "service_mesh_control_plane",
			Type:       "linkerd_control_plane_down",
			Severity:   models.SeverityFatal,
			Title:      "Linkerd Control Plane Down",
			Message:    fmt.Sprintf("Linkerd deployment %s has zero available replicas", deployment),
			Labels: map[string]string{
				"mesh":       "linkerd",
				"namespace":  namespace,
				"deployment": deployment,
			},
			Metrics: map[string]float64{
				"available_replicas": float64(sample.Value),
			},
			Hint:        "Check pod status: kubectl get pods -n linkerd; Check logs: kubectl logs -n linkerd -l app=" + deployment,
			BlastRadius: 15,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// LinkerdProxyInjectionDetector detects linkerd pods in CrashLoopBackOff
type LinkerdProxyInjectionDetector struct {
	interval time.Duration
}

func NewLinkerdProxyInjectionDetector() *LinkerdProxyInjectionDetector {
	return &LinkerdProxyInjectionDetector{
		interval: 30 * time.Second,
	}
}

func (d *LinkerdProxyInjectionDetector) Name() string {
	return "servicemesh_linkerd_injection"
}

func (d *LinkerdProxyInjectionDetector) EntityTypes() []string {
	return []string{"service_mesh_control_plane"}
}

func (d *LinkerdProxyInjectionDetector) Interval() time.Duration {
	return d.interval
}

func (d *LinkerdProxyInjectionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_pod_container_status_waiting_reason{namespace="linkerd",reason="CrashLoopBackOff"} > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("linkerd proxy injection query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])
		container := string(sample.Metric["container"])

		entity := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/linkerd_crash", entity),
			Entity:     entity,
			EntityType: "service_mesh_control_plane",
			Type:       "linkerd_component_crash",
			Severity:   models.SeverityCritical,
			Title:      "Linkerd Component CrashLoopBackOff",
			Message:    fmt.Sprintf("Linkerd pod %s/%s is in CrashLoopBackOff", namespace, pod),
			Labels: map[string]string{
				"mesh":      "linkerd",
				"namespace": namespace,
				"pod":       pod,
				"container": container,
			},
			Metrics: map[string]float64{
				"waiting": float64(sample.Value),
			},
			Hint:        "Proxy injector or identity service failure; Check logs: kubectl logs -n linkerd " + pod,
			BlastRadius: 10,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// IstioControlPlaneDetector detects istiod with zero available replicas
type IstioControlPlaneDetector struct {
	interval time.Duration
}

func NewIstioControlPlaneDetector() *IstioControlPlaneDetector {
	return &IstioControlPlaneDetector{
		interval: 30 * time.Second,
	}
}

func (d *IstioControlPlaneDetector) Name() string {
	return "servicemesh_istio_controlplane"
}

func (d *IstioControlPlaneDetector) EntityTypes() []string {
	return []string{"service_mesh_control_plane"}
}

func (d *IstioControlPlaneDetector) Interval() time.Duration {
	return d.interval
}

func (d *IstioControlPlaneDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_deployment_status_replicas_available{namespace="istio-system",deployment="istiod"} == 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("istio control plane query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		deployment := string(sample.Metric["deployment"])

		entity := fmt.Sprintf("%s/%s", namespace, deployment)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/istio_cp_down", entity),
			Entity:     entity,
			EntityType: "service_mesh_control_plane",
			Type:       "istio_control_plane_down",
			Severity:   models.SeverityFatal,
			Title:      "Istio Control Plane Down",
			Message:    fmt.Sprintf("Istiod deployment %s has zero available replicas", deployment),
			Labels: map[string]string{
				"mesh":       "istio",
				"namespace":  namespace,
				"deployment": deployment,
			},
			Metrics: map[string]float64{
				"available_replicas": float64(sample.Value),
			},
			Hint:        "Check pod status: kubectl get pods -n istio-system; Check logs: kubectl logs -n istio-system -l app=istiod",
			BlastRadius: 15,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// IstioSidecarInjectionDetector detects istio-system pods in CrashLoopBackOff
type IstioSidecarInjectionDetector struct {
	interval time.Duration
}

func NewIstioSidecarInjectionDetector() *IstioSidecarInjectionDetector {
	return &IstioSidecarInjectionDetector{
		interval: 30 * time.Second,
	}
}

func (d *IstioSidecarInjectionDetector) Name() string {
	return "servicemesh_istio_injection"
}

func (d *IstioSidecarInjectionDetector) EntityTypes() []string {
	return []string{"service_mesh_control_plane"}
}

func (d *IstioSidecarInjectionDetector) Interval() time.Duration {
	return d.interval
}

func (d *IstioSidecarInjectionDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_pod_container_status_waiting_reason{namespace="istio-system",reason="CrashLoopBackOff"} > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("istio sidecar injection query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])
		container := string(sample.Metric["container"])

		entity := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/istio_crash", entity),
			Entity:     entity,
			EntityType: "service_mesh_control_plane",
			Type:       "istio_component_crash",
			Severity:   models.SeverityCritical,
			Title:      "Istio Component CrashLoopBackOff",
			Message:    fmt.Sprintf("Istio pod %s/%s is in CrashLoopBackOff", namespace, pod),
			Labels: map[string]string{
				"mesh":      "istio",
				"namespace": namespace,
				"pod":       pod,
				"container": container,
			},
			Metrics: map[string]float64{
				"waiting": float64(sample.Value),
			},
			Hint:        "Sidecar injector or pilot failure; Check logs: kubectl logs -n istio-system " + pod,
			BlastRadius: 10,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}
