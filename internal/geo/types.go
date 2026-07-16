package geo

// GeoInfo contains the geolocation data retrieved for a single IP address.
type GeoInfo struct {
	CountryCode   string
	ContinentCode string
	ASN           uint
	ASOrg         string
	Latitude      float64
	Longitude     float64
}

// IsEmpty returns true if no geolocation data was found.
func (g GeoInfo) IsEmpty() bool {
	return g.CountryCode == "" && g.ContinentCode == "" && g.ASN == 0
}
