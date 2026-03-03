package monitor

import (
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/infranow/internal/models"
)

func TestPlainText_Empty(t *testing.T) {
	got := PlainText(nil, time.Now())
	if got != noProblemsMessage {
		t.Errorf("PlainText(nil) = %q, want %q", got, noProblemsMessage)
	}
}

func TestPlainText_SingleProblem(t *testing.T) {
	now := time.Now()
	problems := []*models.Problem{
		{
			Severity:  models.SeverityCritical,
			Entity:    "production/payment-api",
			Title:     "OOMKilled 3 times",
			FirstSeen: now.Add(-5 * time.Minute),
			Count:     3,
		},
	}
	got := PlainText(problems, now)

	if !strings.Contains(got, "CRIT") {
		t.Error("expected CRIT severity")
	}
	if !strings.Contains(got, "production/payment-api") {
		t.Error("expected entity")
	}
	if !strings.Contains(got, "OOMKilled 3 times") {
		t.Error("expected title")
	}
	if !strings.Contains(got, "5m") {
		t.Error("expected 5m age")
	}
	if !strings.Contains(got, "3") {
		t.Error("expected count 3")
	}
	// Header row must be present
	if !strings.Contains(got, "SEV") {
		t.Error("expected header row")
	}
}

func TestPlainText_MultipleProblems(t *testing.T) {
	now := time.Now()
	problems := []*models.Problem{
		{
			Severity:  models.SeverityFatal,
			Entity:    "kube-system/coredns",
			Title:     "CrashLoopBackOff",
			FirstSeen: now.Add(-2 * time.Hour),
			Count:     12,
		},
		{
			Severity:  models.SeverityWarning,
			Entity:    "monitoring/prometheus",
			Title:     "Disk usage 91%",
			FirstSeen: now.Add(-30 * time.Second),
			Count:     1,
		},
	}
	got := PlainText(problems, now)
	lines := strings.Split(strings.TrimSpace(got), "\n")

	// Header + separator + 2 data rows = 4 lines
	if len(lines) != 4 {
		t.Errorf("expected 4 lines, got %d:\n%s", len(lines), got)
	}
	if !strings.Contains(lines[2], "FATAL") {
		t.Error("first data row should be FATAL")
	}
	if !strings.Contains(lines[3], "WARN") {
		t.Error("second data row should be WARN")
	}
}

func TestPlainText_TruncatesLongEntity(t *testing.T) {
	now := time.Now()
	problems := []*models.Problem{
		{
			Severity:  models.SeverityWarning,
			Entity:    "very-long-namespace/very-long-deployment-name-that-exceeds-thirty-chars",
			Title:     "Something wrong",
			FirstSeen: now,
			Count:     1,
		},
	}
	got := PlainText(problems, now)
	if !strings.Contains(got, "...") {
		t.Error("expected truncation with ellipsis for long entity")
	}
}

func TestPlainTextSummary(t *testing.T) {
	tests := []struct {
		name     string
		problems []*models.Problem
		want     string
	}{
		{
			name:     "empty",
			problems: nil,
			want:     noProblemsMessage,
		},
		{
			name: "mixed",
			problems: []*models.Problem{
				{Severity: models.SeverityFatal},
				{Severity: models.SeverityCritical},
				{Severity: models.SeverityCritical},
				{Severity: models.SeverityWarning},
			},
			want: "4 problems: 1 fatal, 2 critical, 1 warning",
		},
		{
			name: "warnings only",
			problems: []*models.Problem{
				{Severity: models.SeverityWarning},
			},
			want: "1 problems: 0 fatal, 0 critical, 1 warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlainTextSummary(tt.problems)
			if got != tt.want {
				t.Errorf("PlainTextSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHighestSeverity(t *testing.T) {
	tests := []struct {
		name     string
		problems []*models.Problem
		want     models.Severity
	}{
		{
			name:     "empty",
			problems: nil,
			want:     models.Severity(""),
		},
		{
			name: "fatal wins",
			problems: []*models.Problem{
				{Severity: models.SeverityWarning},
				{Severity: models.SeverityFatal},
				{Severity: models.SeverityCritical},
			},
			want: models.SeverityFatal,
		},
		{
			name: "critical wins over warning",
			problems: []*models.Problem{
				{Severity: models.SeverityWarning},
				{Severity: models.SeverityCritical},
			},
			want: models.SeverityCritical,
		},
		{
			name: "warning only",
			problems: []*models.Problem{
				{Severity: models.SeverityWarning},
			},
			want: models.SeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HighestSeverity(tt.problems)
			if got != tt.want {
				t.Errorf("HighestSeverity() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHumanAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"hours and minutes", 3*time.Hour + 30*time.Minute, "3h30m"},
		{"days", 48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanAge(tt.d)
			if got != tt.want {
				t.Errorf("humanAge(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestPlainText_IncidentGrouping(t *testing.T) {
	now := time.Now()
	problems := []*models.Problem{
		{
			ID:         "p1",
			Severity:   models.SeverityCritical,
			Entity:     "prod/payment-api",
			Title:      "OOMKilled 3 times",
			FirstSeen:  now.Add(-5 * time.Minute),
			Count:      3,
			IncidentID: "memory_pressure/cluster",
		},
		{
			ID:         "p2",
			Severity:   models.SeverityCritical,
			Entity:     "node-1",
			Title:      "High memory usage 94%",
			FirstSeen:  now.Add(-5 * time.Minute),
			Count:      3,
			IncidentID: "memory_pressure/cluster",
		},
		{
			ID:        "p3",
			Severity:  models.SeverityWarning,
			Entity:    "monitoring/prometheus",
			Title:     "Disk usage 91%",
			FirstSeen: now.Add(-30 * time.Second),
			Count:     1,
		},
	}
	got := PlainText(problems, now)

	// Should contain incident header
	if !strings.Contains(got, "--- memory_pressure/cluster (2 problems) ---") {
		t.Error("expected incident header")
	}

	// Uncorrelated problem should appear after incidents
	lines := strings.Split(strings.TrimSpace(got), "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "Disk usage 91%") {
		t.Errorf("uncorrelated problem should be last, got: %s", lastLine)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world this is long", 10, "hello w..."},
		{"very short max", "hello", 2, "he"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
