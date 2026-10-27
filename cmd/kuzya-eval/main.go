// kuzya-eval прогоняет tests/cases.json через llama-server и считает долю верных вызовов инструментов.
// Проверяется только первый ответ модели; инструменты не выполняются.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"kuzya/internal/agent"
	"kuzya/internal/llm"
	"kuzya/internal/rules"
	"kuzya/internal/tools"
)

type Case struct {
	ID   int            `json:"id"`
	Q    string         `json:"q"`
	Tool string         `json:"tool"` // "" — инструмент вызывать не нужно
	Args map[string]any `json:"args"`
}

type Result struct {
	Case
	GotTool string         `json:"got_tool"`
	GotArgs map[string]any `json:"got_args,omitempty"`
	ToolOK  bool           `json:"tool_ok"`
	ArgsOK  bool           `json:"args_ok"`
	Reply   string         `json:"reply,omitempty"`
	Err     string         `json:"err,omitempty"`
	Seconds float64        `json:"seconds"`
	Source  string         `json:"source,omitempty"` // model или rules
}

func main() {
	casesPath := flag.String("cases", "tests/cases.json", "файл с запросами")
	out := flag.String("out", "", "куда писать подробный результат (jsonl)")
	systemFile := flag.String("system", "", "файл с системным промптом (по умолчанию agent.System)")
	full := flag.Bool("full", false, "полный промпт и подробные описания инструментов (для базовых моделей)")
	useRules := flag.Bool("rules", false, "детерминированные правила до и после модели")
	flag.Parse()

	system := agent.System
	schemas := tools.Schemas
	if *full {
		system = agent.SystemFull
		schemas = tools.SchemasFull
	}
	if *systemFile != "" {
		b, err := os.ReadFile(*systemFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		system = strings.TrimSpace(string(b))
	}

	raw, err := os.ReadFile(*casesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var cases []Case
	if err := json.Unmarshal(raw, &cases); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := llm.New()
	var results []Result
	for _, c := range cases {
		r := Result{Case: c}
		start := time.Now()
		r.Source = "model"
		if *useRules {
			if msg, ok := rules.Refusal(c.Q); ok {
				r.Source = "rules"
				r.Reply = msg
				r.ToolOK = c.Tool == ""
				r.ArgsOK = r.ToolOK
				r.Seconds = time.Since(start).Seconds()
				results = append(results, r)
				mark := "ok  "
				if !r.ArgsOK {
					mark = "FAIL"
				}
				fmt.Printf("%s %2d %-48q ждали %-16s получили %-16s %v %.0fс\n", mark, c.ID, c.Q, orNone(c.Tool), orNone(r.GotTool), r.GotArgs, r.Seconds)
				continue
			}
			if tool, args, ok := rules.Preroute(c.Q); ok {
				r.Source = "rules"
				r.GotTool = tool
				r.GotArgs = args
				r.ToolOK = r.GotTool == c.Tool
				r.ArgsOK = r.ToolOK && argsMatch(c.Args, r.GotArgs)
				r.Seconds = time.Since(start).Seconds()
				results = append(results, r)
				mark := "ok  "
				if !r.ArgsOK {
					mark = "FAIL"
				}
				fmt.Printf("%s %2d %-48q ждали %-16s получили %-16s %v %.0fс\n", mark, c.ID, c.Q, orNone(c.Tool), orNone(r.GotTool), r.GotArgs, r.Seconds)
				continue
			}
		}
		msgs := []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: c.Q}}
		reply, err := client.Chat(msgs, schemas())
		r.Seconds = time.Since(start).Seconds()
		if err != nil {
			r.Err = err.Error()
		} else {
			r.Reply = strings.TrimSpace(reply.Content)
			if len(reply.ToolCalls) > 0 {
				f := reply.ToolCalls[0].Function
				r.GotTool = f.Name
				_ = json.Unmarshal([]byte(f.Arguments), &r.GotArgs)
				if *useRules && r.GotArgs == nil {
					r.GotArgs = map[string]any{}
				}
				if *useRules {
					r.GotTool, r.GotArgs = rules.Repair(c.Q, r.GotTool, r.GotArgs)
				}
			}
			r.ToolOK = r.GotTool == c.Tool
			r.ArgsOK = r.ToolOK && argsMatch(c.Args, r.GotArgs)
		}
		results = append(results, r)
		mark := "ok  "
		if !r.ArgsOK {
			mark = "FAIL"
		}
		fmt.Printf("%s %2d %-48q ждали %-16s получили %-16s %v %.0fс\n", mark, c.ID, c.Q, orNone(c.Tool), orNone(r.GotTool), r.GotArgs, r.Seconds)
	}

	summarize(results)
	if *out != "" {
		f, err := os.Create(*out)
		if err == nil {
			defer f.Close()
			enc := json.NewEncoder(f)
			for _, r := range results {
				_ = enc.Encode(r)
			}
		}
	}
}

func orNone(s string) string {
	if s == "" {
		return "(без вызова)"
	}
	return s
}

func norm(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(x), ".service"))
		s = strings.TrimSuffix(s, "%")
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		return strings.ToLower(s)
	}
	return fmt.Sprint(v)
}

// argsMatch: все ожидаемые аргументы должны совпасть, лишние у модели допустимы.
func argsMatch(want, got map[string]any) bool {
	for k, v := range want {
		g, ok := got[k]
		if !ok {
			return false
		}
		// "+" и "-" в ожидаемом значении означают только знак числа (для delta)
		if w, isStr := v.(string); isStr && (w == "+" || w == "-") {
			f, err := strconv.ParseFloat(norm(g), 64)
			if err != nil || (w == "+" && f <= 0) || (w == "-" && f >= 0) {
				return false
			}
			continue
		}
		if norm(g) != norm(v) {
			return false
		}
	}
	return true
}

func summarize(rs []Result) {
	type stat struct{ n, tool, args int }
	per := map[string]*stat{}
	var all stat
	var errs int
	var secs float64
	for _, r := range rs {
		s := per[orNone(r.Tool)]
		if s == nil {
			s = &stat{}
			per[orNone(r.Tool)] = s
		}
		for _, t := range []*stat{s, &all} {
			t.n++
			if r.ToolOK {
				t.tool++
			}
			if r.ArgsOK {
				t.args++
			}
		}
		if r.Err != "" {
			errs++
		}
		secs += r.Seconds
	}
	names := make([]string, 0, len(per))
	for n := range per {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Println("\nпо инструментам (верный инструмент / верный инструмент и аргументы):")
	for _, n := range names {
		s := per[n]
		fmt.Printf("  %-18s %d/%d  %d/%d\n", n, s.tool, s.n, s.args, s.n)
	}
	fmt.Printf("\nИТОГО: верный инструмент %d/%d (%.0f%%), с аргументами %d/%d (%.0f%%), ошибок запроса %d, среднее время %.1f с\n",
		all.tool, all.n, 100*float64(all.tool)/float64(all.n), all.args, all.n, 100*float64(all.args)/float64(all.n), errs, secs/float64(all.n))
}
