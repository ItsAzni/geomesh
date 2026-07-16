package health_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/health"
)

func TestStore_SetAndGet(t *testing.T) {
	s := health.NewStore()
	s.Set("1.1.1.1", true)
	if !s.IsHealthy("1.1.1.1") {
		t.Error("expected 1.1.1.1 to be healthy")
	}
	s.Set("1.1.1.1", false)
	if s.IsHealthy("1.1.1.1") {
		t.Error("expected 1.1.1.1 to be unhealthy after Set(false)")
	}
}

func TestStore_UnknownDefaultsHealthy(t *testing.T) {
	s := health.NewStore()

	if !s.IsHealthy("9.9.9.9") {
		t.Error("unknown endpoint should default to healthy")
	}
}

func TestTCPCheck_Success(t *testing.T) {

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	port := l.Addr().(*net.TCPAddr).Port
	ep := config.EndpointConfig{
		Address: "127.0.0.1",
		HealthCheck: &config.HealthCheckConfig{
			Type:    "tcp",
			Port:    port,
			Timeout: 2,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !health.CheckTCP(ctx, ep) {
		t.Error("expected TCP check to succeed")
	}
}

func TestTCPCheck_Failure(t *testing.T) {

	ep := config.EndpointConfig{
		Address: "127.0.0.1",
		HealthCheck: &config.HealthCheckConfig{
			Type:    "tcp",
			Port:    19999,
			Timeout: 1,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if health.CheckTCP(ctx, ep) {
		t.Error("expected TCP check to fail for closed port")
	}
}

func TestHTTPCheck_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	port := srv.Listener.Addr().(*net.TCPAddr).Port
	ep := config.EndpointConfig{
		Address: "127.0.0.1",
		HealthCheck: &config.HealthCheckConfig{
			Type:    "http",
			Port:    port,
			Path:    "/health",
			Timeout: 2,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !health.CheckHTTP(ctx, ep) {
		t.Error("expected HTTP check to succeed")
	}
}

func TestHTTPCheck_FailOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	port := srv.Listener.Addr().(*net.TCPAddr).Port
	ep := config.EndpointConfig{
		Address: "127.0.0.1",
		HealthCheck: &config.HealthCheckConfig{
			Type:    "http",
			Port:    port,
			Path:    "/health",
			Timeout: 2,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if health.CheckHTTP(ctx, ep) {
		t.Error("expected HTTP check to fail on 404")
	}
}
