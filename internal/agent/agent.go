// Package agent — цикл «модель -> инструмент -> результат -> модель».
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"kuzya/internal/llm"
	"kuzya/internal/rules"
	"kuzya/internal/tools"
)

const (
	// System — компактный промпт (1176 байт), на нём обучена nafanya.
	System = `Ты Кузя, офлайн-агент skladenOS. Работай только через инструменты и не вызывай инструмент, если просьба не совпадает с его описанием. Умеешь: заряд батареи; статус Wi-Fi и подключение к сохранённой сети; список сервисов; журнал и перезапуск семи сервисов; запрет/разрешение сна; громкость, mute, шаг громкости; яркость и шаг яркости. Не умеешь: включать/выключать Wi-Fi, останавливать службы, ssh/sshd, файлы, диск, звонки, будильник, фонарик, браузер, перевод, вычисления, питание телефона. Остальное — ответь текстом. «Остановить» не равно «перезапустить»; «все службы» — не одна служба. mute on — выключить звук, off — вернуть; nosleep on — запретить сон, off — разрешить.`
	// SystemFull — прежний подробный промпт; только для сравнений и базовых моделей.
	SystemFull = "Ты Кузя, домовой skladenOS. Работаешь офлайн, действуешь только через инструменты. Умеешь только: заряд батареи; статус Wi-Fi и подключение к уже сохранённой сети; список запущенных служб; журнал и перезапуск нескольких служб из списка; запрет и разрешение сна телефона; громкость и выключение звука; яркость экрана. Всё остальное не умеешь (включение и выключение Wi-Fi, остановка служб, питание телефона, диск, файлы, программы, звонки, перевод, вычисления и так далее). Просьбу нужно выполнить ровно как сказана: «остановить» не то же самое, что «перезапустить», «все службы» не одна служба. Если подходящего инструмента нет, не вызывай инструменты: коротко ответь текстом, что так не умеешь."
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

// runRemote — удалённая модель вызывает инструменты телефона; опасные действия
// подтверждает пользователь, как и в локальном режиме.
func (a *Agent) runRemote(q, url, key, model string) (string, error) {
	msgs := []llm.Message{
		{Role: "system", Content: "Ты — удалённый помощник агента Кузя на телефоне skladenOS. Ты можешь вызывать инструменты телефона и управлять локальной моделью (local_status, local_switch_model); опасные действия подтверждает пользователь. Отвечай кратко, по-русски.\n\n" + localFacts},
		{Role: "user", Content: q},
	}
	remoteTools := append(append([]map[string]any{}, tools.Schemas()...), remoteSchemas()...)
	last := ""
	for i := 0; i < MaxSteps; i++ {
		reply, err := a.LLM.ChatRemote(msgs, remoteTools, url, key, model)
		if err != nil {
			return "", err
		}
		reply.Role = "assistant"
		msgs = append(msgs, reply)
		if len(reply.ToolCalls) == 0 {
			return strings.TrimSpace(reply.Content), nil
		}
		for _, c := range reply.ToolCalls {
			var args map[string]any
			if c.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(c.Function.Arguments), &args)
			}
			out, handled := a.remoteCall(c.Function.Name, args)
			if !handled {
				var ok bool
				out, ok = a.call(c.Function)
				if !ok {
					return fmt.Sprintf("Отменено: «%s» не выполнен.", c.Function.Name), nil
				}
			}
			last = out
			msgs = append(msgs, llm.Message{Role: "tool", ToolCallID: c.ID, Content: out})
		}
	}
	if last != "" {
		return last, nil
	}
	return "Не уложился в лимит шагов.", nil
}

// remoteEnv читает настройки удалённой модели из окружения.
func remoteEnv() (string, string, string, bool) {
	url, key, model := os.Getenv("KUZYA_REMOTE_URL"), os.Getenv("KUZYA_REMOTE_KEY"), os.Getenv("KUZYA_REMOTE_MODEL")
	return url, key, model, url != "" && key != "" && model != ""
}

func (a *Agent) Reset() {
	a.History = []llm.Message{{Role: "system", Content: System}}
}

// Run добавляет реплику пользователя и крутит цикл до ответа без вызовов.
func (a *Agent) Run(prompt string) (string, error) {
	// «@»/«!» — удалённый мозг с локальными инструментами; «?» — только разговор.
	if strings.HasPrefix(strings.TrimSpace(prompt), "@") || strings.HasPrefix(strings.TrimSpace(prompt), "!") || strings.HasPrefix(strings.TrimSpace(prompt), "?") {
		mark := strings.TrimSpace(prompt)[:1]
		q := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), mark))
		url, key, model, ok := remoteEnv()
		if mark == "@" || mark == "!" {
			if !ok {
				return "Удалённая модель не настроена: задай KUZYA_REMOTE_URL, KUZYA_REMOTE_KEY и KUZYA_REMOTE_MODEL.", nil
			}
			return a.runRemote(q, url, key, model)
		}
		if ok {
			reply, err := a.LLM.ChatRemote([]llm.Message{
				{Role: "system", Content: "Ты — удалённый помощник. Отвечай кратко, по-русски."},
				{Role: "user", Content: q},
			}, nil, url, key, model)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(reply.Content), nil
		}
		// «?» без удалённой модели — локальный текстовый ответ (без инструментов).
		msgs := []llm.Message{
			{Role: "system", Content: "Ты — Кузя, офлайн-агент skladenOS. Отвечай коротко, по-русски, обычным текстом. Инструментов сейчас нет, не предлагай их."},
			{Role: "user", Content: q},
		}
		reply, err := a.LLM.Chat(msgs, nil)
		if err != nil {
			return "", err
		}
		reply.ToolCalls = nil
		return strings.TrimSpace(reply.Content), nil
	}

	// Вопрос о возможностях — статический ответ, модель для него не нужна.
	if msg, ok := rules.Help(prompt); ok {
		a.History = append(a.History, llm.Message{Role: "user", Content: prompt}, llm.Message{Role: "assistant", Content: msg})
		a.trim()
		return msg, nil
	}
	// Жёсткие негативы и очевидные команды решаем без модели: это и точнее, и дешевле.
	if msg, ok := rules.Refusal(prompt); ok {
		a.History = append(a.History, llm.Message{Role: "user", Content: prompt}, llm.Message{Role: "assistant", Content: msg})
		a.trim()
		return msg, nil
	}
	if tool, args, ok := rules.Preroute(prompt); ok {
		raw, _ := json.Marshal(args)
		out, confirmed := a.call(llm.FunctionCall{Name: tool, Arguments: string(raw)})
		if !confirmed {
			out = fmt.Sprintf("Отменено: «%s» не выполнен.", tool)
		}
		a.History = append(a.History, llm.Message{Role: "user", Content: prompt}, llm.Message{Role: "assistant", Content: out})
		a.trim()
		return out, nil
	}

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
			if c.Function.Name != "" {
				var args map[string]any
				if c.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(c.Function.Arguments), &args)
				}
				name, fixed := rules.Repair(prompt, c.Function.Name, args)
				if name != c.Function.Name || fixed != nil {
					raw, _ := json.Marshal(fixed)
					c.Function.Name, c.Function.Arguments = name, string(raw)
				}
			}
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
