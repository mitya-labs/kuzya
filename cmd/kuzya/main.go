package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kuzya/internal/agent"
	"kuzya/internal/llm"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	confirm := func(name string) bool {
		fmt.Printf("Кузя хочет выполнить «%s». Разрешить? [y/N, д/н] ", name)
		s, _ := in.ReadString('\n')
		return isYes(s)
	}
	a := agent.New(llm.New(), confirm)

	if len(os.Args) > 1 { // один запрос, для скриптов и SXMO
		answer(a, confirm, strings.Join(os.Args[1:], " "))
		return
	}
	printLogo(os.Stdout, os.Stdout)
	fmt.Println("Кузя, домовой skladenOS")
	fmt.Println("? вопрос — ответить текстом, без инструментов")
	fmt.Println("@ вопрос — спросить удалённую модель (если настроена)")
	fmt.Println("/reset — забыть разговор, /exit — выйти")
	for {
		fmt.Print("> ")
		line, err := in.ReadString('\n')
		line = strings.TrimSpace(line)
		switch line {
		case "":
		case "/exit", "/quit", "/выход":
			return
		case "/reset", "/сброс":
			a.Reset()
			fmt.Println("Забыл.")
		case "/help", "/?", "/помощь":
			fmt.Println("? вопрос — ответить текстом, без инструментов")
			fmt.Println("@ вопрос — спросить удалённую модель (если настроена)")
			fmt.Println("/reset (/сброс) — забыть разговор, /exit (/quit, /выход) — выйти, Ctrl-D — тоже выход.")
		default:
			if strings.HasPrefix(line, "/") {
				fmt.Println("Нет такой команды. /help — список.")
				break
			}
			a = answer(a, confirm, line)
		}
		if err != nil {
			return
		}
	}
}

// isYes — согласие на опасное действие: y, yes, д, да (любой регистр); всё остальное, включая пустой ввод, — отказ.
func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "д", "да":
		return true
	}
	return false
}

const modelLarge = "nafanya-1.7b-Q4_K_M"

// answer печатает ответ и возвращает агента: при отказе маленькой модели
// в текстовом режиме он переключается на 1.7B и повторяет запрос.
func answer(a *agent.Agent, confirm func(string) bool, prompt string) *agent.Agent {
	out, err := a.Run(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		return a
	}
	if strings.HasPrefix(strings.TrimSpace(prompt), "?") && isChatRefusal(out) {
		if isSmallModel() {
			fmt.Println("0.8B не умеет общаться — переключаюсь на 1.7B…")
			if err := switchModel(modelLarge); err == nil {
				os.Setenv("KUZYA_MODEL", modelLarge)
				a = agent.New(llm.New(), confirm)
				if retry, err := a.Run(prompt); err == nil {
					out = retry
				}
			}
		}
		if isChatRefusal(out) {
			fmt.Println(out)
			fmt.Println("Локальная модель не умеет общаться. Для разговора: @ вопрос (удалённая модель, если настроена).")
			return a
		}
	}
	fmt.Println(out)
	return a
}

func isSmallModel() bool {
	return strings.Contains(os.Getenv("KUZYA_MODEL"), "0.8b")
}

func isChatRefusal(s string) bool {
	t := strings.ToLower(s)
	for _, m := range []string{
		"не умею", "не могу", "не входит в мои возможности", "этого я не",
		"такого инструмента", "инструмента у меня нет", "нет такого инструмента", "прости",
	} {
		if strings.Contains(t, m) {
			return true
		}
	}
	return false
}

// switchModel перезапускает kuzya-server с другой моделью. Путь и имя фиксированы,
// пользовательский ввод сюда не попадает.
func switchModel(name string) error {
	kh := os.Getenv("KUZYA_HOME")
	if kh == "" {
		kh = filepath.Join(os.Getenv("HOME"), "kuzya")
	}
	_ = exec.Command(filepath.Join(kh, "kuzya-server"), "stop").Run()
	cmd := exec.Command(filepath.Join(kh, "kuzya-server"), "start")
	cmd.Env = append(os.Environ(), "KUZYA_MODEL="+name)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
