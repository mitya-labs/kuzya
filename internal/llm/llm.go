// Package llm — клиент к llama-server (OpenAI-совместимый /v1/chat/completions).
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Client struct {
	URL  string
	HTTP *http.Client
}

func New() *Client {
	url := os.Getenv("KUZYA_URL")
	if url == "" {
		url = "http://127.0.0.1:8080/v1/chat/completions"
	}
	return &Client{URL: url, HTTP: &http.Client{Timeout: 5 * time.Minute}}
}

// Chat отправляет историю и схемы инструментов, возвращает ответ модели.
func (c *Client) Chat(msgs []Message, tools []map[string]any) (Message, error) {
	// max_tokens: ответ с вызовом инструмента короткий; предел отсекает зацикленную генерацию (были запросы по 155–300 с).
	body := map[string]any{"messages": msgs, "temperature": 0.2, "max_tokens": 256}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	b, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}
	resp, err := c.HTTP.Post(c.URL, "application/json", bytes.NewReader(b))
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("llama-server: %s: %s", resp.Status, data)
	}
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Message{}, err
	}
	if len(out.Choices) == 0 {
		return Message{}, fmt.Errorf("llama-server: пустой ответ")
	}
	return out.Choices[0].Message, nil
}
