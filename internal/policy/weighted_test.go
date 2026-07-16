package policy_test

import (
	"testing"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/geo"
	"github.com/itsazni/geomesh/internal/policy"
)

func TestWeightedPolicy_Distribution(t *testing.T) {
	route := config.RouteConfig{
		Policy: "weighted",
		Endpoints: []config.EndpointConfig{
			{Address: "1.1.1.1", Weight: 3},
			{Address: "2.2.2.2", Weight: 1},
		},
	}

	p := policy.NewWeightedPolicy(route)
	counts := map[string]int{}
	const N = 4000

	for i := 0; i < N; i++ {
		eps, err := p.Select(geo.GeoInfo{}, nil)
		if err != nil || len(eps) == 0 {
			t.Fatalf("select failed at iteration %d: %v", i, err)
		}
		counts[eps[0].Address]++
	}

	ratio := float64(counts["1.1.1.1"]) / float64(N)
	if ratio < 0.65 || ratio > 0.85 {
		t.Errorf("expected ~75%% for weight=3, got %.1f%%", ratio*100)
	}
}
