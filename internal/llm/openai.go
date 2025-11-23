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

type OpenAI struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAI(apiKey, model string) *OpenAI {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

type openaiRequest struct {
	Model    string           `json:"model"`
	Messages []openaiMessage  `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAI) Enrich(ctx context.Context, taskDesc string, beacons []config.Beacon, projects []config.Project) (*Enrichment, error) {
	prompt := buildPrompt(taskDesc, beacons, projects)

	reqBody := openaiRequest{
		Model: o.model,
		Messages: []openaiMessage{
			{Role: "system", Content: "You are a task enrichment assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openaiResp openaiResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openaiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", openaiResp.Error.Message)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	return parseEnrichmentResponse(openaiResp.Choices[0].Message.Content)
}
