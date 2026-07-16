package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/zone"
)

var startTime = time.Now()

func handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"uptime":     time.Since(startTime).String(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
	})
}

func handleZones(reg *zone.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"zones": reg.Zones(),
		})
	}
}

func handleHealth(store *health.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"endpoints": store.All(),
		})
	}
}

func handleReload(reloadFn func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if reloadFn == nil {
			http.Error(w, "reload not configured", http.StatusNotImplemented)
			return
		}
		slog.Info("manual reload triggered via API", "remote", r.RemoteAddr)
		if err := reloadFn(); err != nil {
			slog.Error("manual reload failed", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		slog.Info("manual reload completed")
		writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
