package policy

import (
	"errors"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
)

// GeoPolicy routes based on geolocation: country > continent > ASN.
type GeoPolicy struct {
	route config.RouteConfig
}

func NewGeoPolicy(route config.RouteConfig) *GeoPolicy {
	return &GeoPolicy{route: route}
}

func (g *GeoPolicy) Select(geoInfo geo.GeoInfo, healthyFn func(addr string) bool) ([]config.EndpointConfig, error) {

	for _, region := range g.route.Regions {
		if matchRegion(region, geoInfo) {
			eps := healthyFilter(region.Endpoints, healthyFn)
			if len(eps) > 0 {
				return eps, nil
			}
		}
	}

	if g.route.Default != "" {
		for _, region := range g.route.Regions {
			if region.Name == g.route.Default {
				return healthyFilter(region.Endpoints, healthyFn), nil
			}
		}
	}

	return nil, errors.New("geo: no matching region and no default configured")
}

// matchRegion verifies if the GeoInfo satisfies any of the region's criteria.
func matchRegion(region config.RegionConfig, info geo.GeoInfo) bool {

	for _, c := range region.Countries {
		if c == info.CountryCode {
			return true
		}
	}

	for _, cont := range region.Continents {
		if cont == info.ContinentCode {
			return true
		}
	}

	for _, asn := range region.ASNs {
		if info.ASN != 0 && asn == info.ASN {
			return true
		}
	}
	return false
}
