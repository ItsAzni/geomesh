package zone_test

import (
	"testing"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/zone"
)

func TestRegistry_ExactMatch(t *testing.T) {
	r := zone.NewRegistry()
	r.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "play", Policy: "first", Endpoints: []config.EndpointConfig{{Address: "1.1.1.1", Weight: 1}}},
			},
		},
	})

	route, ok := r.Lookup("play.example.com.")
	if !ok {
		t.Fatal("expected to find route for play.example.com.")
	}
	if route.Name != "play" {
		t.Errorf("expected route name 'play', got %q", route.Name)
	}
}

func TestRegistry_NotFound_ReturnsRefused(t *testing.T) {
	r := zone.NewRegistry()
	r.Load([]config.ZoneConfig{})

	_, ok := r.Lookup("unknown.other.com.")
	if ok {
		t.Error("expected not found for zone not in registry")
	}
}

func TestRegistry_Wildcard(t *testing.T) {
	r := zone.NewRegistry()
	r.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "*", Policy: "first", Endpoints: []config.EndpointConfig{{Address: "9.9.9.9", Weight: 1}}},
				{Name: "specific", Policy: "first", Endpoints: []config.EndpointConfig{{Address: "1.1.1.1", Weight: 1}}},
			},
		},
	})

	specific, ok := r.Lookup("specific.example.com.")
	if !ok {
		t.Fatal("expected specific match")
	}
	if specific.Endpoints[0].Address != "1.1.1.1" {
		t.Errorf("expected specific endpoint, got %v", specific.Endpoints[0].Address)
	}

	wild, ok := r.Lookup("anything.example.com.")
	if !ok {
		t.Fatal("expected wildcard to match")
	}
	if wild.Endpoints[0].Address != "9.9.9.9" {
		t.Errorf("expected wildcard endpoint, got %v", wild.Endpoints[0].Address)
	}
}

func TestRegistry_Reload(t *testing.T) {
	r := zone.NewRegistry()
	r.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "v1", Policy: "first", Endpoints: []config.EndpointConfig{{Address: "1.1.1.1", Weight: 1}}},
			},
		},
	})

	r.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "v2", Policy: "first", Endpoints: []config.EndpointConfig{{Address: "2.2.2.2", Weight: 1}}},
			},
		},
	})

	_, found := r.Lookup("v1.example.com.")
	if found {
		t.Error("expected v1 to be absent after reload")
	}
	route, found := r.Lookup("v2.example.com.")
	if !found {
		t.Fatal("expected v2 after reload")
	}
	if route.Endpoints[0].Address != "2.2.2.2" {
		t.Errorf("expected 2.2.2.2, got %v", route.Endpoints[0].Address)
	}
}
