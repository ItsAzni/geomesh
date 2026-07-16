package policy_test

import (
	"testing"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
	"github.com/itsazni/geomesh/internal/policy"
)

func TestGeoPolicy_MatchCountry(t *testing.T) {
	route := config.RouteConfig{
		Policy:  "geo",
		Default: "us",
		Regions: []config.RegionConfig{
			{
				Name:      "asia",
				Countries: []string{"ID", "SG", "MY"},
				Endpoints: []config.EndpointConfig{{Address: "103.10.10.10", Weight: 1}},
			},
			{
				Name:      "us",
				Countries: []string{"US", "CA"},
				Endpoints: []config.EndpointConfig{{Address: "45.10.10.10", Weight: 1}},
			},
		},
	}

	p := policy.NewGeoPolicy(route)

	eps, err := p.Select(geo.GeoInfo{CountryCode: "ID", ContinentCode: "AS"}, nil)
	if err != nil {
		t.Fatalf("select error: %v", err)
	}
	if len(eps) == 0 || eps[0].Address != "103.10.10.10" {
		t.Errorf("expected asia endpoint 103.10.10.10, got %v", eps)
	}
}

func TestGeoPolicy_MatchContinent(t *testing.T) {
	route := config.RouteConfig{
		Policy:  "geo",
		Default: "us",
		Regions: []config.RegionConfig{
			{
				Name:       "europe",
				Continents: []string{"EU"},
				Endpoints:  []config.EndpointConfig{{Address: "10.0.0.1", Weight: 1}},
			},
			{
				Name:      "us",
				Countries: []string{"US"},
				Endpoints: []config.EndpointConfig{{Address: "45.10.10.10", Weight: 1}},
			},
		},
	}

	p := policy.NewGeoPolicy(route)

	eps, err := p.Select(geo.GeoInfo{CountryCode: "DE", ContinentCode: "EU"}, nil)
	if err != nil {
		t.Fatalf("select error: %v", err)
	}
	if len(eps) == 0 || eps[0].Address != "10.0.0.1" {
		t.Errorf("expected europe endpoint, got %v", eps)
	}
}

func TestGeoPolicy_FallbackToDefault(t *testing.T) {
	route := config.RouteConfig{
		Policy:  "geo",
		Default: "us",
		Regions: []config.RegionConfig{
			{
				Name:      "us",
				Countries: []string{"US", "CA"},
				Endpoints: []config.EndpointConfig{{Address: "45.10.10.10", Weight: 1}},
			},
		},
	}

	p := policy.NewGeoPolicy(route)

	eps, err := p.Select(geo.GeoInfo{CountryCode: "AU", ContinentCode: "OC"}, nil)
	if err != nil {
		t.Fatalf("select error: %v", err)
	}
	if len(eps) == 0 || eps[0].Address != "45.10.10.10" {
		t.Errorf("expected default us endpoint, got %v", eps)
	}
}

func TestGeoPolicy_HealthFilter(t *testing.T) {
	route := config.RouteConfig{
		Policy:  "geo",
		Default: "asia",
		Regions: []config.RegionConfig{
			{
				Name:      "asia",
				Countries: []string{"ID"},
				Endpoints: []config.EndpointConfig{
					{Address: "1.1.1.1", Weight: 1},
					{Address: "2.2.2.2", Weight: 1},
				},
			},
		},
	}

	p := policy.NewGeoPolicy(route)

	healthyFn := func(addr string) bool { return addr == "2.2.2.2" }

	eps, err := p.Select(geo.GeoInfo{CountryCode: "ID"}, healthyFn)
	if err != nil {
		t.Fatalf("select error: %v", err)
	}
	if len(eps) == 0 || eps[0].Address != "2.2.2.2" {
		t.Errorf("expected only healthy endpoint 2.2.2.2, got %v", eps)
	}
}
