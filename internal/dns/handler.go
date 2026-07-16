package dns

import (
	"log/slog"
	"net"
	"time"

	"github.com/itsazni/geomesh/internal/geo"
	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/policy"
	"github.com/itsazni/geomesh/internal/zone"
	mdns "github.com/miekg/dns"
)

// Handler implements the dns.Handler interface for GeoMesh.
type Handler struct {
	registry *zone.Registry
	geoRes   *geo.Resolver
	store    *health.Store
}

// NewHandler initializes a new Handler.
// geoRes may be nil — geo routing will be disabled and will fall back to the default region.
func NewHandler(registry *zone.Registry, geoRes *geo.Resolver, store *health.Store) *Handler {
	return &Handler{registry: registry, geoRes: geoRes, store: store}
}

// ServeDNS processes incoming DNS requests (invoked by miekg/dns).
func (h *Handler) ServeDNS(w mdns.ResponseWriter, r *mdns.Msg) {
	start := time.Now()

	m := new(mdns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false

	if len(r.Question) == 0 {
		slog.Warn("dns query with no questions received",
			"from", w.RemoteAddr(),
			"id", r.Id,
		)
		m.Rcode = mdns.RcodeFormatError
		w.WriteMsg(m)
		return
	}

	q := r.Question[0]
	fqdn := q.Name
	qtype := mdns.TypeToString[q.Qtype]
	from := w.RemoteAddr().String()

	slog.Debug("dns query received",
		"name", fqdn,
		"type", qtype,
		"from", from,
		"id", r.Id,
		"opcode", mdns.OpcodeToString[r.Opcode],
	)

	route, ok := h.registry.Lookup(fqdn)
	if !ok {
		slog.Debug("zone not found, returning REFUSED",
			"name", fqdn,
			"from", from,
		)
		m.Rcode = mdns.RcodeRefused
		w.WriteMsg(m)
		logQuery(fqdn, qtype, from, "REFUSED", 0, time.Since(start))
		return
	}

	slog.Debug("route matched",
		"name", fqdn,
		"route", route.Name,
		"policy", route.Policy,
	)

	var geoInfo geo.GeoInfo
	clientIP := ExtractClientIP(r, w.RemoteAddr())

	if h.geoRes != nil {
		if clientIP.IsValid() && !clientIP.IsPrivate() && !clientIP.IsLoopback() {
			var err error
			geoInfo, err = h.geoRes.Lookup(clientIP)
			if err != nil {
				slog.Warn("geo lookup failed",
					"ip", clientIP,
					"err", err,
				)
			} else {
				slog.Debug("geo lookup result",
					"ip", clientIP,
					"country", geoInfo.CountryCode,
					"continent", geoInfo.ContinentCode,
					"asn", geoInfo.ASN,
					"lat", geoInfo.Latitude,
					"lon", geoInfo.Longitude,
				)
			}
		} else {
			slog.Debug("skipping geo lookup",
				"ip", clientIP,
				"reason", classifySkipReason(clientIP),
			)
		}
	}

	p, err := policy.NewPolicy(route)
	if err != nil {
		slog.Error("policy creation failed",
			"route", route.Name,
			"policy", route.Policy,
			"err", err,
		)
		m.Rcode = mdns.RcodeServerFailure
		w.WriteMsg(m)
		logQuery(fqdn, qtype, from, "SERVFAIL", 0, time.Since(start))
		return
	}

	healthyFn := func(addr string) bool {
		healthy := h.store.IsHealthy(addr)
		slog.Debug("endpoint health check",
			"addr", addr,
			"healthy", healthy,
		)
		return healthy
	}

	endpoints, err := p.Select(geoInfo, healthyFn)
	if err != nil || len(endpoints) == 0 {
		slog.Warn("no endpoints selected",
			"name", fqdn,
			"route", route.Name,
			"policy", route.Policy,
			"country", geoInfo.CountryCode,
			"err", err,
		)
		m.Rcode = mdns.RcodeNameError
		w.WriteMsg(m)
		logQuery(fqdn, qtype, from, "NXDOMAIN", 0, time.Since(start))
		return
	}

	slog.Debug("endpoints selected",
		"name", fqdn,
		"policy", route.Policy,
		"count", len(endpoints),
		"first", endpoints[0].Address,
	)

	m.Answer = BuildAnswer(q, endpoints, route.TTL)
	if len(m.Answer) == 0 {

		slog.Debug("answer empty after build, Qtype mismatch",
			"name", fqdn,
			"type", qtype,
		)
		m.Rcode = mdns.RcodeNameError
	}

	if opt := r.IsEdns0(); opt != nil {
		m.SetEdns0(opt.UDPSize(), opt.Do())
	}

	elapsed := time.Since(start)
	logQuery(fqdn, qtype, from, mdns.RcodeToString[m.Rcode], len(m.Answer), elapsed)

	w.WriteMsg(m)
}

// logQuery emits a consistent info-level log line for every completed DNS query.
func logQuery(name, qtype, from, rcode string, answers int, elapsed time.Duration) {
	slog.Info("dns query",
		"name", name,
		"type", qtype,
		"from", from,
		"rcode", rcode,
		"answers", answers,
		"duration_ms", elapsed.Milliseconds(),
	)
}

// classifySkipReason returns a human-readable reason for skipping geo lookup.
func classifySkipReason(ip interface {
	IsPrivate() bool
	IsLoopback() bool
	IsValid() bool
}) string {
	if !ip.IsValid() {
		return "invalid ip"
	}
	if ip.IsLoopback() {
		return "loopback"
	}
	if ip.IsPrivate() {
		return "private"
	}
	return "unknown"
}

// Ensure net.Addr is accessible for private IP validation
var _ = net.ParseIP
