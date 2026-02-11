package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient implements MetricsProvider for Prometheus
type PrometheusClient struct {
	url    string
	client api.Client
	api    promv1.API
}

// NewPrometheusClient creates a new Prometheus metrics provider
func NewPrometheusClient(url string, timeout time.Duration) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusClient{
		url:    url,
		client: client,
		api:    promv1.NewAPI(client),
	}, nil
}

// QueryRange performs a range query over a time window
func (p *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Matrix, error) {
	result, warnings, err := p.api.QueryRange(ctx, query, promv1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, fmt.Errorf("query range failed: %w", err)
	}

	// Prometheus warnings are informational (e.g., query hints) — not actionable for infranow
	_ = warnings

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return matrix, nil
}

// QueryInstant performs an instant query at a specific time
func (p *PrometheusClient) QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
	result, warnings, err := p.api.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("instant query failed: %w", err)
	}

	// Prometheus warnings are informational (e.g., query hints) — not actionable for infranow
	_ = warnings

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return vector, nil
}

// Health checks if the Prometheus server is reachable
func (p *PrometheusClient) Health(ctx context.Context) error {
	_, err := p.api.Runtimeinfo(ctx)
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}
	return nil
}
