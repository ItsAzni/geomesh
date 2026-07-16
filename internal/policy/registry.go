package policy

import (
	"fmt"

	"github.com/itsazni/geomesh/internal/config"
)

// NewPolicy instantiates the appropriate Policy from a RouteConfig.
func NewPolicy(route config.RouteConfig) (Policy, error) {
	switch route.Policy {
	case "geo":
		return NewGeoPolicy(route), nil
	case "weighted":
		return NewWeightedPolicy(route), nil
	case "roundrobin", "round_robin":
		return NewRoundRobinPolicy(route), nil
	case "random":
		return NewRandomPolicy(route), nil
	case "failover":
		return NewFailoverPolicy(route), nil
	case "first":
		return NewFirstPolicy(route), nil
	case "latency":
		return NewLatencyPolicy(route), nil
	default:
		return nil, fmt.Errorf("unknown policy: %q", route.Policy)
	}
}
