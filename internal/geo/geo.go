package geo

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

var db *geoip2.Reader

func Init(path string) error {
	var err error
	db, err = geoip2.Open(path)
	return err
}

func Region(ipStr string) string {
	if db == nil {
		return ""
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	rec, err := db.City(ip)
	if err != nil {
		return ""
	}
	return rec.Continent.Code
}
