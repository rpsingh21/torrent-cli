package component

import (
	"fmt"
	"strings"
	"time"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGT"[exp])
}

func formatSpeed(b int64) string {
	if b <= 0 {
		return "0 B/s"
	}
	return formatBytes(b) + "/s"
}

// padRight truncates or pads a plain ASCII/UTF-8 string to an exact visual column width
func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		if width > 1 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	if len(runes) < width {
		return s + strings.Repeat(" ", width-len(runes))
	}
	return s
}

func formatETA(seconds int64) string {
	if seconds <= 0 {
		return "-"
	}
	if seconds > 356400 { // > 99 hours
		return "∞"
	}
	d := time.Duration(seconds) * time.Second
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatPeerAddr(addr string, width int) string {
	if len(addr) <= width {
		return addr + strings.Repeat(" ", width-len(addr))
	}

	if width <= 10 {
		return padRight(addr, width)
	}

	// Keep the last ":port" + last 4 hex chars of the IP intact
	tailLen := min(11, width/2) // e.g., "7334]:65535" (11 chars)
	headLen := width - tailLen - 1

	return addr[:headLen] + "…" + addr[len(addr)-tailLen:]
}
