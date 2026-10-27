package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	"golang.org/x/term"
)

const (
	defaultRemoteURL   = "https://api.deepseek.com/chat/completions"
	defaultRemoteModel = "deepseek-chat"
)

// remoteEnvFile — настройки удалённой модели на телефоне (не в git).
func remoteEnvFile() string { return filepath.Join(kuzyaHomeDir(), "remote.env") }

// loadRemoteConfig подхватывает файл при старте; переменные окружения имеют приоритет.
func loadRemoteConfig() {
	data, err := os.ReadFile(remoteEnvFile())
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// Файл настроек главнее окружения: иначе значения из ~/.bashrc
		// перекрывали бы то, что пользователь только что сохранил.
		_ = os.Setenv(k, v)
	}
}

func saveRemoteConfig(url, key, model string) error {
	path := remoteEnvFile()
	body := fmt.Sprintf("KUZYA_REMOTE_URL=%s\nKUZYA_REMOTE_KEY=%s\nKUZYA_REMOTE_MODEL=%s\n", url, key, model)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	_ = os.Setenv("KUZYA_REMOTE_URL", url)
	_ = os.Setenv("KUZYA_REMOTE_KEY", key)
	_ = os.Setenv("KUZYA_REMOTE_MODEL", model)
	return nil
}

func maskKey(k string) string {
	if k == "" {
		return "(не задан)"
	}
	if len(k) <= 8 {
		return "****"
	}
	return k[:3] + "… (" + fmt.Sprint(len(k)) + " симв.)"
}

func remoteStatus() string {
	url, key, model := os.Getenv("KUZYA_REMOTE_URL"), os.Getenv("KUZYA_REMOTE_KEY"), os.Getenv("KUZYA_REMOTE_MODEL")
	state := "выключена"
	if url != "" && key != "" && model != "" {
		state = "настроена"
	}
	return fmt.Sprintf("удалённая модель: %s\n  URL: %s\n  модель: %s\n  ключ: %s\n  файл: %s",
		state, orDash(url), orDash(model), maskKey(key), remoteEnvFile())
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// remoteSetup спрашивает URL, ключ и модель и сохраняет их в remote.env.
func remoteSetup(l *liner.State) {
	urlDef := os.Getenv("KUZYA_REMOTE_URL")
	if urlDef == "" {
		urlDef = defaultRemoteURL
	}
	modelDef := os.Getenv("KUZYA_REMOTE_MODEL")
	if modelDef == "" {
		modelDef = defaultRemoteModel
	}

	url, err := l.Prompt(fmt.Sprintf("URL [%s]: ", urlDef))
	if err != nil {
		return
	}
	url = strings.TrimSpace(url)
	if url == "" {
		url = urlDef
	}
	model, err := l.Prompt(fmt.Sprintf("Модель [%s]: ", modelDef))
	if err != nil {
		return
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = modelDef
	}

	fmt.Print("Ключ (ввод скрыт): ")
	var key string
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			fmt.Println(dim("ключ не прочитан: " + err.Error()))
			return
		}
		key = strings.TrimSpace(string(b))
	} else {
		k, err := l.Prompt("")
		if err != nil {
			return
		}
		key = strings.TrimSpace(k)
	}
	if key == "" {
		key = os.Getenv("KUZYA_REMOTE_KEY")
	}
	if key == "" {
		fmt.Println(dim("ключ не задан — настройка не сохранена"))
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Println(dim("URL должен начинаться с http:// или https://"))
		return
	}
	if err := saveRemoteConfig(url, key, model); err != nil {
		fmt.Println(dim("не сохранил: " + err.Error()))
		return
	}
	fmt.Println(dim("сохранено в " + remoteEnvFile() + " (права 600)"))
}
