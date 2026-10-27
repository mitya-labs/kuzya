// Package tools — реестр инструментов. Модель видит только их, не shell.
// Новые инструменты добавлять только здесь; всё, что меняет систему, — Dangerous: true.
package tools

import "sort"

type Tool struct {
	Name        string
	Description string
	// Short — описание одной строкой для компактной схемы (на ней обучена nafanya).
	// Если пусто, в компактной схеме берётся Description.
	Short     string
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

// Schemas возвращает компактные описания инструментов в формате OpenAI tools.
// Компактная схема (короткое описание + параметры без description) — та, на которой
// обучена nafanya; подробные описания остались в SchemasFull для базовых моделей.
func Schemas() []map[string]any { return schemas(true) }

// SchemasFull возвращает подробные описания инструментов и их параметров.
func SchemasFull() []map[string]any { return schemas(false) }

func schemas(compact bool) []map[string]any {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		t := registry[n]
		desc := t.Description
		props := t.Params
		if compact {
			if t.Short != "" {
				desc = t.Short
			}
			props = withoutDescriptions(props)
		}
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
				"name": t.Name, "description": desc, "parameters": params,
			},
		})
	}
	return out
}

// withoutDescriptions копирует свойства параметров без ключа description.
// enum, type и прочие ограничения сохраняются.
func withoutDescriptions(props map[string]any) map[string]any {
	if props == nil {
		return nil
	}
	out := make(map[string]any, len(props))
	for name, raw := range props {
		p, ok := raw.(map[string]any)
		if !ok {
			out[name] = raw
			continue
		}
		cp := make(map[string]any, len(p))
		for k, v := range p {
			if k == "description" {
				continue
			}
			cp[k] = v
		}
		out[name] = cp
	}
	return out
}
