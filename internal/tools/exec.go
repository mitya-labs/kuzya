package tools

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	execTimeout = 10 * time.Second
	maxOutput   = 4000
)

// execCmd запускает команду без shell: аргументы не интерпретируются.
// Переменная — чтобы подменять в тестах.
var execCmd = func(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if len(s) > maxOutput {
		s = s[:maxOutput] + "\n…обрезано"
	}
	if ctx.Err() == context.DeadlineExceeded {
		return s, fmt.Errorf("%s: таймаут %s", name, execTimeout)
	}
	if err != nil {
		if s == "" {
			return "", fmt.Errorf("%s: %w", name, err)
		}
		return s, fmt.Errorf("%s: %w: %s", name, err, s)
	}
	return s, nil
}

func strArg(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("нужен строковый аргумент %q", key)
	}
	return strings.TrimSpace(v), nil
}

// intArg принимает число или строку с числом (маленькие модели присылают и так, и так).
func intArg(args map[string]any, key string, lo, hi int) (int, error) {
	var n int
	switch v := args[key].(type) {
	case float64:
		n = int(v)
	case string:
		p, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(v), "%"))
		if err != nil {
			return 0, fmt.Errorf("аргумент %q должен быть числом", key)
		}
		n = p
	default:
		return 0, fmt.Errorf("нужен числовой аргумент %q", key)
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("%q должен быть от %d до %d", key, lo, hi)
	}
	return n, nil
}

var (
	unitRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@:._-]{0,60}$`)
	connRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} ._-]{0,31}$`)
)

// unitName приводит имя к виду «имя.service» и проверяет его.
func unitName(s string) (string, error) {
	if !strings.HasSuffix(s, ".service") {
		s += ".service"
	}
	if !unitRe.MatchString(s) {
		return "", fmt.Errorf("недопустимое имя сервиса %q", s)
	}
	return s, nil
}
