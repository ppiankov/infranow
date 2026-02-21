package baseline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ppiankov/infranow/internal/models"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name          string
		current       []*models.Problem
		baseline      []*models.Problem
		wantNew       int
		wantResolved  int
		wantUnchanged int
	}{
		{
			name:          "new problem detected",
			current:       []*models.Problem{{ID: "a"}, {ID: "b"}},
			baseline:      []*models.Problem{{ID: "a"}},
			wantNew:       1,
			wantResolved:  0,
			wantUnchanged: 1,
		},
		{
			name:          "problem resolved",
			current:       []*models.Problem{{ID: "a"}},
			baseline:      []*models.Problem{{ID: "a"}, {ID: "b"}},
			wantNew:       0,
			wantResolved:  1,
			wantUnchanged: 1,
		},
		{
			name:          "unchanged",
			current:       []*models.Problem{{ID: "a"}},
			baseline:      []*models.Problem{{ID: "a"}},
			wantNew:       0,
			wantResolved:  0,
			wantUnchanged: 1,
		},
		{
			name:          "empty baseline",
			current:       []*models.Problem{{ID: "a"}},
			baseline:      []*models.Problem{},
			wantNew:       1,
			wantResolved:  0,
			wantUnchanged: 0,
		},
		{
			name:          "empty current",
			current:       []*models.Problem{},
			baseline:      []*models.Problem{{ID: "a"}},
			wantNew:       0,
			wantResolved:  1,
			wantUnchanged: 0,
		},
		{
			name:          "mixed changes",
			current:       []*models.Problem{{ID: "a"}, {ID: "c"}},
			baseline:      []*models.Problem{{ID: "a"}, {ID: "b"}},
			wantNew:       1,
			wantResolved:  1,
			wantUnchanged: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Baseline{Problems: tt.baseline}
			comp := Compare(tt.current, b)

			if comp.Summary.NewCount != tt.wantNew {
				t.Errorf("new count = %d, want %d", comp.Summary.NewCount, tt.wantNew)
			}
			if comp.Summary.ResolvedCount != tt.wantResolved {
				t.Errorf("resolved count = %d, want %d", comp.Summary.ResolvedCount, tt.wantResolved)
			}
			if comp.Summary.UnchangedCount != tt.wantUnchanged {
				t.Errorf("unchanged count = %d, want %d", comp.Summary.UnchangedCount, tt.wantUnchanged)
			}
			if len(comp.New) != tt.wantNew {
				t.Errorf("len(New) = %d, want %d", len(comp.New), tt.wantNew)
			}
			if len(comp.Resolved) != tt.wantResolved {
				t.Errorf("len(Resolved) = %d, want %d", len(comp.Resolved), tt.wantResolved)
			}
			if len(comp.Unchanged) != tt.wantUnchanged {
				t.Errorf("len(Unchanged) = %d, want %d", len(comp.Unchanged), tt.wantUnchanged)
			}
		})
	}
}

func TestSaveAndLoadBaseline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")

	problems := []*models.Problem{
		{ID: "p1", Entity: "ns/pod", Severity: models.SeverityCritical},
		{ID: "p2", Entity: "ns/pod2", Severity: models.SeverityWarning},
	}
	metadata := map[string]string{"version": "test"}

	if err := SaveBaseline(problems, path, metadata); err != nil {
		t.Fatalf("SaveBaseline failed: %v", err)
	}

	loaded, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	if loaded.Timestamp.IsZero() {
		t.Errorf("timestamp should not be zero")
	}
	if len(loaded.Problems) != 2 {
		t.Errorf("expected 2 problems, got %d", len(loaded.Problems))
	}
	if loaded.Metadata["version"] != "test" {
		t.Errorf("metadata version = %q, want %q", loaded.Metadata["version"], "test")
	}

	// Verify problem IDs roundtrip
	ids := make(map[string]bool)
	for _, p := range loaded.Problems {
		ids[p.ID] = true
	}
	if !ids["p1"] || !ids["p2"] {
		t.Errorf("problem IDs not preserved: got %v", ids)
	}
}

func TestLoadBaseline_FileNotFound(t *testing.T) {
	_, err := LoadBaseline("/nonexistent/path/baseline.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadBaseline_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadBaseline(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
