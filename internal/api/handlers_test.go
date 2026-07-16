package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itsazni/geomesh/internal/api"
	"github.com/itsazni/geomesh/internal/config"
	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/zone"
)

func TestStatusHandler(t *testing.T) {
	reg := zone.NewRegistry()
	store := health.NewStore()
	srv := api.NewServer(":0", reg, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
	if _, ok := body["uptime"]; !ok {
		t.Error("expected 'uptime' field in response")
	}
}

func TestZonesHandler(t *testing.T) {
	reg := zone.NewRegistry()
	reg.Load([]config.ZoneConfig{{Name: "example.com"}})
	store := health.NewStore()
	srv := api.NewServer(":0", reg, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/zones", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	store := health.NewStore()
	store.Set("1.1.1.1", true)
	store.Set("2.2.2.2", false)

	reg := zone.NewRegistry()
	srv := api.NewServer(":0", reg, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "1.1.1.1") {
		t.Error("expected 1.1.1.1 to be present in health response")
	}
}

func TestReloadHandler_Post(t *testing.T) {
	called := false
	reloadFn := func() error { called = true; return nil }

	reg := zone.NewRegistry()
	store := health.NewStore()
	srv := api.NewServer(":0", reg, store, reloadFn)

	req := httptest.NewRequest(http.MethodPost, "/api/reload", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("expected reloadFn to be invoked")
	}
}

func TestReloadHandler_GetMethodNotAllowed(t *testing.T) {
	reg := zone.NewRegistry()
	store := health.NewStore()
	srv := api.NewServer(":0", reg, store, func() error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/api/reload", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
