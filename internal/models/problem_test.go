package models

import (
	"testing"
	"time"
)

func TestProblemScore(t *testing.T) {
	tests := []struct {
		name        string
		severity    Severity
		blastRadius int
		persistence float64
		minScore    float64
	}{
		{
			name:        "fatal with high blast radius",
			severity:    SeverityFatal,
			blastRadius: 10,
			persistence: 3600, // 1 hour
			minScore:    200,  // 100 * 2.0 * 2.0
		},
		{
			name:        "critical with no blast radius",
			severity:    SeverityCritical,
			blastRadius: 0,
			persistence: 0,
			minScore:    50, // 50 * 1.0 * 1.0
		},
		{
			name:        "warning",
			severity:    SeverityWarning,
			blastRadius: 0,
			persistence: 0,
			minScore:    10, // 10 * 1.0 * 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Problem{
				Severity:    tt.severity,
				BlastRadius: tt.blastRadius,
				Persistence: tt.persistence,
			}

			score := p.Score()
			if score < tt.minScore {
				t.Errorf("expected score >= %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestUpdatePersistence(t *testing.T) {
	firstSeen := time.Now().Add(-5 * time.Minute)
	lastSeen := time.Now()

	p := &Problem{
		FirstSeen: firstSeen,
		LastSeen:  lastSeen,
	}

	p.UpdatePersistence()

	expectedSeconds := lastSeen.Sub(firstSeen).Seconds()
	if p.Persistence != expectedSeconds {
		t.Errorf("expected persistence %.2f seconds, got %.2f", expectedSeconds, p.Persistence)
	}
}

func TestSeverityOrdering(t *testing.T) {
	fatal := &Problem{Severity: SeverityFatal}
	critical := &Problem{Severity: SeverityCritical}
	warning := &Problem{Severity: SeverityWarning}

	if fatal.Score() <= critical.Score() {
		t.Error("fatal should score higher than critical")
	}

	if critical.Score() <= warning.Score() {
		t.Error("critical should score higher than warning")
	}
}
