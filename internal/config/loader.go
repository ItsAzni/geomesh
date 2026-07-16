package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (*Config, error) {
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
				if err := loadInto(p, cfg); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if err := loadInto(path, cfg); err != nil {
			return nil, err
		}
	}

	applyDefaults(cfg)
	return cfg, nil
}

func loadInto(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
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

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":53"
	}
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "info"
	}
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

	applyDefaults(cfg)
	return cfg, nil
}
