package handler

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/terminally-online/tunnel/internal/config"
	"github.com/terminally-online/tunnel/internal/llm"
	"github.com/terminally-online/tunnel/internal/memory"
	"github.com/terminally-online/tunnel/internal/sms"
)

type SMSHandler struct {
	smsClient    *sms.Client
	llmClient    llm.Client
	model        *config.Model
	memoryStore  *memory.Store
	memoryConfig *config.Memory
}

func NewSMSHandler(smsClient *sms.Client, llmClient llm.Client, model *config.Model, memoryStore *memory.Store, memoryConfig *config.Memory) *SMSHandler {
	return &SMSHandler{
		smsClient:    smsClient,
		llmClient:    llmClient,
		model:        model,
		memoryStore:  memoryStore,
		memoryConfig: memoryConfig,
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

	prompt := body
	var session *memory.Session

	if h.memoryStore != nil && h.memoryConfig != nil && h.memoryConfig.Enabled {
		var err error
		session, err = h.memoryStore.Get(msg.From)
		if err != nil {
			log.Printf("Failed to load session for %s: %v", msg.From, err)
		} else {
			prompt = session.BuildPrompt(body)
		}
	}

	params := llm.GenerateParams{
		Prompt: prompt,

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

	if session != nil {
		session.AddTurn("user", body)
		session.AddTurn("assistant", response)

		if session.TurnCount >= h.memoryConfig.SummaryInterval {
			h.summarizeSession(session)
		}

		if err := h.memoryStore.Save(session); err != nil {
			log.Printf("Failed to save session for %s: %v", msg.From, err)
		}
	}

	if err := h.smsClient.Send(msg.From, response); err != nil {
		log.Printf("Failed to send SMS to %s: %v", msg.From, err)
	}
}

func (h *SMSHandler) summarizeSession(session *memory.Session) {
	prompt := memory.BuildSummarizationPrompt(session, h.memoryConfig.SummaryPrompt)

	params := llm.GenerateParams{
		Prompt:      prompt,
		MaxTokens:   h.model.MaxTokens,
		Temperature: 0.3,
	}

	summary, err := h.llmClient.Generate(context.Background(), params)
	if err != nil {
		log.Printf("Failed to summarize session %s: %v", session.ID, err)
		return
	}

	session.Summary = strings.TrimSpace(summary)
	session.ClearHistory()
	log.Printf("Summarized session %s", session.ID)
}
