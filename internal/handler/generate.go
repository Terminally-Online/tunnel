package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/terminally-online/tunnel/internal/config"
	"github.com/terminally-online/tunnel/internal/llm"
)

type GenerateHandler struct {
	models  map[string]*config.Model
	clients map[string]llm.Client
}

func NewGenerateHandler(models map[string]*config.Model) *GenerateHandler {
	clients := make(map[string]llm.Client)
	for name, model := range models {
		clients[name] = llm.NewHTTPClient(model.URL, model.APIKey)
	}
	return &GenerateHandler{
		models:  models,
		clients: clients,
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
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

	params := llm.GenerateParams{
		Prompt: req.Prompt,

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

	h.writeJSON(w, http.StatusOK, generateResponse{Text: text})
}

func (h *GenerateHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *GenerateHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
