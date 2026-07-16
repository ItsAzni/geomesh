package health

import (
	"context"
	"fmt"
	"net"
	"time"
)

func CheckUDP(ctx context.Context, addr string, port int) bool {
	target := fmt.Sprintf("%s:%d", addr, port)

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "udp", target)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, err = conn.Write([]byte("\n"))
	if err != nil {
		return false
	}

	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1)
	conn.Read(buf)

	return true
}
