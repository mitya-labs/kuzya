// Package rules — детерминированный слой поверх модели.
//
// Задача: снять с модели то, что она стабильно путает, и не тратить батарею на
// очевидные запросы. Правила узкие: если сработало — инструмент вызывается без
// модели (Preroute) или ответ отклоняется (Refusal); после ответа модели аргументы
// можно починить (Repair). Всё остальное идёт в модель как раньше.
package rules

import (
	"regexp"
	"strconv"
	"strings"
)

const refusal = "Этого я не умею."

func norm(q string) string {
	q = strings.ToLower(q)
	q = strings.ReplaceAll(q, "ё", "е")
	q = strings.ReplaceAll(q, "wi-fi", "wifi")
	q = strings.ReplaceAll(q, "wi fi", "wifi")
	return strings.Join(strings.Fields(q), " ")
}

const (
	unitAlt = `(служб[а-я]*|сервис[а-я]*|демон[а-я]*|networkmanager|bluetooth|блютуз|modemmanager|avahi|resolved|timesyncd|power-profiles)`
	verbAlt = `(перезапус|перезагруз|рестарт|ребут|останов|приостанов)`
)

var (
	reUnsupported = regexp.MustCompile(`ютуб|youtube|камер|скриншот|будильник|погод|смс|позвони|звонок|фонар|браузер|файл|диск|перевод|калькул|видео|музык|телеграм|whatsapp|инстаграм|фото|врем|час`)
	reSupported   = regexp.MustCompile(`заряд|батаре|аккумулятор|вайфай|wifi|вафл|громк|громч|звук|тише|яркост|подсветк|служб|сервис|демон|лог|журнал|спать|сон|спит|засып|гасн|беззвуч|mute|networkmanager|bluetooth|блютуз|modemmanager|avahi|resolved|timesyncd|power-profiles|keentic`)
	reBtToggle    = regexp.MustCompile(`(активир|включ|выключ|отключ|вруб|отруб)[а-я]*\s+(bluetooth|блютуз)`)
	reSSH         = regexp.MustCompile(`(^|[^a-z])sshd?([^a-z]|$)`)
	reWifiVerb    = regexp.MustCompile(`(включ|вруб|выключ|отключ|отруб|погаси|активир)[а-я]*\s+(вайфай|wifi|вафл|беспроводн)`)
	reAllUnits    = regexp.MustCompile(verbAlt + `[а-я]*.*(все|всех|кажд)[а-я]*\s+` + unitAlt + `|` + verbAlt + `[а-я]*.*все`)
	reStopUnit    = regexp.MustCompile(`(останов|выключ|приостанов|погаси)[а-я]*\s+` + unitAlt)
	reArith       = regexp.MustCompile(`(скольк|посчита|вычисли|умнож|раздели|подели|сложи)[а-я]*.*\d+\s*(\*|x|х|умнож|плюс|минус|раздел|дел|слож)|(умнож|раздел|подел|сложи|посчита)[а-я]*.*\d+.*\d+`)
	reAllLogs     = regexp.MustCompile(`(все|всех|кажд)[а-я]*\s+(служб|сервис|демон)`)
	reNetName     = regexp.MustCompile(`[A-Za-z][A-Za-z0-9._-]*[0-9][A-Za-z0-9._-]*|[A-Za-z][A-Za-z0-9._-]*-[A-Za-z0-9._-]+`)
	reBattery     = regexp.MustCompile(`(заряд|батаре|аккумулятор)`)
	reServices    = regexp.MustCompile(`(какие|список|что|перечисли)[а-я]*.*(служб|сервис|демон)`)
	reWifi        = regexp.MustCompile(`(вафл|вайфа|wifi|беспроводн)`)
	reLogs        = regexp.MustCompile(`(лог|журнал|строк|строч)`)
	reRestart     = regexp.MustCompile(verbAlt)
	reVolume      = regexp.MustCompile(`(громк|громч|звук|тише|потише)`)
	reAskLevel    = regexp.MustCompile(`(скольк|какая|какой|уровень|чему|покажи|стоит|выставлен)`)
	reDigit       = regexp.MustCompile(`\d{1,3}`)
	reCountWords  = regexp.MustCompile(`строк|строч|штук`)
	rePercent     = regexp.MustCompile(`процент|%`)
	reWord        = regexp.MustCompile(`[а-я]+`)
	reLines       = regexp.MustCompile(`(\d{1,3})\s*(строк|строч|штук)|(строк|строч)\s*(\d{1,3})|штук\s*(\d{1,3})`)
	reUp          = regexp.MustCompile(`(прибавь|добавь|погромче|громче)`)
	reDown        = regexp.MustCompile(`(потише|тише|сбавь|убавь|слишком громко|слишком шумно)`)
	reBright      = regexp.MustCompile(`(яркост|подсветк|экран)`)
	reBrightUp    = regexp.MustCompile(`(ярче|светлее|поярче)`)
	reBrightDn    = regexp.MustCompile(`(темнее|потемнее|тусклее|притуши|приглуши|убавь)`)
	reMuteOff     = regexp.MustCompile(`(размуть|сними[а-я]*.*беззвуч|верни звук|включи звук|убери беззвуч)`)
	reMuteOn      = regexp.MustCompile(`(заглуши|выключи звук|выруби звук|беззвучн|mute)`)
	reSleepOff    = regexp.MustCompile(`(разреши|можно|можешь|пусть|пускай|сними)[а-я]*.*(спать|сон|засып|гасн)|сними запрет`)
	reSleepOn     = regexp.MustCompile(`(запрети|не давай|не дай|не гас|не выключай).*(спать|сон|засып|гасн|гаси|экран)|не спит`)
	reConnect     = regexp.MustCompile(`(подключ|переключ|перейди|перекинь|включи|вруби|сделай активной)`)
)

var numWords = map[string]int{
	"ноль": 0, "один": 1, "два": 2, "три": 3, "четыре": 4, "пять": 5,
	"шесть": 6, "семь": 7, "восемь": 8, "девять": 9, "десять": 10,
	"одиннадцать": 11, "двенадцать": 12, "тринадцать": 13, "четырнадцать": 14,
	"пятнадцать": 15, "шестнадцать": 16, "семнадцать": 17, "восемнадцать": 18,
	"девятнадцать": 19, "двадцать": 20, "тридцать": 30, "сорок": 40,
	"пятьдесят": 50, "шестьдесят": 60, "семьдесят": 70, "восемьдесят": 80,
	"девяносто": 90, "сто": 100,
}

var units = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`networkmanager`), "NetworkManager"},
	{regexp.MustCompile(`bluetooth|блютуз`), "bluetooth"},
	{regexp.MustCompile(`modemmanager`), "ModemManager"},
	{regexp.MustCompile(`avahi`), "avahi-daemon"},
	{regexp.MustCompile(`resolved`), "systemd-resolved"},
	{regexp.MustCompile(`timesyncd`), "systemd-timesyncd"},
	{regexp.MustCompile(`power-profiles`), "power-profiles-daemon"},
}

// Refusal возвращает готовый отказ, если запрос заведомо вне возможностей Кузи.
func Refusal(q string) (string, bool) {
	t := norm(q)
	switch {
	case reSSH.MatchString(t):
		return refusal, true
	case reWifiVerb.MatchString(t) && !hasNetworkName(q):
		return refusal, true
	case reAllUnits.MatchString(t):
		return refusal, true
	case reStopUnit.MatchString(t):
		return refusal, true
	case reArith.MatchString(t):
		return refusal, true
	case reLogs.MatchString(t) && reAllLogs.MatchString(t):
		return refusal, true
	case reBtToggle.MatchString(t):
		return refusal, true
	case reUnsupported.MatchString(t) && !reSupported.MatchString(t):
		return refusal, true
	}
	return "", false
}

// Preroute возвращает вызов инструмента, который можно сделать без модели.
func Preroute(q string) (string, map[string]any, bool) {
	if _, refused := Refusal(q); refused {
		return "", nil, false
	}
	t := norm(q)
	// Составные просьбы («заряд и время», «яркость, потом звук») отдаём модели:
	// одно правило не должно молча выполнить только первую часть.
	if strings.Contains(t, " и ") || strings.Contains(t, " потом ") || strings.Contains(t, " затем ") {
		return "", nil, false
	}
	unit, hasUnit := findUnit(t)

	switch {
	case reBattery.MatchString(t):
		return "battery_status", map[string]any{}, true
	case hasUnit && reLogs.MatchString(t) && !reRestart.MatchString(t):
		args := map[string]any{"unit": unit}
		if n, found := findLines(t); found {
			args["lines"] = n
		} else if n, found := numberIn(t); found && reCountWords.MatchString(t) {
			args["lines"] = n
		}
		return "service_logs", args, true
	case hasUnit && reRestart.MatchString(t):
		return "service_restart", map[string]any{"unit": unit}, true
	case reVolume.MatchString(t) && hasVolumeNumber(t):
		if n, err := firstNumber(t); err == nil {
			return "volume_set", map[string]any{"percent": n}, true
		}
	case reMuteOff.MatchString(t):
		return "mute_set", map[string]any{"mode": "off"}, true
	case reMuteOn.MatchString(t):
		return "mute_set", map[string]any{"mode": "on"}, true
	case reSleepOff.MatchString(t):
		return "nosleep_set", map[string]any{"mode": "off"}, true
	case reSleepOn.MatchString(t):
		return "nosleep_set", map[string]any{"mode": "on"}, true
	case reConnect.MatchString(t) && hasNetworkName(q):
		return "wifi_connect", map[string]any{"connection": networkName(q)}, true
	case reServices.MatchString(t) && !reRestart.MatchString(t):
		return "services_running", map[string]any{}, true
	case reWifi.MatchString(t) && !reLogs.MatchString(t) && !hasNetworkName(q):
		return "wifi_status", map[string]any{}, true
	case reBright.MatchString(t) && reDigit.MatchString(t):
		if n, err := firstDigit(t); err == nil {
			return "brightness_set", map[string]any{"percent": n}, true
		}
	case reVolume.MatchString(t) && reUp.MatchString(t):
		return "volume_step", map[string]any{"direction": "up"}, true
	case reVolume.MatchString(t) && reDown.MatchString(t):
		return "volume_step", map[string]any{"direction": "down"}, true
	case reVolume.MatchString(t) && reAskLevel.MatchString(t):
		return "volume_get", map[string]any{}, true
	case reBrightUp.MatchString(t) && !reVolume.MatchString(t):
		return "brightness_step", map[string]any{"direction": "up"}, true
	case reBrightDn.MatchString(t) && !reVolume.MatchString(t):
		return "brightness_step", map[string]any{"direction": "down"}, true
	}
	return "", nil, false
}

// Repair исправляет вызов, если модель ошиблась в известном месте.
func Repair(q, tool string, args map[string]any) (string, map[string]any) {
	t := norm(q)
	if args == nil {
		args = map[string]any{}
	}

	if reMuteOff.MatchString(t) && (tool == "volume_step" || tool == "volume_set") {
		return "mute_set", map[string]any{"mode": "off"}
	}
	if reMuteOn.MatchString(t) && (tool == "volume_step" || tool == "volume_set") {
		return "mute_set", map[string]any{"mode": "on"}
	}
	if tool == "service_logs" || tool == "service_restart" {
		if unit, ok := findUnit(t); ok {
			args["unit"] = unit
		}
		if tool == "service_logs" {
			delete(args, "lines")
			if n, ok := findLines(t); ok {
				args["lines"] = n
			}
		}
	}
	if tool == "mute_set" {
		if reMuteOff.MatchString(t) {
			args["mode"] = "off"
		} else if reMuteOn.MatchString(t) {
			args["mode"] = "on"
		}
	}
	if tool == "nosleep_set" {
		if reSleepOff.MatchString(t) {
			args["mode"] = "off"
		} else if reSleepOn.MatchString(t) {
			args["mode"] = "on"
		}
	}
	if tool == "volume_step" && !reVolume.MatchString(t) && reBright.MatchString(t) {
		tool = "brightness_step"
	}
	if tool == "volume_step" && !reVolume.MatchString(t) && reBrightDn.MatchString(t) {
		tool = "brightness_step"
	}
	if tool == "brightness_step" && reVolume.MatchString(t) && !reBright.MatchString(t) {
		tool = "volume_step"
	}
	if tool == "volume_step" {
		if reDown.MatchString(t) {
			args["direction"] = "down"
		}
		if reUp.MatchString(t) {
			args["direction"] = "up"
		}
	}
	if tool == "brightness_step" {
		if reBrightDn.MatchString(t) {
			args["direction"] = "down"
		}
		if reBrightUp.MatchString(t) {
			args["direction"] = "up"
		}
	}
	return tool, args
}

func findUnit(t string) (string, bool) {
	for _, u := range units {
		if u.re.MatchString(t) {
			return u.name, true
		}
	}
	return "", false
}

func findLines(t string) (int, bool) {
	m := reLines.FindStringSubmatch(t)
	if m == nil {
		return 0, false
	}
	for _, s := range m[1:] {
		if s == "" {
			continue
		}
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 200 {
			return n, true
		}
	}
	return 0, false
}

func firstDigit(t string) (int, error) {
	return strconv.Atoi(reDigit.FindString(t))
}

// numberIn понимает цифры и числительные словами (для «двадцать строк», «десять процентов»).
func numberIn(t string) (int, bool) {
	if m := reDigit.FindString(t); m != "" {
		n, err := strconv.Atoi(m)
		return n, err == nil
	}
	for _, w := range reWord.FindAllString(t, -1) {
		if n, ok := numWords[w]; ok {
			return n, true
		}
	}
	return 0, false
}

func hasVolumeNumber(t string) bool {
	if !reVolume.MatchString(t) {
		return false
	}
	if reDigit.MatchString(t) || rePercent.MatchString(t) {
		return true
	}
	_, ok := numberIn(t)
	return ok
}

func firstNumber(t string) (int, error) {
	n, ok := numberIn(t)
	if !ok {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

func hasNetworkName(q string) bool {
	return reNetName.MatchString(norm(q))
}

func networkName(q string) string {
	return reNetName.FindString(q)
}
