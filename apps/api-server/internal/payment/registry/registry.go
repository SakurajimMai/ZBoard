// Package payment/registry provides a simple map-based registry of payment
// providers, keyed by name. The server initializes providers from config and
// registers them here; handlers look up by name at runtime.
package registry

import (
	"fmt"
	"sync"

	"github.com/zboard/api-server/internal/payment"
)

// Registry holds all configured payment providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]payment.Provider
}

func New() *Registry {
	return &Registry{providers: make(map[string]payment.Provider)}
}

func (r *Registry) Register(p payment.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (payment.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("payment provider %q not registered", name)
	}
	return p, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}
