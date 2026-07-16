package policy

import (
	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// Policy dictates how an endpoint is selected for a given DNS query.
type Policy interface {
	Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error)
}

// healthyFilter filters endpoints based on their health status.
// If all are unhealthy or healthyFn is nil, returns all endpoints (fail-open behavior).
func healthyFilter(endpoints []config.EndpointConfig, healthyFn func(addr string) bool) []config.EndpointConfig {
	if healthyFn == nil {
		return endpoints
	}
	var healthy []config.EndpointConfig
	for _, ep := range endpoints {
		if healthyFn(ep.Address) {
			healthy = append(healthy, ep)
		}
	}
	if len(healthy) == 0 {
		return endpoints
	}
	return healthy
}
