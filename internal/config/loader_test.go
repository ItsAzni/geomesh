package config_test

import (
	"os"
	"path/filepath"
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

func TestLoadServerConfigSeparate(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Flat format
	flatPath := filepath.Join(tmpDir, "server_flat.yaml")
	flatContent := `
listen: ":5353"
api: ":9090"
log_level: debug
log_json: true
`
	if err := os.WriteFile(flatPath, []byte(flatContent), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := config.LoadServerConfig(flatPath)
	if err != nil {
		t.Fatalf("failed to load flat server config: %v", err)
	}
	if srv.Listen != ":5353" || srv.API != ":9090" || srv.LogLevel != "debug" || !srv.LogJSON {
		t.Errorf("unexpected flat srv values: %+v", srv)
	}

	// 2. Nested under server:
	nestedPath := filepath.Join(tmpDir, "server_nested.yaml")
	nestedContent := `
server:
  listen: ":1053"
  api: ""
  log_level: warn
  log_json: false
`
	if err := os.WriteFile(nestedPath, []byte(nestedContent), 0644); err != nil {
		t.Fatal(err)
	}
	srv2, err := config.LoadServerConfig(nestedPath)
	if err != nil {
		t.Fatalf("failed to load nested server config: %v", err)
	}
	if srv2.Listen != ":1053" || srv2.API != "" || srv2.LogLevel != "warn" || srv2.LogJSON {
		t.Errorf("unexpected nested srv values: %+v", srv2)
	}
}

func TestDirectoryConfigForbidsServer(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "zone_with_server.yaml")
	content := `
server:
  listen: ":53"
zones:
  - name: forbidden.local
    routes:
      - name: www
        policy: first
        endpoints:
          - address: 1.1.1.1
`
	if err := os.WriteFile(badFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadZonesConfig(tmpDir)
	if err == nil {
		t.Fatal("expected error when server section is present in directory config, got nil")
	}
	if !strings.Contains(err.Error(), "server configuration is not allowed in config directory") {
		t.Errorf("expected server prohibition error, got: %v", err)
	}
}

func TestDirectoryConfigZonesAndGeoIP(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. GeoIP config
	geoipPath := filepath.Join(tmpDir, "geoip.yaml")
	geoipContent := `
geoip:
  city_db: /path/to/city.mmdb
  asn_db: /path/to/asn.mmdb
`
	if err := os.WriteFile(geoipPath, []byte(geoipContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Zone A
	zoneAPath := filepath.Join(tmpDir, "zone_a.yaml")
	zoneAContent := `
zones:
  - name: a.example.com
    routes:
      - name: app
        policy: first
        endpoints:
          - address: 10.0.0.1
`
	if err := os.WriteFile(zoneAPath, []byte(zoneAContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Zone B
	zoneBPath := filepath.Join(tmpDir, "zone_b.yaml")
	zoneBContent := `
zones:
  - name: b.example.com
    routes:
      - name: app
        policy: first
        endpoints:
          - address: 10.0.0.2
`
	if err := os.WriteFile(zoneBPath, []byte(zoneBContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadZonesConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load zones config directory: %v", err)
	}

	if cfg.GeoIP.CityDB != "/path/to/city.mmdb" || cfg.GeoIP.ASNDB != "/path/to/asn.mmdb" {
		t.Errorf("geoip not loaded properly: %+v", cfg.GeoIP)
	}
	if len(cfg.Zones) != 2 {
		t.Fatalf("expected 2 zones merged, got %d", len(cfg.Zones))
	}
	if cfg.Zones[0].Name != "a.example.com" && cfg.Zones[1].Name != "a.example.com" {
		t.Errorf("zone a missing")
	}
}

func TestLoadServerAndZonesCombined(t *testing.T) {
	tmpDir := t.TempDir()

	serverPath := filepath.Join(tmpDir, "server.yaml")
	serverContent := `
listen: ":5354"
api: ":8081"
log_level: warn
log_json: true
`
	if err := os.WriteFile(serverPath, []byte(serverContent), 0644); err != nil {
		t.Fatal(err)
	}

	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	zonePath := filepath.Join(configDir, "example.com.yaml")
	zoneContent := `
zones:
  - name: example.com
    routes:
      - name: test
        policy: first
        endpoints:
          - address: 1.1.1.1
`
	if err := os.WriteFile(zonePath, []byte(zoneContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(serverPath, configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Listen != ":5354" || cfg.Server.API != ":8081" || cfg.Server.LogLevel != "warn" || !cfg.Server.LogJSON {
		t.Errorf("unexpected server config: %+v", cfg.Server)
	}
	if len(cfg.Zones) != 1 || cfg.Zones[0].Name != "example.com" {
		t.Errorf("unexpected zones: %+v", cfg.Zones)
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
