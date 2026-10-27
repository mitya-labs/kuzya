package main

import (
	"os"
	"strings"
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

// rule — тонкая линия для рамок в духе Claude/opencode.
func rule() string { return dim(strings.Repeat("─", 46)) }

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
