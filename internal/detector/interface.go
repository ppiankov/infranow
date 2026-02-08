package detector

import (
	"context"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// Detector identifies problems from metrics
type Detector interface {
	// Name returns detector identifier (e.g., "kubernetes_oom_kills")
	Name() string

	// EntityTypes returns which entity types this detector handles
	EntityTypes() []string

	// Detect runs detection logic and returns problems found
	Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error)

	// Interval returns how often this detector should run
	Interval() time.Duration
}
