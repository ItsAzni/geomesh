package geo

import (
	"fmt"
	"net/netip"
	"sync"

	"github.com/oschwald/geoip2-golang/v2"
)

// Resolver performs GeoIP lookups utilizing MaxMind GeoLite2 databases.
// Thread-safe: Lookup and Reload can be invoked concurrently.
type Resolver struct {
	mu     sync.RWMutex
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

// NewResolver initializes a new Resolver using the paths to GeoLite2 .mmdb files.
func NewResolver(cityDBPath, asnDBPath string) (*Resolver, error) {
	cityDB, err := geoip2.Open(cityDBPath)
	if err != nil {
		return nil, fmt.Errorf("open city db %q: %w", cityDBPath, err)
	}
	asnDB, err := geoip2.Open(asnDBPath)
	if err != nil {
		cityDB.Close()
		return nil, fmt.Errorf("open asn db %q: %w", asnDBPath, err)
	}
	return &Resolver{cityDB: cityDB, asnDB: asnDB}, nil
}

// Lookup queries GeoIP information for a specified IP address.
// Returns empty GeoInfo (not an error) for private or unmapped IPs.
func (r *Resolver) Lookup(ip netip.Addr) (GeoInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var info GeoInfo

	city, err := r.cityDB.City(ip)
	if err == nil && city.HasData() {
		info.CountryCode = city.Country.ISOCode
		info.ContinentCode = city.Continent.Code
		if city.Location.Latitude != nil {
			info.Latitude = *city.Location.Latitude
		}
		if city.Location.Longitude != nil {
			info.Longitude = *city.Location.Longitude
		}
	}

	asn, err := r.asnDB.ASN(ip)
	if err == nil && asn.HasData() {
		info.ASN = uint(asn.AutonomousSystemNumber)
		info.ASOrg = asn.AutonomousSystemOrganization
	}

	return info, nil
}

// Reload updates the databases from disk without downtime.
// Useful when GeoLite2 files are updated via cron job.
func (r *Resolver) Reload(cityDBPath, asnDBPath string) error {
	newCity, err := geoip2.Open(cityDBPath)
	if err != nil {
		return fmt.Errorf("reload city db: %w", err)
	}
	newASN, err := geoip2.Open(asnDBPath)
	if err != nil {
		newCity.Close()
		return fmt.Errorf("reload asn db: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cityDB.Close()
	r.asnDB.Close()
	r.cityDB = newCity
	r.asnDB = newASN
	return nil
}

// Close gracefully terminates the database connections.
func (r *Resolver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cityDB != nil {
		r.cityDB.Close()
	}
	if r.asnDB != nil {
		r.asnDB.Close()
	}
}
