// Package agent — цикл «модель -> инструмент -> результат -> модель».
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"kuzya/internal/llm"
	"kuzya/internal/tools"
)

const (
	System     = "Ты Кузя, домовой skladenOS. Работаешь офлайн, действуешь только через инструменты. Умеешь только: заряд батареи; статус Wi-Fi и подключение к уже сохранённой сети; список запущенных служб; журнал и перезапуск нескольких служб из списка; запрет и разрешение сна телефона; громкость и выключение звука; яркость экрана. Всё остальное не умеешь (включение и выключение Wi-Fi, остановка служб, питание телефона, диск, файлы, программы, звонки, перевод, вычисления и так далее). Просьбу нужно выполнить ровно как сказана: «остановить» не то же самое, что «перезапустить», «все службы» не одна служба. Если подходящего инструмента нет, не вызывай инструменты: коротко ответь текстом, что так не умеешь."
	MaxSteps   = 8
	maxHistory = 24 // сообщений кроме системного; контекст на телефоне маленький
)

type Agent struct {
	LLM     *llm.Client
	Confirm func(tool string) bool
	History []llm.Message
}

func New(c *llm.Client, confirm func(string) bool) *Agent {
	a := &Agent{LLM: c, Confirm: confirm}
	a.Reset()
	return a
}

func (a *Agent) Reset() {
	a.History = []llm.Message{{Role: "system", Content: System}}
}

// Run добавляет реплику пользователя и крутит цикл до ответа без вызовов.
func (a *Agent) Run(prompt string) (string, error) {
	a.History = append(a.History, llm.Message{Role: "user", Content: prompt})
	a.trim()
	last := "" // результат последнего выполненного инструмента
	for i := 0; i < MaxSteps; i++ {
		reply, err := a.LLM.Chat(a.History, tools.Schemas())
		if err != nil {
			return "", err
		}
		reply.Role = "assistant"
		a.History = append(a.History, reply)
		if len(reply.ToolCalls) == 0 {
			// Маленькая модель иногда молчит после результата инструмента: показываем сам результат.
			if strings.TrimSpace(reply.Content) == "" && last != "" {
				return last, nil
			}
			return reply.Content, nil
		}
		denied := ""
		for _, c := range reply.ToolCalls {
			out, ok := a.call(c.Function)
			if ok {
				last = out
			} else {
				denied = c.Function.Name
			}
			a.History = append(a.History, llm.Message{Role: "tool", ToolCallID: c.ID, Content: out})
		}
		// После отказа модель не зовём: она отвечала «так не умею», хотя действие просто отменено.
		if denied != "" {
			msg := fmt.Sprintf("Отменено: «%s» не выполнен.", denied)
			a.History = append(a.History, llm.Message{Role: "assistant", Content: msg})
			return msg, nil
		}
	}
	return "Не уложился в лимит шагов.", nil
}

// call выполняет вызов; false — пользователь отказал в подтверждении.
func (a *Agent) call(f llm.FunctionCall) (string, bool) {
	t, ok := tools.Get(f.Name)
	if !ok {
		return "нет такого инструмента: " + f.Name, true
	}
	args := map[string]any{}
	if f.Arguments != "" {
		if err := json.Unmarshal([]byte(f.Arguments), &args); err != nil {
			return "неверные аргументы: " + err.Error(), true
		}
	}
	if t.Dangerous && !a.Confirm(f.Name) {
		return "пользователь отказал", false
	}
	out, err := t.Run(args)
	if err != nil {
		return fmt.Sprintf("ошибка: %v", err), true
	}
	return out, true
}

// trim отбрасывает старые сообщения, не разрывая пары «вызов — результат».
func (a *Agent) trim() {
	if len(a.History) <= maxHistory+1 {
		return
	}
	rest := a.History[len(a.History)-maxHistory:]
	for len(rest) > 0 && rest[0].Role != "user" {
		rest = rest[1:]
	}
	a.History = append([]llm.Message{a.History[0]}, rest...)
}
