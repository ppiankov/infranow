package baseline

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ppiankov/infranow/internal/models"
)

// Baseline represents a saved snapshot of problems
type Baseline struct {
	Timestamp time.Time         `json:"timestamp"`
	Problems  []*models.Problem `json:"problems"`
	Metadata  map[string]string `json:"metadata"`
}

// Comparison represents the diff between current and baseline states
type Comparison struct {
	New       []*models.Problem `json:"new"`
	Resolved  []*models.Problem `json:"resolved"`
	Unchanged []*models.Problem `json:"unchanged"`
	Summary   ComparisonSummary `json:"summary"`
}

// ComparisonSummary provides counts of changes
type ComparisonSummary struct {
	NewCount       int `json:"new_count"`
	ResolvedCount  int `json:"resolved_count"`
	UnchangedCount int `json:"unchanged_count"`
}

// SaveBaseline saves a problem snapshot to a file
func SaveBaseline(problems []*models.Problem, path string, metadata map[string]string) error {
	b := &Baseline{
		Timestamp: time.Now(),
		Problems:  problems,
		Metadata:  metadata,
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadBaseline loads a baseline from a file
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// Compare compares current problems against a baseline
func Compare(current []*models.Problem, baseline *Baseline) *Comparison {
	baselineMap := make(map[string]*models.Problem)
	for _, p := range baseline.Problems {
		baselineMap[p.ID] = p
	}

	currentMap := make(map[string]*models.Problem)
	for _, p := range current {
		currentMap[p.ID] = p
	}

	comp := &Comparison{
		New:       []*models.Problem{},
		Resolved:  []*models.Problem{},
		Unchanged: []*models.Problem{},
	}

	// Find new and unchanged
	for id, p := range currentMap {
		if _, exists := baselineMap[id]; exists {
			comp.Unchanged = append(comp.Unchanged, p)
		} else {
			comp.New = append(comp.New, p)
		}
	}

	// Find resolved
	for id, p := range baselineMap {
		if _, exists := currentMap[id]; !exists {
			comp.Resolved = append(comp.Resolved, p)
		}
	}

	comp.Summary = ComparisonSummary{
		NewCount:       len(comp.New),
		ResolvedCount:  len(comp.Resolved),
		UnchangedCount: len(comp.Unchanged),
	}

	return comp
}
