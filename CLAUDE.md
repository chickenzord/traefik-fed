# traefik-fed

A lightweight Go service that aggregates routing configurations from multiple Traefik instances into a single configuration, enabling a central Traefik to route traffic across a distributed setup.

## Problem Statement

In a multi-host Docker environment with Traefik on each host:
- Each host runs Traefik with Docker provider (auto-discovery)
- A central/public Traefik needs to route to these internal Traefiks
- Manual configuration of routes is tedious and error-prone

## Solution

traefik-fed polls multiple upstream Traefik API endpoints, discovers their routers, and generates a unified Traefik configuration that routes traffic to the appropriate upstream based on hostname rules.

## Architecture
```
Internet → Central Traefik (reads from traefik-fed)
              ↓
         traefik-fed
              ↓ (polls APIs)
    ┌─────────┴─────────┐
    ↓                   ↓
Host1 Traefik      Host2 Traefik
    ↓                   ↓
  Services          Services
```

## Features

- **Multi-upstream support**: Poll multiple Traefik API endpoints
- **Router filtering**: Select routers by name, provider, or custom criteria
- **Dual output**: Serve config via HTTP and/or write to file periodically
- **Automatic service creation**: Generates loadbalancer services pointing to upstream Traefik instances
- **Status filtering**: Filter routers by status (enabled/disabled)

## Configuration
```yaml
# config.yaml (example)
upstreams:
  - name: host1
    admin_url: http://192.168.1.10:8080     # Traefik admin URL
    server_url: http://192.168.1.10:80      # URL to route traffic to

  - name: host2
    admin_url: http://192.168.1.11:8080
    server_url: http://192.168.1.11:80

routers:
  selector:
    # Filter routers to aggregate (optional)
    # Examples: by provider, router name prefix, labels, etc.
    provider: docker
    status: enabled  # Filter by status (default: enabled)

output:
  http:
    enabled: true
    port: 8080
    path: /config

  file:
    enabled: true
    path: /etc/traefik/dynamic/federation.yml
    interval: 30s  # Update interval

server:
  poll_interval: 10s  # How often to poll upstream Traefiks

log:
  format: plain       # Log format: plain, json (default: plain)
  level: info         # Log level: debug, info, warn, error (default: info)
```

## Generated Output Example
```yaml
# Generated Traefik configuration
http:
  routers:
    host1-webapp:  # Original: webapp@docker
      rule: "Host(`app.example.com`)"
      service: host1-traefik
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

    host2-api:  # Original: api@docker
      rule: "Host(`api.example.com`)"
      service: host2-traefik
      entryPoints:
        - websecure
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

## Usage
```bash
# Run with config file
traefik-fed --config config.yaml

# Config file defaults to config.yaml
traefik-fed
```

## Integration with Traefik

**As HTTP Provider:**
```yaml
# Central Traefik config
providers:
  http:
    endpoint: "http://localhost:8080/config"
    pollInterval: "10s"
```

**As File Provider:**
```yaml
# Central Traefik config
providers:
  file:
    filename: /etc/traefik/dynamic/federation.yml
    watch: true
```

## Development Notes

- Language: Go
- Dependencies: Minimal (standard library + YAML/JSON parsing)
- Traefik API: Uses `/api/http/routers` and `/api/http/services` endpoints
- Error handling: Graceful degradation if upstream unavailable
- Logging: Structured logging for debugging

## Implementation Status

- [x] Basic HTTP server with `/config` endpoint
- [x] Poll single upstream Traefik API
- [x] Parse router configurations
- [x] Generate Traefik-compatible YAML/JSON
- [x] Support multiple upstreams
- [x] Add router filtering/selection (provider, status)
- [x] File output with periodic updates
- [x] Configuration validation
- [x] Health check endpoint
- [ ] Metrics/observability

## License

MIT (or your preferred license)
