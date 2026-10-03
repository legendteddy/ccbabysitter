package main

import (
	"fmt"
	"math"
	"strconv"
)

// fmtTokens writes a token count short, as the page does: as it is under a
// thousand, then in thousands, millions or billions, with one decimal
// below ten of the unit.
func fmtTokens(n int64) string {
	if n < 1000 {
		if n < 0 {
			n = 0
		}
		return strconv.FormatInt(n, 10)
	}
	units := []string{"k", "M", "B"}
	i := 0
	value := float64(n) / 1000
	for i < len(units)-1 && math.Round(value) >= 1000 {
		i++
		value = float64(n) / math.Pow(1000, float64(i+1))
	}
	if value < 9.95 {
		return strconv.FormatFloat(value, 'f', 1, 64) + units[i]
	}
	return strconv.FormatFloat(math.Round(value), 'f', 0, 64) + units[i]
}

// fmtDuration writes an uptime as the page does: 45s, 12m, 3h 5m, 2d 4h.
func fmtDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	d, h, m := seconds/86400, seconds%86400/3600, seconds%3600/60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", seconds)
}
