package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	if cfg.DiscoveryDepth != 3 {
		t.Errorf("expected DiscoveryDepth 3, got %d", cfg.DiscoveryDepth)
	}
	if cfg.DiscoveryInterval != 5*time.Minute {
		t.Errorf("expected DiscoveryInterval 5m, got %v", cfg.DiscoveryInterval)
	}
}

func TestDefaultProjectConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultProjectConfig()
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if !cfg.Collect.Git {
		t.Error("expected Collect.Git to be true")
	}
	if len(cfg.Ignore) != 4 {
		t.Errorf("expected 4 default ignore patterns, got %d", len(cfg.Ignore))
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("missing file returns defaults", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "missing.yaml")

		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := DefaultConfig()
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("expected %v, got %v", expected, cfg)
		}
	})

	t.Run("valid yaml", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "config.yaml")

		yamlData := []byte("discoveryDepth: 10\ndiscoveryInterval: 10m\nroots:\n  - /some/path\n")
		if err := os.WriteFile(path, yamlData, 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.DiscoveryDepth != 10 {
			t.Errorf("expected depth 10, got %d", cfg.DiscoveryDepth)
		}
		if cfg.DiscoveryInterval != 10*time.Minute {
			t.Errorf("expected interval 10m, got %v", cfg.DiscoveryInterval)
		}
		if len(cfg.Roots) != 1 || cfg.Roots[0] != "/some/path" {
			t.Errorf("expected root /some/path, got %v", cfg.Roots)
		}
	})
}

func TestLoadProjectConfig(t *testing.T) {
	t.Parallel()

	t.Run("partial yaml merges with defaults", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, ".logy.yaml")

		yamlData := []byte("enabled: false\ncollect:\n  git: false\n")
		if err := os.WriteFile(path, yamlData, 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadProjectConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Enabled != false {
			t.Error("expected Enabled to be false")
		}
		if cfg.Collect.Git != false {
			t.Error("expected Collect.Git to be false")
		}
		if cfg.Collect.Terminal != true {
			t.Error("expected Collect.Terminal to be true")
		}
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.Roots = []string{t.TempDir()} // TempDir is always absolute

	if err := Validate(cfg); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	invalidCfg := cfg
	invalidCfg.Roots = []string{"relative/path"}
	if err := Validate(invalidCfg); err == nil {
		t.Error("expected error for relative path")
	}

	invalidCfg2 := cfg
	invalidCfg2.DiscoveryDepth = 0
	if err := Validate(invalidCfg2); err == nil {
		t.Error("expected error for zero depth")
	}
}

func TestValidateProject(t *testing.T) {
	t.Parallel()
	cfg := DefaultProjectConfig()

	if err := ValidateProject(cfg); err != nil {
		t.Errorf("expected valid project config, got error: %v", err)
	}

	invalidCfg := cfg
	invalidCfg.Ignore = []string{"["} // invalid glob
	if err := ValidateProject(invalidCfg); err == nil {
		t.Error("expected error for invalid glob pattern")
	}
}

func TestSaveConfigAtomicRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := DefaultConfig()
	cfg.Roots = []string{t.TempDir()}
	cfg.DiscoveryDepth = 4

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() overwrite error = %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.DiscoveryDepth != 4 {
		t.Fatalf("DiscoveryDepth = %d, want 4", loaded.DiscoveryDepth)
	}
	if len(loaded.Roots) != 1 || loaded.Roots[0] != cfg.Roots[0] {
		t.Fatalf("Roots = %v, want %v", loaded.Roots, cfg.Roots)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed Config
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(cfg, parsed) {
		t.Errorf("round trip failed.\nExpected: %+v\nGot: %+v", cfg, parsed)
	}
}
