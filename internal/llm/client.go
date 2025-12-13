package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GenerateParams struct {
	Prompt string

	MaxTokens   int
	Temperature float64

	TopP     float64
	TopK     int
	MinP     float64
	TopA     float64
	TypicalP float64
	TFS      float64

	RepetitionPenalty      float64
	RepetitionPenaltyRange int
	PresencePenalty        float64
	FrequencyPenalty       float64

	MirostatMode int
	MirostatTau  float64
	MirostatEta  float64

	DryMultiplier       float64
	DryBase             float64
	DryAllowedLength    int
	DrySequenceBreakers []string

	SmoothingFactor float64
	EpsilonCutoff   float64
	EtaCutoff       float64

	DynamicTemperatureMin      float64
	DynamicTemperatureMax      float64
	DynamicTemperatureExponent float64

	Seed          int
	StopSequences []string
}

type Client interface {
	Generate(ctx context.Context, params GenerateParams) (string, error)
}

type HTTPClient struct {
	endpointURL string
	apiKey      string
	httpClient  *http.Client
}

func NewHTTPClient(endpointURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		endpointURL: endpointURL,
		apiKey:      apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type request struct {
	Prompt string `json:"prompt"`

	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`

	TopP     float64 `json:"top_p,omitempty"`
	TopK     int     `json:"top_k,omitempty"`
	MinP     float64 `json:"min_p,omitempty"`
	TopA     float64 `json:"top_a,omitempty"`
	TypicalP float64 `json:"typical_p,omitempty"`
	TFS      float64 `json:"tfs,omitempty"`

	RepetitionPenalty      float64 `json:"repetition_penalty,omitempty"`
	RepetitionPenaltyRange int     `json:"repetition_penalty_range,omitempty"`
	PresencePenalty        float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty       float64 `json:"frequency_penalty,omitempty"`

	MirostatMode int     `json:"mirostat_mode,omitempty"`
	MirostatTau  float64 `json:"mirostat_tau,omitempty"`
	MirostatEta  float64 `json:"mirostat_eta,omitempty"`

	DryMultiplier       float64  `json:"dry_multiplier,omitempty"`
	DryBase             float64  `json:"dry_base,omitempty"`
	DryAllowedLength    int      `json:"dry_allowed_length,omitempty"`
	DrySequenceBreakers []string `json:"dry_sequence_breakers,omitempty"`

	SmoothingFactor float64 `json:"smoothing_factor,omitempty"`
	EpsilonCutoff   float64 `json:"epsilon_cutoff,omitempty"`
	EtaCutoff       float64 `json:"eta_cutoff,omitempty"`

	DynamicTemperatureMin      float64 `json:"dynamic_temperature_min,omitempty"`
	DynamicTemperatureMax      float64 `json:"dynamic_temperature_max,omitempty"`
	DynamicTemperatureExponent float64 `json:"dynamic_temperature_exponent,omitempty"`

	Seed          int      `json:"seed,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

type response struct {
	Text string `json:"text"`
}

func (c *HTTPClient) Generate(ctx context.Context, params GenerateParams) (string, error) {
	reqBody := request{
		Prompt: params.Prompt,

		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,

		TopP:     params.TopP,
		TopK:     params.TopK,
		MinP:     params.MinP,
		TopA:     params.TopA,
		TypicalP: params.TypicalP,
		TFS:      params.TFS,

		RepetitionPenalty:      params.RepetitionPenalty,
		RepetitionPenaltyRange: params.RepetitionPenaltyRange,
		PresencePenalty:        params.PresencePenalty,
		FrequencyPenalty:       params.FrequencyPenalty,

		MirostatMode: params.MirostatMode,
		MirostatTau:  params.MirostatTau,
		MirostatEta:  params.MirostatEta,

		DryMultiplier:       params.DryMultiplier,
		DryBase:             params.DryBase,
		DryAllowedLength:    params.DryAllowedLength,
		DrySequenceBreakers: params.DrySequenceBreakers,

		SmoothingFactor: params.SmoothingFactor,
		EpsilonCutoff:   params.EpsilonCutoff,
		EtaCutoff:       params.EtaCutoff,

		DynamicTemperatureMin:      params.DynamicTemperatureMin,
		DynamicTemperatureMax:      params.DynamicTemperatureMax,
		DynamicTemperatureExponent: params.DynamicTemperatureExponent,

		Seed:          params.Seed,
		StopSequences: params.StopSequences,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpointURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("LLM error (status %d): %s", resp.StatusCode, string(body))
	}

	var llmResp response
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return strings.TrimSpace(llmResp.Text), nil
}
