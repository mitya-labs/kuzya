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
		Short:       "Заряд батареи и состояние (заряжается/разряжается)",
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
		Short:       "Показать активную Wi-Fi сеть и сохранённые сети",
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
		Short:       "Подключиться к сохранённой Wi-Fi сети по имени",
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
		Short:       "Список запущенных системных сервисов",
		Run: func(map[string]any) (string, error) {
			return execCmd("systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--plain", "--no-pager")
		},
	})

	Register(Tool{
		Name:        "service_logs",
		Description: "Последние строки журнала сервиса из списка: NetworkManager, bluetooth, ModemManager, avahi-daemon, systemd-resolved, systemd-timesyncd, power-profiles-daemon.",
		Short:       "Последние строки журнала сервиса из списка",
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
		Short:       "Перезапустить сервис из списка",
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
		Short:       "Запретить сон (on) или разрешить (off)",
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
		Description: "Установить громкость звука в процентах (0–100): «громкость 40», «на половину», «на максимум». Не яркость экрана.",
		Short:       "Установить громкость 0–100",
		Params:      map[string]any{"percent": num("громкость, 0–100")},
		Required:    []string{"percent"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			p, err := intArg(args, "percent", 0, 100)
			if err != nil {
				return "", err
			}
			return setVolume(p)
		},
	})

	Register(Tool{
		Name:        "volume_step",
		Description: "Сделать звук громче или тише на шаг, когда число не названо: «погромче», «потише». Не яркость экрана.",
		Short:       "Сделать громче или тише на шаг",
		Params:      map[string]any{"direction": str("up — громче, down — тише")},
		Required:    []string{"direction"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			d, err := stepArg(args, 10)
			if err != nil {
				return "", err
			}
			cur, err := volumePercent()
			if err != nil {
				return "", err
			}
			return setVolume(clamp(cur+d, 0, 100))
		},
	})

	Register(Tool{
		Name:        "brightness_set",
		Description: "Установить яркость экрана (подсветки) в процентах (5–100): «яркость 60», «на максимум». Не громкость звука.",
		Short:       "Установить яркость экрана 5–100",
		Params:      map[string]any{"percent": num("яркость, 5–100")},
		Required:    []string{"percent"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			p, err := intArg(args, "percent", 5, 100)
			if err != nil {
				return "", err
			}
			return setBrightness(p)
		},
	})

	Register(Tool{
		Name:        "brightness_step",
		Description: "Сделать экран ярче или темнее на шаг, когда число не названо: «ярче», «потемнее». Не громкость звука.",
		Short:       "Сделать экран ярче или темнее на шаг",
		Params:      map[string]any{"direction": str("up — ярче, down — темнее")},
		Required:    []string{"direction"},
		Dangerous:   true,
		Run: func(args map[string]any) (string, error) {
			d, err := stepArg(args, 20)
			if err != nil {
				return "", err
			}
			cur, err := brightnessPercent()
			if err != nil {
				return "", err
			}
			return setBrightness(clamp(cur+d, 5, 100))
		},
	})
}

func setVolume(p int) (string, error) {
	if _, err := execCmd("wpctl", "set-volume", sink, strconv.Itoa(p)+"%"); err != nil {
		return "", err
	}
	out, err := execCmd("wpctl", "get-volume", sink)
	if err != nil {
		return "", err
	}
	return volumeInfo(out), nil
}

func brightnessMax() (int, error) {
	max, err := readFirst("class/backlight/*/max_brightness")
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(max)
	if err != nil || m <= 0 {
		return 0, fmt.Errorf("max_brightness: %q", max)
	}
	return m, nil
}

func brightnessPercent() (int, error) {
	m, err := brightnessMax()
	if err != nil {
		return 0, err
	}
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

func setBrightness(p int) (string, error) {
	m, err := brightnessMax()
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
}
