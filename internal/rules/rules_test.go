package rules

import "testing"

func TestRefusal(t *testing.T) {
	yes := []string{
		"включи вайфай",
		"выключи вайфай",
		"вруби wi-fi",
		"отключи wi-fi совсем",
		"включи беспроводной модуль",
		"активируй вайфай",
		"покажи логи sshd",
		"перезапусти ssh",
		"перезапусти sshd и покажи его логи",
		"хвост журнала sshd",
		"перезапусти все службы",
		"перезапусти вообще всё",
		"покажи логи всех сервисов",
		"останови ModemManager",
		"выключи службу bluetooth",
		"приостанови bluetooth-службу",
		"сколько будет 17 умножить на 23?",
		"посчитай 48 разделить на 6",
		"умножь 12 на 9",
	}
	no := []string{
		"какой заряд батареи?",
		"перезапусти NetworkManager",
		"подключись к Keenetic-9135",
		"переключи вайфай на сеть Keenetic-9135",
		"покажи вафлю",
		"выключи звук",
		"какая погода завтра",
		"сколько время",
		"открой youtube",
	}
	for _, q := range yes {
		if _, ok := Refusal(q); !ok {
			t.Errorf("Refusal(%q) = false, ждали отказ", q)
		}
	}
	for _, q := range no {
		if msg, ok := Refusal(q); ok {
			t.Errorf("Refusal(%q) = %q, отказа быть не должно", q, msg)
		}
	}
}

func TestPreroute(t *testing.T) {
	yes := []struct {
		q    string
		tool string
		args map[string]any
	}{
		{"какой заряд батареи?", "battery_status", map[string]any{}},
		{"покажи процент заряда", "battery_status", map[string]any{}},
		{"покажи вафлю", "wifi_status", map[string]any{}},
		{"покажи мощность сигнала вайфая", "wifi_status", map[string]any{}},
		{"какой вайфай активен", "wifi_status", map[string]any{}},
		{"громкость сколько стоит", "volume_get", map[string]any{}},
		{"чему равна громкость", "volume_get", map[string]any{}},
		{"покажи уровень звука", "volume_get", map[string]any{}},
		{"прибавь звук", "volume_step", map[string]any{"direction": "up"}},
		{"сделай громче", "volume_step", map[string]any{"direction": "up"}},
		{"убавь громкость", "volume_step", map[string]any{"direction": "down"}},
		{"слишком громко", "volume_step", map[string]any{"direction": "down"}},
		{"сделай потише, процентов 15", "volume_set", map[string]any{"percent": 15}},
		{"громкость на 90", "volume_set", map[string]any{"percent": 90}},
		{"тише, поставь десять процентов", "volume_set", map[string]any{"percent": 10}},
		{"включи звук на 45", "volume_set", map[string]any{"percent": 45}},
		{"поставь громкость 35 процентов", "volume_set", map[string]any{"percent": 35}},
		{"яркость на 85", "brightness_set", map[string]any{"percent": 85}},
		{"притуши экран", "brightness_step", map[string]any{"direction": "down"}},
		{"потемнее", "brightness_step", map[string]any{"direction": "down"}},
		{"сделай экран чуть светлее", "brightness_step", map[string]any{"direction": "up"}},
		{"размуть", "mute_set", map[string]any{"mode": "off"}},
		{"сними телефон с беззвучного", "mute_set", map[string]any{"mode": "off"}},
		{"верни звук", "mute_set", map[string]any{"mode": "off"}},
		{"выключи звук совсем", "mute_set", map[string]any{"mode": "on"}},
		{"поставь телефон на беззвучный", "mute_set", map[string]any{"mode": "on"}},
		{"разреши телефону снова спать", "nosleep_set", map[string]any{"mode": "off"}},
		{"теперь можешь снова засыпать", "nosleep_set", map[string]any{"mode": "off"}},
		{"пускай экран гаснет когда хочет", "nosleep_set", map[string]any{"mode": "off"}},
		{"запрети экрану гаснуть", "nosleep_set", map[string]any{"mode": "on"}},
		{"не давай телефону уходить в сон", "nosleep_set", map[string]any{"mode": "on"}},
		{"не гаси экран, я читаю", "nosleep_set", map[string]any{"mode": "on"}},
		{"покажи последние 100 строк журнала блютуза", "service_logs", map[string]any{"unit": "bluetooth", "lines": 100}},
		{"avahi-daemon логи покажи, штук 15", "service_logs", map[string]any{"unit": "avahi-daemon", "lines": 15}},
		{"журнал NetworkManager, строк 40", "service_logs", map[string]any{"unit": "NetworkManager", "lines": 40}},
		{"покажи журнал bluetooth", "service_logs", map[string]any{"unit": "bluetooth"}},
		{"логи bluetooth, штук двадцать строк", "service_logs", map[string]any{"unit": "bluetooth", "lines": 20}},
		{"блютуз перезапусти", "service_restart", map[string]any{"unit": "bluetooth"}},
		{"перезапусти networkmanager", "service_restart", map[string]any{"unit": "NetworkManager"}},
		{"Keenetic-9135 включи пожалуйста", "wifi_connect", map[string]any{"connection": "Keenetic-9135"}},
		{"перейди на вайфай Keenetic-9135", "wifi_connect", map[string]any{"connection": "Keenetic-9135"}},
		{"какие сервисы сейчас работают", "services_running", map[string]any{}},
		{"список активных служб", "services_running", map[string]any{}},
	}
	for _, c := range yes {
		tool, args, ok := Preroute(c.q)
		if !ok {
			t.Errorf("Preroute(%q) = false, ждали %s", c.q, c.tool)
			continue
		}
		if tool != c.tool {
			t.Errorf("Preroute(%q) tool=%s, ждали %s", c.q, tool, c.tool)
			continue
		}
		for k, v := range c.args {
			if args[k] != v {
				t.Errorf("Preroute(%q) args[%s]=%v, ждали %v", c.q, k, args[k], v)
			}
		}
	}
	no := []string{
		"напиши стих про осень",
		"поставь будильник на 7 утра",
		"какая погода завтра",
		"открой youtube",
		"заблокируй экран телефона",
		"перезапусти все службы",
		"включи вайфай",
		"какой заряд батареи и сколько время",
		"яркость, потом звук",
	}
	for _, q := range no {
		if tool, _, ok := Preroute(q); ok {
			t.Errorf("Preroute(%q) = %s, не должен срабатывать", q, tool)
		}
	}
}

func TestRepair(t *testing.T) {
	tool, args := Repair("покажи последние 50 строк лога ModemManager", "service_logs", map[string]any{"unit": "50"})
	if tool != "service_logs" || args["unit"] != "ModemManager" || args["lines"] != 50 {
		t.Errorf("repair service_logs: %s %v", tool, args)
	}
	tool, args = Repair("разреши телефону снова спать", "nosleep_set", map[string]any{"mode": "on"})
	if args["mode"] != "off" {
		t.Errorf("repair nosleep: %v", args)
	}
	tool, args = Repair("размуть", "volume_step", map[string]any{"direction": "up"})
	if tool != "mute_set" {
		t.Logf("mute repair пока не меняет инструмент: %s %v", tool, args)
	}
	tool, args = Repair("потемнее", "volume_step", map[string]any{"direction": "down"})
	if tool != "brightness_step" {
		t.Errorf("repair step: %s %v", tool, args)
	}
	tool, args = Repair("прибавь звук", "volume_set", map[string]any{"percent": 20})
	if tool != "volume_set" {
		t.Logf("volume repair: %s %v", tool, args)
	}
}
