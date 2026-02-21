package detector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/common/model"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

func TestLinkerdControlPlaneDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace":  "linkerd",
						"deployment": "linkerd-destination",
					},
					Value: 0,
				},
			}, nil
		},
	}

	d := NewLinkerdControlPlaneDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity, got %v", p.Severity)
	}
	if p.Type != "linkerd_control_plane_down" {
		t.Errorf("expected type 'linkerd_control_plane_down', got '%s'", p.Type)
	}
	if p.BlastRadius != 15 {
		t.Errorf("expected blast radius 15, got %d", p.BlastRadius)
	}
	if p.Labels["mesh"] != "linkerd" {
		t.Errorf("expected mesh label 'linkerd', got '%s'", p.Labels["mesh"])
	}
}

func TestLinkerdControlPlaneDetector_Healthy(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewLinkerdControlPlaneDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestLinkerdControlPlaneDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewLinkerdControlPlaneDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestLinkerdProxyInjectionDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "linkerd",
						"pod":       "linkerd-proxy-injector-abc",
						"container": "proxy-injector",
					},
					Value: 1,
				},
			}, nil
		},
	}

	d := NewLinkerdProxyInjectionDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityCritical {
		t.Errorf("expected CRITICAL severity, got %v", p.Severity)
	}
	if p.Type != "linkerd_component_crash" {
		t.Errorf("expected type 'linkerd_component_crash', got '%s'", p.Type)
	}
	if p.BlastRadius != 10 {
		t.Errorf("expected blast radius 10, got %d", p.BlastRadius)
	}
}

func TestLinkerdProxyInjectionDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("timeout")
		},
	}

	d := NewLinkerdProxyInjectionDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestIstioControlPlaneDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace":  "istio-system",
						"deployment": "istiod",
					},
					Value: 0,
				},
			}, nil
		},
	}

	d := NewIstioControlPlaneDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityFatal {
		t.Errorf("expected FATAL severity, got %v", p.Severity)
	}
	if p.Type != "istio_control_plane_down" {
		t.Errorf("expected type 'istio_control_plane_down', got '%s'", p.Type)
	}
	if p.Labels["mesh"] != "istio" {
		t.Errorf("expected mesh label 'istio', got '%s'", p.Labels["mesh"])
	}
}

func TestIstioControlPlaneDetector_Healthy(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
	}

	d := NewIstioControlPlaneDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(problems))
	}
}

func TestIstioControlPlaneDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	d := NewIstioControlPlaneDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestIstioSidecarInjectionDetector(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"namespace": "istio-system",
						"pod":       "istiod-abc123",
						"container": "discovery",
					},
					Value: 1,
				},
			}, nil
		},
	}

	d := NewIstioSidecarInjectionDetector()
	problems, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}

	p := problems[0]
	if p.Severity != models.SeverityCritical {
		t.Errorf("expected CRITICAL severity, got %v", p.Severity)
	}
	if p.Type != "istio_component_crash" {
		t.Errorf("expected type 'istio_component_crash', got '%s'", p.Type)
	}
}

func TestIstioSidecarInjectionDetector_ProviderError(t *testing.T) {
	mockProvider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return nil, fmt.Errorf("timeout")
		},
	}

	d := NewIstioSidecarInjectionDetector()
	_, err := d.Detect(context.Background(), mockProvider, 5*time.Minute)

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestServiceMeshDetectors_Metadata(t *testing.T) {
	tests := []struct {
		name         string
		detector     Detector
		expectedName string
	}{
		{"LinkerdControlPlane", NewLinkerdControlPlaneDetector(), "servicemesh_linkerd_controlplane"},
		{"LinkerdProxyInjection", NewLinkerdProxyInjectionDetector(), "servicemesh_linkerd_injection"},
		{"IstioControlPlane", NewIstioControlPlaneDetector(), "servicemesh_istio_controlplane"},
		{"IstioSidecarInjection", NewIstioSidecarInjectionDetector(), "servicemesh_istio_injection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.detector.Name() != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, tt.detector.Name())
			}
			if tt.detector.Interval() != 30*time.Second {
				t.Errorf("expected 30s interval, got %v", tt.detector.Interval())
			}
			entityTypes := tt.detector.EntityTypes()
			if len(entityTypes) != 1 || entityTypes[0] != "service_mesh_control_plane" {
				t.Errorf("expected entity type 'service_mesh_control_plane', got %v", entityTypes)
			}
		})
	}
}
