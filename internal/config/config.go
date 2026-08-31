package config

import (
	"time"
)

// Config represents the global configuration for Logy.
type Config struct {
	DataDir           string        `yaml:"dataDir"`
	Roots             []string      `yaml:"roots"`
	DiscoveryDepth    int           `yaml:"discoveryDepth"`
	DiscoveryInterval time.Duration `yaml:"discoveryInterval"`
	AI                AIConfig      `yaml:"ai"`
}

// AIConfig holds the AI-specific configuration.
type AIConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
	Model    string `yaml:"model"`
	KeyEnv   string `yaml:"keyEnv"`
}

// ProjectConfig represents the per-project configuration.
type ProjectConfig struct {
	Enabled bool          `yaml:"enabled"`
	Collect CollectConfig `yaml:"collect"`
	Store   StoreConfig   `yaml:"store"`
	Ignore  []string      `yaml:"ignore"`
	Redact  []string      `yaml:"redact"`
}

// CollectConfig specifies what data sources to collect.
type CollectConfig struct {
	Git        bool `yaml:"git"`
	Terminal   bool `yaml:"terminal"`
	Filesystem bool `yaml:"filesystem"`
	Agents     bool `yaml:"agents"`
}

// StoreConfig specifies what artifacts to store.
type StoreConfig struct {
	Diffs       bool `yaml:"diffs"`
	Transcripts bool `yaml:"transcripts"`
}
