package traefik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// Client handles communication with Traefik API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Traefik API client
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

// Observability represents observability settings
type Observability struct {
	AccessLogs     bool   `json:"accessLogs"`
	Metrics        bool   `json:"metrics"`
	Tracing        bool   `json:"tracing"`
	TraceVerbosity string `json:"traceVerbosity"`
}

// RouterInfo represents a router from the Traefik API
type RouterInfo struct {
	EntryPoints   []string       `json:"entryPoints"`
	Middlewares   []string       `json:"middlewares,omitempty"`
	Service       string         `json:"service"`
	Rule          string         `json:"rule"`
	RuleSyntax    string         `json:"ruleSyntax"`
	Priority      int            `json:"priority"`
	Observability *Observability `json:"observability,omitempty"`
	Status        string         `json:"status"`
	Using         []string       `json:"using"`
	Name          string         `json:"name"`
	Provider      string         `json:"provider"`
	TLS           *dynamic.RouterTLSConfig `json:"tls,omitempty"`
}

// GetRouters fetches all HTTP routers from the Traefik API
func (c *Client) GetRouters() ([]*RouterInfo, error) {
	url := fmt.Sprintf("%s/http/routers", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Traefik API returns an array of routers
	var routers []*RouterInfo
	if err := json.Unmarshal(body, &routers); err != nil {
		return nil, fmt.Errorf("failed to parse routers: %w", err)
	}

	return routers, nil
}

// FilterRouters filters routers based on provider and status
func FilterRouters(routers []*RouterInfo, provider, status string) []*RouterInfo {
	filtered := make([]*RouterInfo, 0)
	for _, router := range routers {
		// Always exclude internal provider
		if router.Provider == "internal" {
			continue
		}

		// Filter by provider if specified
		if provider != "" && router.Provider != provider {
			continue
		}

		// Filter by status if specified
		if status != "" && router.Status != status {
			continue
		}

		filtered = append(filtered, router)
	}

	return filtered
}
