package policy_test

import (
	"testing"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
	"github.com/itsazni/geomesh/internal/policy"
)

func TestRoundRobinPolicy_Rotation(t *testing.T) {
	route := config.RouteConfig{
		Policy: "roundrobin",
		Endpoints: []config.EndpointConfig{
			{Address: "1.1.1.1", Weight: 1},
			{Address: "2.2.2.2", Weight: 1},
			{Address: "3.3.3.3", Weight: 1},
		},
	}

	p := policy.NewRoundRobinPolicy(route)
	seen := []string{}
	for i := 0; i < 6; i++ {
		eps, err := p.Select(geo.GeoInfo{}, nil)
		if err != nil || len(eps) == 0 {
			t.Fatalf("select error at step %d: %v", i, err)
		}
		seen = append(seen, eps[0].Address)
	}

	expected := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "1.1.1.1", "2.2.2.2", "3.3.3.3"}
	for i, e := range expected {
		if seen[i] != e {
			t.Errorf("step %d: expected %q, got %q", i, e, seen[i])
		}
	}
}

func TestRoundRobinPolicy_ConcurrentSafe(t *testing.T) {
	route := config.RouteConfig{
		Policy: "roundrobin",
		Endpoints: []config.EndpointConfig{
			{Address: "1.1.1.1", Weight: 1},
			{Address: "2.2.2.2", Weight: 1},
		},
	}

	p := policy.NewRoundRobinPolicy(route)
	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func() {
			p.Select(geo.GeoInfo{}, nil)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}

}
