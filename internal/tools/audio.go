package tools

import (
	"fmt"
	"strconv"
	"strings"
)

const sink = "@DEFAULT_AUDIO_SINK@"

// volumeInfo превращает вывод wpctl («Volume: 0.40 [MUTED]») в понятную строку.
func volumeInfo(out string) string {
	f := strings.Fields(out)
	if len(f) < 2 {
		return out
	}
	v, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return out
	}
	s := fmt.Sprintf("громкость %d%%", int(v*100+0.5))
	if strings.Contains(out, "MUTED") {
		s += ", звук выключен"
	}
	return s
}

// volumePercent читает текущую громкость в процентах из вывода wpctl get-volume.
func volumePercent() (int, error) {
	out, err := execCmd("wpctl", "get-volume", sink)
	if err != nil {
		return 0, err
	}
	f := strings.Fields(out)
	if len(f) < 2 {
		return 0, fmt.Errorf("не прочитал громкость: %q", out)
	}
	v, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return 0, fmt.Errorf("не прочитал громкость: %q", out)
	}
	return int(v*100 + 0.5), nil
}

func init() {
	Register(Tool{
		Name:        "volume_get",
		Description: "Текущая громкость в процентах и выключен ли звук.",
		Run: func(map[string]any) (string, error) {
			out, err := execCmd("wpctl", "get-volume", sink)
			if err != nil {
				return "", err
			}
			return volumeInfo(out), nil
		},
	})

	Register(Tool{
		Name:        "mute_set",
		Description: "Заглушить звук (mode=on: «выключи звук», «заглуши», mute) или вернуть звук (mode=off: «включи звук», «верни звук», unmute). Уровень громкости не меняет.",
		Params: map[string]any{"mode": map[string]any{
			"type": "string", "enum": []string{"on", "off"}, "description": "on — заглушить звук, off — вернуть звук",
		}},
		Required:  []string{"mode"},
		Dangerous: true,
		Run: func(args map[string]any) (string, error) {
			m, err := strArg(args, "mode")
			if err != nil {
				return "", err
			}
			if m != "on" && m != "off" {
				return "", fmt.Errorf("mode должен быть on или off")
			}
			v := "0"
			if m == "on" {
				v = "1"
			}
			if _, err := execCmd("wpctl", "set-mute", sink, v); err != nil {
				return "", err
			}
			out, err := execCmd("wpctl", "get-volume", sink)
			if err != nil {
				return "", err
			}
			return volumeInfo(out), nil
		},
	})
}
