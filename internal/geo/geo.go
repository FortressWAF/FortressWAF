package geo

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Record struct {
	CountryCode         string  `json:"country_code"`
	CountryName         string  `json:"country_name"`
	City                string  `json:"city"`
	ISP                 string  `json:"isp"`
	ASN                 uint    `json:"asn"`
	ASOrg               string  `json:"as_org"`
	Timezone            string  `json:"timezone"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	IsAnonymousProxy    bool    `json:"is_anonymous_proxy"`
	IsSatelliteProvider bool    `json:"is_satellite_provider"`
}

type Lookup struct {
	mu         sync.RWMutex
	db         *geoip2.Reader
	asnDB      *geoip2.Reader
	cityDBPath string
	asnDBPath  string
	available  bool
}

func NewLookup(cityDBPath, asnDBPath string) *Lookup {
	l := &Lookup{
		cityDBPath: cityDBPath,
		asnDBPath:  asnDBPath,
	}

	if err := l.open(); err != nil {
		slog.Warn("geoip database not available, using stub", "error", err)
	}

	return l
}

func (l *Lookup) open() error {
	var err error

	if l.cityDBPath != "" {
		l.db, err = geoip2.Open(l.cityDBPath)
		if err != nil {
			return fmt.Errorf("open city database: %w", err)
		}
	}

	if l.asnDBPath != "" {
		l.asnDB, err = geoip2.Open(l.asnDBPath)
		if err != nil {
			if l.db != nil {
				l.db.Close()
			}
			return fmt.Errorf("open asn database: %w", err)
		}
	}

	l.available = true
	return nil
}

func (l *Lookup) LookupIP(ipStr string) *Record {
	if !l.available {
		return &Record{CountryCode: "XX", CountryName: "Unknown"}
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &Record{CountryCode: "XX", CountryName: "Invalid IP"}
	}

	record := &Record{}

	if l.db != nil {
		city, err := l.db.City(ip)
		if err == nil {
			record.CountryCode = city.Country.IsoCode
			record.CountryName = city.Country.Names["en"]
			if city.City.Names != nil {
				record.City = city.City.Names["en"]
			}
			record.Timezone = city.Location.TimeZone
			record.Latitude = city.Location.Latitude
			record.Longitude = city.Location.Longitude
			record.IsAnonymousProxy = city.Traits.IsAnonymousProxy
			record.IsSatelliteProvider = city.Traits.IsSatelliteProvider
		}
	}

	if l.asnDB != nil {
		asn, err := l.asnDB.ASN(ip)
		if err == nil {
			record.ASN = asn.AutonomousSystemNumber
			record.ASOrg = asn.AutonomousSystemOrganization
		}
	}

	return record
}

func (l *Lookup) CountryCode(ipStr string) string {
	record := l.LookupIP(ipStr)
	return record.CountryCode
}

func (l *Lookup) ASN(ipStr string) uint {
	record := l.LookupIP(ipStr)
	return record.ASN
}

func (l *Lookup) IsCountryAllowed(ipStr string, allowedCountries []string) bool {
	if len(allowedCountries) == 0 {
		return true
	}

	code := l.CountryCode(ipStr)
	for _, c := range allowedCountries {
		if code == c {
			return true
		}
	}
	return false
}

func (l *Lookup) IsCountryBlocked(ipStr string, blockedCountries []string) bool {
	if len(blockedCountries) == 0 {
		return false
	}

	code := l.CountryCode(ipStr)
	for _, c := range blockedCountries {
		if code == c {
			return true
		}
	}
	return false
}

func (l *Lookup) EnforceDataResidency(ipStr string, allowedRegions []string) bool {
	if len(allowedRegions) == 0 {
		return true
	}

	code := l.CountryCode(ipStr)
	for _, region := range allowedRegions {
		if code == region {
			return true
		}
	}
	return false
}

func (l *Lookup) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.db != nil {
		l.db.Close()
	}
	if l.asnDB != nil {
		l.asnDB.Close()
	}
	l.available = false
}

func (l *Lookup) Reload() error {
	l.Close()
	return l.open()
}

func (l *Lookup) IsAvailable() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.available
}

var _ = slog.Debug
