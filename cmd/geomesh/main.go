package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/itsazni/geomesh/internal/api"
	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/dns"
	"github.com/itsazni/geomesh/internal/geo"
	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/logger"
	"github.com/itsazni/geomesh/internal/zone"
)

var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	configFlag := flag.String("config", "config", "Path to config file or directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GeoMesh - Authoritative GeoDNS Server\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  geomesh [flags] [config-path]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("GeoMesh version: %s\n", Version)
		os.Exit(0)
	}

	configPath := *configFlag
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	// Auto-create default config directory if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config path %q does not exist. Creating it...\n", configPath)
		if err := os.MkdirAll(configPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: failed to create config directory: %v\n", err)
			os.Exit(1)
		}
	}

	cfg, err := config.LoadFile(configPath)
	if err != nil {

		fmt.Fprintf(os.Stderr, "FATAL: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: invalid config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.Server.LogLevel, cfg.Server.LogJSON)

	slog.Info("GeoMesh starting",
		"version", Version,
		"config", configPath,
		"log_level", cfg.Server.LogLevel,
	)

	geoResolver, err := geo.NewResolver(cfg.GeoIP.CityDB, cfg.GeoIP.ASNDB)
	if err != nil {
		slog.Warn("GeoIP databases not loaded, geo and latency routing will not function",
			"city_db", cfg.GeoIP.CityDB,
			"asn_db", cfg.GeoIP.ASNDB,
			"err", err,
		)
	} else {
		slog.Info("GeoIP databases loaded",
			"city_db", cfg.GeoIP.CityDB,
			"asn_db", cfg.GeoIP.ASNDB,
		)
		defer geoResolver.Close()
	}

	healthStore := health.NewStore()
	var healthChecker *health.Checker
	zoneReg := zone.NewRegistry()

	reloadFull := func(newCfg *config.Config) error {
		if err := config.Validate(newCfg); err != nil {
			return err
		}

		zoneCount := len(newCfg.Zones)
		zoneReg.Load(newCfg.Zones)
		slog.Info("zone registry updated", "zones", zoneCount)

		if healthChecker != nil {
			healthChecker.Stop()
		}
		healthChecker = health.NewChecker(healthStore)

		var allEndpoints []config.EndpointConfig
		for _, z := range newCfg.Zones {
			for _, r := range z.Routes {
				for _, ep := range r.Endpoints {
					allEndpoints = append(allEndpoints, ep)
				}
				for _, reg := range r.Regions {
					for _, ep := range reg.Endpoints {
						allEndpoints = append(allEndpoints, ep)
					}
				}
			}
		}
		healthChecker.AddEndpoints(allEndpoints)
		healthChecker.Start()
		return nil
	}

	if err := reloadFull(cfg); err != nil {
		slog.Error("failed to initialize core", "err", err)
		os.Exit(1)
	}

	configWatcher, err := config.NewWatcher(configPath, func(newCfg *config.Config) {
		slog.Info("config change detected, reloading")
		if err := reloadFull(newCfg); err != nil {
			slog.Error("config reload failed", "err", err)
			return
		}
		slog.Info("config reload complete")
	})
	if err != nil {
		slog.Error("failed to setup config watcher", "err", err)
		os.Exit(1)
	}
	defer configWatcher.Close()
	go configWatcher.Start()

	dnsHandler := dns.NewHandler(zoneReg, geoResolver, healthStore)
	dnsServer := dns.NewServer(cfg.Server.Listen, dnsHandler)
	go func() {
		if err := dnsServer.Start(); err != nil {
			slog.Error("DNS server failed", "err", err)
			os.Exit(1)
		}
	}()

	if cfg.Server.API != "" {
		apiServer := api.NewServer(cfg.Server.API, zoneReg, healthStore, func() error {
			newCfg, err := config.LoadFile(configPath)
			if err != nil {
				return err
			}
			return reloadFull(newCfg)
		})
		go func() {
			if err := apiServer.Start(); err != nil {
				slog.Error("API server failed", "err", err)
				os.Exit(1)
			}
		}()
	}

	slog.Info("GeoMesh ready",
		"version", Version,
		"dns", cfg.Server.Listen,
		"api", cfg.Server.API,
	)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	slog.Info("shutdown signal received", "signal", s.String())
	dnsServer.Shutdown()
	slog.Info("GeoMesh stopped")
}
