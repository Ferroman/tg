package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bf/tg/internal/config"
)

type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(baseURL, model string) *Ollama {
	if model == "" {
		model = "llama3.2"
	}
	return &Ollama{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (o *Ollama) Enrich(ctx context.Context, taskDesc string, beacons []config.Beacon, projects []config.Project) (*Enrichment, error) {
	prompt := buildPrompt(taskDesc, beacons, projects)

	reqBody := ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("API error: %s", ollamaResp.Error)
	}

	return parseEnrichmentResponse(ollamaResp.Response)
}
