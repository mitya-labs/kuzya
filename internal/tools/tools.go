// Package tools — реестр инструментов. Модель видит только их, не shell.
// Новые инструменты добавлять только здесь; всё, что меняет систему, — Dangerous: true.
package tools

import "sort"

type Tool struct {
	Name        string
	Description string
	// Params — JSON-схема свойств: {"имя": {"type": "string", "description": "..."}}.
	Params    map[string]any
	Required  []string
	Dangerous bool
	Run       func(args map[string]any) (string, error)
}

var registry = map[string]Tool{}

func Register(t Tool) { registry[t.Name] = t }

func Get(name string) (Tool, bool) {
	t, ok := registry[name]
	return t, ok
}

// Schemas возвращает описания инструментов в формате OpenAI tools.
func Schemas() []map[string]any {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		t := registry[n]
		props := t.Params
		if props == nil {
			props = map[string]any{}
		}
		params := map[string]any{"type": "object", "properties": props}
		if len(t.Required) > 0 {
			params["required"] = t.Required
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": t.Name, "description": t.Description, "parameters": params,
			},
		})
	}
	return out
}
