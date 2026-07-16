package dns

import (
	"net"

	"github.com/itsazni/geomesh/internal/config"
	mdns "github.com/miekg/dns"
)

// BuildAnswer constructs DNS Answer RRs from a list of endpoints.
// Only generates records that correspond to the query's Qtype.
func BuildAnswer(question mdns.Question, endpoints []config.EndpointConfig, ttl uint32) []mdns.RR {
	var answers []mdns.RR

	for _, ep := range endpoints {
		hdr := mdns.RR_Header{
			Name:  question.Name,
			Class: mdns.ClassINET,
			Ttl:   ttl,
		}

		ip := net.ParseIP(ep.Address)

		switch question.Qtype {
		case mdns.TypeA:
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			hdr.Rrtype = mdns.TypeA
			answers = append(answers, &mdns.A{Hdr: hdr, A: ip4})

		case mdns.TypeAAAA:
			if ip == nil {
				continue
			}
			if ip.To4() != nil {
				continue
			}
			hdr.Rrtype = mdns.TypeAAAA
			answers = append(answers, &mdns.AAAA{Hdr: hdr, AAAA: ip.To16()})

		case mdns.TypeANY:

			if ip != nil {
				if ip4 := ip.To4(); ip4 != nil {
					h := hdr
					h.Rrtype = mdns.TypeA
					answers = append(answers, &mdns.A{Hdr: h, A: ip4})
				} else {
					h := hdr
					h.Rrtype = mdns.TypeAAAA
					answers = append(answers, &mdns.AAAA{Hdr: h, AAAA: ip.To16()})
				}
			}
		}
	}
	return answers
}
