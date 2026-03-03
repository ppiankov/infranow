package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListContexts(t *testing.T) {
	// Write a minimal kubeconfig with known contexts
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "config")
	data := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://prod-us.example.com
  name: prod-us
- cluster:
    server: https://prod-eu.example.com
  name: prod-eu
- cluster:
    server: https://staging.example.com
  name: staging
contexts:
- context:
    cluster: prod-us
    user: admin
  name: prod-us-east
- context:
    cluster: prod-eu
    user: admin
  name: prod-eu-west
- context:
    cluster: staging
    user: dev
  name: staging
current-context: prod-us-east
users:
- name: admin
  user: {}
- name: dev
  user: {}
`
	if err := os.WriteFile(kubeconfig, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	contexts, err := ListContexts(kubeconfig)
	if err != nil {
		t.Fatalf("ListContexts: %v", err)
	}

	// Should be sorted alphabetically
	expected := []string{"prod-eu-west", "prod-us-east", "staging"}
	if len(contexts) != len(expected) {
		t.Fatalf("got %d contexts, want %d", len(contexts), len(expected))
	}
	for i, ctx := range contexts {
		if ctx != expected[i] {
			t.Errorf("context[%d] = %q, want %q", i, ctx, expected[i])
		}
	}
}

func TestListContexts_FileNotFound(t *testing.T) {
	_, err := ListContexts("/nonexistent/path/kubeconfig")
	if err == nil {
		t.Fatal("expected error for missing kubeconfig")
	}
}

func TestMatchContexts(t *testing.T) {
	contexts := []string{"prod-eu-west", "prod-us-east", "staging", "dev-local"}

	tests := []struct {
		name     string
		patterns string
		want     []string
	}{
		{"empty pattern returns all", "", []string{"prod-eu-west", "prod-us-east", "staging", "dev-local"}},
		{"glob prefix", "prod-*", []string{"prod-eu-west", "prod-us-east"}},
		{"glob suffix", "*-east", []string{"prod-us-east"}},
		{"exact match", "staging", []string{"staging"}},
		{"no matches", "nonexistent-*", nil},
		{"multiple patterns", "prod-*,staging", []string{"prod-eu-west", "prod-us-east", "staging"}},
		{"pattern with spaces", "prod-* , staging", []string{"prod-eu-west", "prod-us-east", "staging"}},
		{"wildcard all", "*", []string{"prod-eu-west", "prod-us-east", "staging", "dev-local"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchContexts(contexts, tt.patterns)
			if len(got) != len(tt.want) {
				t.Fatalf("MatchContexts(%q) = %v (len %d), want %v (len %d)", tt.patterns, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("result[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolveKubeconfigPath(t *testing.T) {
	t.Run("explicit path wins", func(t *testing.T) {
		got := resolveKubeconfigPath("/custom/kubeconfig")
		if got != "/custom/kubeconfig" {
			t.Errorf("got %q, want /custom/kubeconfig", got)
		}
	})

	t.Run("KUBECONFIG env fallback", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "/env/kubeconfig")
		got := resolveKubeconfigPath("")
		if got != "/env/kubeconfig" {
			t.Errorf("got %q, want /env/kubeconfig", got)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "")
		got := resolveKubeconfigPath("")
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".kube", "config")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
