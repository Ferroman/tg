package llm

import (
	"context"
	"fmt"

	"github.com/bf/tg/internal/config"
)

// Enrichment represents the LLM's suggestions for a task
type Enrichment struct {
	Description string   `json:"description"`
	Beacons     []string `json:"beacons"`     // e.g., ["b.great.dev", "b.organized"]
	Directions  []string `json:"directions"`  // e.g., ["d.sw.design", "d.test.write"]
	Project     string   `json:"project"`
	Priority    string   `json:"priority"`    // H, M, L or empty
	Due         string   `json:"due"`         // taskwarrior due format or empty
	Effort      string   `json:"effort"`      // E (easy), N (normal), D (difficult)
	Impact      string   `json:"impact"`      // H (high), M (medium), L (low)
	Estimate    string   `json:"estimate"`    // 15m, 30m, 1h, 2h, 4h, 8h, 2d
	Fun         string   `json:"fun"`         // H (high/fun), M (medium), L (low/boring)
	IsWaste     bool     `json:"is_waste"`    // true if task doesn't align with any beacon
	Reasoning   string   `json:"reasoning"`   // explanation for the enrichment
}

// Provider is the interface for LLM backends
type Provider interface {
	Enrich(ctx context.Context, taskDesc string, beacons []config.Beacon, projects []config.Project) (*Enrichment, error)
}

// New creates a new LLM provider based on config
func New(cfg *config.Config) (Provider, error) {
	switch cfg.LLM.Provider {
	case "anthropic":
		apiKey := cfg.GetAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set (configured env: %s)", cfg.LLM.APIKeyEnv)
		}
		return NewAnthropic(apiKey, cfg.LLM.Model), nil
	case "openai":
		apiKey := cfg.GetAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set (configured env: %s)", cfg.LLM.APIKeyEnv)
		}
		return NewOpenAI(apiKey, cfg.LLM.Model), nil
	case "ollama":
		baseURL := cfg.LLM.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return NewOllama(baseURL, cfg.LLM.Model), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLM.Provider)
	}
}
