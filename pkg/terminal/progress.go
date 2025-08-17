package terminal

import (
	"fmt"
	"strings"
	"time"
)

// ProgressBar represents a terminal progress bar
type ProgressBar struct {
	total   int
	current int
	width   int
	prefix  string
	start   time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		total:  total,
		width:  40,
		prefix: prefix,
		start:  time.Now(),
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int) {
	p.current = current
	p.render()
}

// Increment increments the progress by 1
func (p *ProgressBar) Increment() {
	p.current++
	p.render()
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.current = p.total
	p.render()
	fmt.Println()
}

func (p *ProgressBar) render() {
	if !IsTerminal() {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	elapsed := time.Since(p.start).Seconds()
	rate := float64(p.current) / elapsed

	fmt.Printf("\r%s [%s] %d/%d (%.0f/s)", p.prefix, bar, p.current, p.total, rate)
}
