package monitor

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ppiankov/infranow/internal/detector"
	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

// Watcher orchestrates problem detection and state management
type Watcher struct {
	provider metrics.MetricsProvider
	registry *detector.Registry

	mu       sync.RWMutex
	problems map[string]*models.Problem // Keyed by Problem.ID

	updateChan chan struct{} // Notify UI of changes
	stopChan   chan struct{}
	stopped    bool
}

// NewWatcher creates a new watcher instance
func NewWatcher(provider metrics.MetricsProvider, registry *detector.Registry) *Watcher {
	return &Watcher{
		provider:   provider,
		registry:   registry,
		problems:   make(map[string]*models.Problem),
		updateChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}
}

// Start begins the monitoring loop
func (w *Watcher) Start(ctx context.Context) error {
	detectors := w.registry.All()
	if len(detectors) == 0 {
		return nil
	}

	// Start each detector in its own goroutine
	var wg sync.WaitGroup
	for _, d := range detectors {
		wg.Add(1)
		go func(det detector.Detector) {
			defer wg.Done()
			w.runDetector(ctx, det)
		}(d)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Mark as stopped and wait for all detectors to finish
	w.mu.Lock()
	w.stopped = true
	w.mu.Unlock()

	wg.Wait()
	close(w.updateChan)

	return nil
}

// runDetector runs a single detector at its specified interval
func (w *Watcher) runDetector(ctx context.Context, d detector.Detector) {
	ticker := time.NewTicker(d.Interval())
	defer ticker.Stop()

	// Run immediately on start
	w.executeDetector(ctx, d)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.executeDetector(ctx, d)
		}
	}
}

// executeDetector runs detection logic and updates problem state
func (w *Watcher) executeDetector(ctx context.Context, d detector.Detector) {
	// Create context with timeout for this detection cycle
	detCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	problems, err := d.Detect(detCtx, w.provider, 5*time.Minute)
	if err != nil {
		// Log error but continue
		// TODO: Add proper logging
		return
	}

	if len(problems) > 0 {
		w.updateProblems(problems)
	}
}

// updateProblems merges detected problems with existing state
func (w *Watcher) updateProblems(detected []*models.Problem) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	updated := false

	for _, p := range detected {
		if existing, ok := w.problems[p.ID]; ok {
			// Update existing problem
			existing.Count++
			existing.LastSeen = now
			existing.Metrics = p.Metrics
			existing.UpdatePersistence()
			updated = true
		} else {
			// New problem
			p.FirstSeen = now
			p.LastSeen = now
			p.Count = 1
			p.UpdatePersistence()
			w.problems[p.ID] = p
			updated = true
		}
	}

	// Prune stale problems (not seen in last 2 minutes)
	staleThreshold := now.Add(-2 * time.Minute)
	for id, p := range w.problems {
		if p.LastSeen.Before(staleThreshold) {
			delete(w.problems, id)
			updated = true
		}
	}

	// Notify UI if there were changes
	if updated {
		select {
		case w.updateChan <- struct{}{}:
		default:
			// Channel already has a pending notification
		}
	}
}

// GetProblems returns current problems sorted by score
func (w *Watcher) GetProblems() []*models.Problem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	list := make([]*models.Problem, 0, len(w.problems))
	for _, p := range w.problems {
		// Create a copy to avoid race conditions
		pCopy := *p
		list = append(list, &pCopy)
	}

	// Sort by score descending
	sort.Slice(list, func(i, j int) bool {
		return list[i].Score() > list[j].Score()
	})

	return list
}

// GetProblemsByRecency returns problems sorted by most recent first
func (w *Watcher) GetProblemsByRecency() []*models.Problem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	list := make([]*models.Problem, 0, len(w.problems))
	for _, p := range w.problems {
		pCopy := *p
		list = append(list, &pCopy)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastSeen.After(list[j].LastSeen)
	})

	return list
}

// GetProblemsByCount returns problems sorted by count descending
func (w *Watcher) GetProblemsByCount() []*models.Problem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	list := make([]*models.Problem, 0, len(w.problems))
	for _, p := range w.problems {
		pCopy := *p
		list = append(list, &pCopy)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Count > list[j].Count
	})

	return list
}

// GetSummary returns problem count by severity
func (w *Watcher) GetSummary() map[models.Severity]int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	summary := map[models.Severity]int{
		models.SeverityFatal:    0,
		models.SeverityCritical: 0,
		models.SeverityWarning:  0,
	}

	for _, p := range w.problems {
		summary[p.Severity]++
	}

	return summary
}

// UpdateChan returns the channel for UI update notifications
func (w *Watcher) UpdateChan() <-chan struct{} {
	return w.updateChan
}

// Stop signals the watcher to stop
func (w *Watcher) Stop() {
	close(w.stopChan)
}
