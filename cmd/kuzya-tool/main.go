// kuzya-tool вызывает один инструмент напрямую, без модели и без подтверждения: проверка на устройстве.
// Использование: kuzya-tool имя ['{"аргумент":"значение"}']
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"kuzya/internal/tools"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "использование: kuzya-tool имя ['{\"json\":\"аргументы\"}']")
		os.Exit(2)
	}
	t, ok := tools.Get(os.Args[1])
	if !ok {
		fmt.Fprintln(os.Stderr, "нет такого инструмента:", os.Args[1])
		os.Exit(2)
	}
	args := map[string]any{}
	if len(os.Args) > 2 {
		if err := json.Unmarshal([]byte(os.Args[2]), &args); err != nil {
			fmt.Fprintln(os.Stderr, "аргументы:", err)
			os.Exit(2)
		}
	}
	out, err := t.Run(args)
	fmt.Print(out)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		fmt.Println()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}
