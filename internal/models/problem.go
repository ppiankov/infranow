package models

import (
	"fmt"
	"strings"
	"time"
)

// Severity represents the urgency level of a problem
type Severity string

const (
	SeverityFatal    Severity = "FATAL"    // Service down, data loss
	SeverityCritical Severity = "CRITICAL" // Degraded performance, risk of failure
	SeverityWarning  Severity = "WARNING"  // Anomaly detected, no immediate impact
)

// Scoring weights for problem importance ranking
const (
	scoreFatal    = 100.0
	scoreCritical = 50.0
	scoreWarning  = 10.0

	// Per-unit blast radius weight applied to base score
	blastRadiusWeight = 0.1

	// Persistence is normalized to hours for scoring
	secondsPerHour = 3600.0
)

// Problem represents a unified infrastructure issue
type Problem struct {
	// Identity
	ID         string // Unique identifier (entity + type hash)
	Entity     string // What: "namespace/deployment/pod", "kafka/broker-1", "postgres/primary"
	EntityType string // Kind: "kubernetes_pod", "kafka_broker", "database"
	Type       string // Issue type: "high_error_rate", "disk_full", "replication_lag"

	// Classification
	Severity Severity
	Title    string // Short description
	Message  string // Detailed message

	// Temporal
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int // How many times detected

	// Impact
	BlastRadius int     // Estimated affected entities
	Persistence float64 // Duration in seconds
	Volatility  float64 // Rate of change (problems/minute)

	// Context
	Labels  map[string]string  // source, namespace, cluster, etc.
	Metrics map[string]float64 // Raw metric values for evidence
	Hint    string             // One-line actionable guidance
}

// Score calculates problem importance for ranking
func (p *Problem) Score() float64 {
	severityWeight := map[Severity]float64{
		SeverityFatal:    scoreFatal,
		SeverityCritical: scoreCritical,
		SeverityWarning:  scoreWarning,
	}

	base := severityWeight[p.Severity]
	blastRadiusMultiplier := 1.0 + (float64(p.BlastRadius) * blastRadiusWeight)
	persistenceMultiplier := 1.0 + (p.Persistence / secondsPerHour)

	return base * blastRadiusMultiplier * persistenceMultiplier
}

// UpdatePersistence calculates the persistence duration based on first and last seen times
func (p *Problem) UpdatePersistence() {
	p.Persistence = p.LastSeen.Sub(p.FirstSeen).Seconds()
}

// AtLeast checks if this severity is at least as severe as the threshold
func (s Severity) AtLeast(threshold Severity) bool {
	order := map[Severity]int{
		SeverityWarning:  1,
		SeverityCritical: 2,
		SeverityFatal:    3,
	}
	return order[s] >= order[threshold]
}

// ParseSeverity parses a severity string (case-insensitive)
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToUpper(s) {
	case "WARNING":
		return SeverityWarning, nil
	case "CRITICAL":
		return SeverityCritical, nil
	case "FATAL":
		return SeverityFatal, nil
	default:
		return "", fmt.Errorf("invalid severity: %s (must be WARNING, CRITICAL, or FATAL)", s)
	}
}
