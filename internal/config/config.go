package config

import (
	"fmt"
	"os"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Upstreams []Upstream   `yaml:"upstreams"`
	Routers   RouterConfig `yaml:"routers"`
	Output    OutputConfig `yaml:"output"`
	Server    ServerConfig `yaml:"server"`
}

// Upstream represents a Traefik instance to poll
type Upstream struct {
	Name      string `yaml:"name"`       // Identifier for this upstream
	AdminURL  string `yaml:"admin_url"`  // Traefik admin/dashboard URL (e.g., http://100.64.1.2:8080)
	ServerURL string `yaml:"server_url"` // Full URL to route traffic to (e.g., http://100.64.1.2:80)
}

// RouterConfig defines how to filter and configure routers
type RouterConfig struct {
	Selector RouterSelector `yaml:"selector"`
	Defaults RouterDefaults `yaml:"defaults"`
}

// RouterSelector defines filtering criteria for routers
type RouterSelector struct {
	Provider string `yaml:"provider"`
	Status   string `yaml:"status"`
}

// RouterDefaults defines default values applied to all generated routers
type RouterDefaults struct {
	EntryPoints []string                 `yaml:"entrypoints"`
	Middlewares []string                 `yaml:"middlewares"`
	TLS         *dynamic.RouterTLSConfig `yaml:"tls"`
}

// OutputConfig defines where to output the aggregated configuration
type OutputConfig struct {
	HTTP HTTPOutput `yaml:"http"`
	File FileOutput `yaml:"file"`
}

// HTTPOutput configuration for HTTP server
type HTTPOutput struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// FileOutput configuration for file-based output
type FileOutput struct {
	Enabled  bool          `yaml:"enabled"`
	Path     string        `yaml:"path"`
	Interval time.Duration `yaml:"interval"`
}

// ServerConfig defines server behavior
type ServerConfig struct {
	PollInterval time.Duration `yaml:"poll_interval"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Server.PollInterval == 0 {
		cfg.Server.PollInterval = 10 * time.Second
	}

	if cfg.Output.HTTP.Path == "" {
		cfg.Output.HTTP.Path = "/config"
	}

	if cfg.Output.File.Interval == 0 {
		cfg.Output.File.Interval = 30 * time.Second
	}

	if cfg.Routers.Selector.Status == "" {
		cfg.Routers.Selector.Status = "enabled"
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Upstreams) == 0 {
		return fmt.Errorf("at least one upstream must be configured")
	}

	for i, upstream := range c.Upstreams {
		if upstream.Name == "" {
			return fmt.Errorf("upstream %d: name is required", i)
		}
		if upstream.AdminURL == "" {
			return fmt.Errorf("upstream %s: admin_url is required", upstream.Name)
		}
		if upstream.ServerURL == "" {
			return fmt.Errorf("upstream %s: server_url is required", upstream.Name)
		}
	}

	if !c.Output.HTTP.Enabled && !c.Output.File.Enabled {
		return fmt.Errorf("at least one output method (HTTP or File) must be enabled")
	}

	if c.Output.HTTP.Enabled && c.Output.HTTP.Port <= 0 {
		return fmt.Errorf("HTTP output port must be specified")
	}

	if c.Output.File.Enabled && c.Output.File.Path == "" {
		return fmt.Errorf("file output path must be specified")
	}

	return nil
}
