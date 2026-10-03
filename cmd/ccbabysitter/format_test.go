package main

import "testing"

func TestFormatsMatchThePage(t *testing.T) {
	for n, want := range map[int64]string{0: "0", 999: "999", 1000: "1.0k", 9940: "9.9k", 9950: "10k", 12300: "12k", 999499: "999k", 999500: "1.0M", 4500000: "4.5M", 2000000000: "2.0B"} {
		if got := fmtTokens(n); got != want {
			t.Errorf("fmtTokens(%d) = %q, want %q", n, got, want)
		}
	}
	for s, want := range map[int64]string{0: "0s", 45: "45s", 720: "12m", 11100: "3h 5m", 187200: "2d 4h"} {
		if got := fmtDuration(s); got != want {
			t.Errorf("fmtDuration(%d) = %q, want %q", s, got, want)
		}
	}
}
