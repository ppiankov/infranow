package monitor

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/ppiankov/infranow/internal/models"
)

// SARIF 2.1.0 types — minimal subset for valid output.

const sarifSchema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string            `json:"id"`
	ShortDescription sarifMessage      `json:"shortDescription"`
	DefaultConfig    sarifRuleDefaults `json:"defaultConfiguration"`
}

type sarifRuleDefaults struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifLocation struct {
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
}

type sarifLogicalLocation struct {
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
	Kind               string `json:"kind"`
}

var severityToLevel = map[models.Severity]string{
	models.SeverityFatal:    "error",
	models.SeverityCritical: "error",
	models.SeverityWarning:  "warning",
}

// SARIF renders problems as SARIF 2.1.0 JSON for GitHub Code Scanning.
func SARIF(problems []*models.Problem, toolVersion string) ([]byte, error) {
	// Collect unique rules from problem types
	ruleIndex := make(map[string]bool)
	var rules []sarifRule
	for _, p := range problems {
		ruleID := "infranow/" + p.Type
		if !ruleIndex[ruleID] {
			ruleIndex[ruleID] = true
			rules = append(rules, sarifRule{
				ID:               ruleID,
				ShortDescription: sarifMessage{Text: p.Title},
				DefaultConfig:    sarifRuleDefaults{Level: severityToLevel[p.Severity]},
			})
		}
	}
	if rules == nil {
		rules = []sarifRule{}
	}

	// Build results
	results := make([]sarifResult, 0, len(problems))
	for _, p := range problems {
		level := severityToLevel[p.Severity]
		if level == "" {
			level = "note"
		}

		msg := p.Title
		if p.Message != "" {
			msg += " — " + p.Message
		}
		if p.Hint != "" {
			msg += " [hint: " + p.Hint + "]"
		}

		kind := p.EntityType
		if kind == "" {
			kind = "infrastructure/resource"
		}

		r := sarifResult{
			RuleID:  "infranow/" + p.Type,
			Level:   level,
			Message: sarifMessage{Text: msg},
			Locations: []sarifLocation{
				{
					LogicalLocations: []sarifLogicalLocation{
						{
							Name:               entityName(p.Entity),
							FullyQualifiedName: p.Entity,
							Kind:               kind,
						},
					},
				},
			},
		}
		results = append(results, r)
	}

	log := sarifLog{
		Version: "2.1.0",
		Schema:  sarifSchema,
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "infranow",
						Version:        toolVersion,
						InformationURI: "https://github.com/ppiankov/infranow",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	return json.MarshalIndent(log, "", "  ")
}

// entityName returns the last path segment of an entity string.
func entityName(entity string) string {
	name := path.Base(entity)
	if name == "." || name == "/" {
		return entity
	}
	return name
}

// FormatSARIFSummary returns a one-line summary for stderr.
func FormatSARIFSummary(problems []*models.Problem) string {
	ruleCount := sarifRuleCount(problems)
	return fmt.Sprintf("SARIF: %d results, %d rules", len(problems), ruleCount)
}

func sarifRuleCount(problems []*models.Problem) int {
	seen := make(map[string]bool)
	for _, p := range problems {
		seen[p.Type] = true
	}
	return len(seen)
}
