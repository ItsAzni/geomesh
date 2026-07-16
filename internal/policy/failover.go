package policy

import (
	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// FailoverPolicy returns the first healthy endpoint.
// The sequence of endpoints in the configuration dictates the failover priority.
type FailoverPolicy struct {
	route config.RouteConfig
}

func NewFailoverPolicy(route config.RouteConfig) *FailoverPolicy {
	return &FailoverPolicy{route: route}
}

func (f *FailoverPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	if healthyFn != nil {
		for _, ep := range f.route.Endpoints {
			if healthyFn(ep.Address) {
				return []config.EndpointConfig{ep}, nil
			}
		}
	}

	if len(f.route.Endpoints) > 0 {
		return []config.EndpointConfig{f.route.Endpoints[0]}, nil
	}
	return nil, nil
}
