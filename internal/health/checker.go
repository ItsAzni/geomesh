package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/itsazni/geomesh/internal/config"
)

// Checker executes health check goroutines for all endpoints periodically.
// Operates independently from DNS request handling.
type Checker struct {
	store     *Store
	endpoints []config.EndpointConfig
	cancel    context.CancelFunc
}

// NewChecker initializes a new Checker connected to the specified store.
func NewChecker(store *Store) *Checker {
	return &Checker{store: store}
}

// AddEndpoints registers endpoints for health monitoring.
// Only endpoints with HealthCheck != nil are monitored.
func (c *Checker) AddEndpoints(eps []config.EndpointConfig) {
	c.endpoints = append(c.endpoints, eps...)
}

// Start launches the health check goroutines. It is non-blocking.
func (c *Checker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	started := 0
	for _, ep := range c.endpoints {
		if ep.HealthCheck == nil {
			slog.Debug("health check skipped, no config",
				"addr", ep.Address,
			)
			continue
		}
		slog.Debug("health check registered",
			"addr", ep.Address,
			"type", ep.HealthCheck.Type,
			"port", ep.HealthCheck.Port,
			"interval", ep.HealthCheck.Interval,
		)
		go c.runLoop(ctx, ep)
		started++
	}
	slog.Info("health checker started", "endpoints", started)
}

// Stop terminates all health check goroutines.
func (c *Checker) Stop() {
	if c.cancel != nil {
		slog.Debug("health checker stopping")
		c.cancel()
	}
}

func (c *Checker) runLoop(ctx context.Context, ep config.EndpointConfig) {
	hc := ep.HealthCheck
	interval := time.Duration(hc.Interval) * time.Second
	if interval == 0 {
		interval = 30 * time.Second
	}

	c.check(ctx, ep)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("health check loop stopped", "addr", ep.Address)
			return
		case <-ticker.C:
			c.check(ctx, ep)
		}
	}
}

func (c *Checker) check(ctx context.Context, ep config.EndpointConfig) {
	hc := ep.HealthCheck
	retries := hc.Retries
	if retries <= 0 {
		retries = 1
	}
	timeout := time.Duration(hc.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	var ok bool
	var attempt int

	for i := 0; i < retries; i++ {
		attempt = i + 1
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		switch hc.Type {
		case "tcp":
			ok = CheckTCP(checkCtx, ep)
		case "http", "https":
			ok = CheckHTTP(checkCtx, ep)
		case "mcjava":
			ok = CheckMinecraft(checkCtx, ep)
		case "mcbedrock":
			ok = CheckBedrock(checkCtx, ep)
		case "udp":
			ok = CheckUDP(checkCtx, ep.Address, hc.Port)
		default:
			ok = true
		}
		cancel()
		if ok {
			break
		}
		slog.Debug("health check attempt failed",
			"addr", ep.Address,
			"type", hc.Type,
			"attempt", attempt,
			"retries", retries,
		)
	}

	elapsed := time.Since(start)
	slog.Debug("health check completed",
		"addr", ep.Address,
		"type", hc.Type,
		"healthy", ok,
		"attempts", attempt,
		"duration_ms", elapsed.Milliseconds(),
	)

	prev := c.store.IsHealthy(ep.Address)
	c.store.Set(ep.Address, ok)

	if prev != ok {
		if ok {
			slog.Info("endpoint recovered",
				"addr", ep.Address,
				"type", hc.Type,
				"port", hc.Port,
				"after_ms", elapsed.Milliseconds(),
			)
		} else {
			slog.Warn("endpoint marked unhealthy",
				"addr", ep.Address,
				"type", hc.Type,
				"port", hc.Port,
				"retries", retries,
				"after_ms", elapsed.Milliseconds(),
			)
		}
	}
}
