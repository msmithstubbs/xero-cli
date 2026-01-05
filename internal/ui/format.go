package ui

import (
	"fmt"
	"strings"
)

func Pad(value string, width int) string {
	if len(value) > width {
		if width <= 3 {
			return value[:width]
		}
		return value[:width-3] + "..."
	}
	if len(value) < width {
		return value + strings.Repeat(" ", width-len(value))
	}
	return value
}

func FormatRow(columns ...string) string {
	return strings.Join(columns, " | ")
}

func PrintHeaderLine(width int) {
	fmt.Println(strings.Repeat("=", width))
}
