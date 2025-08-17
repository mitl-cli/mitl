// Package terminal provides terminal output utilities.
package terminal

import (
	"fmt"
	"os"
)

// Color codes for terminal output
const (
	Reset  = "\033[0m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

// IsTerminal checks if output is to a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Colorize returns text with color codes if terminal supports it
func Colorize(color, text string) string {
	if !IsTerminal() || os.Getenv("NO_COLOR") != "" {
		return text
	}
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

// Success prints green text
func Success(text string) string {
	return Colorize(Green, text)
}

// Error prints red text
func Error(text string) string {
	return Colorize(Red, text)
}

// Warning prints yellow text
func Warning(text string) string {
	return Colorize(Yellow, text)
}

// Info prints cyan text
func Info(text string) string {
	return Colorize(Cyan, text)
}

// BoldText returns bold text
func BoldText(text string) string {
	if !IsTerminal() || os.Getenv("NO_COLOR") != "" {
		return text
	}
	return fmt.Sprintf("%s%s%s", Bold, text, Reset)
}
