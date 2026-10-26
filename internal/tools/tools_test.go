package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type call struct {
	name string
	args []string
}

// stub подменяет execCmd и запоминает вызовы.
func stub(t *testing.T, out string) *[]call {
	var calls []call
	old := execCmd
	execCmd = func(name string, args ...string) (string, error) {
		calls = append(calls, call{name, args})
		return out, nil
	}
	t.Cleanup(func() { execCmd = old })
	return &calls
}

func run(t *testing.T, name string, args map[string]any) (string, error) {
	tool, ok := Get(name)
	if !ok {
		t.Fatalf("нет инструмента %s", name)
	}
	return tool.Run(args)
}

func TestMutatingToolsAreDangerous(t *testing.T) {
	for _, n := range []string{"wifi_connect", "service_restart", "nosleep_set", "volume_set", "mute_set", "brightness_set"} {
		tool, ok := Get(n)
		if !ok || !tool.Dangerous {
			t.Errorf("%s должен существовать и быть Dangerous", n)
		}
	}
	for _, n := range []string{"battery_status", "wifi_status", "services_running", "service_logs", "volume_get"} {
		if tool, ok := Get(n); !ok || tool.Dangerous {
			t.Errorf("%s должен существовать и быть только на чтение", n)
		}
	}
}

func TestSchemasRequiredAreDeclared(t *testing.T) {
	for _, s := range Schemas() {
		fn := s["function"].(map[string]any)
		params := fn["parameters"].(map[string]any)
		props := params["properties"].(map[string]any)
		if req, ok := params["required"].([]string); ok {
			for _, r := range req {
				if _, ok := props[r]; !ok {
					t.Errorf("%v: required %q не описан в properties", fn["name"], r)
				}
			}
		}
	}
}

func TestServiceRestartAllowlist(t *testing.T) {
	calls := stub(t, "")
	for _, bad := range []string{"ssh", "sxmo.service", "dbus", "x; reboot", "../etc"} {
		if _, err := run(t, "service_restart", map[string]any{"unit": bad}); err == nil {
			t.Errorf("%q не должен перезапускаться", bad)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("команды не должны запускаться: %v", *calls)
	}
	if _, err := run(t, "service_restart", map[string]any{"unit": "NetworkManager"}); err != nil {
		t.Fatal(err)
	}
	c := (*calls)[0]
	if c.name != "sudo" || strings.Join(c.args, " ") != "-n systemctl restart NetworkManager.service" {
		t.Errorf("неожиданная команда: %v", c)
	}
}

func TestServiceLogsArgs(t *testing.T) {
	calls := stub(t, "log")
	if _, err := run(t, "service_logs", map[string]any{"unit": "bluetooth", "lines": "50"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join((*calls)[0].args, " "); got != "-u bluetooth.service -n 50 --no-pager" {
		t.Errorf("args=%q", got)
	}
	for _, a := range []map[string]any{{"unit": "-n"}, {"unit": "a b"}, {"unit": "ssh"}, {"unit": "sxmo"}, {"unit": "bluetooth", "lines": 9999}} {
		if _, err := run(t, "service_logs", a); err == nil {
			t.Errorf("%v должен быть отклонён", a)
		}
	}
}

func TestWifiConnectValidation(t *testing.T) {
	calls := stub(t, "ok")
	if _, err := run(t, "wifi_connect", map[string]any{"connection": "--help"}); err == nil {
		t.Error("имя с ведущим дефисом должно отклоняться")
	}
	if _, err := run(t, "wifi_connect", map[string]any{"connection": "Keenetic-9135"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join((*calls)[0].args, " "); got != "connection up id Keenetic-9135" {
		t.Errorf("args=%q", got)
	}
}

func TestVolumeAndMute(t *testing.T) {
	calls := stub(t, "Volume: 0.40 [MUTED]")
	out, err := run(t, "volume_set", map[string]any{"percent": 40.0})
	if err != nil || out != "громкость 40%, звук выключен" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if got := strings.Join((*calls)[0].args, " "); got != "set-volume @DEFAULT_AUDIO_SINK@ 40%" {
		t.Errorf("args=%q", got)
	}
	*calls = nil
	if _, err := run(t, "volume_set", map[string]any{"delta": -10.0}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join((*calls)[1].args, " "); got != "set-volume @DEFAULT_AUDIO_SINK@ 30%" {
		t.Errorf("delta -10 от 40%%: args=%q", got)
	}
	if _, err := run(t, "volume_set", map[string]any{"percent": 150.0}); err == nil {
		t.Error("150% должно отклоняться")
	}
	if _, err := run(t, "mute_set", map[string]any{"mode": "maybe"}); err == nil {
		t.Error("mode=maybe должен отклоняться")
	}
	*calls = nil
	if _, err := run(t, "mute_set", map[string]any{"mode": "on"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join((*calls)[0].args, " "); got != "set-mute @DEFAULT_AUDIO_SINK@ 1" {
		t.Errorf("args=%q", got)
	}
}

func TestSysfsTools(t *testing.T) {
	dir := t.TempDir()
	old := sysfs
	sysfs = dir
	t.Cleanup(func() { sysfs = old })
	write := func(rel, v string) {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(v), 0o644)
	}
	write("class/power_supply/qcom-battery/capacity", "65\n")
	write("class/power_supply/qcom-battery/status", "Discharging\n")
	write("class/backlight/backlight/max_brightness", "4095\n")
	write("class/backlight/backlight/brightness", "100\n")

	out, err := run(t, "battery_status", nil)
	if err != nil || out != "заряд 65%, состояние: Discharging" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, err := run(t, "brightness_set", map[string]any{"percent": 50.0}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "class/backlight/backlight/brightness"))
	if string(b) != "2047" {
		t.Errorf("brightness=%q, ждали 2047", b)
	}
	// delta: от текущих 50% (2047/4095) на +20 → 70%; на -100 не ниже 5%
	if out, err := run(t, "brightness_set", map[string]any{"delta": 20.0}); err != nil || out != "яркость 70%" {
		t.Errorf("delta +20: out=%q err=%v", out, err)
	}
	if out, err := run(t, "brightness_set", map[string]any{"delta": -100.0}); err != nil || out != "яркость 5%" {
		t.Errorf("delta -100: out=%q err=%v", out, err)
	}
	if _, err := run(t, "brightness_set", map[string]any{"percent": 50.0, "delta": 5.0}); err == nil {
		t.Error("percent вместе с delta должно отклоняться")
	}
	if _, err := run(t, "brightness_set", nil); err == nil {
		t.Error("без percent и delta должно отклоняться")
	}
	if _, err := run(t, "brightness_set", map[string]any{"percent": 0.0}); err == nil {
		t.Error("яркость 0% должна отклоняться (чёрный экран)")
	}
}
