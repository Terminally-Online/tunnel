# Tunnel

A single passage to any text generation backend.

## How It Works

Requests enter the tunnel via HTTP or SMS and exit through whichever model you've configured. Your applications don't need to know what's on the other side.

**Entry Points**
- HTTP API for direct integration
- Twilio SMS for conversational interfaces

**Exits**
- Local models (llama.cpp, Ollama, vLLM)
- Self-hosted inference
- Cloud endpoints

## Models

Each model is a named configuration with its own endpoint and sampler settings. Define as many as you need:

```toml
[sms]
url = "${SMS_MODEL_URL}"
temperature = 0.9
max_tokens = 512

[analyst]
url = "${ANALYST_MODEL_URL}"
temperature = 0.3
max_tokens = 2048
```

Requests specify which model to use. The tunnel handles the rest.

## Usage

```bash
cp config.example.toml config.toml
go build -o tunnel ./cmd/server
./tunnel
```

## License

MIT
