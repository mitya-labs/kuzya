package main

import "testing"

func TestPickLogoFitsTerminal(t *testing.T) {
	cases := []struct{ avail, wantRows int }{
		{0, 19},  // размер неизвестен: целиком
		{40, 19}, // большой терминал
		{24, 19},
		{20, 10}, // альбомный экран телефона: уменьшаем в 2 раза
		{14, 7},
		{11, 5},
		{9, 0}, // места нет
	}
	for _, c := range cases {
		g := pickLogo(c.avail)
		got := len(g) / 2
		if g == nil {
			got = 0
		}
		if got != c.wantRows {
			t.Errorf("avail=%d: %d строк, ожидали %d", c.avail, got, c.wantRows)
		}
		if g != nil && c.avail > 0 && got+5 > c.avail {
			t.Errorf("avail=%d: логотип не влезает (%d строк)", c.avail, got)
		}
	}
}
