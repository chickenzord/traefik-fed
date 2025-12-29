package output

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"gopkg.in/yaml.v3"
)

// HTTPServer serves the aggregated configuration via HTTP
type HTTPServer struct {
	port   int
	path   string
	logger *slog.Logger

	mu     sync.RWMutex
	config *dynamic.HTTPConfiguration
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port int, path string, logger *slog.Logger) *HTTPServer {
	return &HTTPServer{
		port:   port,
		path:   path,
		logger: logger,
		config: &dynamic.HTTPConfiguration{},
	}
}

// UpdateConfig updates the cached configuration
func (s *HTTPServer) UpdateConfig(config *dynamic.HTTPConfiguration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.path, s.handleConfig)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("starting HTTP server", "addr", addr, "path", s.path)

	return http.ListenAndServe(addr, mux)
}

// handleConfig serves the aggregated configuration
func (s *HTTPServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	// Support both JSON and YAML based on Accept header
	acceptHeader := r.Header.Get("Accept")

	if acceptHeader == "application/json" || r.URL.Query().Get("format") == "json" {
		s.serveJSON(w, config)
	} else {
		s.serveYAML(w, config)
	}
}

// serveJSON serves configuration as JSON
func (s *HTTPServer) serveJSON(w http.ResponseWriter, config *dynamic.HTTPConfiguration) {
	w.Header().Set("Content-Type", "application/json")

	// Wrap in http key for Traefik format
	output := map[string]interface{}{
		"http": config,
	}

	if err := json.NewEncoder(w).Encode(output); err != nil {
		s.logger.Error("failed to encode JSON", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// serveYAML serves configuration as YAML
func (s *HTTPServer) serveYAML(w http.ResponseWriter, config *dynamic.HTTPConfiguration) {
	w.Header().Set("Content-Type", "application/x-yaml")

	// Wrap in http key for Traefik format
	output := map[string]interface{}{
		"http": config,
	}

	if err := yaml.NewEncoder(w).Encode(output); err != nil {
		s.logger.Error("failed to encode YAML", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleHealth provides a health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
