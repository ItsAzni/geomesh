package policy

import (
	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// FirstPolicy always returns the initial endpoint, ignoring health status.
type FirstPolicy struct {
	route config.RouteConfig
}

func NewFirstPolicy(route config.RouteConfig) *FirstPolicy {
	return &FirstPolicy{route: route}
}

func (f *FirstPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	if len(f.route.Endpoints) == 0 {
		return nil, nil
	}
	return []config.EndpointConfig{f.route.Endpoints[0]}, nil
}
