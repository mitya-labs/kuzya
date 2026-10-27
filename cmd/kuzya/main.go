package main

import (
	"bufio"
	"fmt"
	"os"
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
		answer(a, strings.Join(os.Args[1:], " "))
		return
	}
	printLogo(os.Stdout, os.Stdout)
	fmt.Println("Кузя, домовой skladenOS")
	fmt.Println("? вопрос — ответить текстом, без инструментов")
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
			fmt.Println("/reset (/сброс) — забыть разговор, /exit (/quit, /выход) — выйти, Ctrl-D — тоже выход.")
		default:
			if strings.HasPrefix(line, "/") {
				fmt.Println("Нет такой команды. /help — список.")
				break
			}
			answer(a, line)
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

func answer(a *agent.Agent, prompt string) {
	out, err := a.Run(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		return
	}
	fmt.Println(out)
}
