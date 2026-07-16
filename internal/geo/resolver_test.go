package geo_test

import (
	"net/netip"
	"testing"

	"github.com/itsazni/geomesh/internal/geo"
)

func TestResolverCountry(t *testing.T) {
	r, err := geo.NewResolver(
		"../../test-fixtures/GeoLite2-City.mmdb",
		"../../test-fixtures/GeoLite2-ASN.mmdb",
	)
	if err != nil {
		t.Skipf("GeoIP database not found (run 'make download-geoip'), skipping: %v", err)
	}
	defer r.Close()

	ip, _ := netip.ParseAddr("8.8.8.8")
	info, err := r.Lookup(ip)
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if info.CountryCode != "US" {
		t.Errorf("expected country US for 8.8.8.8, got %q", info.CountryCode)
	}
	if info.ContinentCode != "NA" {
		t.Errorf("expected continent NA for 8.8.8.8, got %q", info.ContinentCode)
	}
}

func TestResolverASN(t *testing.T) {
	r, err := geo.NewResolver(
		"../../test-fixtures/GeoLite2-City.mmdb",
		"../../test-fixtures/GeoLite2-ASN.mmdb",
	)
	if err != nil {
		t.Skipf("GeoIP database not found, skipping: %v", err)
	}
	defer r.Close()

	ip, _ := netip.ParseAddr("8.8.8.8")
	info, err := r.Lookup(ip)
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if info.ASN == 0 {
		t.Error("expected non-zero ASN for 8.8.8.8")
	}
	if info.ASN != 15169 {
		t.Errorf("expected ASN 15169 (Google), got %d", info.ASN)
	}
}

func TestResolverPrivateIP(t *testing.T) {
	r, err := geo.NewResolver(
		"../../test-fixtures/GeoLite2-City.mmdb",
		"../../test-fixtures/GeoLite2-ASN.mmdb",
	)
	if err != nil {
		t.Skipf("GeoIP database not found, skipping: %v", err)
	}
	defer r.Close()

	ip, _ := netip.ParseAddr("192.168.1.1")
	info, err := r.Lookup(ip)
	if err != nil {
		t.Fatalf("unexpected error for private IP: %v", err)
	}

	if info.CountryCode != "" {
		t.Errorf("expected empty country for private IP, got %q", info.CountryCode)
	}
}

func TestResolverIPv6(t *testing.T) {
	r, err := geo.NewResolver(
		"../../test-fixtures/GeoLite2-City.mmdb",
		"../../test-fixtures/GeoLite2-ASN.mmdb",
	)
	if err != nil {
		t.Skipf("GeoIP database not found, skipping: %v", err)
	}
	defer r.Close()

	ip, _ := netip.ParseAddr("2001:4860:4860::8888")
	info, err := r.Lookup(ip)
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}
	if info.CountryCode == "" {
		t.Error("expected non-empty country code for Google IPv6")
	}
}
