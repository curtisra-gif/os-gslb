package dns

import (
	"log"
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
	log.Printf("[DEBUG] Received query from IP: %s", ip)

	if !h.Throttler.Allow(ip) {
		log.Printf("[DEBUG] Request throttled for IP: %s", ip)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeRefused)
		_ = w.WriteMsg(m)
		return
	}

	m := new(dns.Msg)
	m.SetReply(r)

	for _, q := range r.Question {
		log.Printf("[DEBUG] Query for domain: %s, type: %d", q.Name, q.Qtype)
		if q.Qtype == dns.TypeA {
			region := h.GeoLocator.Lookup(ip)
			log.Printf("[DEBUG] Mapped IP %s to region: %s", ip, region)
			backendIP := h.Router.Route(region)
			log.Printf("[DEBUG] Region %s routed to backend IP: %s", region, backendIP)
			if backendIP != "" {
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30},
					A:   net.ParseIP(backendIP),
				})
			} else {
				log.Printf("[WARN] No backend IP found for region: %s", region)
			}
		}
	}

	_ = w.WriteMsg(m)
	log.Printf("[DEBUG] Response sent for IP: %s", ip)
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
