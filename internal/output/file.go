package output

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"gopkg.in/yaml.v3"
)

// FileWriter writes the aggregated configuration to a file
type FileWriter struct {
	path     string
	interval time.Duration
	logger   *slog.Logger
}

// NewFileWriter creates a new file writer
func NewFileWriter(path string, interval time.Duration, logger *slog.Logger) *FileWriter {
	return &FileWriter{
		path:     path,
		interval: interval,
		logger:   logger,
	}
}

// Start starts the periodic file writing
func (w *FileWriter) Start(configChan <-chan *dynamic.HTTPConfiguration) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var currentConfig *dynamic.HTTPConfiguration

	// Write initial config if available
	select {
	case config := <-configChan:
		currentConfig = config
		if err := w.writeConfig(config); err != nil {
			w.logger.Error("failed to write initial config", "error", err)
		}
	case <-time.After(1 * time.Second):
		// No initial config yet
	}

	for {
		select {
		case config := <-configChan:
			currentConfig = config
			if err := w.writeConfig(config); err != nil {
				w.logger.Error("failed to write config", "error", err)
			}
		case <-ticker.C:
			if currentConfig != nil {
				if err := w.writeConfig(currentConfig); err != nil {
					w.logger.Error("failed to write config on timer", "error", err)
				}
			}
		}
	}
}

// writeConfig writes the configuration to the file
func (w *FileWriter) writeConfig(config *dynamic.HTTPConfiguration) error {
	// Ensure directory exists
	dir := filepath.Dir(w.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tmpPath := w.path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	// Wrap in http key for Traefik format
	output := map[string]interface{}{
		"http": config,
	}

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, w.path); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	w.logger.Info("wrote configuration to file", "path", w.path)
	return nil
}
