package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sysrqio/switchllm/internal/metrics"
	"github.com/sysrqio/switchllm/internal/router"
	"github.com/sysrqio/switchllm/internal/upstream"
)

const (
	// Rough list-price delta used for savings display only (not billing).
	simpleCostUSD  = 0.00015
	premiumCostUSD = 0.0025
)

type Server struct {
	router      *router.Router
	client      upstream.Client
	metrics     *metrics.Collector
	log         *slog.Logger
	showSavings bool
}

func New(r *router.Router, client upstream.Client, m *metrics.Collector, log *slog.Logger, showSavings bool) *Server {
	return &Server{router: r, client: client, metrics: m, log: log, showSavings: showSavings}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /v1/metrics", s.handleMetrics)
	mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	b, err := s.metrics.JSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var req upstream.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	userText := extractUserText(req.Messages)
	decision := s.router.Route(userText)
	save := float64(0)
	if !decision.Premium {
		save = premiumCostUSD - simpleCostUSD
	}
	s.metrics.RecordRoute(decision.Premium, save)

	model := decision.Model
	if req.Model != "" && req.Model != "auto" && req.Model != "switchllm-auto" {
		model = req.Model
	} else {
		req.Model = model
	}

	s.log.Info("route",
		"model", model,
		"complexity", decision.Complexity,
		"premium", decision.Premium,
	)
	if save > 0 && s.showSavings {
		s.log.Info("estimated savings vs premium", "usd", save, "model", model)
	}

	stream := req.Stream || strings.Contains(r.Header.Get("Accept"), "text/event-stream")
	if stream {
		s.streamCompletion(w, req, model)
		return
	}
	s.jsonCompletion(w, req, model)
}

func (s *Server) jsonCompletion(w http.ResponseWriter, req upstream.ChatCompletionRequest, model string) {
	req.Model = model
	resp, err := s.client.ChatCompletion(req)
	if err != nil && s.client.Live() && upstream.IsRetryable(err) {
		s.metrics.RecordFallback()
		req.Model = s.router.PremiumModel()
		s.log.Warn("fallback to premium", "reason", err.Error())
		resp, err = s.client.ChatCompletion(req)
	}
	if err != nil {
		status := http.StatusBadGateway
		var he *upstream.HTTPError
		if upstreamErr, ok := err.(*upstream.HTTPError); ok {
			he = upstreamErr
			status = he.Status
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) streamCompletion(w http.ResponseWriter, req upstream.ChatCompletionRequest, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	req.Model = model
	writeChunk := func(b []byte) error {
		if _, err := w.Write([]byte("data: ")); err != nil {
			return err
		}
		if _, err := w.Write(bytesTrimNewline(b)); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	err := s.client.ChatCompletionStream(req, writeChunk)
	if err != nil && s.client.Live() && upstream.IsRetryable(err) {
		s.metrics.RecordFallback()
		req.Model = s.router.PremiumModel()
		s.log.Warn("stream fallback to premium", "reason", err.Error())
		err = s.client.ChatCompletionStream(req, writeChunk)
	}
	if err != nil {
		s.log.Error("stream error", "err", err)
		return
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func bytesTrimNewline(b []byte) []byte {
	return bytes.TrimSpace(b)
}

func extractUserText(messages []upstream.ChatMessage) string {
	var parts []string
	for _, m := range messages {
		if m.Role == "user" && m.Content != "" {
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, "\n")
}
