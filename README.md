# traefik-fed

A lightweight Go service that federates routing configurations from multiple Traefik instances into a single unified configuration.

## Problem

When running Traefik on multiple hosts (e.g., Docker hosts with Traefik auto-discovery), you need a central/public-facing Traefik to route traffic to these distributed instances. Manually configuring routes for each service is tedious and error-prone.

## Solution

traefik-fed polls multiple upstream Traefik API endpoints, discovers their routers, and generates a unified Traefik configuration. Your central Traefik can consume this via HTTP or File provider.

## Architecture

```
Internet → Central Traefik (consumes traefik-fed config)
              ↓
         traefik-fed (polls APIs)
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
Host1 Traefik      Host2 Traefik
    ↓                   ↓
  Services          Services
```

## Features

- **Multi-upstream support**: Poll multiple Traefik API endpoints
- **Flexible filtering**: Filter routers by provider (docker, file, kubernetes) and status (enabled)
- **Dual output modes**: Serve via HTTP endpoint and/or write to file
- **Automatic service creation**: Generates loadbalancer services pointing to upstream Traefik instances
- **Clean naming**: Router names are prefixed with upstream identifier (e.g., `host1-myapp`)

## Installation

```bash
# Build from source
go build -o traefik-fed ./cmd/traefik-fed

# Or use go install
go install github.com/chickenzord/traefik-fed/cmd/traefik-fed@latest
```

## Configuration

Create a `config.yaml`:

```yaml
upstreams:
  - name: host1
    admin_url: http://192.168.1.10:8080
    server_url: http://192.168.1.10:80

  - name: host2
    admin_url: http://192.168.1.11:8080
    server_url: http://192.168.1.11:80

routers:
  selector:
    provider: docker       # Filter by provider (optional)
    status: enabled        # Filter by status (default: enabled)

output:
  http:
    enabled: true
    port: 8080
    path: /config

  file:
    enabled: true
    path: /etc/traefik/dynamic/federation.yml
    interval: 30s

server:
  poll_interval: 10s
```

### Configuration Reference

**Upstreams**:
- `name`: Unique identifier for this upstream (used as router name prefix)
- `admin_url`: Traefik admin/dashboard URL (typically port 8080, `/api` is appended automatically)
- `server_url`: URL where the central Traefik should forward traffic

**Router Selector**:
- `provider`: Filter routers by provider (`docker`, `file`, `kubernetes`, etc.) - optional
- `status`: Filter by status (`enabled` or `disabled`) - defaults to `enabled`
- Note: Routers from the `internal` provider (API, dashboard) are always excluded

**Output**:
- `http.enabled`: Enable HTTP endpoint
- `http.port`: Port to listen on
- `http.path`: Path for config endpoint
- `file.enabled`: Enable file output
- `file.path`: Path to write configuration file
- `file.interval`: How often to write file

**Server**:
- `poll_interval`: How often to poll upstream Traefik APIs

## Usage

```bash
# Run with config file
./traefik-fed --config config.yaml

# Config file defaults to config.yaml in current directory
./traefik-fed
```

## Integration with Traefik

### HTTP Provider (Recommended)

Configure your central Traefik to use the HTTP provider:

```yaml
# traefik.yml (central/public Traefik)
providers:
  http:
    endpoint: "http://localhost:8080/config"
    pollInterval: "10s"
```

### File Provider

Alternatively, use the file provider:

```yaml
# traefik.yml (central/public Traefik)
providers:
  file:
    filename: /etc/traefik/dynamic/federation.yml
    watch: true
```

## Generated Configuration Example

Given upstream routers:
- Host1: `webapp` with rule `Host('app.example.com')`
- Host2: `api` with rule `Host('api.example.com')`

traefik-fed generates:

```yaml
http:
  routers:
    host1-webapp:
      rule: "Host(`app.example.com`)"
      service: host1-traefik
      entryPoints: [websecure]
      tls:
        certResolver: letsencrypt

    host2-api:
      rule: "Host(`api.example.com`)"
      service: host2-traefik
      entryPoints: [websecure]
      tls:
        certResolver: letsencrypt

  services:
    host1-traefik:
      loadBalancer:
        servers:
          - url: "http://192.168.1.10:80"

    host2-traefik:
      loadBalancer:
        servers:
          - url: "http://192.168.1.11:80"
```

## API Endpoints

When HTTP output is enabled:

- `GET /config` - Returns aggregated configuration (YAML by default)
- `GET /config?format=json` - Returns configuration as JSON
- `GET /health` - Health check endpoint

## Use Cases

- **Multi-host Docker setup**: Each host runs Traefik with Docker provider, central Traefik routes to appropriate host
- **Hybrid infrastructure**: Combine routes from multiple Traefik instances (Docker, Kubernetes, etc.)
- **Edge routing**: Central Traefik at edge routes to internal Traefik instances

## Requirements

- Go 1.21+ (for building)
- Traefik v3.x upstreams
- Network access to upstream Traefik API endpoints

## License

MIT
