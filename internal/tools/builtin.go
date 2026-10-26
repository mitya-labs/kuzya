package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Сервисы, с которыми работает Кузя: перезапуск и чтение журнала. ssh, dbus, sxmo и т.п. не входят:
// перезапуск рвёт связь или интерфейс, а в их журналах больше чувствительных данных.
var allowedUnits = map[string]bool{
	"NetworkManager.service":        true,
	"bluetooth.service":             true,
	"ModemManager.service":          true,
	"avahi-daemon.service":          true,
	"systemd-resolved.service":      true,
	"systemd-timesyncd.service":     true,
	"power-profiles-daemon.service": true,
}

// sysfs — корень /sys, подменяется в тестах.
var sysfs = "/sys"

func readFirst(pattern string) (string, error) {
	files, _ := filepath.Glob(filepath.Join(sysfs, pattern))
	if len(files) == 0 {
		return "", fmt.Errorf("нет %s", pattern)
	}
	b, err := os.ReadFile(files[0])
	return strings.TrimSpace(string(b)), err
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }

func init() {
	Register(Tool{
		Name:        "battery_status",
		Description: "Заряд батареи в процентах и состояние (заряжается/разряжается).",
		Run: func(map[string]any) (string, error) {
			c, err := readFirst("class/power_supply/*battery/capacity")
			if err != nil {
				return "", err
			}
			st, _ := readFirst("class/power_supply/*battery/status")
			return fmt.Sprintf("заряд %s%%, состояние: %s", c, st), nil
		},
	})

	Register(Tool{
		Name:        "wifi_status",
		Description: "Только показать состояние Wi-Fi: активное подключение и список сохранённых сетей (имена для wifi_connect). Ничего не подключает.",
		Run: func(map[string]any) (string, error) {
			dev, err := execCmd("nmcli", "-t", "-f", "DEVICE,TYPE,STATE,CONNECTION", "device", "status")
			if err != nil {
				return "", err
			}
			saved, _ := execCmd("nmcli", "-t", "-f", "NAME,TYPE", "connection", "show")
			return "устройства:\n" + dev + "\nсохранённые подключения:\n" + saved, nil
		},
	})

	Register(Tool{
		Name:        "wifi_connect",
		Description: "Подключиться или переключиться на сохранённую Wi-Fi сеть, когда пользователь просит подключить и называет имя сети («подключись к…», «перейди на…», connect). Пароль не вводится.",
		Params:      map[string]any{"connection": str("имя сохранённого подключения из wifi_status")},
		Required:    []string{"connection"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			name, err := strArg(args, "connection")
			if err != nil {
				return "", err
			}
			if !connRe.MatchString(name) {
				return "", fmt.Errorf("недопустимое имя подключения %q", name)
			}
			return execCmd("nmcli", "connection", "up", "id", name)
		},
	})

	Register(Tool{
		Name:        "services_running",
		Description: "Список запущенных системных сервисов («что запущено», «что крутится в системе», «какие демоны»).",
		Run: func(map[string]any) (string, error) {
			return execCmd("systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--plain", "--no-pager")
		},
	})

	Register(Tool{
		Name:        "service_logs",
		Description: "Последние строки журнала сервиса из списка: NetworkManager, bluetooth, ModemManager, avahi-daemon, systemd-resolved, systemd-timesyncd, power-profiles-daemon.",
		Params: map[string]any{
			"unit":  str("имя сервиса из списка, например NetworkManager"),
			"lines": num("сколько строк, по умолчанию 30, максимум 200"),
		},
		Required: []string{"unit"},
		Run: func(args map[string]any) (string, error) {
			raw, err := strArg(args, "unit")
			if err != nil {
				return "", err
			}
			unit, err := unitName(raw)
			if err != nil {
				return "", err
			}
			if !allowedUnits[unit] {
				return "", fmt.Errorf("журнал сервиса %s Кузе недоступен", unit)
			}
			n := 30
			if _, ok := args["lines"]; ok {
				if n, err = intArg(args, "lines", 1, 200); err != nil {
					return "", err
				}
			}
			return execCmd("journalctl", "-u", unit, "-n", strconv.Itoa(n), "--no-pager")
		},
	})

	Register(Tool{
		Name:        "service_restart",
		Description: "Перезапустить сервис из белого списка: NetworkManager, bluetooth, ModemManager, avahi-daemon, systemd-resolved, systemd-timesyncd, power-profiles-daemon.",
		Params:      map[string]any{"unit": str("имя сервиса")},
		Required:    []string{"unit"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			raw, err := strArg(args, "unit")
			if err != nil {
				return "", err
			}
			unit, err := unitName(raw)
			if err != nil {
				return "", err
			}
			if !allowedUnits[unit] {
				return "", fmt.Errorf("сервис %s нельзя перезапускать через Кузю", unit)
			}
			if _, err := execCmd("sudo", "-n", "systemctl", "restart", unit); err != nil {
				return "", err
			}
			return unit + " перезапущен", nil
		},
	})

	Register(Tool{
		Name:        "nosleep_set",
		Description: "Не давать телефону засыпать и гасить экран (mode=on: «не давай засыпать», «запрети засыпать») или разрешить сон (mode=off: «разреши спать», «можно засыпать»).",
		Params: map[string]any{"mode": map[string]any{
			"type": "string", "enum": []string{"on", "off"}, "description": "on — запретить сон, off — разрешить сон",
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
			return execCmd("phone-nosleep", m)
		},
	})

	Register(Tool{
		Name:        "volume_set",
		Description: "Установить громкость звука: percent — значение 0–100 («громкость 40», «на половину», «на максимум»); delta — сдвиг от текущей громкости («погромче» +10, «потише» −10). Не яркость экрана.",
		Params: map[string]any{
			"percent": num("громкость, 0–100"),
			"delta":   num("сдвиг в процентных пунктах: плюс — громче, минус — тише"),
		},
		Dangerous: true,
		Run: func(args map[string]any) (string, error) {
			p, err := levelArg(args, 0, 100, volumePercent)
			if err != nil {
				return "", err
			}
			if _, err := execCmd("wpctl", "set-volume", sink, strconv.Itoa(p)+"%"); err != nil {
				return "", err
			}
			out, err := execCmd("wpctl", "get-volume", sink)
			if err != nil {
				return "", err
			}
			return volumeInfo(out), nil
		},
	})

	Register(Tool{
		Name:        "brightness_set",
		Description: "Установить яркость экрана (подсветки): percent — значение 5–100 («яркость 60», «на максимум»); delta — сдвиг от текущей яркости («ярче» +20, «темнее» −20). Не громкость звука.",
		Params: map[string]any{
			"percent": num("яркость, 5–100"),
			"delta":   num("сдвиг в процентных пунктах: плюс — ярче, минус — темнее"),
		},
		Dangerous: true,
		Run: func(args map[string]any) (string, error) {
			max, err := readFirst("class/backlight/*/max_brightness")
			if err != nil {
				return "", err
			}
			m, err := strconv.Atoi(max)
			if err != nil || m <= 0 {
				return "", fmt.Errorf("max_brightness: %q", max)
			}
			current := func() (int, error) {
				cur, err := readFirst("class/backlight/*/brightness")
				if err != nil {
					return 0, err
				}
				c, err := strconv.Atoi(cur)
				if err != nil {
					return 0, fmt.Errorf("brightness: %q", cur)
				}
				return (c*100 + m/2) / m, nil
			}
			p, err := levelArg(args, 5, 100, current)
			if err != nil {
				return "", err
			}
			files, _ := filepath.Glob(filepath.Join(sysfs, "class/backlight/*/brightness"))
			if len(files) == 0 {
				return "", fmt.Errorf("нет подсветки экрана")
			}
			if err := os.WriteFile(files[0], []byte(strconv.Itoa(m*p/100)), 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("яркость %d%%", p), nil
		},
	})
}
