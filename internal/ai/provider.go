package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"logy/internal/config"
)

// Prompt is the sanitized message pair sent to an AI provider.
type Prompt struct {
	System string
	User   string
}

// Provider generates text from a prompt. Implementations must not persist API keys.
type Provider interface {
	Generate(ctx context.Context, prompt Prompt) (string, error)
}

// HTTPProvider calls an OpenAI-compatible chat completions endpoint.
type HTTPProvider struct {
	Endpoint string
	Model    string
	APIKey   string // from env at construction, not persisted
	Client   *http.Client
}

// NewHTTPProvider builds a provider from config. When AI is disabled it returns (nil, nil).
// When enabled, endpoint and API key (via KeyEnv) are required.
func NewHTTPProvider(cfg config.AIConfig) (*HTTPProvider, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("ai endpoint is required when AI is enabled")
	}
	keyEnv := strings.TrimSpace(cfg.KeyEnv)
	if keyEnv == "" {
		return nil, fmt.Errorf("ai keyEnv is required when AI is enabled")
	}
	apiKey := strings.TrimSpace(os.Getenv(keyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("ai api key missing: set environment variable %s", keyEnv)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "default"
	}
	return &HTTPProvider{
		Endpoint: endpoint,
		Model:    model,
		APIKey:   apiKey,
		Client:   &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Generate sends the prompt to the configured endpoint and returns the assistant text.
func (p *HTTPProvider) Generate(ctx context.Context, prompt Prompt) (string, error) {
	if p == nil {
		return "", fmt.Errorf("ai provider is nil")
	}
	body, err := json.Marshal(chatRequest{
		Model: p.Model,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read ai response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode ai response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("ai error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai response contained no choices")
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("ai response was empty")
	}
	return text, nil
}

// AnswerWithOptionalAI returns the deterministic answer unless useAI is set and a
// provider is wired. On provider failure it returns the deterministic answer with a note.
func AnswerWithOptionalAI(deterministic string, useAI bool, provider Provider, prompt Prompt) string {
	if !useAI || provider == nil {
		return deterministic
	}
	out, err := provider.Generate(context.Background(), prompt)
	if err != nil {
		return deterministic + "\n(note: AI unavailable; showing offline answer)"
	}
	return out
}
