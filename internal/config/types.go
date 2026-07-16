package config

// Config is the root configuration structure for GeoMesh.
type Config struct {
	Server ServerConfig `yaml:"server"`
	GeoIP  GeoIPConfig  `yaml:"geoip"`
	Zones  []ZoneConfig `yaml:"zones"`
}

// ServerConfig defines DNS server and API settings.
type ServerConfig struct {
	Listen   string `yaml:"listen"`
	API      string `yaml:"api"`
	LogJSON  bool   `yaml:"log_json"`
	LogLevel string `yaml:"log_level"`
}

// GeoIPConfig defines paths to MaxMind GeoLite2 databases.
type GeoIPConfig struct {
	CityDB string `yaml:"city_db"`
	ASNDB  string `yaml:"asn_db"`
}

// ZoneConfig represents a single DNS zone.
type ZoneConfig struct {
	Name   string        `yaml:"name"`
	Routes []RouteConfig `yaml:"routes"`
}

// RouteConfig represents a specific hostname route within a zone.
type RouteConfig struct {
	Name        string             `yaml:"name"`
	Policy      string             `yaml:"policy"`
	Default     string             `yaml:"default"`
	TTL         uint32             `yaml:"ttl"`
	RecordType  string             `yaml:"type"`
	Regions     []RegionConfig     `yaml:"regions"`
	Endpoints   []EndpointConfig   `yaml:"endpoints"`
	HealthCheck *HealthCheckConfig `yaml:"health_check"`
}

// RegionConfig represents a geographic region for geo routing.
type RegionConfig struct {
	Name        string             `yaml:"name"`
	Countries   []string           `yaml:"countries"`
	Continents  []string           `yaml:"continents"`
	ASNs        []uint             `yaml:"asns"`
	CIDRs       []string           `yaml:"cidrs"`
	Endpoints   []EndpointConfig   `yaml:"endpoints"`
	HealthCheck *HealthCheckConfig `yaml:"health_check"`
}

// EndpointConfig defines a target IP or hostname.
type EndpointConfig struct {
	Address     string             `yaml:"address"`
	Weight      int                `yaml:"weight"`
	Latitude    float64            `yaml:"latitude"`
	Longitude   float64            `yaml:"longitude"`
	HealthCheck *HealthCheckConfig `yaml:"health_check"`
}

// HealthCheckConfig defines parameters for endpoint health checks.
type HealthCheckConfig struct {
	Type     string `yaml:"type"`
	Port     int    `yaml:"port"`
	Path     string `yaml:"path"`
	Interval int    `yaml:"interval"`
	Timeout  int    `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}
