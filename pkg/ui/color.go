package ui

import (
	"fmt"
	"strconv"
)

// ContrastTextColor returns "#fff" or "#000" based on the perceived luminance
// of the given hex color (e.g. "#ff8800" or "ff8800").
func ContrastTextColor(hex string) string {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return "#000"
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return "#000"
	}
	// Relative luminance (sRGB coefficients)
	luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if luminance > 150 {
		return "#000"
	}
	return "#fff"
}

// BadgeStyle returns an inline style string with background and contrasting text color.
func BadgeStyle(bgColor string) string {
	return fmt.Sprintf("background-color: %s; color: %s", bgColor, ContrastTextColor(bgColor))
}
