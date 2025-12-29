package dns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"

	"gslb/internal/lb"
	"gslb/internal/model"
)

type Server struct {
	Records []model.Record
}

func extractClientIP(w dns.ResponseWriter, r *dns.Msg) string {
	clientIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())

	if opt := r.IsEdns0(); opt != nil {
		for _, o := range opt.Option {
			if ecs, ok := o.(*dns.EDNS0_SUBNET); ok {
				return ecs.Address.String()
			}
		}
	}
	return clientIP
}

func (s *Server) Serve(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	clientIP := extractClientIP(w, r)

	for _, q := range r.Question {
		for _, record := range s.Records {
			if q.Name == record.Name {
				ip := lb.Select(record.IPs, clientIP)
				if ip != "" {
					rr, _ := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
					msg.Answer = append(msg.Answer, rr)
				}
			}
		}
	}
	w.WriteMsg(msg)
}
