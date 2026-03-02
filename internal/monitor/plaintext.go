package monitor

import (
	"fmt"
	"strings"
	"time"

	"github.com/ppiankov/infranow/internal/models"
)

const noProblemsMessage = "No problems detected."

// PlainText renders problems as a fixed-width text table suitable for
// piped output and CI logs. No ANSI colors or escape sequences.
func PlainText(problems []*models.Problem, now time.Time) string {
	if len(problems) == 0 {
		return noProblemsMessage
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-8s %-30s %-40s %-10s %s\n", "SEV", "ENTITY", "TITLE", "AGE", "COUNT")
	fmt.Fprintf(&b, "%-8s %-30s %-40s %-10s %s\n", "---", "------", "-----", "---", "-----")
	for _, p := range problems {
		sev := shortSeverity(p.Severity)
		entity := truncate(p.Entity, 30)
		title := truncate(p.Title, 40)
		age := humanAge(now.Sub(p.FirstSeen))
		fmt.Fprintf(&b, "%-8s %-30s %-40s %-10s %d\n", sev, entity, title, age, p.Count)
	}
	return b.String()
}

// PlainTextSummary returns a one-line summary of problem counts by severity.
func PlainTextSummary(problems []*models.Problem) string {
	if len(problems) == 0 {
		return noProblemsMessage
	}

	var fatal, critical, warning int
	for _, p := range problems {
		switch p.Severity {
		case models.SeverityFatal:
			fatal++
		case models.SeverityCritical:
			critical++
		case models.SeverityWarning:
			warning++
		}
	}
	return fmt.Sprintf("%d problems: %d fatal, %d critical, %d warning",
		len(problems), fatal, critical, warning)
}

// HighestSeverity returns the highest severity among problems.
// Returns empty string if no problems.
func HighestSeverity(problems []*models.Problem) models.Severity {
	highest := models.Severity("")
	order := map[models.Severity]int{
		models.SeverityWarning:  1,
		models.SeverityCritical: 2,
		models.SeverityFatal:    3,
	}
	for _, p := range problems {
		if order[p.Severity] > order[highest] {
			highest = p.Severity
		}
	}
	return highest
}

func shortSeverity(s models.Severity) string {
	switch s {
	case models.SeverityFatal:
		return "FATAL"
	case models.SeverityCritical:
		return "CRIT"
	case models.SeverityWarning:
		return "WARN"
	default:
		return string(s)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
