package config

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultConfig returns a Config initialized with sensible default values.
func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	dataDir := "~/.logy/data"
	if err == nil {
		dataDir = filepath.Join(home, ".logy", "data")
	}

	return Config{
		DataDir:           dataDir,
		Roots:             []string{},
		DiscoveryDepth:    3,
		DiscoveryInterval: 5 * time.Minute,
		AI: AIConfig{
			Enabled:  false,
			Endpoint: "",
			Model:    "",
			KeyEnv:   "",
		},
	}
}

// DefaultProjectConfig returns a ProjectConfig initialized with sensible default values.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Enabled: true,
		Collect: CollectConfig{
			Git:        true,
			Terminal:   true,
			Filesystem: false,
			Agents:     true,
		},
		Store: StoreConfig{
			Diffs:       false,
			Transcripts: false,
		},
		Ignore: []string{
			".env*",
			"secrets/**",
			"node_modules/**",
			"vendor/**",
		},
		Redact: []string{
			"token",
			"password",
			"api_key",
		},
	}
}
