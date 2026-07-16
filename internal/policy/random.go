package policy

import (
	"math/rand"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// RandomPolicy selects an endpoint randomly (uniform distribution).
type RandomPolicy struct {
	route config.RouteConfig
}

func NewRandomPolicy(route config.RouteConfig) *RandomPolicy {
	return &RandomPolicy{route: route}
}

func (r *RandomPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	eps := healthyFilter(r.route.Endpoints, healthyFn)
	if len(eps) == 0 {
		return nil, nil
	}
	return []config.EndpointConfig{eps[rand.Intn(len(eps))]}, nil
}
