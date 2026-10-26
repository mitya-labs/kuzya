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
		fmt.Printf("Кузя хочет выполнить «%s». Разрешить? [y/N] ", name)
		s, _ := in.ReadString('\n')
		return strings.EqualFold(strings.TrimSpace(s), "y")
	}
	a := agent.New(llm.New(), confirm)

	if len(os.Args) > 1 { // один запрос, для скриптов и SXMO
		answer(a, strings.Join(os.Args[1:], " "))
		return
	}
	printLogo(os.Stdout, os.Stdout)
	fmt.Println("Кузя, домовой skladenOS")
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
		default:
			answer(a, line)
		}
		if err != nil {
			return
		}
	}
}

func answer(a *agent.Agent, prompt string) {
	out, err := a.Run(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		return
	}
	fmt.Println(out)
}
