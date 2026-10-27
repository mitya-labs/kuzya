package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kuzya/internal/llm"
	"kuzya/internal/tools"
)

// Фейковый сервер: первый ответ — вызов инструмента, второй — текст second.
func fake(t *testing.T, second string) *llm.Client {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		msg := map[string]any{"role": "assistant", "content": second}
		if n == 1 {
			msg = map[string]any{"role": "assistant", "tool_calls": []map[string]any{{
				"id": "1", "type": "function",
				"function": map[string]any{"name": "echo", "arguments": `{"text":"hi"}`},
			}}}
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": msg}}})
	}))
	t.Cleanup(srv.Close)
	return &llm.Client{URL: srv.URL, HTTP: srv.Client()}
}

func TestToolLoopAndConfirm(t *testing.T) {
	ran := 0
	tools.Register(tools.Tool{
		Name: "echo", Description: "test", Dangerous: true,
		Run: func(args map[string]any) (string, error) { ran++; return args["text"].(string), nil },
	})
	a := New(fake(t, "готово"), func(string) bool { return false })
	out, err := a.Run("сделай")
	if err != nil || out != "Отменено: «echo» не выполнен." {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if ran != 0 {
		t.Fatal("опасный инструмент выполнен без подтверждения")
	}
	if last := a.History[len(a.History)-2]; last.Content != "пользователь отказал" || a.History[len(a.History)-1].Role != "assistant" {
		t.Fatalf("tool result=%q", last.Content)
	}
}

func TestEmptyReplyShowsToolResult(t *testing.T) {
	tools.Register(tools.Tool{
		Name: "echo", Description: "test", Dangerous: true,
		Run: func(args map[string]any) (string, error) { return args["text"].(string), nil },
	})
	a := New(fake(t, ""), func(string) bool { return true })
	out, err := a.Run("сделай")
	if err != nil || out != "hi" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestTrimKeepsUserStart(t *testing.T) {
	a := New(nil, nil)
	for i := 0; i < 40; i++ {
		a.History = append(a.History, llm.Message{Role: "user", Content: "q"}, llm.Message{Role: "tool", Content: "r"})
	}
	a.trim()
	if a.History[0].Role != "system" || a.History[1].Role != "user" {
		t.Fatalf("плохая обрезка: %v", a.History[:2])
	}
}

func TestTextOnlyPrefix(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": "привет"},
		}}})
	}))
	t.Cleanup(srv.Close)
	a := New(&llm.Client{URL: srv.URL, HTTP: srv.Client()}, nil)
	out, err := a.Run("? привет, как дела?")
	if err != nil || out != "привет" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, ok := body["tools"]; ok {
		t.Fatal("в режиме «?» инструменты отправляться не должны")
	}
}

func TestRemoteNotConfigured(t *testing.T) {
	t.Setenv("KUZYA_REMOTE_URL", "")
	t.Setenv("KUZYA_REMOTE_KEY", "")
	t.Setenv("KUZYA_REMOTE_MODEL", "")
	a := New(nil, nil)
	out, err := a.Run("@привет")
	if err != nil || !strings.Contains(out, "не настроена") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestQuestionUsesRemote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": "привет"},
		}}})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KUZYA_REMOTE_URL", srv.URL)
	t.Setenv("KUZYA_REMOTE_KEY", "k")
	t.Setenv("KUZYA_REMOTE_MODEL", "m")
	a := New(&llm.Client{}, nil)
	out, err := a.Run("? как дела")
	if err != nil || out != "привет" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
