package config

import (
	"errors"
	"path/filepath"
)

// Validate checks that the global configuration is valid.
func Validate(cfg Config) error {
	for _, root := range cfg.Roots {
		if !filepath.IsAbs(root) {
			return errors.New("root path must be absolute")
		}
	}
	if cfg.DiscoveryDepth <= 0 {
		return errors.New("discovery depth must be greater than 0")
	}
	if cfg.DiscoveryInterval <= 0 {
		return errors.New("discovery interval must be greater than 0")
	}
	return nil
}

// ValidateProject checks that the project configuration is valid.
func ValidateProject(cfg ProjectConfig) error {
	for _, pattern := range cfg.Ignore {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return err
		}
	}
	return nil
}
