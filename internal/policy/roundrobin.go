package policy

import (
	"sync/atomic"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// RoundRobinPolicy rotates through endpoints sequentially (lock-free via atomic operations).
type RoundRobinPolicy struct {
	route   config.RouteConfig
	counter atomic.Uint64
}

func NewRoundRobinPolicy(route config.RouteConfig) *RoundRobinPolicy {
	return &RoundRobinPolicy{route: route}
}

func (r *RoundRobinPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	eps := healthyFilter(r.route.Endpoints, healthyFn)
	if len(eps) == 0 {
		return nil, nil
	}
	idx := int(r.counter.Add(1)-1) % len(eps)
	return []config.EndpointConfig{eps[idx]}, nil
}
