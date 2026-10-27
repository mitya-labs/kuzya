package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatRemote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization=%q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "remote-model" {
			t.Errorf("model=%v", body["model"])
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": "ответ"},
		}}})
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client()}
	msg, err := c.ChatRemote([]Message{{Role: "user", Content: "привет"}}, srv.URL, "test-key", "remote-model")
	if err != nil || msg.Content != "ответ" {
		t.Fatalf("msg=%q err=%v", msg.Content, err)
	}
}
