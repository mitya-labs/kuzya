package main

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// colorOn — можно ли раскрашивать вывод: терминал, не NO_COLOR и не TERM=dumb.
var colorOn = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

func paint(code, s string) string {
	if !colorOn || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func dim(s string) string    { return paint("2", s) }
func bold(s string) string   { return paint("1", s) }
func cyan(s string) string   { return paint("1;36", s) }
func yellow(s string) string { return paint("1;33", s) }
func red(s string) string    { return paint("1;31", s) }
func green(s string) string  { return paint("1;32", s) }

// clearScreen очищает видимый экран перед запуском REPL.
// Отключается через KUZYA_NOCLEAR, TERM=dumb и при выводе не в терминал.
func clearScreen() {
	if !isTerminal(os.Stdout) || os.Getenv("TERM") == "dumb" || os.Getenv("KUZYA_NOCLEAR") != "" {
		return
	}
	print("\x1b[H\x1b[2J")
}

// rule — тонкая линия во всю ширину терминала (в духе Claude/opencode).
func rule() string {
	w := 46
	if tw, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && tw > 20 {
		w = tw - 1
	}
	return dim(strings.Repeat("─", w))
}

// modelLabel — короткое имя текущей модели для шапки и подсказок.
func modelLabel() string {
	m := os.Getenv("KUZYA_MODEL")
	if m == "" {
		m = modelLarge
	}
	m = strings.TrimPrefix(m, "nafanya-")
	m = strings.TrimSuffix(m, "-Q4_K_M")
	return m
}
