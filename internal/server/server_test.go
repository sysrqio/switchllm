package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sysrqio/switchllm/internal/config"
	"github.com/sysrqio/switchllm/internal/metrics"
	"github.com/sysrqio/switchllm/internal/router"
	"github.com/sysrqio/switchllm/internal/upstream"
)

type stubClient struct {
	live   bool
	calls  []string
	failOn map[string]error
}

func (s *stubClient) Live() bool { return s.live }

func (s *stubClient) ChatCompletion(req upstream.ChatCompletionRequest) (*upstream.ChatCompletionResponse, error) {
	s.calls = append(s.calls, req.Model)
	if err, ok := s.failOn[req.Model]; ok {
		return nil, err
	}
	fr := "stop"
	resp, err := upstream.NewMock().ChatCompletion(req)
	if err != nil {
		return nil, err
	}
	resp.Model = req.Model
	resp.Choices[0].Message.Content = "ok"
	resp.Choices[0].FinishReason = &fr
	return resp, nil
}

func (s *stubClient) ChatCompletionStream(req upstream.ChatCompletionRequest, onChunk func([]byte) error) error {
	resp, err := s.ChatCompletion(req)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(map[string]interface{}{
		"choices": []map[string]interface{}{
			{"delta": map[string]string{"content": resp.Choices[0].Message.Content}},
		},
	})
	return onChunk(b)
}

func TestHealthzAndMetrics(t *testing.T) {
	cfg, err := config.Load("../../switchllm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	r := router.New(cfg, nil)
	s := New(r, upstream.NewMock(), metrics.New(true), slog.Default(), false)

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("healthz status %d", res.StatusCode)
	}

	res2, err := http.Get(ts.URL + "/v1/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	var snap map[string]interface{}
	if err := json.NewDecoder(res2.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if _, ok := snap["requests_total"]; !ok {
		t.Fatal("missing requests_total")
	}
}

func TestChatCompletions_dryRun(t *testing.T) {
	cfg, _ := config.Load("../../switchllm.yaml")
	r := router.New(cfg, nil)
	m := metrics.New(true)
	s := New(r, upstream.NewMock(), m, slog.Default(), true)

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	payload := `{"model":"auto","messages":[{"role":"user","content":"hello"}]}`
	res, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, body)
	}
	var out upstream.ChatCompletionResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Choices[0].Message.Content == "" {
		t.Fatal("empty completion")
	}
	if m.Snapshot().RequestsTotal != 1 {
		t.Fatalf("metrics requests=%d", m.Snapshot().RequestsTotal)
	}
}

func TestFallbackOn429(t *testing.T) {
	cfg, _ := config.Load("../../switchllm.yaml")
	r := router.New(cfg, nil)
	stub := &stubClient{
		live: true,
		failOn: map[string]error{
			cfg.Routing.SimpleModel: &upstream.HTTPError{Status: 429, Body: "rate limit"},
		},
	}
	s := New(r, stub, metrics.New(false), slog.Default(), false)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	payload := `{"model":"auto","messages":[{"role":"user","content":"hi"}]}`
	res, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if len(stub.calls) < 2 {
		t.Fatalf("expected fallback call, calls=%v", stub.calls)
	}
	if stub.calls[len(stub.calls)-1] != cfg.Routing.PremiumModel {
		t.Fatalf("last model %s want premium %s", stub.calls[len(stub.calls)-1], cfg.Routing.PremiumModel)
	}
}
