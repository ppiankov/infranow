package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeContext holds a resolved kubeconfig context with its REST config and clientset.
type KubeContext struct {
	Name       string
	RestConfig *rest.Config
	Clientset  *kubernetes.Clientset
}

// ListContexts returns sorted context names from the kubeconfig file.
// Path resolution: explicit arg → KUBECONFIG env → ~/.kube/config.
func ListContexts(kubeconfigPath string) ([]string, error) {
	kubeconfig := resolveKubeconfigPath(kubeconfigPath)
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	contexts := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	sort.Strings(contexts)
	return contexts, nil
}

// MatchContexts filters context names against comma-separated glob patterns.
// Empty pattern returns all contexts unchanged.
func MatchContexts(contexts []string, patterns string) []string {
	if patterns == "" {
		return contexts
	}
	pats := strings.Split(patterns, ",")
	for i := range pats {
		pats[i] = strings.TrimSpace(pats[i])
	}
	var matched []string
	for _, ctx := range contexts {
		for _, pat := range pats {
			ok, err := filepath.Match(pat, ctx)
			if err != nil {
				continue // Invalid pattern, skip
			}
			if ok {
				matched = append(matched, ctx)
				break
			}
		}
	}
	return matched
}

// NewKubeContext loads a specific context from kubeconfig and returns
// the REST config and Clientset bound to that context.
func NewKubeContext(kubeconfigPath, contextName string) (*KubeContext, error) {
	kubeconfig := resolveKubeconfigPath(kubeconfigPath)

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config for context %q: %w", contextName, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset for context %q: %w", contextName, err)
	}

	return &KubeContext{
		Name:       contextName,
		RestConfig: config,
		Clientset:  clientset,
	}, nil
}

// resolveKubeconfigPath returns the kubeconfig path using standard resolution order.
func resolveKubeconfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".kube", "config")
	}
	return filepath.Join(home, ".kube", "config")
}
