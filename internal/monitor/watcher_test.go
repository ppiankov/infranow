package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/common/model"

	"github.com/ppiankov/infranow/internal/detector"
	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

func newTestWatcher(maxConcurrency int) *Watcher {
	provider := &metrics.MockProvider{
		QueryInstantFunc: func(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
			return model.Vector{}, nil
		},
		HealthFunc: func(ctx context.Context) error {
			return nil
		},
	}
	registry := detector.NewRegistry()
	return NewWatcher(provider, registry, maxConcurrency, 30*time.Second)
}

func TestNewWatcher(t *testing.T) {
	w := newTestWatcher(5)

	if w.provider == nil {
		t.Fatal("provider should not be nil")
	}
	if w.problems == nil {
		t.Fatal("problems map should be initialized")
	}
	if !w.prometheusHealthy {
		t.Error("should start healthy")
	}
	if w.semaphore == nil {
		t.Fatal("semaphore should be initialized when maxConcurrency > 0")
	}
	if cap(w.semaphore) != 5 {
		t.Errorf("semaphore capacity = %d, want 5", cap(w.semaphore))
	}
}

func TestNewWatcher_UnlimitedConcurrency(t *testing.T) {
	w := newTestWatcher(0)

	if w.semaphore != nil {
		t.Error("semaphore should be nil when maxConcurrency = 0")
	}
}

func TestUpdateProblems_NewProblem(t *testing.T) {
	w := newTestWatcher(0)

	detected := []*models.Problem{
		{ID: "test/problem1", Severity: models.SeverityCritical},
	}

	w.updateProblems(detected)

	w.mu.RLock()
	defer w.mu.RUnlock()

	p, ok := w.problems["test/problem1"]
	if !ok {
		t.Fatal("problem should be added to map")
	}
	if p.Count != 1 {
		t.Errorf("count = %d, want 1", p.Count)
	}
	if p.FirstSeen.IsZero() {
		t.Error("FirstSeen should be set")
	}
	if p.LastSeen.IsZero() {
		t.Error("LastSeen should be set")
	}
}

func TestUpdateProblems_UpdateExisting(t *testing.T) {
	w := newTestWatcher(0)

	// Insert initial problem
	initial := []*models.Problem{
		{ID: "test/problem1", Severity: models.SeverityCritical},
	}
	w.updateProblems(initial)

	w.mu.RLock()
	firstSeen := w.problems["test/problem1"].FirstSeen
	w.mu.RUnlock()

	// Small delay to ensure LastSeen changes
	time.Sleep(time.Millisecond)

	// Update same problem
	update := []*models.Problem{
		{ID: "test/problem1", Severity: models.SeverityCritical},
	}
	w.updateProblems(update)

	w.mu.RLock()
	defer w.mu.RUnlock()

	p := w.problems["test/problem1"]
	if p.Count != 2 {
		t.Errorf("count = %d, want 2", p.Count)
	}
	if !p.FirstSeen.Equal(firstSeen) {
		t.Error("FirstSeen should not change on update")
	}
	if !p.LastSeen.After(firstSeen) {
		t.Error("LastSeen should be updated")
	}
}

func TestUpdateProblems_StalePruning(t *testing.T) {
	w := newTestWatcher(0)

	// Manually insert a stale problem
	w.mu.Lock()
	w.problems["stale/problem"] = &models.Problem{
		ID:       "stale/problem",
		LastSeen: time.Now().Add(-2 * time.Minute),
	}
	w.mu.Unlock()

	// Trigger update with empty list
	w.updateProblems([]*models.Problem{})

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, ok := w.problems["stale/problem"]; ok {
		t.Error("stale problem should be pruned")
	}
}

func TestUpdateProblems_NotifiesUpdateChan(t *testing.T) {
	w := newTestWatcher(0)

	detected := []*models.Problem{
		{ID: "test/notify", Severity: models.SeverityWarning},
	}
	w.updateProblems(detected)

	select {
	case <-w.updateChan:
		// expected
	default:
		t.Error("expected notification on updateChan")
	}
}

func TestUpdateProblems_NoNotificationWhenUnchanged(t *testing.T) {
	w := newTestWatcher(0)

	// No detected problems, no existing problems -> nothing changes
	w.updateProblems([]*models.Problem{})

	select {
	case <-w.updateChan:
		t.Error("should not notify when nothing changed")
	default:
		// expected
	}
}

func TestGetProblems_SortedByScore(t *testing.T) {
	w := newTestWatcher(0)

	w.mu.Lock()
	now := time.Now()
	w.problems["a"] = &models.Problem{ID: "a", Severity: models.SeverityWarning, LastSeen: now}
	w.problems["b"] = &models.Problem{ID: "b", Severity: models.SeverityFatal, LastSeen: now}
	w.problems["c"] = &models.Problem{ID: "c", Severity: models.SeverityCritical, LastSeen: now}
	w.mu.Unlock()

	problems := w.GetProblems()

	if len(problems) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(problems))
	}
	if problems[0].Severity != models.SeverityFatal {
		t.Errorf("first problem should be FATAL, got %v", problems[0].Severity)
	}
	if problems[2].Severity != models.SeverityWarning {
		t.Errorf("last problem should be WARNING, got %v", problems[2].Severity)
	}
}

func TestGetProblems_ReturnsCopies(t *testing.T) {
	w := newTestWatcher(0)

	w.mu.Lock()
	w.problems["a"] = &models.Problem{ID: "a", Severity: models.SeverityCritical, LastSeen: time.Now(), Count: 1}
	w.mu.Unlock()

	problems := w.GetProblems()
	problems[0].Count = 999

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.problems["a"].Count == 999 {
		t.Error("mutation of returned problem should not affect internal state")
	}
}

func TestGetSummary(t *testing.T) {
	w := newTestWatcher(0)

	w.mu.Lock()
	now := time.Now()
	w.problems["a"] = &models.Problem{ID: "a", Severity: models.SeverityFatal, LastSeen: now}
	w.problems["b"] = &models.Problem{ID: "b", Severity: models.SeverityCritical, LastSeen: now}
	w.problems["c"] = &models.Problem{ID: "c", Severity: models.SeverityCritical, LastSeen: now}
	w.problems["d"] = &models.Problem{ID: "d", Severity: models.SeverityWarning, LastSeen: now}
	w.mu.Unlock()

	summary := w.GetSummary()

	if summary[models.SeverityFatal] != 1 {
		t.Errorf("fatal count = %d, want 1", summary[models.SeverityFatal])
	}
	if summary[models.SeverityCritical] != 2 {
		t.Errorf("critical count = %d, want 2", summary[models.SeverityCritical])
	}
	if summary[models.SeverityWarning] != 1 {
		t.Errorf("warning count = %d, want 1", summary[models.SeverityWarning])
	}
}

func TestGetProblemsByRecency(t *testing.T) {
	w := newTestWatcher(0)

	now := time.Now()
	w.mu.Lock()
	w.problems["a"] = &models.Problem{ID: "a", LastSeen: now.Add(-2 * time.Minute)}
	w.problems["b"] = &models.Problem{ID: "b", LastSeen: now}
	w.problems["c"] = &models.Problem{ID: "c", LastSeen: now.Add(-1 * time.Minute)}
	w.mu.Unlock()

	problems := w.GetProblemsByRecency()

	if len(problems) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(problems))
	}
	if problems[0].ID != "b" {
		t.Errorf("first problem should be most recent (b), got %s", problems[0].ID)
	}
	if problems[2].ID != "a" {
		t.Errorf("last problem should be oldest (a), got %s", problems[2].ID)
	}
}

func TestGetProblemsByCount(t *testing.T) {
	w := newTestWatcher(0)

	now := time.Now()
	w.mu.Lock()
	w.problems["a"] = &models.Problem{ID: "a", Count: 5, LastSeen: now}
	w.problems["b"] = &models.Problem{ID: "b", Count: 10, LastSeen: now}
	w.problems["c"] = &models.Problem{ID: "c", Count: 1, LastSeen: now}
	w.mu.Unlock()

	problems := w.GetProblemsByCount()

	if len(problems) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(problems))
	}
	if problems[0].Count != 10 {
		t.Errorf("first problem count = %d, want 10", problems[0].Count)
	}
	if problems[2].Count != 1 {
		t.Errorf("last problem count = %d, want 1", problems[2].Count)
	}
}

func TestGetPrometheusHealth(t *testing.T) {
	w := newTestWatcher(0)

	w.mu.Lock()
	w.prometheusHealthy = false
	checkTime := time.Now()
	w.lastPrometheusCheck = checkTime
	w.mu.Unlock()

	healthy, lastCheck := w.GetPrometheusHealth()
	if healthy {
		t.Error("expected unhealthy")
	}
	if !lastCheck.Equal(checkTime) {
		t.Error("lastCheck time mismatch")
	}
}

func TestUpdateChan(t *testing.T) {
	w := newTestWatcher(0)

	ch := w.UpdateChan()
	if ch == nil {
		t.Fatal("UpdateChan should not return nil")
	}
}

func TestGetPrometheusStats(t *testing.T) {
	w := newTestWatcher(0)

	w.mu.Lock()
	w.queryCount = 100
	w.errorCount = 25
	w.prometheusHealthy = true
	w.mu.Unlock()

	stats := w.GetPrometheusStats()

	if !stats.Healthy {
		t.Error("expected healthy = true")
	}
	if stats.QueryCount != 100 {
		t.Errorf("query count = %d, want 100", stats.QueryCount)
	}
	if stats.ErrorCount != 25 {
		t.Errorf("error count = %d, want 25", stats.ErrorCount)
	}
	if stats.ErrorRate != 0.25 {
		t.Errorf("error rate = %f, want 0.25", stats.ErrorRate)
	}
}
