package policy

import (
	"errors"
	"math"
	"sort"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// LatencyPolicy routes based on distance using Haversine formula
type LatencyPolicy struct {
	route config.RouteConfig
}

func NewLatencyPolicy(route config.RouteConfig) *LatencyPolicy {
	return &LatencyPolicy{route: route}
}

func (l *LatencyPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {
	if geoInfo.IsEmpty() || (geoInfo.Latitude == 0 && geoInfo.Longitude == 0) {

		eps := healthyFilter(l.route.Endpoints, healthyFn)
		if len(eps) > 0 {
			return eps, nil
		}
		return nil, errors.New("latency: no endpoints available")
	}

	eps := healthyFilter(l.route.Endpoints, healthyFn)
	if len(eps) == 0 {
		return nil, errors.New("latency: no endpoints available")
	}

	type endpointDistance struct {
		ep       config.EndpointConfig
		distance float64
	}

	var distances []endpointDistance
	for _, ep := range eps {
		dist := haversine(geoInfo.Latitude, geoInfo.Longitude, ep.Latitude, ep.Longitude)
		distances = append(distances, endpointDistance{ep: ep, distance: dist})
	}

	sort.Slice(distances, func(i, j int) bool {
		return distances[i].distance < distances[j].distance
	})

	return []config.EndpointConfig{distances[0].ep}, nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1 = lat1 * math.Pi / 180.0
	lat2 = lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
