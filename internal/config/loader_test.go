package config_test

import (
	"strings"
	"testing"

	"github.com/itsazni/geomesh/internal/config"
)

const sampleYAML = `
server:
  listen: ":53"
  api: ":8080"
  log_level: info
  log_json: false

geoip:
  city_db: /app/geoip/GeoLite2-City.mmdb
  asn_db: /app/geoip/GeoLite2-ASN.mmdb

zones:
  - name: example.com
    routes:
      - name: play
        policy: geo
        default: us
        ttl: 60
        type: A
        regions:
          - name: asia
            countries: [ID, SG, MY, TH, VN, PH, JP, KR, HK, TW]
            endpoints:
              - address: 103.10.10.10
                weight: 1
              - address: 103.10.10.11
                weight: 2
            health_check:
              type: tcp
              port: 80
              interval: 30
              timeout: 5
              retries: 3
          - name: us
            countries: [US, CA, MX]
            endpoints:
              - address: 45.10.10.10
                weight: 1
            health_check:
              type: http
              port: 80
              path: /health
              interval: 30
              timeout: 5
              retries: 3
      - name: www
        policy: roundrobin
        ttl: 300
        endpoints:
          - address: 1.2.3.4
          - address: 5.6.7.8
`

func TestLoadServerConfig(t *testing.T) {
	cfg, err := config.LoadReader(strings.NewReader(sampleYAML))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.Server.Listen != ":53" {
		t.Errorf("expected listen :53, got %q", cfg.Server.Listen)
	}
	if cfg.Server.API != ":8080" {
		t.Errorf("expected api :8080, got %q", cfg.Server.API)
	}
}

func TestLoadZones(t *testing.T) {
	cfg, err := config.LoadReader(strings.NewReader(sampleYAML))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(cfg.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(cfg.Zones))
	}
	if cfg.Zones[0].Name != "example.com" {
		t.Errorf("expected zone example.com, got %q", cfg.Zones[0].Name)
	}
	if len(cfg.Zones[0].Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(cfg.Zones[0].Routes))
	}
}

func TestLoadGeoRoute(t *testing.T) {
	cfg, err := config.LoadReader(strings.NewReader(sampleYAML))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	route := cfg.Zones[0].Routes[0]
	if route.Name != "play" {
		t.Errorf("expected route 'play', got %q", route.Name)
	}
	if route.Policy != "geo" {
		t.Errorf("expected policy 'geo', got %q", route.Policy)
	}
	if route.Default != "us" {
		t.Errorf("expected default 'us', got %q", route.Default)
	}
	if len(route.Regions) != 2 {
		t.Errorf("expected 2 regions, got %d", len(route.Regions))
	}
	asia := route.Regions[0]
	if asia.Name != "asia" {
		t.Errorf("expected region 'asia', got %q", asia.Name)
	}
	if len(asia.Countries) != 10 {
		t.Errorf("expected 10 countries, got %d", len(asia.Countries))
	}
	if len(asia.Endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(asia.Endpoints))
	}
	if asia.Endpoints[1].Weight != 2 {
		t.Errorf("expected weight=2, got %d", asia.Endpoints[1].Weight)
	}
	if asia.HealthCheck == nil {
		t.Fatal("expected health check config")
	}
	if asia.HealthCheck.Type != "tcp" {
		t.Errorf("expected tcp health check, got %q", asia.HealthCheck.Type)
	}
}

func TestLoadDefaults(t *testing.T) {
	minimal := `
server:
  listen: ":53"
zones:
  - name: test.local
    routes:
      - name: www
        policy: first
        endpoints:
          - address: 1.1.1.1
`
	cfg, err := config.LoadReader(strings.NewReader(minimal))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	route := cfg.Zones[0].Routes[0]

	if route.TTL != 60 {
		t.Errorf("expected default TTL 60, got %d", route.TTL)
	}

	if route.RecordType != "A" {
		t.Errorf("expected default type A, got %q", route.RecordType)
	}
}

func TestValidateUnknownPolicy(t *testing.T) {
	bad := `
server:
  listen: ":53"
zones:
  - name: test.local
    routes:
      - name: www
        policy: invalid_policy
        endpoints:
          - address: 1.1.1.1
`
	cfg, err := config.LoadReader(strings.NewReader(bad))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if err := config.Validate(cfg); err == nil {
		t.Error("expected validation error for unknown policy")
	}
}
