package main

import (
	"log"

	"github.com/miekg/dns"

	"gslb/internal/config"
	dnssrv "gslb/internal/dns"
	"gslb/internal/geo"
	"gslb/internal/health"
)

func main() {
	cfg := config.Load("config.json")

	if err := geo.Init("build/GeoLite2-City.mmdb"); err != nil {
		log.Fatal(err)
	}

	hm := health.NewManager()
	for i := range cfg.Records {
		hm.Start(&cfg.Records[i])
	}

	server := &dnssrv.Server{Records: cfg.Records}
	dns.HandleFunc(".", server.Serve)

	log.Fatal(dns.ListenAndServe(":53", "udp", nil))
}
