package dns

import (
	"fmt"
	"log/slog"

	mdns "github.com/miekg/dns"
)

// Server manages the authoritative DNS server over UDP and TCP.
type Server struct {
	addr    string
	handler mdns.Handler
	udpSrv  *mdns.Server
	tcpSrv  *mdns.Server
}

// NewServer creates a new Server instance.
// addr: e.g., ":53" or ":5353" for testing without root privileges.
func NewServer(addr string, handler mdns.Handler) *Server {
	return &Server{addr: addr, handler: handler}
}

// Start initiates the UDP and TCP DNS listeners. Blocks on the first encountered error.
func (s *Server) Start() error {
	s.udpSrv = &mdns.Server{
		Addr:    s.addr,
		Net:     "udp",
		Handler: s.handler,
	}
	s.tcpSrv = &mdns.Server{
		Addr:    s.addr,
		Net:     "tcp",
		Handler: s.handler,
	}

	errCh := make(chan error, 2)

	go func() {
		slog.Info("DNS UDP server listening", "addr", s.addr)
		if err := s.udpSrv.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("dns udp: %w", err)
		}
	}()

	go func() {
		slog.Info("DNS TCP server listening", "addr", s.addr)
		if err := s.tcpSrv.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("dns tcp: %w", err)
		}
	}()

	return <-errCh
}

// Shutdown gracefully terminates the server.
func (s *Server) Shutdown() {
	if s.udpSrv != nil {
		s.udpSrv.Shutdown()
	}
	if s.tcpSrv != nil {
		s.tcpSrv.Shutdown()
	}
}
