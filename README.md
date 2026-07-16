# GeoMesh

GeoMesh is an authoritative GeoDNS server written in Go. It routes DNS queries to the most appropriate endpoint based on the requester's geographic location or network characteristics, using a modular and configurable routing policy engine.

It is designed to be operationally simple: a single static binary, YAML configuration, and optional GeoIP databases are all that is required.

---

## Table of Contents

- [Deployment Guide](#deployment-guide)
  - [Method 1: Pre-built Binary](#method-1-pre-built-binary-recommended)
  - [Method 2: Docker](#method-2-docker)
  - [Method 3: Compile from Source](#method-3-compile-from-source)
- [DNS Delegation (Subdomain Setup)](#dns-delegation-subdomain-setup)
- [Configuration](#configuration)
- [Routing Policies](#routing-policies)
- [Health Check Types](#health-check-types)
- [Multi-File Configuration](#multi-file-configuration)
- [Hot Reload](#hot-reload)
- [REST API](#rest-api)

---

## Deployment Guide

GeoMesh is distributed as a static binary with no runtime dependencies. Choose the method that fits your infrastructure.

---

### Method 1: Pre-built Binary (Recommended)

This is the recommended approach for production deployments. No Go toolchain, Docker, or any other dependency is required.

**Step 1 — Download the Binary**

Go to the [Releases](https://github.com/itsazni/geomesh/releases) page and download the archive for your platform. Make the binary executable:

```bash
chmod +x geomesh-linux-amd64
mv geomesh-linux-amd64 /usr/local/bin/geomesh
```

**Step 2 — Download GeoIP Databases**

GeoMesh uses [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) for geographic lookups. These are free but require a MaxMind account and license key.

Sign up at [maxmind.com](https://www.maxmind.com/en/geolite2/signup), generate a license key, then run:

```bash
curl -sSL https://raw.githubusercontent.com/itsazni/geomesh/master/scripts/download-geoip.sh | bash -s -- YOUR_LICENSE_KEY ./geoip
```

This downloads `GeoLite2-City.mmdb` and `GeoLite2-ASN.mmdb` into `./geoip/`.

**Step 3 — Create Configuration**

```bash
mkdir -p config
curl -sSL https://raw.githubusercontent.com/itsazni/geomesh/master/examples/geomesh.yaml -o config/geomesh.yaml
```

Edit `config/geomesh.yaml` to define your zones, routes, and endpoints.

**Step 4 — Run**

```bash
# Load all YAML files in a directory
geomesh config/

# Or point to a single file
geomesh config/geomesh.yaml

# Check version
geomesh --version
```

---

### Method 2: Docker

**Step 1 — Clone the Repository**

```bash
git clone https://github.com/itsazni/geomesh.git
cd geomesh
```

**Step 2 — Download GeoIP Databases**

```bash
MAXMIND_LICENSE_KEY=your_key bash scripts/download-geoip.sh
```

**Step 3 — Configure and Run**

```bash
cp examples/geomesh.yaml config/geomesh.yaml
docker compose up -d
docker compose logs -f
```

The default `docker-compose.yml` exposes port `53` (UDP/TCP) for DNS and port `8080` for the REST API.

---

### Method 3: Compile from Source

Requires Go 1.23 or later.

```bash
git clone https://github.com/itsazni/geomesh.git
cd geomesh

make build
make download-geoip MAXMIND_LICENSE_KEY=your_key

cp examples/geomesh.yaml config/geomesh.yaml
./bin/geomesh config/
```

---

## DNS Delegation (Subdomain Setup)

If you manage your primary domain (e.g., `example.com`) through a provider like Cloudflare but want to use GeoMesh to route a specific subdomain (e.g., `geomesh.example.com`), you need to set up **NS Delegation**.

1. **Get your GeoMesh Server IP**: Assume your GeoMesh instance is running on `203.0.113.50`.
2. **Create an A record**: In Cloudflare (or your current DNS provider), create an `A` record pointing your nameserver hostname to your GeoMesh IP.
   - **Type:** `A`
   - **Name:** `ns1.geomesh` (this results in `ns1.geomesh.example.com`)
   - **Content:** `203.0.113.50`
   - **Proxy status:** **DNS only** (Grey cloud - essential for DNS traffic)
3. **Delegate the Subdomain**: Create an `NS` record delegating the target subdomain to your new nameserver.
   - **Type:** `NS`
   - **Name:** `geomesh` (this delegates the `geomesh.example.com` zone)
   - **Content:** `ns1.geomesh.example.com`

Now, any DNS query for `geomesh.example.com` (or subdomains like `play.geomesh.example.com`) will be forwarded to your GeoMesh server.

---

## Configuration

GeoMesh is configured with YAML. Below is a minimal working example followed by a full annotated example.

**Minimal:**
```yaml
server:
  listen: ":53"

zones:
  - name: example.com
    routes:
      - name: www
        policy: roundrobin
        endpoints:
          - address: 1.2.3.4
          - address: 5.6.7.8
```

**Full annotated example:**
```yaml
server:
  listen: ":53"        # DNS address (UDP + TCP)
  api: ":8080"         # REST API address, remove to disable
  log_level: info      # debug | info | warn | error
  log_json: false      # true = structured JSON logs

geoip:
  city_db: /app/geoip/GeoLite2-City.mmdb
  asn_db:  /app/geoip/GeoLite2-ASN.mmdb

zones:
  - name: example.com
    routes:
      - name: play
        policy: geo
        default: us       # fallback region if no match
        ttl: 60
        regions:
          - name: asia
            countries: [ID, SG, MY, TH, VN]
            endpoints:
              - address: 103.10.10.10
                weight: 2
              - address: 103.10.10.11
                weight: 1
            health_check:
              type: tcp
              port: 80
              interval: 30
              timeout: 5
              retries: 3
          - name: us
            countries: [US, CA, MX]
            endpoints:
              - address: 45.10.10.10
            health_check:
              type: http
              port: 80
              path: /health
              interval: 30
              timeout: 5
              retries: 3

      - name: api
        policy: latency
        ttl: 30
        endpoints:
          - address: 1.2.3.4
            latitude: 37.7749
            longitude: -122.4194   # San Francisco
          - address: 5.6.7.8
            latitude: 1.3521
            longitude: 103.8198    # Singapore
        health_check:
          type: tcp
          port: 443
          interval: 15
          timeout: 3
          retries: 2

      - name: mc
        policy: failover
        ttl: 30
        endpoints:
          - address: 204.77.4.220
          - address: 185.207.166.156
        health_check:
          type: mcjava
          port: 25565
          interval: 15
          timeout: 5
          retries: 3
```

---

## Routing Policies

| Policy | Description |
|---|---|
| `geo` | Routes the client to the region matching their IP address (country, continent, ASN, or CIDR). Falls back to the `default` region if no match is found. Requires GeoIP databases. |
| `latency` | Routes to the endpoint with the shortest great-circle distance from the client using the Haversine formula. Each endpoint must have `latitude` and `longitude` set. Requires GeoIP databases. |
| `weighted` | Distributes traffic proportionally. An endpoint with `weight: 3` receives three times more traffic than one with `weight: 1`. |
| `roundrobin` | Cycles through all healthy endpoints sequentially. |
| `random` | Selects a healthy endpoint at random for each query. |
| `failover` | Always uses the first healthy endpoint. Falls back to the next one only when the primary is unhealthy. |
| `first` | Always returns the first endpoint in the list, regardless of health status. |

---

## Health Check Types

Health checks can be defined at the route level (applies to all endpoints), the region level, or the endpoint level (highest priority).

| Type | Protocol | Description |
|---|---|---|
| `tcp` | TCP | Opens a TCP connection. Healthy if the connection succeeds within the timeout. |
| `http` | HTTP | Sends a `GET` request to the configured `path`. Healthy if a `2xx` status code is returned. |
| `https` | HTTPS | Same as `http` but over TLS. |
| `udp` | UDP | Sends a probe packet. Healthy if no ICMP port-unreachable error is received. |
| `mcjava` | TCP | Sends a Minecraft Java Edition Server List Ping handshake. |
| `mcbedrock` | UDP | Sends a Minecraft Bedrock Edition (RakNet) Server List Ping handshake. |

Health check configuration fields: `type`, `port`, `path` (HTTP only), `interval` (seconds), `timeout` (seconds), `retries`.

---

## Multi-File Configuration

GeoMesh can load a single YAML file or an entire directory. When a directory is given, all `.yaml` and `.yml` files within it are merged into a single configuration. This is useful for splitting zones across files.

```
config/
├── server.yaml
├── example.com.yaml
└── internal.net.yaml
```

```bash
geomesh config/
```

---

## Hot Reload

GeoMesh watches the configuration path for changes using `fsnotify`. When a file is modified, GeoMesh automatically re-parses and validates the new configuration, updates the zone registry, and restarts health checks — all without a restart. Changes take effect within 200 milliseconds.

A manual reload can also be triggered via the REST API:

```bash
curl -X POST http://localhost:8080/api/reload
```

---

## REST API

When `server.api` is configured, the following endpoints are available:

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/status` | Server version and uptime |
| `GET` | `/api/zones` | All loaded zones and their routes |
| `GET` | `/api/health` | Current health status of all monitored endpoints |
| `POST` | `/api/reload` | Trigger a manual configuration reload |
| `GET` | `/metrics` | Prometheus metrics |
