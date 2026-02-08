package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// OOMKillDetector detects containers that have been OOM killed
type OOMKillDetector struct {
	interval time.Duration
}

func NewOOMKillDetector() *OOMKillDetector {
	return &OOMKillDetector{
		interval: 30 * time.Second,
	}
}

func (d *OOMKillDetector) Name() string {
	return "kubernetes_oom_kills"
}

func (d *OOMKillDetector) EntityTypes() []string {
	return []string{"kubernetes_pod"}
}

func (d *OOMKillDetector) Interval() time.Duration {
	return d.interval
}

func (d *OOMKillDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `increase(kube_pod_container_status_restarts_total{reason="OOMKilled"}[5m]) > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("oom kill query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])
		container := string(sample.Metric["container"])

		entity := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/oomkill", entity),
			Entity:     entity,
			EntityType: "kubernetes_pod",
			Type:       "oom_kill",
			Severity:   models.SeverityCritical,
			Title:      "Container OOM Killed",
			Message:    fmt.Sprintf("Container %s in pod %s/%s was OOM killed", container, namespace, pod),
			Labels: map[string]string{
				"namespace": namespace,
				"pod":       pod,
				"container": container,
			},
			Metrics: map[string]float64{
				"restart_count": float64(sample.Value),
			},
			Hint:        "Container memory limit too low or memory leak detected",
			BlastRadius: 1,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// CrashLoopBackOffDetector detects pods in CrashLoopBackOff state
type CrashLoopBackOffDetector struct {
	interval time.Duration
}

func NewCrashLoopBackOffDetector() *CrashLoopBackOffDetector {
	return &CrashLoopBackOffDetector{
		interval: 30 * time.Second,
	}
}

func (d *CrashLoopBackOffDetector) Name() string {
	return "kubernetes_crashloop"
}

func (d *CrashLoopBackOffDetector) EntityTypes() []string {
	return []string{"kubernetes_pod"}
}

func (d *CrashLoopBackOffDetector) Interval() time.Duration {
	return d.interval
}

func (d *CrashLoopBackOffDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"} > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("crashloop query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])
		container := string(sample.Metric["container"])

		entity := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/crashloop", entity),
			Entity:     entity,
			EntityType: "kubernetes_pod",
			Type:       "crashloopbackoff",
			Severity:   models.SeverityFatal,
			Title:      "Pod CrashLoopBackOff",
			Message:    fmt.Sprintf("Pod %s/%s is in CrashLoopBackOff state", namespace, pod),
			Labels: map[string]string{
				"namespace": namespace,
				"pod":       pod,
				"container": container,
			},
			Metrics: map[string]float64{
				"waiting": float64(sample.Value),
			},
			Hint:        "Application startup failure or fatal runtime error",
			BlastRadius: 1,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// ImagePullBackOffDetector detects pods unable to pull images
type ImagePullBackOffDetector struct {
	interval time.Duration
}

func NewImagePullBackOffDetector() *ImagePullBackOffDetector {
	return &ImagePullBackOffDetector{
		interval: 30 * time.Second,
	}
}

func (d *ImagePullBackOffDetector) Name() string {
	return "kubernetes_imagepull"
}

func (d *ImagePullBackOffDetector) EntityTypes() []string {
	return []string{"kubernetes_pod"}
}

func (d *ImagePullBackOffDetector) Interval() time.Duration {
	return d.interval
}

func (d *ImagePullBackOffDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	query := `kube_pod_container_status_waiting_reason{reason=~"ImagePullBackOff|ErrImagePull"} > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("image pull query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])
		container := string(sample.Metric["container"])

		entity := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/imagepull", entity),
			Entity:     entity,
			EntityType: "kubernetes_pod",
			Type:       "imagepullbackoff",
			Severity:   models.SeverityCritical,
			Title:      "Image Pull Failed",
			Message:    fmt.Sprintf("Pod %s/%s cannot pull container image", namespace, pod),
			Labels: map[string]string{
				"namespace": namespace,
				"pod":       pod,
				"container": container,
			},
			Metrics: map[string]float64{
				"waiting": float64(sample.Value),
			},
			Hint:        "Image not found or registry authentication failure",
			BlastRadius: 1,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// PodPendingDetector detects pods stuck in Pending state
type PodPendingDetector struct {
	interval time.Duration
}

func NewPodPendingDetector() *PodPendingDetector {
	return &PodPendingDetector{
		interval: 30 * time.Second,
	}
}

func (d *PodPendingDetector) Name() string {
	return "kubernetes_pending"
}

func (d *PodPendingDetector) EntityTypes() []string {
	return []string{"kubernetes_pod"}
}

func (d *PodPendingDetector) Interval() time.Duration {
	return d.interval
}

func (d *PodPendingDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	// Detect pods currently in Pending phase for more than 5 minutes
	// Query: only pods where phase="Pending" AND value=1 (currently active)
	query := `kube_pod_status_phase{phase="Pending"} == 1 and on(namespace, pod) ((time() - kube_pod_created) > 300)`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pending pod query failed: %w", err)
	}

	problems := make([]*models.Problem, 0)
	for _, sample := range result {
		// Only process if value is 1 (pod is currently pending)
		if sample.Value != 1 {
			continue
		}

		namespace := string(sample.Metric["namespace"])
		pod := string(sample.Metric["pod"])

		entity := fmt.Sprintf("%s/%s", namespace, pod)
		problem := &models.Problem{
			ID:         fmt.Sprintf("%s/pending", entity),
			Entity:     entity,
			EntityType: "kubernetes_pod",
			Type:       "pending",
			Severity:   models.SeverityCritical,
			Title:      "Pod Pending",
			Message:    fmt.Sprintf("Pod %s/%s has been pending for >5 minutes", namespace, pod),
			Labels: map[string]string{
				"namespace": namespace,
				"pod":       pod,
			},
			Metrics: map[string]float64{
				"phase": float64(sample.Value),
			},
			Hint:        "Insufficient cluster resources or scheduling constraints",
			BlastRadius: 1,
		}
		problems = append(problems, problem)
	}

	return problems, nil
}
