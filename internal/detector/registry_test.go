package detector

import (
	"context"
	"testing"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

type stubDetector struct {
	name string
}

func (s *stubDetector) Name() string            { return s.name }
func (s *stubDetector) EntityTypes() []string   { return []string{"test"} }
func (s *stubDetector) Interval() time.Duration { return 30 * time.Second }
func (s *stubDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, window time.Duration) ([]*models.Problem, error) {
	return nil, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("new registry count = %d, want 0", r.Count())
	}

	d1 := &stubDetector{name: "det-1"}
	d2 := &stubDetector{name: "det-2"}

	r.Register(d1)
	r.Register(d2)

	if r.Count() != 2 {
		t.Errorf("count after register = %d, want 2", r.Count())
	}

	got, ok := r.Get("det-1")
	if !ok {
		t.Fatal("expected to find det-1")
	}
	if got.Name() != "det-1" {
		t.Errorf("got name %q, want %q", got.Name(), "det-1")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent detector")
	}

	all := r.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d, want 2", len(all))
	}

	r.Unregister("det-1")
	if r.Count() != 1 {
		t.Errorf("count after unregister = %d, want 1", r.Count())
	}
}
