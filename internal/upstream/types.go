package upstream

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message,omitempty"`
		Delta        ChatMessage `json:"delta,omitempty"`
		FinishReason *string     `json:"finish_reason"`
	} `json:"choices"`
}

type Client interface {
	ChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error)
	ChatCompletionStream(req ChatCompletionRequest, onChunk func([]byte) error) error
	Live() bool
}
