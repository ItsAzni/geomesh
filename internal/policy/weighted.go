package policy

import (
	"math/rand"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// WeightedPolicy selects an endpoint utilizing a weighted random distribution.
type WeightedPolicy struct {
	route config.RouteConfig
}

func NewWeightedPolicy(route config.RouteConfig) *WeightedPolicy {
	return &WeightedPolicy{route: route}
}

func (w *WeightedPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	eps := healthyFilter(w.route.Endpoints, healthyFn)
	if len(eps) == 0 {
		return nil, nil
	}

	total := 0
	for _, ep := range eps {
		wt := ep.Weight
		if wt <= 0 {
			wt = 1
		}
		total += wt
	}

	r := rand.Intn(total)
	for _, ep := range eps {
		wt := ep.Weight
		if wt <= 0 {
			wt = 1
		}
		r -= wt
		if r < 0 {
			return []config.EndpointConfig{ep}, nil
		}
	}
	return []config.EndpointConfig{eps[0]}, nil
}
