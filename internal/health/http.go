package health

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itsazni/geomesh/internal/config"
)

// CheckHTTP executes an HTTP/HTTPS health check.
// It considers the endpoint healthy if the response code is 2xx or 3xx.
func CheckHTTP(ctx context.Context, ep config.EndpointConfig) bool {
	hc := ep.HealthCheck
	scheme := hc.Type
	path := hc.Path
	if path == "" {
		path = "/"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, ep.Address, hc.Port, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "GeoMesh-HealthCheck/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
