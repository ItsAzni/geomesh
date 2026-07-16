package zone

import (
	"strings"
	"sync"

	"github.com/itsazni/geomesh/internal/config"
)

// Registry stores routes and performs lookups by FQDN (thread-safe).
// It supports exact matches and wildcards (*).
type Registry struct {
	mu        sync.RWMutex
	routes    map[string]config.RouteConfig
	wildcards map[string]config.RouteConfig
}

// NewRegistry initializes a new, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		routes:    make(map[string]config.RouteConfig),
		wildcards: make(map[string]config.RouteConfig),
	}
}

// Load populates the registry with zones, replacing existing data atomically.
// Safe to invoke while the server is operating (config hot reload).
func (r *Registry) Load(zones []config.ZoneConfig) {
	newRoutes := make(map[string]config.RouteConfig)
	newWildcards := make(map[string]config.RouteConfig)

	for _, z := range zones {
		zoneFQDN := toFQDN(z.Name)
		for _, route := range z.Routes {
			if route.Name == "*" {
				newWildcards[zoneFQDN] = route
			} else if route.Name == "@" || route.Name == "" {
				newRoutes[zoneFQDN] = route
			} else {
				fqdn := toFQDN(route.Name + "." + z.Name)
				newRoutes[fqdn] = route
			}
		}
	}

	r.mu.Lock()
	r.routes = newRoutes
	r.wildcards = newWildcards
	r.mu.Unlock()
}

// Lookup queries the route for a given FQDN.
// Priority: exact match > wildcard match.
// Returns (zero, false) if no match is found — DNS handler should return REFUSED.
func (r *Registry) Lookup(fqdn string) (config.RouteConfig, bool) {
	fqdn = toFQDN(fqdn)

	r.mu.RLock()
	defer r.mu.RUnlock()

	if route, ok := r.routes[fqdn]; ok {
		return route, true
	}

	if idx := strings.Index(fqdn, "."); idx >= 0 {
		zoneFQDN := fqdn[idx+1:]
		if route, ok := r.wildcards[zoneFQDN]; ok {
			return route, true
		}
	}

	return config.RouteConfig{}, false
}

// Zones returns a list of all registered zone names.
func (r *Registry) Zones() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for k := range r.routes {
		if idx := strings.Index(k, "."); idx >= 0 {
			seen[k[idx+1:]] = struct{}{}
		}
	}
	for k := range r.wildcards {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func toFQDN(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}
	return name
}
