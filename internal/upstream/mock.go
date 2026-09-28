package upstream

import (
	"encoding/json"
	"fmt"
	"time"
)

type MockClient struct{}

func NewMock() *MockClient {
	return &MockClient{}
}

func (m *MockClient) Live() bool { return false }

func (m *MockClient) ChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	content := fmt.Sprintf("[switchllm dry-run] model=%s reply to %d messages", req.Model, len(req.Messages))
	fr := "stop"
	return &ChatCompletionResponse{
		ID:      "chatcmpl-mock",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []struct {
			Index        int         `json:"index"`
			Message      ChatMessage `json:"message,omitempty"`
			Delta        ChatMessage `json:"delta,omitempty"`
			FinishReason *string     `json:"finish_reason"`
		}{
			{Index: 0, Message: ChatMessage{Role: "assistant", Content: content}, FinishReason: &fr},
		},
	}, nil
}

func (m *MockClient) ChatCompletionStream(req ChatCompletionRequest, onChunk func([]byte) error) error {
	resp, err := m.ChatCompletion(req)
	if err != nil {
		return err
	}
	content := resp.Choices[0].Message.Content
	chunk := map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion.chunk",
		"created": resp.Created,
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{"role": "assistant", "content": content},
			},
		},
	}
	b, _ := json.Marshal(chunk)
	if err := onChunk(append(b, '\n')); err != nil {
		return err
	}
	fr := "stop"
	done := map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion.chunk",
		"created": resp.Created,
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{"index": 0, "delta": map[string]string{}, "finish_reason": fr},
		},
	}
	b2, _ := json.Marshal(done)
	return onChunk(append(b2, '\n'))
}
