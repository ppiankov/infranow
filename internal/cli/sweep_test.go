package cli

import (
	"errors"
	"testing"

	"github.com/ppiankov/infranow/internal/models"
)

func TestMergeResults(t *testing.T) {
	tests := []struct {
		name         string
		results      []ClusterResult
		wantProblems int
		wantFailures int
	}{
		{
			name:         "empty",
			results:      nil,
			wantProblems: 0,
			wantFailures: 0,
		},
		{
			name: "all succeed",
			results: []ClusterResult{
				{Context: "a", Problems: []*models.Problem{
					{Severity: models.SeverityWarning, Entity: "a/pod1"},
				}},
				{Context: "b", Problems: []*models.Problem{
					{Severity: models.SeverityCritical, Entity: "b/pod2"},
				}},
			},
			wantProblems: 2,
			wantFailures: 0,
		},
		{
			name: "mixed success and failure",
			results: []ClusterResult{
				{Context: "ok", Problems: []*models.Problem{
					{Severity: models.SeverityFatal, Entity: "ok/pod"},
				}},
				{Context: "bad", Error: errTestCluster},
			},
			wantProblems: 1,
			wantFailures: 1,
		},
		{
			name: "all fail",
			results: []ClusterResult{
				{Context: "bad1", Error: errTestCluster},
				{Context: "bad2", Error: errTestCluster},
			},
			wantProblems: 0,
			wantFailures: 2,
		},
		{
			name: "sorted by score descending",
			results: []ClusterResult{
				{Context: "a", Problems: []*models.Problem{
					{Severity: models.SeverityWarning, Entity: "warn"},
				}},
				{Context: "b", Problems: []*models.Problem{
					{Severity: models.SeverityFatal, Entity: "fatal"},
				}},
			},
			wantProblems: 2,
			wantFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems, failures := mergeResults(tt.results)
			if len(problems) != tt.wantProblems {
				t.Errorf("got %d problems, want %d", len(problems), tt.wantProblems)
			}
			if len(failures) != tt.wantFailures {
				t.Errorf("got %d failures, want %d", len(failures), tt.wantFailures)
			}
		})
	}

	// Verify score ordering
	t.Run("score ordering", func(t *testing.T) {
		results := []ClusterResult{
			{Context: "a", Problems: []*models.Problem{
				{Severity: models.SeverityWarning, Entity: "low"},
			}},
			{Context: "b", Problems: []*models.Problem{
				{Severity: models.SeverityFatal, Entity: "high"},
			}},
		}
		problems, _ := mergeResults(results)
		if len(problems) != 2 {
			t.Fatalf("got %d problems, want 2", len(problems))
		}
		if problems[0].Score() < problems[1].Score() {
			t.Error("problems not sorted by score descending")
		}
	})
}

func TestCountBySeverity(t *testing.T) {
	problems := []*models.Problem{
		{Severity: models.SeverityFatal},
		{Severity: models.SeverityCritical},
		{Severity: models.SeverityCritical},
		{Severity: models.SeverityWarning},
		{Severity: models.SeverityWarning},
		{Severity: models.SeverityWarning},
	}

	if got := countBySeverity(problems, models.SeverityFatal); got != 1 {
		t.Errorf("fatal: got %d, want 1", got)
	}
	if got := countBySeverity(problems, models.SeverityCritical); got != 2 {
		t.Errorf("critical: got %d, want 2", got)
	}
	if got := countBySeverity(problems, models.SeverityWarning); got != 3 {
		t.Errorf("warning: got %d, want 3", got)
	}
	if got := countBySeverity(nil, models.SeverityFatal); got != 0 {
		t.Errorf("nil: got %d, want 0", got)
	}
}

func TestApplySweepFilters(t *testing.T) {
	problems := []*models.Problem{
		{Entity: "[ctx] prod/pod1", Labels: map[string]string{"namespace": "production"}},
		{Entity: "[ctx] stg/pod2", Labels: map[string]string{"namespace": "staging"}},
		{Entity: "[ctx] disk-full", Labels: map[string]string{}},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		sweepIncludeNS = ""
		sweepExcludeNS = ""
		got := applySweepFilters(problems)
		if len(got) != 3 {
			t.Errorf("got %d, want 3", len(got))
		}
	})

	t.Run("include filter", func(t *testing.T) {
		sweepIncludeNS = "production"
		sweepExcludeNS = ""
		got := applySweepFilters(problems)
		// production matches, staging doesn't, empty namespace passes through
		if len(got) != 2 {
			t.Errorf("got %d, want 2", len(got))
		}
	})

	t.Run("exclude filter", func(t *testing.T) {
		sweepIncludeNS = ""
		sweepExcludeNS = "staging"
		got := applySweepFilters(problems)
		// production matches, staging excluded, empty namespace passes through
		if len(got) != 2 {
			t.Errorf("got %d, want 2", len(got))
		}
	})

	// Reset for other tests
	sweepIncludeNS = ""
	sweepExcludeNS = ""
}

// errTestCluster is a sentinel error for testing.
var errTestCluster = errors.New("test cluster error")
