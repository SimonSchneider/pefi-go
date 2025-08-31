package ui

import (
	"math"
	"strconv"
)

func FormatWithThousands(val float64) string {
	// Round the value to the nearest integer
	rounded := math.Round(val)
	s := strconv.FormatInt(int64(rounded), 10)
	n := len(s)
	neg := false
	if n > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
		n--
	}
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var out []byte
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	out = append(out, s[:pre]...)
	for i := pre; i < n; i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
