package dns

import (
	"net"
	"net/netip"

	mdns "github.com/miekg/dns"
)

// ExtractClientIP extracts the client IP address from the DNS request.
// Priority: EDNS Client Subnet (ECS) > remote addr.
// ECS is utilized when queries arrive via a DNS resolver rather than directly from the client.
func ExtractClientIP(r *mdns.Msg, remoteAddr net.Addr) netip.Addr {

	if opt := r.IsEdns0(); opt != nil {
		for _, o := range opt.Option {
			if subnet, ok := o.(*mdns.EDNS0_SUBNET); ok {
				if len(subnet.Address) > 0 && !subnet.Address.IsUnspecified() {
					addr, err := netip.ParseAddr(subnet.Address.String())
					if err == nil {
						return addr
					}
				}
			}
		}
	}

	host, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		return netip.Addr{}
	}
	addr, _ := netip.ParseAddr(host)
	return addr
}
