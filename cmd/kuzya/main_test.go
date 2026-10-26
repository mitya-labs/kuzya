package main

import "testing"

func TestIsYes(t *testing.T) {
	for _, s := range []string{"y", "Y", "yes", "д", "Д", "да", "Да", " да \n"} {
		if !isYes(s) {
			t.Errorf("%q должно быть согласием", s)
		}
	}
	for _, s := range []string{"", "n", "N", "н", "Н", "нет", "no", "ага", "yy", "\n"} {
		if isYes(s) {
			t.Errorf("%q должно быть отказом", s)
		}
	}
}
