# switchllm

**switchllm** is a local, OpenAI-compatible HTTP proxy that routes each chat request to a **simple** or **premium** model using lightweight heuristics (keywords, length, code blocks). Bring your own API keys (BYOK), keep traffic on your machine, and optionally log estimated savings when simple models suffice.

- **License:** MIT  
- **Module:** `github.com/sysrqio/switchllm`  
- **Telemetry:** none — no usage data is uploaded

## Quick start

```bash
go build -o switchllm ./cmd/switchllm

# Mock upstream (no API key)
./switchllm start --dry-run --show-savings --config switchllm.yaml

# Live mode
export OPENAI_API_KEY=sk-...
./switchllm start --port 8080 --threshold 0.35
```

Point any OpenAI-compatible client at `http://127.0.0.1:8080/v1`.

## CLI

```text
switchllm start [flags]

  --port          Listen port (default 8080)
  --config        YAML config path (default switchllm.yaml)
  --threshold     Premium routing threshold 0–1 (default 0.35)
  --log-level     debug | info | warn | error
  --show-savings  Log estimated $ savings when using the simple model
  --dry-run       Mock upstream (for local dev and CI)
```

If `OPENAI_API_KEY` is unset, the server automatically uses mock upstream (same as `--dry-run`).

`ANTHROPIC_API_KEY` is recognized for future provider support; the MVP forwards chat completions to the configured OpenAI-compatible endpoint.

## HTTP API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness JSON `{"status":"ok"}` |
| `GET` | `/v1/metrics` | Routing counters and estimated savings |
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat (JSON or SSE stream) |

### Chat routing

- Set `"model": "auto"` or `"model": "switchllm-auto"` to enable heuristic routing.
- Explicit model names are passed through unchanged.
- On **429** or **5xx** from upstream in live mode, switchllm **retries once** on the configured premium model.

### Example (non-streaming)

```bash
curl -s http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "auto",
    "messages": [{"role":"user","content":"Summarize this in one line."}]
  }'
```

### Example (streaming)

```bash
curl -N http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "auto",
    "stream": true,
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

## Configuration

See [`switchllm.yaml`](switchllm.yaml):

- `routing.simple_model` / `routing.premium_model`
- `routing.threshold` (overridable via `--threshold`)
- `complexity_keywords` — substring hints that increase the complexity score

## Development

```bash
make test
make build
make run
```

Docker:

```bash
docker build -t switchllm:local .
docker run --rm -p 8080:8080 -e OPENAI_API_KEY switchllm:local start --dry-run
```

## How routing works (MVP)

The classifier scores each user message in `[0, 1]` using:

- Message length buckets  
- Code-fence / multiline heuristics  
- Configurable keyword hits  
- Light punctuation / multi-question signals  

Scores **≥ threshold** → premium model; otherwise → simple model.

This MVP intentionally avoids ONNX or remote scoring services so you can run and test fully offline.

## Security notes

- API keys are read from the environment only.
- No external analytics or phone-home endpoints are implemented.
- Run behind your own network controls in production.
