package main

import (
	"log"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var db *geoip2.Reader

var logger *zap.Logger

func initLogger() {
    config := zap.NewProductionConfig()
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    var err error
    logger, err = config.Build()
    if err != nil {
        panic(err)
    }
}

func main() {
	initLogger()
    defer logger.Sync() // Flushes buffer before exit

    cfg, err := LoadConfig("config.yaml")
    if err != nil {
        logger.Fatal("failed to load config", zap.Error(err))
    }

	db, err = geoip2.Open(cfg.Server.GeoDBPath)
	if err != nil {
		log.Fatalf("GeoIP error: %v", err)
	}

	go RunHealthChecks(cfg)

	// Initialize RRL using config values
	rrl = NewRRLimiter(cfg.Server.Throttling.RPS, cfg.Server.Throttling.Burst)

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
