package metrics

import (
	"context"
	"time"

	"github.com/prometheus/common/model"
)

// MockProvider implements MetricsProvider for testing
type MockProvider struct {
	QueryRangeFunc   func(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Matrix, error)
	QueryInstantFunc func(ctx context.Context, query string, ts time.Time) (model.Vector, error)
	HealthFunc       func(ctx context.Context) error
}

// QueryRange calls the mock function if set, otherwise returns empty result
func (m *MockProvider) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Matrix, error) {
	if m.QueryRangeFunc != nil {
		return m.QueryRangeFunc(ctx, query, start, end, step)
	}
	return model.Matrix{}, nil
}

// QueryInstant calls the mock function if set, otherwise returns empty result
func (m *MockProvider) QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
	if m.QueryInstantFunc != nil {
		return m.QueryInstantFunc(ctx, query, ts)
	}
	return model.Vector{}, nil
}

// Health calls the mock function if set, otherwise returns nil
func (m *MockProvider) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}
