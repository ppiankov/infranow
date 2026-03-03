package correlator

import (
	"testing"

	"github.com/ppiankov/infranow/internal/models"
)

func TestCorrelate_Empty(t *testing.T) {
	result := Correlate(nil)
	if result != nil {
		t.Errorf("Correlate(nil) = %v, want nil", result)
	}
}

func TestCorrelate_SingleProblem_NoMatch(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "oom_kill", Labels: map[string]string{"namespace": "prod"}},
	}
	Correlate(problems)

	if problems[0].IncidentID != "" {
		t.Errorf("single problem should not form incident, got %q", problems[0].IncidentID)
	}
}

func TestCorrelate_MemoryPressure(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "oom_kill", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p2", Type: "high_memory", Labels: map[string]string{"node": "node-1"}},
	}
	Correlate(problems)

	for _, p := range problems {
		if p.IncidentID != "memory_pressure/cluster" {
			t.Errorf("problem %s: IncidentID = %q, want %q", p.ID, p.IncidentID, "memory_pressure/cluster")
		}
		if p.IncidentType != "memory_pressure" {
			t.Errorf("problem %s: IncidentType = %q, want %q", p.ID, p.IncidentType, "memory_pressure")
		}
	}

	if len(problems[0].RelatedIDs) != 1 || problems[0].RelatedIDs[0] != "p2" {
		t.Errorf("p1 RelatedIDs = %v, want [p2]", problems[0].RelatedIDs)
	}
	if len(problems[1].RelatedIDs) != 1 || problems[1].RelatedIDs[0] != "p1" {
		t.Errorf("p2 RelatedIDs = %v, want [p1]", problems[1].RelatedIDs)
	}
}

func TestCorrelate_DeploymentFailure_SameNamespace(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "crashloopbackoff", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p2", Type: "imagepullbackoff", Labels: map[string]string{"namespace": "prod"}},
	}
	Correlate(problems)

	if problems[0].IncidentID != "deployment_failure/prod" {
		t.Errorf("p1 IncidentID = %q, want %q", problems[0].IncidentID, "deployment_failure/prod")
	}
	if problems[1].IncidentID != "deployment_failure/prod" {
		t.Errorf("p2 IncidentID = %q, want %q", problems[1].IncidentID, "deployment_failure/prod")
	}
}

func TestCorrelate_DeploymentFailure_DifferentNamespaces(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "crashloopbackoff", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p2", Type: "imagepullbackoff", Labels: map[string]string{"namespace": "staging"}},
	}
	Correlate(problems)

	if problems[0].IncidentID != "" {
		t.Errorf("different namespaces should not correlate, p1 got %q", problems[0].IncidentID)
	}
	if problems[1].IncidentID != "" {
		t.Errorf("different namespaces should not correlate, p2 got %q", problems[1].IncidentID)
	}
}

func TestCorrelate_LinkerdMeshFailure(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "linkerd_control_plane_down", Labels: map[string]string{"mesh": "linkerd"}},
		{ID: "p2", Type: "linkerd_component_crash", Labels: map[string]string{"mesh": "linkerd"}},
	}
	Correlate(problems)

	if problems[0].IncidentID != "linkerd_mesh_failure/linkerd" {
		t.Errorf("p1 IncidentID = %q, want %q", problems[0].IncidentID, "linkerd_mesh_failure/linkerd")
	}
	if problems[1].IncidentID != "linkerd_mesh_failure/linkerd" {
		t.Errorf("p2 IncidentID = %q, want %q", problems[1].IncidentID, "linkerd_mesh_failure/linkerd")
	}
}

func TestCorrelate_IstioCertCascade(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "istio_cert_expiry", Labels: map[string]string{"mesh": "istio"}},
		{ID: "p2", Type: "istio_control_plane_down", Labels: map[string]string{"mesh": "istio"}},
	}
	Correlate(problems)

	if problems[0].IncidentID != "istio_cert_cascade/istio" {
		t.Errorf("p1 IncidentID = %q, want %q", problems[0].IncidentID, "istio_cert_cascade/istio")
	}
}

func TestCorrelate_OnlyOneTypePresent(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "oom_kill", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p2", Type: "oom_kill", Labels: map[string]string{"namespace": "staging"}},
	}
	Correlate(problems)

	for _, p := range problems {
		if p.IncidentID != "" {
			t.Errorf("same type only should not form incident, %s got %q", p.ID, p.IncidentID)
		}
	}
}

func TestCorrelate_AlreadyClaimed(t *testing.T) {
	// istio_control_plane_down participates in both istio_cert_cascade and istio_mesh_failure.
	// First rule alphabetically (istio_cert_cascade) should claim it.
	problems := []*models.Problem{
		{ID: "p1", Type: "istio_cert_expiry", Labels: map[string]string{"mesh": "istio"}},
		{ID: "p2", Type: "istio_control_plane_down", Labels: map[string]string{"mesh": "istio"}},
		{ID: "p3", Type: "istio_component_crash", Labels: map[string]string{"mesh": "istio"}},
	}
	Correlate(problems)

	// p1 and p2 should be in istio_cert_cascade
	if problems[0].IncidentType != "istio_cert_cascade" {
		t.Errorf("p1 IncidentType = %q, want istio_cert_cascade", problems[0].IncidentType)
	}
	if problems[1].IncidentType != "istio_cert_cascade" {
		t.Errorf("p2 IncidentType = %q, want istio_cert_cascade", problems[1].IncidentType)
	}

	// p3 should not be in any incident (p2 was claimed, so istio_mesh_failure
	// only has one type left)
	if problems[2].IncidentID != "" {
		t.Errorf("p3 should not be in incident, got %q", problems[2].IncidentID)
	}
}

func TestCorrelate_Deterministic(t *testing.T) {
	makeProblems := func() []*models.Problem {
		return []*models.Problem{
			{ID: "p1", Type: "oom_kill", Labels: map[string]string{"namespace": "prod"}},
			{ID: "p2", Type: "high_memory", Labels: map[string]string{"node": "n1"}},
			{ID: "p3", Type: "crashloopbackoff", Labels: map[string]string{"namespace": "prod"}},
			{ID: "p4", Type: "imagepullbackoff", Labels: map[string]string{"namespace": "prod"}},
		}
	}

	run1 := makeProblems()
	Correlate(run1)
	run2 := makeProblems()
	Correlate(run2)

	for i := range run1 {
		if run1[i].IncidentID != run2[i].IncidentID {
			t.Errorf("non-deterministic: problem %d IncidentID %q vs %q", i, run1[i].IncidentID, run2[i].IncidentID)
		}
	}
}

func TestCorrelate_MultipleMemoryProblems(t *testing.T) {
	// Multiple oom_kills + one high_memory should all correlate
	problems := []*models.Problem{
		{ID: "p1", Type: "oom_kill", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p2", Type: "oom_kill", Labels: map[string]string{"namespace": "staging"}},
		{ID: "p3", Type: "high_memory", Labels: map[string]string{"node": "n1"}},
	}
	Correlate(problems)

	for _, p := range problems {
		if p.IncidentID != "memory_pressure/cluster" {
			t.Errorf("problem %s: IncidentID = %q, want memory_pressure/cluster", p.ID, p.IncidentID)
		}
	}

	// Each problem should have 2 related IDs
	if len(problems[0].RelatedIDs) != 2 {
		t.Errorf("p1 RelatedIDs = %v, want 2 entries", problems[0].RelatedIDs)
	}
}

func TestCorrelate_UnrelatedProblemsUntouched(t *testing.T) {
	problems := []*models.Problem{
		{ID: "p1", Type: "disk_full", Labels: map[string]string{"node": "n1"}},
		{ID: "p2", Type: "pending", Labels: map[string]string{"namespace": "prod"}},
		{ID: "p3", Type: "high_error_rate", Labels: map[string]string{"service": "api"}},
	}
	Correlate(problems)

	for _, p := range problems {
		if p.IncidentID != "" {
			t.Errorf("unrelated problem %s should not correlate, got %q", p.ID, p.IncidentID)
		}
	}
}

func TestExtractMatchKey(t *testing.T) {
	p := &models.Problem{
		Labels: map[string]string{"namespace": "prod", "mesh": "linkerd"},
	}

	tests := []struct {
		labelKey string
		want     string
	}{
		{"namespace", "prod"},
		{"mesh", "linkerd"},
		{"", "cluster"},
		{"missing", ""},
	}

	for _, tt := range tests {
		got := extractMatchKey(p, tt.labelKey)
		if got != tt.want {
			t.Errorf("extractMatchKey(%q) = %q, want %q", tt.labelKey, got, tt.want)
		}
	}
}

func TestWithoutSelf(t *testing.T) {
	ids := []string{"a", "b", "c"}
	got := withoutSelf(ids, "b")
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Errorf("withoutSelf = %v, want [a c]", got)
	}
}
