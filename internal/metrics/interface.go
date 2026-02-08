package metrics

import (
	"context"
	"time"

	"github.com/prometheus/common/model"
)

// MetricsProvider defines backend-agnostic metrics access
type MetricsProvider interface {
	// QueryRange performs a range query over a time window
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Matrix, error)

	// QueryInstant performs an instant query at a specific time
	QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error)

	// Health checks if the metrics backend is reachable
	Health(ctx context.Context) error
}
