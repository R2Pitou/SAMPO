package config

import (
	"encoding/json"
	"os"
)

// StorageProvider represents a configured physical/virtual storage backend.
type StorageProvider struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`       // local, smb, webdav, git, s3, etc.
	Path       string            `json:"path"`       // root directory or connection string
	Capabilities map[string]string `json:"capabilities"` // e.g. latency: "low", cost: "none", reliability: "high"
}

// Policy represents user intent configuration.
type Policy struct {
	ID       string `json:"id"`
	Type     string `json:"type"`     // replicate, archive, cleanup
	Target   string `json:"target"`   // object, project, tag
	Value    string `json:"value"`    // e.g. "3" for replicas, "30d" for age, "hot" for tier
	Priority int    `json:"priority"` // evaluation priority
}

// Config defines the complete configuration for MAS-H.
type Config struct {
	Providers []StorageProvider `json:"providers"`
	Policies  []Policy          `json:"policies"`
	Services  []string          `json:"services"` // active services to run (e.g. tuoni, seshat, observer)
	Port      int               `json:"port"`     // default gateway port
	DBPath    string            `json:"db_path"`  // sqlite database filepath (optional)
}

// LoadConfig reads and decodes the configuration from a JSON file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
