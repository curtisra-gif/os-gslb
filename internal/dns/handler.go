package dns

import (
	"net"

	"github.com/curtisra-gif/os-gslb/internal/config"
	"github.com/curtisra-gif/os-gslb/internal/throttler"
	"github.com/miekg/dns"
)

type Handler struct {
	Cfg        *config.Config
	Throttler  *throttler.Throttler
	GeoLocator *GeoLocator
	Router     *Router
}

func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	ip := extractClientIP(w, r) // handles EDNS Client Subnet

	if !h.Throttler.Allow(ip) {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeRefused)
		_ = w.WriteMsg(m)
		return
	}

	m := new(dns.Msg)
	m.SetReply(r)

	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			region := h.GeoLocator.Lookup(ip)
			backendIP := h.Router.Route(region)
			if backendIP != "" {
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30},
					A:   net.ParseIP(backendIP),
				})
			}
		}
	}

	_ = w.WriteMsg(m)
}

// Example ECS extraction
func extractClientIP(w dns.ResponseWriter, r *dns.Msg) string {
	for _, edns := range r.Extra {
		if o, ok := edns.(*dns.OPT); ok && o != nil {
			for _, option := range o.Option {
				if ecs, ok := option.(*dns.EDNS0_SUBNET); ok {
					return ecs.Address.String()
				}
			}
		}
	}
	host, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	return host
}
