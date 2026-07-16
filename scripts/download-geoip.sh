#!/usr/bin/env bash
# download-geoip.sh — Download MaxMind GeoLite2 databases
# Usage: ./scripts/download-geoip.sh <LICENSE_KEY> [output-dir]
#
# Requires a free MaxMind account: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data

set -euo pipefail

LICENSE_KEY="${1:-${MAXMIND_LICENSE_KEY:-}}"
OUTPUT_DIR="${2:-./geoip}"

if [ -z "$LICENSE_KEY" ]; then
  echo "Error: MaxMind license key is required."
  echo ""
  echo "How to get a free license key:"
  echo "  1. Sign up at https://www.maxmind.com/en/geolite2/signup"
  echo "  2. Log in to your MaxMind account"
  echo "  3. Go to My Account > Manage License Keys > Generate new license key"
  echo ""
  echo "Usage: $0 <LICENSE_KEY> [output-dir]"
  echo "       MAXMIND_LICENSE_KEY=<key> $0"
  exit 1
fi

MAXMIND_BASE_URL="https://download.maxmind.com/app/geoip_download"
EDITIONS=("GeoLite2-City" "GeoLite2-ASN")

mkdir -p "$OUTPUT_DIR"

for edition in "${EDITIONS[@]}"; do
  echo "Downloading $edition..."
  
  url="${MAXMIND_BASE_URL}?edition_id=${edition}&license_key=${LICENSE_KEY}&suffix=tar.gz"
  tmpfile=$(mktemp /tmp/geolite2-XXXXXX.tar.gz)
  
  # Download
  if command -v curl &> /dev/null; then
    curl -sSL --fail --output "$tmpfile" "$url" \
      || { echo "Failed to download $edition. Ensure license key is valid."; rm -f "$tmpfile"; exit 1; }
  elif command -v wget &> /dev/null; then
    wget -q --output-document="$tmpfile" "$url" \
      || { echo "Failed to download $edition."; rm -f "$tmpfile"; exit 1; }
  else
    echo "Requires curl or wget to download."
    exit 1
  fi
  
  # Extract .mmdb from tarball
  mmdb_file=$(tar -tzf "$tmpfile" | grep '\.mmdb$' | head -1)
  if [ -z "$mmdb_file" ]; then
    echo "No .mmdb file found in archive for $edition"
    rm -f "$tmpfile"
    exit 1
  fi
  
  tar -xzf "$tmpfile" --strip-components=1 -C "$OUTPUT_DIR" "$mmdb_file"
  rm -f "$tmpfile"
  
  echo "$edition.mmdb saved to $OUTPUT_DIR/"
done

echo ""
echo "GeoLite2 databases successfully downloaded to: $OUTPUT_DIR/"
echo ""
echo "Downloaded files:"
ls -lh "$OUTPUT_DIR/"*.mmdb 2>/dev/null || echo "  (no .mmdb found)"
echo ""
echo "Add to GeoMesh configuration:"
echo "  geoip:"
echo "    city_db: $OUTPUT_DIR/GeoLite2-City.mmdb"
echo "    asn_db:  $OUTPUT_DIR/GeoLite2-ASN.mmdb"
