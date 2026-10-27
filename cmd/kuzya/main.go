package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"

	"kuzya/internal/agent"
	"kuzya/internal/llm"
)

const modelLarge = "nafanya-1.7b-Q4_K_M"

func main() {
	loadRemoteConfig()
	// Один запрос: без редактора строк, чтобы работали скрипты и SXMO.
	if len(os.Args) > 1 {
		in := bufio.NewReader(os.Stdin)
		confirm := func(name string) bool {
			fmt.Printf("%s «%s». Разрешить? [y/N, д/н] ", dim("Кузя хочет выполнить"), yellow(name))
			s, _ := in.ReadString('\n')
			return isYes(s)
		}
		a := agent.New(llm.New(), confirm)
		answer(a, confirm, strings.Join(os.Args[1:], " "))
		return
	}

	l := liner.NewLiner()
	defer l.Close()
	l.SetCtrlCAborts(true)
	hist := filepath.Join(os.Getenv("HOME"), ".kuzya_history")
	if f, err := os.Open(hist); err == nil {
		_, _ = l.ReadHistory(f)
		_ = f.Close()
	}
	defer func() {
		if f, err := os.Create(hist); err == nil {
			_, _ = l.WriteHistory(f)
			_ = f.Close()
		}
	}()

	confirm := func(name string) bool {
		s, err := l.Prompt(fmt.Sprintf("Кузя хочет выполнить «%s». Разрешить? [y/N, д/н] ", name))
		if err != nil {
			return false
		}
		return isYes(s)
	}
	a := agent.New(llm.New(), confirm)

	clearScreen()
	printLogo(os.Stdout, os.Stdout)
	fmt.Println(rule())
	fmt.Println(" " + bold("Кузя") + dim(" · офлайн-агент skladenOS"))
	fmt.Println(" " + dim("модель ") + cyan(modelLabel()) + dim("   ·   ? разговор") + dim("   ·   ! удалённая") + dim("   ·   /remote") + dim("   ·   /help"))
	fmt.Println(rule())
	remoteOnly := false
	for {
		prompt := "› "
		if remoteOnly {
			prompt = "r› "
		}
		line, err := l.Prompt(prompt)
		if err == liner.ErrPromptAborted {
			fmt.Println()
			continue
		}
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line != "" {
			l.AppendHistory(line)
		}
		switch line {
		case "":
		case "/exit", "/quit", "/выход":
			return
		case "/reset", "/сброс":
			a.Reset()
			fmt.Println(dim("Забыл."))
		case "/remote setup", "/remote config", "/remote настройка":
			remoteSetup(l)
		case "/remote show", "/remote status":
			fmt.Println(remoteStatus())
		case "/remote", "/remote on", "/remote off", "/удалённо", "/удаленно":
			want := !remoteOnly
			if strings.HasSuffix(line, " on") || line == "/удалённо" || line == "/удаленно" {
				want = true
			}
			if strings.HasSuffix(line, " off") {
				want = false
			}
			if want == remoteOnly {
				break
			}
			if want {
				if !remoteConfigured() {
					fmt.Println(dim("Удалённая модель не настроена: KUZYA_REMOTE_URL/KEY/MODEL."))
					break
				}
				stopServer()
				fmt.Println(dim("режим: только удалённая модель; локальный сервер остановлен"))
				remoteOnly = true
			} else {
				if err := startServer(); err != nil {
					fmt.Fprintln(os.Stderr, red("сервер не поднялся: "+err.Error()))
					break
				}
				fmt.Println(dim("режим: локальная и удалённая; сервер запущен"))
				remoteOnly = false
			}
		case "/help", "/?", "/помощь":
			fmt.Println(rule())
			fmt.Println(" " + cyan("?") + " текст — разговор (удалённая модель, если настроена)")
			fmt.Println(" " + cyan("!") + " или " + cyan("@") + " текст — удалённая модель с локальными инструментами")
			fmt.Println(" " + cyan("/remote") + " — только удалённая модель (локальный сервер останавливается)")
			fmt.Println(" " + cyan("/remote setup") + " — настроить URL, ключ и модель")
			fmt.Println(" " + cyan("/remote show") + " — показать настройки (ключ замаскирован)")
			fmt.Println(" " + cyan("/reset") + " — забыть разговор")
			fmt.Println(" " + cyan("/exit") + " — выйти")
			fmt.Println(rule())
		default:
			if strings.HasPrefix(line, "/") {
				fmt.Println(dim("Нет такой команды. /help — список."))
				break
			}
			if remoteOnly && !strings.HasPrefix(line, "?") && !strings.HasPrefix(line, "!") && !strings.HasPrefix(line, "@") {
				line = "!" + line
			}
			a = answer(a, confirm, line)
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

// answer печатает ответ и возвращает агента: при отказе маленькой модели
// в текстовом режиме он переключается на 1.7B и повторяет запрос.
func answer(a *agent.Agent, confirm func(string) bool, prompt string) *agent.Agent {
	out, err := a.Run(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("ошибка: "+err.Error()))
		return a
	}
	if strings.HasPrefix(strings.TrimSpace(prompt), "?") && isChatRefusal(out) {
		if isSmallModel() {
			fmt.Println(dim("0.8B не умеет общаться — переключаюсь на 1.7B…"))
			if err := switchModel(modelLarge); err == nil {
				os.Setenv("KUZYA_MODEL", modelLarge)
				a = agent.New(llm.New(), confirm)
				if retry, err := a.Run(prompt); err == nil {
					out = retry
				}
			}
		}
		if isChatRefusal(out) {
			fmt.Println()
			fmt.Println(green("●") + " " + out)
			fmt.Println(dim("Локальная модель не умеет общаться. Настрой удалённую: KUZYA_REMOTE_URL/KEY/MODEL."))
			return a
		}
	}
	out = clean(out)
	if strings.TrimSpace(out) == "" {
		out = dim("(пусто)")
	}
	fmt.Println()
	fmt.Println(green("●") + " " + out)
	return a
}

func remoteConfigured() bool {
	return os.Getenv("KUZYA_REMOTE_URL") != "" && os.Getenv("KUZYA_REMOTE_KEY") != "" && os.Getenv("KUZYA_REMOTE_MODEL") != ""
}

func kuzyaHomeDir() string {
	if h := os.Getenv("KUZYA_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "kuzya")
}

func stopServer() {
	_ = exec.Command(filepath.Join(kuzyaHomeDir(), "kuzya-server"), "stop").Run()
}

func startServer() error {
	cmd := exec.Command(filepath.Join(kuzyaHomeDir(), "kuzya-server"), "start")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// clean убирает markdown-разметку из ответов удалённой модели.
func clean(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	return s
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
