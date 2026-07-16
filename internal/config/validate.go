package config

import (
	"fmt"
	"net"
	"strings"
)

var validPolicies = map[string]bool{
	"geo": true, "weighted": true, "roundrobin": true,
	"random": true, "failover": true, "first": true, "latency": true,
}

var validHealthTypes = map[string]bool{
	"tcp": true, "http": true, "https": true, "mcjava": true, "mcbedrock": true, "udp": true,
}

// Validate performs semantic consistency checks on the configuration.
// Should be called after LoadFile/LoadReader.
func Validate(cfg *Config) error {
	if cfg.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	for _, z := range cfg.Zones {
		if z.Name == "" {
			return fmt.Errorf("zone name is required")
		}
		for _, r := range z.Routes {
			if err := validateRoute(z.Name, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRoute(zoneName string, r RouteConfig) error {
	prefix := fmt.Sprintf("zone %q route %q", zoneName, r.Name)

	if !validPolicies[r.Policy] {
		return fmt.Errorf("%s: unknown policy %q (valid: geo, weighted, roundrobin, random, failover, first, latency)", prefix, r.Policy)
	}

	if r.Policy == "geo" && len(r.Regions) == 0 {
		return fmt.Errorf("%s: geo policy requires at least one region", prefix)
	}

	if r.Policy != "geo" && len(r.Endpoints) == 0 {
		return fmt.Errorf("%s: policy %q requires at least one endpoint", prefix, r.Policy)
	}

	allEndpoints := append([]EndpointConfig{}, r.Endpoints...)
	for _, reg := range r.Regions {
		allEndpoints = append(allEndpoints, reg.Endpoints...)
	}
	for _, ep := range allEndpoints {
		if ep.Address == "" {
			return fmt.Errorf("%s: endpoint address cannot be empty", prefix)
		}
		if net.ParseIP(ep.Address) == nil && !isValidHostname(ep.Address) {
			return fmt.Errorf("%s: invalid endpoint address %q", prefix, ep.Address)
		}
		if ep.HealthCheck != nil {
			if err := validateHealthCheck(prefix, ep.HealthCheck); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateHealthCheck(prefix string, hc *HealthCheckConfig) error {
	if !validHealthTypes[hc.Type] {
		return fmt.Errorf("%s: unknown health check type %q (valid: tcp, http, https, mcjava, mcbedrock, udp)", prefix, hc.Type)
	}
	if hc.Type != "udp" && hc.Type != "mcjava" && hc.Type != "mcbedrock" && hc.Type != "tcp" && hc.Type != "http" && hc.Type != "https" {
		return fmt.Errorf("%s: invalid health check type", prefix)
	}
	if hc.Port <= 0 || hc.Port > 65535 {
		return fmt.Errorf("%s: health check port %d out of range", prefix, hc.Port)
	}
	return nil
}

func isValidHostname(s string) bool {
	s = strings.TrimSuffix(s, ".")
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
	}
	return true
}
