package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultServerYAML is the default template for server.yaml.
const DefaultServerYAML = `# GeoMesh Server Configuration
# Docs: https://github.com/itsazni/geomesh

listen: ":53"        # DNS address (UDP + TCP)
api: ":8080"         # REST API address, remove or set to "" to disable
log_level: info      # debug | info | warn | error
log_json: false      # true = structured JSON logs
`

type serverConfigFile struct {
	Server   *ServerConfig `yaml:"server"`
	Listen   *string       `yaml:"listen"`
	API      *string       `yaml:"api"`
	LogLevel *string       `yaml:"log_level"`
	LogJSON  *bool         `yaml:"log_json"`
}

type directoryConfigFile struct {
	Server *ServerConfig `yaml:"server"`
	Listen *string       `yaml:"listen"`
	GeoIP  GeoIPConfig   `yaml:"geoip"`
	Zones  []ZoneConfig  `yaml:"zones"`
}

func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read server config %q: %w", path, err)
	}

	var raw serverConfigFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil && err != io.EOF {
		return nil, fmt.Errorf("parse server config %q: %w", path, err)
	}

	srv := &ServerConfig{
		Listen:   ":53",
		LogLevel: "info",
	}

	if raw.Server != nil {
		if raw.Server.Listen != "" {
			srv.Listen = raw.Server.Listen
		}
		if raw.Server.API != "" {
			srv.API = raw.Server.API
		}
		if raw.Server.LogLevel != "" {
			srv.LogLevel = raw.Server.LogLevel
		}
		srv.LogJSON = raw.Server.LogJSON
	}

	if raw.Listen != nil && *raw.Listen != "" {
		srv.Listen = *raw.Listen
	}
	if raw.API != nil {
		srv.API = *raw.API
	}
	if raw.LogLevel != nil && *raw.LogLevel != "" {
		srv.LogLevel = *raw.LogLevel
	}
	if raw.LogJSON != nil {
		srv.LogJSON = *raw.LogJSON
	}

	return srv, nil
}

func LoadZonesConfig(path string) (*Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat config %q: %w", path, err)
	}

	cfg := &Config{}

	if info.IsDir() {
		err := filepath.Walk(path, func(p string, i os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !i.IsDir() && (strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml")) {
				if err := loadDirectoryFile(p, cfg); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if err := loadSingleFile(path, cfg); err != nil {
			return nil, err
		}
	}

	applyZoneDefaults(cfg)
	return cfg, nil
}

// Load loads both the server config and the zones/geoip config, merging them into a complete Config.
func Load(serverPath, configPath string) (*Config, error) {
	var srv *ServerConfig

	if serverPath != "" {
		if _, err := os.Stat(serverPath); err == nil {
			loadedSrv, err := LoadServerConfig(serverPath)
			if err != nil {
				return nil, err
			}
			srv = loadedSrv
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat server config %q: %w", serverPath, err)
		} else if serverPath != "server.yaml" {
			return nil, fmt.Errorf("server config file not found: %s", serverPath)
		}
	}

	cfg, err := LoadZonesConfig(configPath)
	if err != nil {
		return nil, err
	}

	if srv != nil {
		cfg.Server = *srv
	} else if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":53"
		if cfg.Server.LogLevel == "" {
			cfg.Server.LogLevel = "info"
		}
	}

	return cfg, nil
}

// LoadFile parses configuration from a file or directory.
// When given a directory, server configuration is forbidden inside the directory files.
func LoadFile(path string) (*Config, error) {
	return Load("server.yaml", path)
}

func loadDirectoryFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("open config %q: %w", path, err)
	}

	var parsed directoryConfigFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&parsed); err != nil && err != io.EOF {
		return fmt.Errorf("parse config %q: %w", path, err)
	}

	if parsed.Server != nil || parsed.Listen != nil {
		return fmt.Errorf("config %q: server configuration is not allowed in config directory; server config must be in its own file (e.g. server.yaml), config directory is only for zones and geoip", path)
	}

	if parsed.GeoIP.CityDB != "" {
		cfg.GeoIP.CityDB = parsed.GeoIP.CityDB
	}
	if parsed.GeoIP.ASNDB != "" {
		cfg.GeoIP.ASNDB = parsed.GeoIP.ASNDB
	}

	cfg.Zones = append(cfg.Zones, parsed.Zones...)
	return nil
}

func loadSingleFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("open config %q: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	temp := &Config{}
	if err := dec.Decode(temp); err != nil && err != io.EOF {
		return fmt.Errorf("parse config %q: %w", path, err)
	}

	if temp.Server.Listen != "" {
		cfg.Server.Listen = temp.Server.Listen
	}
	if temp.Server.API != "" {
		cfg.Server.API = temp.Server.API
	}
	if temp.Server.LogLevel != "" {
		cfg.Server.LogLevel = temp.Server.LogLevel
	}
	if temp.Server.LogJSON {
		cfg.Server.LogJSON = temp.Server.LogJSON
	}

	if temp.GeoIP.CityDB != "" {
		cfg.GeoIP.CityDB = temp.GeoIP.CityDB
	}
	if temp.GeoIP.ASNDB != "" {
		cfg.GeoIP.ASNDB = temp.GeoIP.ASNDB
	}

	cfg.Zones = append(cfg.Zones, temp.Zones...)
	return nil
}

func applyZoneDefaults(cfg *Config) {
	for zi := range cfg.Zones {
		for ri := range cfg.Zones[zi].Routes {
			r := &cfg.Zones[zi].Routes[ri]
			if r.TTL == 0 {
				r.TTL = 60
			}
			if r.RecordType == "" {
				r.RecordType = "A"
			}
			for ei := range r.Endpoints {
				if r.Endpoints[ei].Weight == 0 {
					r.Endpoints[ei].Weight = 1
				}
			}
			for regi := range r.Regions {
				for ei := range r.Regions[regi].Endpoints {
					if r.Regions[regi].Endpoints[ei].Weight == 0 {
						r.Regions[regi].Endpoints[ei].Weight = 1
					}
				}
			}
		}
	}
}

// LoadReader parses configuration from an io.Reader (useful for tests).
func LoadReader(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	cfg := &Config{}
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":53"
	}
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "info"
	}

	applyZoneDefaults(cfg)
	return cfg, nil
}
