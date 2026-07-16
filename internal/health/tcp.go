package health

import (
	"context"
	"fmt"
	"net"

	"github.com/itsazni/geomesh/internal/config"
)

// CheckTCP executes a TCP health check by dialing the address:port.
func CheckTCP(ctx context.Context, ep config.EndpointConfig) bool {
	hc := ep.HealthCheck
	addr := fmt.Sprintf("%s:%d", ep.Address, hc.Port)
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
