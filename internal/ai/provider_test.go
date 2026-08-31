package ai

import (
	"os"
	"testing"

	"logy/internal/config"
)

func TestNewHTTPProviderRequiresEndpointAndKeyWhenEnabled(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		p, err := NewHTTPProvider(config.AIConfig{Enabled: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p != nil {
			t.Fatalf("expected nil provider when disabled, got %#v", p)
		}
	})

	t.Run("enabled missing endpoint", func(t *testing.T) {
		_, err := NewHTTPProvider(config.AIConfig{
			Enabled: true,
			KeyEnv:  "LOGY_TEST_AI_KEY_MISSING_ENDPOINT",
		})
		if err == nil {
			t.Fatal("expected error for missing endpoint")
		}
	})

	t.Run("enabled missing key", func(t *testing.T) {
		t.Setenv("LOGY_TEST_AI_KEY_EMPTY", "")
		_, err := NewHTTPProvider(config.AIConfig{
			Enabled:  true,
			Endpoint: "https://example.com/v1/chat/completions",
			Model:    "test-model",
			KeyEnv:   "LOGY_TEST_AI_KEY_EMPTY",
		})
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("enabled with key from env", func(t *testing.T) {
		const envName = "LOGY_TEST_AI_KEY_OK"
		t.Setenv(envName, "secret-not-for-db")
		p, err := NewHTTPProvider(config.AIConfig{
			Enabled:  true,
			Endpoint: "https://example.com/v1/chat/completions",
			Model:    "test-model",
			KeyEnv:   envName,
		})
		if err != nil {
			t.Fatalf("NewHTTPProvider() error = %v", err)
		}
		if p == nil {
			t.Fatal("expected provider")
		}
		if p.APIKey != "secret-not-for-db" {
			t.Fatalf("APIKey = %q, want env value", p.APIKey)
		}
		if got := os.Getenv(envName); got == "" {
			t.Fatal("test env key unexpectedly empty")
		}
	})
}
