package filter

import (
	"path/filepath"
	"strings"

	"github.com/ppiankov/infranow/internal/models"
)

// NamespaceFilter filters problems by namespace patterns
type NamespaceFilter struct {
	includePatterns []string
	excludePatterns []string
}

// NewNamespaceFilter creates a new namespace filter
func NewNamespaceFilter(include, exclude string) *NamespaceFilter {
	return &NamespaceFilter{
		includePatterns: parsePatterns(include),
		excludePatterns: parsePatterns(exclude),
	}
}

func parsePatterns(s string) []string {
	if s == "" {
		return nil
	}
	patterns := strings.Split(s, ",")
	for i, p := range patterns {
		patterns[i] = strings.TrimSpace(p)
	}
	return patterns
}

// Matches checks if a namespace matches the filter
func (f *NamespaceFilter) Matches(namespace string) bool {
	// If no patterns, match all
	if len(f.includePatterns) == 0 && len(f.excludePatterns) == 0 {
		return true
	}

	// Check exclude first (more restrictive)
	for _, pattern := range f.excludePatterns {
		if matchPattern(pattern, namespace) {
			return false
		}
	}

	// If include patterns specified, must match at least one
	if len(f.includePatterns) > 0 {
		for _, pattern := range f.includePatterns {
			if matchPattern(pattern, namespace) {
				return true
			}
		}
		return false
	}

	return true
}

func matchPattern(pattern, value string) bool {
	matched, _ := filepath.Match(pattern, value)
	return matched
}

// Apply filters a list of problems by namespace
func (f *NamespaceFilter) Apply(problems []*models.Problem) []*models.Problem {
	if len(f.includePatterns) == 0 && len(f.excludePatterns) == 0 {
		return problems
	}

	filtered := make([]*models.Problem, 0)
	for _, p := range problems {
		// Extract namespace from entity (format: "namespace/pod/container")
		parts := strings.Split(p.Entity, "/")
		if len(parts) > 0 {
			namespace := parts[0]
			if f.Matches(namespace) {
				filtered = append(filtered, p)
			}
		}
	}

	return filtered
}
