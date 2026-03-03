package correlator

import (
	"sort"

	"github.com/ppiankov/infranow/internal/models"
)

// Rule defines a deterministic correlation pattern. When 2+ distinct
// problem Types from the Types set co-occur with the same match key,
// they form an incident.
type Rule struct {
	Name    string   // incident type, e.g. "memory_pressure"
	Types   []string // problem Types that participate (need 2+ distinct)
	MatchBy string   // label key to group by, or "" for presence-based
}

// DefaultRules are the built-in correlation rules, sorted by Name
// for deterministic evaluation order.
var DefaultRules = []Rule{
	{
		Name:    "deployment_failure",
		Types:   []string{"crashloopbackoff", "imagepullbackoff"},
		MatchBy: "namespace",
	},
	{
		Name:    "istio_cert_cascade",
		Types:   []string{"istio_cert_expiry", "istio_control_plane_down"},
		MatchBy: "mesh",
	},
	{
		Name:    "istio_mesh_failure",
		Types:   []string{"istio_control_plane_down", "istio_component_crash"},
		MatchBy: "mesh",
	},
	{
		Name:    "linkerd_cert_cascade",
		Types:   []string{"linkerd_cert_expiry", "linkerd_control_plane_down"},
		MatchBy: "mesh",
	},
	{
		Name:    "linkerd_mesh_failure",
		Types:   []string{"linkerd_control_plane_down", "linkerd_component_crash"},
		MatchBy: "mesh",
	},
	{
		Name:    "memory_pressure",
		Types:   []string{"oom_kill", "high_memory"},
		MatchBy: "",
	},
}

// Correlate stamps IncidentID, IncidentType, and RelatedIDs on problems
// that match a correlation rule. Problems are modified in-place.
// Same input always produces the same output (deterministic).
func Correlate(problems []*models.Problem) []*models.Problem {
	if len(problems) == 0 {
		return problems
	}

	claimed := make(map[string]bool) // problem ID → already in an incident

	for _, rule := range DefaultRules {
		typeSet := toSet(rule.Types)
		applyRule(rule, problems, typeSet, claimed)
	}

	return problems
}

func applyRule(rule Rule, problems []*models.Problem, typeSet, claimed map[string]bool) {
	// Group matching problems by match key
	groups := make(map[string][]*models.Problem)
	for _, p := range problems {
		if claimed[p.ID] || !typeSet[p.Type] {
			continue
		}
		key := extractMatchKey(p, rule.MatchBy)
		groups[key] = append(groups[key], p)
	}

	// For each group, check if 2+ distinct types are present
	for key, members := range groups {
		if countDistinctTypes(members) < 2 {
			continue
		}

		incidentID := rule.Name + "/" + key

		// Collect all member IDs for RelatedIDs
		ids := make([]string, 0, len(members))
		for _, p := range members {
			ids = append(ids, p.ID)
		}
		sort.Strings(ids) // deterministic order

		// Stamp each member
		for _, p := range members {
			p.IncidentID = incidentID
			p.IncidentType = rule.Name
			p.RelatedIDs = withoutSelf(ids, p.ID)
			claimed[p.ID] = true
		}
	}
}

func extractMatchKey(p *models.Problem, labelKey string) string {
	if labelKey == "" {
		return "cluster" // presence-based: all matching problems in one group
	}
	return p.Labels[labelKey]
}

func countDistinctTypes(problems []*models.Problem) int {
	seen := make(map[string]bool)
	for _, p := range problems {
		seen[p.Type] = true
	}
	return len(seen)
}

func withoutSelf(ids []string, self string) []string {
	result := make([]string, 0, len(ids)-1)
	for _, id := range ids {
		if id != self {
			result = append(result, id)
		}
	}
	return result
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
