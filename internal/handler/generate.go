package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/terminally-online/tunnel/internal/config"
	"github.com/terminally-online/tunnel/internal/llm"
	"github.com/terminally-online/tunnel/internal/memory"
)

type GenerateHandler struct {
	models       map[string]*config.Model
	clients      map[string]llm.Client
	memoryStore  *memory.Store
	memoryConfig *config.Memory
}

func NewGenerateHandler(models map[string]*config.Model, memoryStore *memory.Store, memoryConfig *config.Memory) *GenerateHandler {
	clients := make(map[string]llm.Client)
	for name, model := range models {
		clients[name] = llm.NewHTTPClient(model.URL, model.APIKey)
	}
	return &GenerateHandler{
		models:       models,
		clients:      clients,
		memoryStore:  memoryStore,
		memoryConfig: memoryConfig,
	}
}

type generateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	SessionID string `json:"session_id,omitempty"`
}

type generateResponse struct {
	Text string `json:"text"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		h.writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	model, ok := h.models[req.Model]
	if !ok {
		h.writeError(w, http.StatusBadRequest, "unknown model")
		return
	}

	client := h.clients[req.Model]

	prompt := req.Prompt
	var session *memory.Session

	useMemory := req.SessionID != "" && h.memoryStore != nil && h.memoryConfig != nil && h.memoryConfig.Enabled
	if useMemory {
		var err error
		session, err = h.memoryStore.Get(req.SessionID)
		if err != nil {
			log.Printf("Failed to load session %s: %v", req.SessionID, err)
		} else {
			prompt = session.BuildPrompt(req.Prompt, model.CharacterDescription, model.ContextSize, model.MaxTokens)
		}
	}

	params := llm.GenerateParams{
		Prompt: prompt,

		MaxTokens:   model.MaxTokens,
		Temperature: model.Temperature,

		TopP:     model.TopP,
		TopK:     model.TopK,
		MinP:     model.MinP,
		TopA:     model.TopA,
		TypicalP: model.TypicalP,
		TFS:      model.TFS,

		RepetitionPenalty:      model.RepetitionPenalty,
		RepetitionPenaltyRange: model.RepetitionPenaltyRange,
		PresencePenalty:        model.PresencePenalty,
		FrequencyPenalty:       model.FrequencyPenalty,

		MirostatMode: model.MirostatMode,
		MirostatTau:  model.MirostatTau,
		MirostatEta:  model.MirostatEta,

		DryMultiplier:       model.DryMultiplier,
		DryBase:             model.DryBase,
		DryAllowedLength:    model.DryAllowedLength,
		DrySequenceBreakers: model.DrySequenceBreakers,

		SmoothingFactor: model.SmoothingFactor,
		EpsilonCutoff:   model.EpsilonCutoff,
		EtaCutoff:       model.EtaCutoff,

		DynamicTemperatureMin:      model.DynamicTemperatureMin,
		DynamicTemperatureMax:      model.DynamicTemperatureMax,
		DynamicTemperatureExponent: model.DynamicTemperatureExponent,

		Seed:          model.Seed,
		StopSequences: model.StopSequences,
	}

	text, err := client.Generate(context.Background(), params)
	if err != nil {
		log.Printf("LLM error [%s]: %v", req.Model, err)
		h.writeError(w, http.StatusBadGateway, "upstream error")
		return
	}

	if session != nil {
		session.AddTurn("user", req.Prompt)
		session.AddTurn("assistant", text)

		if model.ContextSize > 0 && session.HistoryRatio(model.ContextSize, model.MaxTokens, model.CharacterDescription) >= h.memoryConfig.SummaryThreshold {
			h.summarizeSession(session, client, model)
		}

		if h.memoryConfig.ProfileInterval > 0 && session.TotalTurnCount > 0 && session.TotalTurnCount%h.memoryConfig.ProfileInterval == 0 {
			h.extractProfile(session, client, model)
		}

		if h.shouldEvolveCharacter(session, model) {
			h.evolveCharacter(session, client, model)
		}

		if err := h.memoryStore.Save(session); err != nil {
			log.Printf("Failed to save session %s: %v", req.SessionID, err)
		}
	}

	h.writeJSON(w, http.StatusOK, generateResponse{Text: text})
}

func (h *GenerateHandler) summarizeSession(session *memory.Session, client llm.Client, model *config.Model) {
	prompt := memory.BuildSummarizationPrompt(session, h.memoryConfig.SummaryPrompt)

	params := llm.GenerateParams{
		Prompt:      prompt,
		MaxTokens:   model.MaxTokens,
		Temperature: 0.3,
	}

	summary, err := client.Generate(context.Background(), params)
	if err != nil {
		log.Printf("Failed to summarize session %s: %v", session.ID, err)
		return
	}

	session.Summary = strings.TrimSpace(summary)
	session.ClearHistory()
	log.Printf("Summarized session %s", session.ID)
}

func (h *GenerateHandler) extractProfile(session *memory.Session, client llm.Client, model *config.Model) {
	prompt := memory.BuildProfileExtractionPrompt(session, h.memoryConfig.ProfilePrompt)

	params := llm.GenerateParams{
		Prompt:      prompt,
		MaxTokens:   model.MaxTokens,
		Temperature: 0.3,
	}

	profile, err := client.Generate(context.Background(), params)
	if err != nil {
		log.Printf("Failed to extract profile for session %s: %v", session.ID, err)
		return
	}

	session.Profile = strings.TrimSpace(profile)
	log.Printf("Extracted profile for session %s", session.ID)
}

func (h *GenerateHandler) shouldEvolveCharacter(session *memory.Session, model *config.Model) bool {
	if model.CharacterDescription == "" {
		return false
	}
	if h.memoryConfig.CharacterInterval <= 0 {
		return false
	}
	if session.TotalTurnCount <= 0 {
		return false
	}
	if session.TotalTurnCount%h.memoryConfig.CharacterInterval != 0 {
		return false
	}
	if !session.CharacterUpdatedAt.IsZero() && time.Since(session.CharacterUpdatedAt) < 24*time.Hour {
		return false
	}
	return true
}

func (h *GenerateHandler) evolveCharacter(session *memory.Session, client llm.Client, model *config.Model) {
	prompt := memory.BuildCharacterEvolutionPrompt(session, model.CharacterDescription)

	params := llm.GenerateParams{
		Prompt:      prompt,
		MaxTokens:   model.MaxTokens,
		Temperature: 0.3,
	}

	deltas, err := client.Generate(context.Background(), params)
	if err != nil {
		log.Printf("Failed to evolve character for session %s: %v", session.ID, err)
		return
	}

	session.CharacterDeltas = strings.TrimSpace(deltas)
	session.CharacterUpdatedAt = time.Now()
	log.Printf("Evolved character for session %s", session.ID)
}

func (h *GenerateHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *GenerateHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
