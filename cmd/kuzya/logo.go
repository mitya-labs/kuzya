package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// logo — Кузя в пикселях: c — коралловый, t — бирюзовый, k — тёмный, точка — пусто.
// Цвета как в assets/kuzya-mid-glasses.svg. Две пиксельные строки на строку терминала (полублоки).
const logo = `...............ccccc..............
..............ccccccccc...........
..............cccccccccc..........
.............cccccccccccc.........
.............ccccccccccccc........
............cccccccccccccc........
............cccccccccc.cccc.......
...........cccccccccccc...cc......
..........ccccccccccccc...........
.........ccccccccccccccc..........
.........cccccccccccccccc.........
........cccccccccccccccccc........
.......cccccccccccccccccccc.......
......ccccccccccccccccccccc.......
......cccccccccccccccccccccc......
.....cccccccccccccccccccccccc.....
....cccccccccccccccccccccccccc....
.......tttttttttttttttttttt.......
.......tttkkkttttttttkkkttt.......
.......ttktttkttttttktttktt.......
.......tkttkktkttttkttkktkt.......
.......tkttkktkkkkkkttkktkt.......
.......tktttttkttttktttttkt.......
.......ttktttkttttttktttktt.......
.......tttkkkttttttttkkkttt.......
.......tttttttttttttttttttt.......
.......tccttttkttttkttttcct.......
........cctttttkkkktttttcc........
.........tttttttttttttttt.........
..........tttttttttttttt..........
...........tttttttttttt...........
............tttttttttt............
.............tttttttt.............
..............tttttt..............
..............tttttt..............
...............tttt...............
................tt................
................tt................
`

type rgb struct{ r, g, b int }

var palette = map[byte]struct {
	rgb  rgb
	c256 int
}{
	'c': {rgb{255, 107, 90}, 203},
	't': {rgb{94, 234, 219}, 80},
	'k': {rgb{13, 14, 21}, 233},
}

// isTerminal — вывод идёт в терминал, а не в файл или канал.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func trueColor() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit" || strings.HasPrefix(os.Getenv("TERM"), "foot")
}

func colorCode(ch byte, fg bool, truecolor bool) string {
	p := palette[ch]
	base := 38
	if !fg {
		base = 48
	}
	if truecolor {
		return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", base, p.rgb.r, p.rgb.g, p.rgb.b)
	}
	return fmt.Sprintf("\x1b[%d;5;%dm", base, p.c256)
}

// termRows возвращает число строк терминала или 0, если узнать не удалось.
func termRows(f *os.File) int {
	var ws struct{ Row, Col, X, Y uint16 }
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if e != 0 {
		return 0
	}
	return int(ws.Row)
}

// shrink уменьшает сетку логотипа в k раз; тёмные пиксели (очки, глаза) весят больше, чтобы детали не пропадали.
func shrink(rows []string, k int) []string {
	if k <= 1 {
		return rows
	}
	h, w := len(rows), len(rows[0])
	var out []string
	for y := 0; y < h; y += k {
		var b strings.Builder
		for x := 0; x < w; x += k {
			cnt := map[byte]float64{}
			total := 0
			for dy := 0; dy < k && y+dy < h; dy++ {
				for dx := 0; dx < k && x+dx < w; dx++ {
					total++
					ch := rows[y+dy][x+dx]
					if ch == 'k' {
						cnt[ch] += 1.5
					} else {
						cnt[ch]++
					}
				}
			}
			best, bestN := byte('.'), 0.0
			for _, ch := range []byte{'c', 't', 'k', '.'} {
				if cnt[ch] > bestN {
					best, bestN = ch, cnt[ch]
				}
			}
			if cnt['.'] > float64(total)/2 {
				best = '.'
			}
			b.WriteByte(best)
		}
		out = append(out, b.String())
	}
	return out
}

// pickLogo выбирает размер логотипа под avail строк терминала (0 — неизвестно): целиком, если есть место,
// иначе уменьшенный в 2–4 раза; nil, если места нет. Под логотипом остаются две строки текста, приглашение и запас.
func pickLogo(avail int) []string {
	const reserve = 5
	full := strings.Split(strings.TrimRight(logo, "\n"), "\n")
	for _, k := range []int{1, 2, 3, 4} {
		g := shrink(full, k)
		if len(g)%2 == 1 {
			g = append(g, strings.Repeat(".", len(g[0])))
		}
		if avail == 0 || len(g)/2+reserve <= avail {
			return g
		}
	}
	return nil
}

// printLogo рисует логотип по размеру терминала: целиком, если есть место, иначе уменьшенный; без терминала,
// с NO_COLOR, TERM=dumb или KUZYA_NOLOGO не рисует ничего. Под логотипом остаётся место на текст и приглашение.
func printLogo(w io.Writer, f *os.File) {
	if !isTerminal(f) || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || os.Getenv("KUZYA_NOLOGO") != "" {
		return
	}
	rows := pickLogo(termRows(f))
	if rows == nil {
		return
	}
	tc := trueColor()
	for y := 0; y < len(rows); y += 2 {
		var b strings.Builder
		for x := 0; x < len(rows[y]); x++ {
			top, bot := rows[y][x], rows[y+1][x]
			switch {
			case top == '.' && bot == '.':
				b.WriteByte(' ')
			case bot == '.':
				b.WriteString(colorCode(top, true, tc) + "▀\x1b[0m")
			case top == '.':
				b.WriteString(colorCode(bot, true, tc) + "▄\x1b[0m")
			default:
				b.WriteString(colorCode(top, true, tc) + colorCode(bot, false, tc) + "▀\x1b[0m")
			}
		}
		fmt.Fprintln(w, b.String())
	}
}
