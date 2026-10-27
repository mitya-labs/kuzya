package agent

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// localFacts — то, что удалённая модель должна знать о локальной.
const localFacts = `Локальные модели телефона:
- nafanya-1.7b (Qwen3-1.7B + LoRA): 100% на heldout-v4 с правилами, 89% без правил; 1.71 ГБ ОЗУ; 8.9 Дж/запрос; загрузка 3-4 с.
- nafanya-0.8b v8 (Qwen3.5-0.8B + LoRA): 98% на heldout-v4 с правилами (39/40), 88% на v3; 0.92 ГБ; 4.1 Дж; загрузка 2 с.
Правила агента (Refusal, Preroute, Repair) живут в kuzya, не в весах модели.
Локальных инструментов 13: battery_status, wifi_status, wifi_connect, services_running, service_logs, service_restart, nosleep_set, volume_get, volume_set, volume_step, mute_set, brightness_set, brightness_step.`

var localModels = map[string]string{
	"1.7b": "nafanya-1.7b-Q4_K_M",
	"0.8b": "nafanya-0.8b-Q4_K_M",
}

// remoteSchemas — дополнительные инструменты, которые видит только удалённая модель.
func remoteSchemas() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "local_status",
				"description": "Показать текущую локальную модель, состояние сервера и доступные модели телефона",
				"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "local_switch_model",
				"description": "Переключить локальную модель телефона; сервер перезапустится",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"model": map[string]any{"type": "string", "enum": []string{"1.7b", "0.8b"}},
					},
					"required": []string{"model"},
				},
			},
		},
	}
}

// remoteCall выполняет инструменты, доступные только удалённой модели.
// Второй результат — был ли вызов обработан здесь.
func (a *Agent) remoteCall(name string, args map[string]any) (string, bool) {
	switch name {
	case "local_status":
		return localStatus(), true
	case "local_switch_model":
		short, _ := args["model"].(string)
		full, ok := localModels[short]
		if !ok {
			return "неизвестная модель: " + short + " (есть 1.7b и 0.8b)", true
		}
		if a.Confirm != nil && !a.Confirm(name) {
			return "пользователь отказал", true
		}
		if err := switchLocalModel(full); err != nil {
			return "ошибка переключения: " + err.Error(), true
		}
		_ = os.Setenv("KUZYA_MODEL", full)
		return "переключил локальную модель на " + full, true
	}
	return "", false
}

func localStatus() string {
	model := os.Getenv("KUZYA_MODEL")
	if model == "" {
		model = readServerModel()
	}
	if model == "" {
		model = "nafanya-1.7b-Q4_K_M"
	}
	state := "сервер не отвечает"
	client := &http.Client{Timeout: time.Second}
	if resp, err := client.Get("http://127.0.0.1:8080/health"); err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			state = "сервер работает"
		}
	}
	var files []string
	dir := filepath.Join(kuzyaHome(), "models")
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".gguf") {
				files = append(files, e.Name())
			}
		}
		sort.Strings(files)
	}
	return fmt.Sprintf("локальная модель: %s\n%s\nмодели на телефоне: %s\n\n%s",
		model, state, strings.Join(files, ", "), localFacts)
}

func readServerModel() string {
	b, err := os.ReadFile(filepath.Join(kuzyaHome(), "server.model"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func kuzyaHome() string {
	if h := os.Getenv("KUZYA_HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "kuzya")
}

// switchLocalModel перезапускает kuzya-server с выбранной моделью.
func switchLocalModel(name string) error {
	kh := kuzyaHome()
	_ = exec.Command(filepath.Join(kh, "kuzya-server"), "stop").Run()
	cmd := exec.Command(filepath.Join(kh, "kuzya-server"), "start")
	cmd.Env = append(os.Environ(), "KUZYA_MODEL="+name)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
