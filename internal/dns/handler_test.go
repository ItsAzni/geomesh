package dns_test

import (
	"net"
	"testing"

	"github.com/itsazni/geomesh/internal/config"
	gdns "github.com/itsazni/geomesh/internal/dns"
	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/zone"
	mdns "github.com/miekg/dns"
)

func TestHandler_FirstPolicy_ReturnsA(t *testing.T) {
	reg := zone.NewRegistry()
	reg.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{
					Name:       "play",
					Policy:     "first",
					TTL:        60,
					RecordType: "A",
					Endpoints:  []config.EndpointConfig{{Address: "1.2.3.4", Weight: 1}},
				},
			},
		},
	})

	store := health.NewStore()
	handler := gdns.NewHandler(reg, nil, store)

	req := new(mdns.Msg)
	req.SetQuestion("play.example.com.", mdns.TypeA)

	rw := &mockWriter{addr: &net.UDPAddr{IP: net.ParseIP("8.8.8.8")}}
	handler.ServeDNS(rw, req)

	if rw.msg == nil {
		t.Fatal("no response written")
	}
	if rw.msg.Rcode != mdns.RcodeSuccess {
		t.Errorf("expected NOERROR, got %s", mdns.RcodeToString[rw.msg.Rcode])
	}
	if len(rw.msg.Answer) == 0 {
		t.Fatal("expected at least one answer")
	}
	a, ok := rw.msg.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", rw.msg.Answer[0])
	}
	if a.A.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", a.A.String())
	}
}

func TestHandler_UnknownZone_ReturnsRefused(t *testing.T) {
	reg := zone.NewRegistry()
	reg.Load(nil)

	store := health.NewStore()
	handler := gdns.NewHandler(reg, nil, store)

	req := new(mdns.Msg)
	req.SetQuestion("unknown.other.com.", mdns.TypeA)

	rw := &mockWriter{addr: &net.UDPAddr{IP: net.ParseIP("8.8.8.8")}}
	handler.ServeDNS(rw, req)

	if rw.msg.Rcode != mdns.RcodeRefused {
		t.Errorf("expected REFUSED for unknown zone, got %s", mdns.RcodeToString[rw.msg.Rcode])
	}
}

func TestHandler_AuthoritativeFlag(t *testing.T) {
	reg := zone.NewRegistry()
	reg.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "www", Policy: "first", TTL: 60, RecordType: "A",
					Endpoints: []config.EndpointConfig{{Address: "5.5.5.5", Weight: 1}}},
			},
		},
	})

	store := health.NewStore()
	handler := gdns.NewHandler(reg, nil, store)

	req := new(mdns.Msg)
	req.SetQuestion("www.example.com.", mdns.TypeA)

	rw := &mockWriter{addr: &net.UDPAddr{IP: net.ParseIP("1.2.3.4")}}
	handler.ServeDNS(rw, req)

	if !rw.msg.Authoritative {
		t.Error("response must have Authoritative bit set")
	}
}

func TestHandler_TTL(t *testing.T) {
	reg := zone.NewRegistry()
	reg.Load([]config.ZoneConfig{
		{
			Name: "example.com",
			Routes: []config.RouteConfig{
				{Name: "api", Policy: "first", TTL: 120, RecordType: "A",
					Endpoints: []config.EndpointConfig{{Address: "7.7.7.7", Weight: 1}}},
			},
		},
	})

	store := health.NewStore()
	handler := gdns.NewHandler(reg, nil, store)

	req := new(mdns.Msg)
	req.SetQuestion("api.example.com.", mdns.TypeA)

	rw := &mockWriter{addr: &net.UDPAddr{IP: net.ParseIP("1.2.3.4")}}
	handler.ServeDNS(rw, req)

	if rw.msg.Answer[0].Header().Ttl != 120 {
		t.Errorf("expected TTL 120, got %d", rw.msg.Answer[0].Header().Ttl)
	}
}

// mockWriter implements dns.ResponseWriter for testing purposes.
type mockWriter struct {
	addr net.Addr
	msg  *mdns.Msg
}

func (m *mockWriter) LocalAddr() net.Addr          { return &net.UDPAddr{} }
func (m *mockWriter) RemoteAddr() net.Addr         { return m.addr }
func (m *mockWriter) WriteMsg(msg *mdns.Msg) error { m.msg = msg; return nil }
func (m *mockWriter) Write(b []byte) (int, error)  { return 0, nil }
func (m *mockWriter) Close() error                 { return nil }
func (m *mockWriter) TsigStatus() error            { return nil }
func (m *mockWriter) TsigTimersOnly(bool)          {}
func (m *mockWriter) Hijack()                      {}
