package main

import (
	"log"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
)

var db *geoip2.Reader

func main() {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	db, err = geoip2.Open(cfg.Server.GeoDBPath)
	if err != nil {
		log.Fatalf("GeoIP error: %v", err)
	}

	go RunHealthChecks(cfg)

	handler := &GSLBHandler{Config: cfg}
	server := &dns.Server{
		Addr:    cfg.Server.ListenAddr,
		Net:     "udp",
		Handler: handler,
	}

	log.Printf("GSLB active on %s", cfg.Server.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
