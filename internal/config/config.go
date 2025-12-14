package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
)

var envPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type Config struct {
	Port   string            `toml:"port"`
	Twilio *Twilio           `toml:"-"`
	Memory *Memory           `toml:"-"`
	Models map[string]*Model `toml:"-"`
}

type Memory struct {
	Enabled          bool    `toml:"enabled"`
	StoragePath      string  `toml:"storage_path"`
	SummaryThreshold float64 `toml:"summary_threshold"`
	SummaryPrompt    string  `toml:"summary_prompt"`
	ProfileInterval  int     `toml:"profile_interval"`
	ProfilePrompt    string  `toml:"profile_prompt"`
	CharacterInterval int    `toml:"character_interval"`
}

type Twilio struct {
	AccountSID string `toml:"account_sid"`
	AuthToken  string `toml:"auth_token"`
	FromNumber string `toml:"from_number"`
	Model      string `toml:"model"`
}

type Model struct {
	URL                  string `toml:"url"`
	APIKey               string `toml:"api_key"`
	CharacterDescription string `toml:"character_description"`

	ContextSize int `toml:"context_size"`
	MaxTokens   int `toml:"max_tokens"`
	Temperature float64 `toml:"temperature"`

	TopP     float64 `toml:"top_p"`
	TopK     int     `toml:"top_k"`
	MinP     float64 `toml:"min_p"`
	TopA     float64 `toml:"top_a"`
	TypicalP float64 `toml:"typical_p"`
	TFS      float64 `toml:"tfs"`

	RepetitionPenalty      float64 `toml:"repetition_penalty"`
	RepetitionPenaltyRange int     `toml:"repetition_penalty_range"`
	PresencePenalty        float64 `toml:"presence_penalty"`
	FrequencyPenalty       float64 `toml:"frequency_penalty"`

	MirostatMode int     `toml:"mirostat_mode"`
	MirostatTau  float64 `toml:"mirostat_tau"`
	MirostatEta  float64 `toml:"mirostat_eta"`

	DryMultiplier      float64  `toml:"dry_multiplier"`
	DryBase            float64  `toml:"dry_base"`
	DryAllowedLength   int      `toml:"dry_allowed_length"`
	DrySequenceBreakers []string `toml:"dry_sequence_breakers"`

	SmoothingFactor float64 `toml:"smoothing_factor"`
	EpsilonCutoff   float64 `toml:"epsilon_cutoff"`
	EtaCutoff       float64 `toml:"eta_cutoff"`

	DynamicTemperatureMin      float64 `toml:"dynamic_temperature_min"`
	DynamicTemperatureMax      float64 `toml:"dynamic_temperature_max"`
	DynamicTemperatureExponent float64 `toml:"dynamic_temperature_exponent"`

	Seed          int      `toml:"seed"`
	StopSequences []string `toml:"stop_sequences"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := expandEnv(string(data))

	var raw map[string]any
	if _, err := toml.Decode(expanded, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg := &Config{
		Port:   "8080",
		Models: make(map[string]*Model),
	}

	if port, ok := raw["port"].(string); ok {
		cfg.Port = port
		delete(raw, "port")
	}

	if twilioSection, ok := raw["twilio"].(map[string]any); ok {
		cfg.Twilio = &Twilio{}
		if v, ok := twilioSection["account_sid"].(string); ok {
			cfg.Twilio.AccountSID = v
		}
		if v, ok := twilioSection["auth_token"].(string); ok {
			cfg.Twilio.AuthToken = v
		}
		if v, ok := twilioSection["from_number"].(string); ok {
			cfg.Twilio.FromNumber = v
		}
		if v, ok := twilioSection["model"].(string); ok {
			cfg.Twilio.Model = v
		}
		delete(raw, "twilio")
	}

	if memorySection, ok := raw["memory"].(map[string]any); ok {
		cfg.Memory = &Memory{
			StoragePath:      "memory.db",
			SummaryThreshold: 0.6,
			ProfileInterval:  25,
			CharacterInterval: 40,
		}
		if v, ok := memorySection["enabled"].(bool); ok {
			cfg.Memory.Enabled = v
		}
		if v, ok := memorySection["storage_path"].(string); ok {
			cfg.Memory.StoragePath = v
		}
		if v, ok := memorySection["summary_threshold"].(float64); ok {
			cfg.Memory.SummaryThreshold = v
		}
		if v, ok := memorySection["summary_prompt"].(string); ok {
			cfg.Memory.SummaryPrompt = v
		}
		if v, ok := memorySection["profile_interval"].(int64); ok {
			cfg.Memory.ProfileInterval = int(v)
		}
		if v, ok := memorySection["profile_prompt"].(string); ok {
			cfg.Memory.ProfilePrompt = v
		}
		if v, ok := memorySection["character_interval"].(int64); ok {
			cfg.Memory.CharacterInterval = int(v)
		}
		delete(raw, "memory")
	}

	for name, v := range raw {
		section, ok := v.(map[string]any)
		if !ok {
			continue
		}

		model := &Model{}

		if url, ok := section["url"].(string); ok {
			model.URL = url
		}
		if apiKey, ok := section["api_key"].(string); ok {
			model.APIKey = apiKey
		}
		if v, ok := section["character_description"].(string); ok {
			model.CharacterDescription = v
		}

		if v, ok := section["context_size"].(int64); ok {
			model.ContextSize = int(v)
		}
		if v, ok := section["max_tokens"].(int64); ok {
			model.MaxTokens = int(v)
		}
		if v, ok := section["temperature"].(float64); ok {
			model.Temperature = v
		}

		if v, ok := section["top_p"].(float64); ok {
			model.TopP = v
		}
		if v, ok := section["top_k"].(int64); ok {
			model.TopK = int(v)
		}
		if v, ok := section["min_p"].(float64); ok {
			model.MinP = v
		}
		if v, ok := section["top_a"].(float64); ok {
			model.TopA = v
		}
		if v, ok := section["typical_p"].(float64); ok {
			model.TypicalP = v
		}
		if v, ok := section["tfs"].(float64); ok {
			model.TFS = v
		}

		if v, ok := section["repetition_penalty"].(float64); ok {
			model.RepetitionPenalty = v
		}
		if v, ok := section["repetition_penalty_range"].(int64); ok {
			model.RepetitionPenaltyRange = int(v)
		}
		if v, ok := section["presence_penalty"].(float64); ok {
			model.PresencePenalty = v
		}
		if v, ok := section["frequency_penalty"].(float64); ok {
			model.FrequencyPenalty = v
		}

		if v, ok := section["mirostat_mode"].(int64); ok {
			model.MirostatMode = int(v)
		}
		if v, ok := section["mirostat_tau"].(float64); ok {
			model.MirostatTau = v
		}
		if v, ok := section["mirostat_eta"].(float64); ok {
			model.MirostatEta = v
		}

		if v, ok := section["dry_multiplier"].(float64); ok {
			model.DryMultiplier = v
		}
		if v, ok := section["dry_base"].(float64); ok {
			model.DryBase = v
		}
		if v, ok := section["dry_allowed_length"].(int64); ok {
			model.DryAllowedLength = int(v)
		}
		if arr, ok := section["dry_sequence_breakers"].([]any); ok {
			for _, s := range arr {
				if str, ok := s.(string); ok {
					model.DrySequenceBreakers = append(model.DrySequenceBreakers, str)
				}
			}
		}

		if v, ok := section["smoothing_factor"].(float64); ok {
			model.SmoothingFactor = v
		}
		if v, ok := section["epsilon_cutoff"].(float64); ok {
			model.EpsilonCutoff = v
		}
		if v, ok := section["eta_cutoff"].(float64); ok {
			model.EtaCutoff = v
		}

		if v, ok := section["dynamic_temperature_min"].(float64); ok {
			model.DynamicTemperatureMin = v
		}
		if v, ok := section["dynamic_temperature_max"].(float64); ok {
			model.DynamicTemperatureMax = v
		}
		if v, ok := section["dynamic_temperature_exponent"].(float64); ok {
			model.DynamicTemperatureExponent = v
		}

		if v, ok := section["seed"].(int64); ok {
			model.Seed = int(v)
		}
		if arr, ok := section["stop_sequences"].([]any); ok {
			for _, s := range arr {
				if str, ok := s.(string); ok {
					model.StopSequences = append(model.StopSequences, str)
				}
			}
		}

		if model.URL == "" {
			return nil, fmt.Errorf("model %q: url is required", name)
		}

		cfg.Models[name] = model
	}

	if len(cfg.Models) == 0 {
		return nil, fmt.Errorf("no models defined")
	}

	return cfg, nil
}

func expandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}
