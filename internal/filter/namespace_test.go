package filter

import (
	"testing"

	"github.com/ppiankov/infranow/internal/models"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		name      string
		include   string
		exclude   string
		namespace string
		want      bool
	}{
		{"no patterns matches all", "", "", "default", true},
		{"include match", "prod", "", "prod", true},
		{"include no match", "prod", "", "staging", false},
		{"exclude match", "", "kube-system", "kube-system", false},
		{"exclude no match", "", "kube-system", "prod", true},
		{"include and exclude both match", "prod,staging", "staging", "staging", false},
		{"include and exclude only include matches", "prod,staging", "kube-system", "prod", true},
		{"wildcard include", "prod-*", "", "prod-us", true},
		{"wildcard exclude", "", "kube-*", "kube-system", false},
		{"wildcard no match", "prod-*", "", "staging", false},
		{"empty namespace", "prod", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewNamespaceFilter(tt.include, tt.exclude)
			got := f.Matches(tt.namespace)
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v (include=%q exclude=%q)", tt.namespace, got, tt.want, tt.include, tt.exclude)
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name     string
		include  string
		exclude  string
		entities []string
		wantLen  int
	}{
		{
			name:     "no filter returns all",
			entities: []string{"prod/pod-1", "staging/pod-2"},
			wantLen:  2,
		},
		{
			name:     "include filters",
			include:  "prod",
			entities: []string{"prod/pod-1", "staging/pod-2", "prod/pod-3"},
			wantLen:  2,
		},
		{
			name:     "exclude filters",
			exclude:  "kube-system",
			entities: []string{"prod/pod-1", "kube-system/coredns", "staging/pod-2"},
			wantLen:  2,
		},
		{
			name:     "multi-segment entity",
			include:  "prod",
			entities: []string{"prod/deploy/pod/container", "staging/pod"},
			wantLen:  1,
		},
		{
			name:     "empty entity excluded",
			include:  "prod",
			entities: []string{"", "prod/pod-1"},
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := make([]*models.Problem, len(tt.entities))
			for i, e := range tt.entities {
				problems[i] = &models.Problem{ID: e, Entity: e}
			}

			f := NewNamespaceFilter(tt.include, tt.exclude)
			result := f.Apply(problems)
			if len(result) != tt.wantLen {
				t.Errorf("Apply() returned %d problems, want %d", len(result), tt.wantLen)
			}
		})
	}
}
