package main

import "testing"

func TestPickLogoFitsTerminal(t *testing.T) {
	cases := []struct{ avail, wantRows int }{
		{0, 16},  // размер неизвестен: целиком
		{40, 16}, // большой терминал
		{20, 16}, // альбомный экран телефона: ровно влезает целиком
		{19, 8},  // на строку меньше: уменьшаем в 2 раза
		{12, 8},
		{10, 6},
		{9, 4},
		{7, 0}, // места нет
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
		if g != nil && c.avail > 0 && got+4 > c.avail {
			t.Errorf("avail=%d: логотип не влезает (%d строк)", c.avail, got)
		}
	}
}
