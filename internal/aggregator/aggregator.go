package aggregator

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/chickenzord/traefik-fed/internal/config"
	"github.com/chickenzord/traefik-fed/internal/traefik"
	traefikDynamic "github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// Aggregator aggregates configurations from multiple Traefik upstreams
type Aggregator struct {
	config  *config.Config
	clients map[string]*traefik.Client
	logger  *slog.Logger
}

// New creates a new aggregator
func New(cfg *config.Config, logger *slog.Logger) *Aggregator {
	clients := make(map[string]*traefik.Client)
	for _, upstream := range cfg.Upstreams {
		// Append /api to admin URL to get the API endpoint
		apiURL := strings.TrimSuffix(upstream.AdminURL, "/") + "/api"
		clients[upstream.Name] = traefik.NewClient(apiURL)
	}

	return &Aggregator{
		config:  cfg,
		clients: clients,
		logger:  logger,
	}
}

// Aggregate fetches and aggregates configurations from all upstreams
func (a *Aggregator) Aggregate() (*traefikDynamic.HTTPConfiguration, error) {
	httpConfig := &traefikDynamic.HTTPConfiguration{
		Routers:  make(map[string]*traefikDynamic.Router),
		Services: make(map[string]*traefikDynamic.Service),
	}

	for _, upstream := range a.config.Upstreams {
		if err := a.aggregateUpstream(upstream, httpConfig); err != nil {
			a.logger.Error("failed to aggregate upstream",
				"upstream", upstream.Name,
				"error", err)
			// Continue with other upstreams even if one fails
			continue
		}
	}

	return httpConfig, nil
}

// aggregateUpstream aggregates configuration from a single upstream
func (a *Aggregator) aggregateUpstream(upstream config.Upstream, httpConfig *traefikDynamic.HTTPConfiguration) error {
	client := a.clients[upstream.Name]

	// Fetch routers from upstream
	routers, err := client.GetRouters()
	if err != nil {
		return fmt.Errorf("failed to fetch routers: %w", err)
	}

	// Apply filters
	filteredRouters := traefik.FilterRouters(routers, a.config.Routers.Selector.Provider, a.config.Routers.Selector.Status)

	a.logger.Info("fetched routers from upstream",
		"upstream", upstream.Name,
		"total", len(routers),
		"filtered", len(filteredRouters))

	// Create a service for this upstream if we have any routers
	if len(filteredRouters) > 0 {
		serviceName := fmt.Sprintf("%s-traefik", upstream.Name)
		httpConfig.Services[serviceName] = &traefikDynamic.Service{
			LoadBalancer: &traefikDynamic.ServersLoadBalancer{
				Servers: []traefikDynamic.Server{
					{
						URL: upstream.ServerURL,
					},
				},
			},
		}

		// Add routers, using router name from API
		for _, router := range filteredRouters {
			// Trim provider suffix from router name (e.g., "memos@docker" -> "memos")
			baseName := router.Name
			if idx := strings.Index(baseName, "@"); idx != -1 {
				baseName = baseName[:idx]
			}

			// Prepend upstream name
			routerName := fmt.Sprintf("%s-%s", upstream.Name, baseName)

			// Create a new router pointing to our upstream service
			newRouter := &traefikDynamic.Router{
				Rule:        router.Rule,
				Service:     serviceName,
				EntryPoints: router.EntryPoints,
				Middlewares: router.Middlewares,
			}

			// Copy TLS configuration if present
			if router.TLS != nil {
				newRouter.TLS = router.TLS
			}

			httpConfig.Routers[routerName] = newRouter
		}
	}

	return nil
}
