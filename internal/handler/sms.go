package handler

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/terminally-online/tunnel/internal/config"
	"github.com/terminally-online/tunnel/internal/llm"
	"github.com/terminally-online/tunnel/internal/sms"
)

type SMSHandler struct {
	smsClient *sms.Client
	llmClient llm.Client
	model     *config.Model
}

func NewSMSHandler(smsClient *sms.Client, llmClient llm.Client, model *config.Model) *SMSHandler {
	return &SMSHandler{
		smsClient: smsClient,
		llmClient: llmClient,
		model:     model,
	}
}

func (h *SMSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	msg, err := sms.ParseInbound(r)
	if err != nil {
		log.Printf("Failed to parse inbound SMS: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("SMS from %s: %s", msg.From, msg.Body)

	go h.processMessage(msg)

	sms.WriteTwiMLResponse(w)
}

func (h *SMSHandler) processMessage(msg *sms.InboundMessage) {
	body := strings.TrimSpace(msg.Body)
	if body == "" {
		return
	}

	params := llm.GenerateParams{
		Prompt: body,

		MaxTokens:   h.model.MaxTokens,
		Temperature: h.model.Temperature,

		TopP:     h.model.TopP,
		TopK:     h.model.TopK,
		MinP:     h.model.MinP,
		TopA:     h.model.TopA,
		TypicalP: h.model.TypicalP,
		TFS:      h.model.TFS,

		RepetitionPenalty:      h.model.RepetitionPenalty,
		RepetitionPenaltyRange: h.model.RepetitionPenaltyRange,
		PresencePenalty:        h.model.PresencePenalty,
		FrequencyPenalty:       h.model.FrequencyPenalty,

		MirostatMode: h.model.MirostatMode,
		MirostatTau:  h.model.MirostatTau,
		MirostatEta:  h.model.MirostatEta,

		DryMultiplier:       h.model.DryMultiplier,
		DryBase:             h.model.DryBase,
		DryAllowedLength:    h.model.DryAllowedLength,
		DrySequenceBreakers: h.model.DrySequenceBreakers,

		SmoothingFactor: h.model.SmoothingFactor,
		EpsilonCutoff:   h.model.EpsilonCutoff,
		EtaCutoff:       h.model.EtaCutoff,

		DynamicTemperatureMin:      h.model.DynamicTemperatureMin,
		DynamicTemperatureMax:      h.model.DynamicTemperatureMax,
		DynamicTemperatureExponent: h.model.DynamicTemperatureExponent,

		Seed:          h.model.Seed,
		StopSequences: h.model.StopSequences,
	}

	response, err := h.llmClient.Generate(context.Background(), params)
	if err != nil {
		log.Printf("LLM error for %s: %v", msg.From, err)
		return
	}

	if err := h.smsClient.Send(msg.From, response); err != nil {
		log.Printf("Failed to send SMS to %s: %v", msg.From, err)
	}
}
