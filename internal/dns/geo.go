package dns

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoLocator struct {
	db *geoip2.Reader
}

func NewGeoLocator(dbPath string) (*GeoLocator, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &GeoLocator{db: db}, nil
}

// Returns a region string based on IP or ECS
func (g *GeoLocator) Lookup(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "NA" // default
	}

	record, err := g.db.City(ip)
	if err != nil || record == nil {
		return "NA"
	}

	country := record.Country.IsoCode
	switch country {
	case "US", "CA":
		return "NA"
	case "DE", "FR", "GB":
		return "EU"
	default:
		return "Default"
	}
}
