package monitor

import (
	"encoding/json"
	"testing"

	"github.com/ppiankov/infranow/internal/models"
)

func TestSARIF_Empty(t *testing.T) {
	data, err := SARIF(nil, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF(nil) error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if log.Version != "2.1.0" {
		t.Errorf("version = %q, want %q", log.Version, "2.1.0")
	}
	if log.Schema != sarifSchema {
		t.Errorf("schema = %q, want %q", log.Schema, sarifSchema)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(log.Runs))
	}
	if len(log.Runs[0].Results) != 0 {
		t.Errorf("results = %d, want 0", len(log.Runs[0].Results))
	}
	if len(log.Runs[0].Tool.Driver.Rules) != 0 {
		t.Errorf("rules = %d, want 0", len(log.Runs[0].Tool.Driver.Rules))
	}
}

func TestSARIF_SingleProblem(t *testing.T) {
	problems := []*models.Problem{
		{
			Type:       "oomkill",
			Severity:   models.SeverityCritical,
			Entity:     "production/payment-api",
			EntityType: "pod",
			Title:      "OOMKilled 3 times",
			Message:    "container exceeded memory limit",
			Hint:       "increase memory request",
		},
	}

	data, err := SARIF(problems, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	run := log.Runs[0]

	// Check tool driver
	if run.Tool.Driver.Name != "infranow" {
		t.Errorf("driver name = %q, want %q", run.Tool.Driver.Name, "infranow")
	}
	if run.Tool.Driver.Version != "0.2.0" {
		t.Errorf("driver version = %q, want %q", run.Tool.Driver.Version, "0.2.0")
	}

	// Check rules
	if len(run.Tool.Driver.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(run.Tool.Driver.Rules))
	}
	rule := run.Tool.Driver.Rules[0]
	if rule.ID != "infranow/oomkill" {
		t.Errorf("rule ID = %q, want %q", rule.ID, "infranow/oomkill")
	}
	if rule.DefaultConfig.Level != "error" {
		t.Errorf("rule level = %q, want %q", rule.DefaultConfig.Level, "error")
	}

	// Check results
	if len(run.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(run.Results))
	}
	r := run.Results[0]
	if r.RuleID != "infranow/oomkill" {
		t.Errorf("ruleId = %q, want %q", r.RuleID, "infranow/oomkill")
	}
	if r.Level != "error" {
		t.Errorf("level = %q, want %q", r.Level, "error")
	}
	if r.Message.Text != "OOMKilled 3 times — container exceeded memory limit [hint: increase memory request]" {
		t.Errorf("message = %q", r.Message.Text)
	}

	// Check location
	if len(r.Locations) != 1 || len(r.Locations[0].LogicalLocations) != 1 {
		t.Fatal("expected 1 logical location")
	}
	loc := r.Locations[0].LogicalLocations[0]
	if loc.Name != "payment-api" {
		t.Errorf("location name = %q, want %q", loc.Name, "payment-api")
	}
	if loc.FullyQualifiedName != "production/payment-api" {
		t.Errorf("fqn = %q, want %q", loc.FullyQualifiedName, "production/payment-api")
	}
	if loc.Kind != "pod" {
		t.Errorf("kind = %q, want %q", loc.Kind, "pod")
	}
}

func TestSARIF_SameTypeDedupesRules(t *testing.T) {
	problems := []*models.Problem{
		{Type: "oomkill", Severity: models.SeverityCritical, Entity: "ns/pod-a", Title: "OOM A"},
		{Type: "oomkill", Severity: models.SeverityCritical, Entity: "ns/pod-b", Title: "OOM B"},
	}

	data, err := SARIF(problems, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	run := log.Runs[0]
	if len(run.Tool.Driver.Rules) != 1 {
		t.Errorf("rules = %d, want 1 (same type should dedup)", len(run.Tool.Driver.Rules))
	}
	if len(run.Results) != 2 {
		t.Errorf("results = %d, want 2", len(run.Results))
	}
}

func TestSARIF_DifferentTypesMultipleRules(t *testing.T) {
	problems := []*models.Problem{
		{Type: "oomkill", Severity: models.SeverityCritical, Entity: "ns/pod-a", Title: "OOM"},
		{Type: "crashloop", Severity: models.SeverityFatal, Entity: "ns/pod-b", Title: "CrashLoop"},
		{Type: "disk_space", Severity: models.SeverityWarning, Entity: "node-1", Title: "Disk 91%"},
	}

	data, err := SARIF(problems, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	run := log.Runs[0]
	if len(run.Tool.Driver.Rules) != 3 {
		t.Errorf("rules = %d, want 3", len(run.Tool.Driver.Rules))
	}
	if len(run.Results) != 3 {
		t.Errorf("results = %d, want 3", len(run.Results))
	}
}

func TestSARIF_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity models.Severity
		want     string
	}{
		{models.SeverityFatal, "error"},
		{models.SeverityCritical, "error"},
		{models.SeverityWarning, "warning"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			problems := []*models.Problem{
				{Type: "test", Severity: tt.severity, Entity: "ns/pod", Title: "test"},
			}

			data, err := SARIF(problems, "0.2.0")
			if err != nil {
				t.Fatalf("SARIF error: %v", err)
			}

			var log sarifLog
			if err := json.Unmarshal(data, &log); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			r := log.Runs[0].Results[0]
			if r.Level != tt.want {
				t.Errorf("level = %q, want %q", r.Level, tt.want)
			}
		})
	}
}

func TestSARIF_MessageFormats(t *testing.T) {
	tests := []struct {
		name    string
		problem *models.Problem
		want    string
	}{
		{
			name:    "title only",
			problem: &models.Problem{Type: "t", Severity: models.SeverityWarning, Entity: "e", Title: "Disk 91%"},
			want:    "Disk 91%",
		},
		{
			name:    "title and message",
			problem: &models.Problem{Type: "t", Severity: models.SeverityWarning, Entity: "e", Title: "Disk 91%", Message: "/dev/sda1"},
			want:    "Disk 91% — /dev/sda1",
		},
		{
			name:    "title message and hint",
			problem: &models.Problem{Type: "t", Severity: models.SeverityWarning, Entity: "e", Title: "Disk 91%", Message: "/dev/sda1", Hint: "clean logs"},
			want:    "Disk 91% — /dev/sda1 [hint: clean logs]",
		},
		{
			name:    "title and hint no message",
			problem: &models.Problem{Type: "t", Severity: models.SeverityWarning, Entity: "e", Title: "Disk 91%", Hint: "clean logs"},
			want:    "Disk 91% [hint: clean logs]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := SARIF([]*models.Problem{tt.problem}, "0.2.0")
			if err != nil {
				t.Fatalf("SARIF error: %v", err)
			}

			var log sarifLog
			if err := json.Unmarshal(data, &log); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			got := log.Runs[0].Results[0].Message.Text
			if got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSARIF_DefaultEntityKind(t *testing.T) {
	problems := []*models.Problem{
		{Type: "test", Severity: models.SeverityWarning, Entity: "ns/something", Title: "test"},
	}

	data, err := SARIF(problems, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	kind := log.Runs[0].Results[0].Locations[0].LogicalLocations[0].Kind
	if kind != "infrastructure/resource" {
		t.Errorf("kind = %q, want %q", kind, "infrastructure/resource")
	}
}

func TestEntityName(t *testing.T) {
	tests := []struct {
		entity string
		want   string
	}{
		{"production/payment-api", "payment-api"},
		{"kube-system/coredns", "coredns"},
		{"single-name", "single-name"},
		{"a/b/c", "c"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.entity, func(t *testing.T) {
			got := entityName(tt.entity)
			if got != tt.want {
				t.Errorf("entityName(%q) = %q, want %q", tt.entity, got, tt.want)
			}
		})
	}
}

func TestFormatSARIFSummary(t *testing.T) {
	problems := []*models.Problem{
		{Type: "oomkill", Severity: models.SeverityCritical},
		{Type: "oomkill", Severity: models.SeverityCritical},
		{Type: "disk_space", Severity: models.SeverityWarning},
	}

	got := FormatSARIFSummary(problems)
	if got != "SARIF: 3 results, 2 rules" {
		t.Errorf("FormatSARIFSummary() = %q, want %q", got, "SARIF: 3 results, 2 rules")
	}
}

func TestSARIF_ValidJSON(t *testing.T) {
	problems := []*models.Problem{
		{Type: "oomkill", Severity: models.SeverityCritical, Entity: "ns/pod", Title: "OOM"},
		{Type: "disk_space", Severity: models.SeverityWarning, Entity: "node-1", Title: "Disk 91%"},
	}

	data, err := SARIF(problems, "0.2.0")
	if err != nil {
		t.Fatalf("SARIF error: %v", err)
	}

	// Verify it's valid JSON
	if !json.Valid(data) {
		t.Error("SARIF output is not valid JSON")
	}

	// Verify it round-trips
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
}
