package detector

import "sync"

// Registry manages detector lifecycle
type Registry struct {
	mu        sync.RWMutex
	detectors map[string]Detector
}

// NewRegistry creates a new detector registry
func NewRegistry() *Registry {
	return &Registry{
		detectors: make(map[string]Detector),
	}
}

// Register adds a detector to the registry
func (r *Registry) Register(d Detector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors[d.Name()] = d
}

// Get retrieves a detector by name
func (r *Registry) Get(name string) (Detector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.detectors[name]
	return d, ok
}

// All returns all registered detectors
func (r *Registry) All() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Detector, 0, len(r.detectors))
	for _, d := range r.detectors {
		list = append(list, d)
	}
	return list
}

// Unregister removes a detector from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.detectors, name)
}

// Count returns the number of registered detectors
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.detectors)
}
