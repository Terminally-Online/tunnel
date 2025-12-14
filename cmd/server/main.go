package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/terminally-online/tunnel/internal/config"
	"github.com/terminally-online/tunnel/internal/handler"
	"github.com/terminally-online/tunnel/internal/llm"
	"github.com/terminally-online/tunnel/internal/memory"
	"github.com/terminally-online/tunnel/internal/sms"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded %d model(s)", len(cfg.Models))
	for name := range cfg.Models {
		log.Printf("  - %s", name)
	}

	var memoryStore *memory.Store
	if cfg.Memory != nil && cfg.Memory.Enabled {
		memoryStore, err = memory.NewStore(cfg.Memory.StoragePath)
		if err != nil {
			log.Fatalf("Failed to initialize memory store: %v", err)
		}
		log.Printf("Memory enabled, storing at %s (summarize at %.0f%% context)", cfg.Memory.StoragePath, cfg.Memory.SummaryThreshold*100)
	}

	mux := http.NewServeMux()

	mux.Handle("/generate", handler.NewGenerateHandler(cfg.Models, memoryStore, cfg.Memory))

	if cfg.Twilio != nil {
		model, ok := cfg.Models[cfg.Twilio.Model]
		if !ok {
			log.Fatalf("Twilio model %q not found", cfg.Twilio.Model)
		}

		smsClient := sms.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.FromNumber)
		llmClient := llm.NewHTTPClient(model.URL, model.APIKey)
		smsHandler := handler.NewSMSHandler(smsClient, llmClient, model, memoryStore, cfg.Memory)

		mux.Handle("/sms", smsHandler)
		log.Printf("Twilio SMS enabled, routing to model %q", cfg.Twilio.Model)
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Starting tunnel on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
