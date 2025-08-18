// Package logger provides simple structured logging with levels and optional colors.
// It supports -v (verbose) and --debug flags. In debug mode, logs are also written
// to $HOME/.mitl/logs/mitl-YYYY-MM-DD.log for troubleshooting.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelVerbose
	LevelDebug
)

// Logger provides structured logging
type Logger struct {
	mu      sync.Mutex
	level   Level
	output  io.Writer
	file    *os.File
	colors  bool
	timings map[string]time.Time
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Initialize sets up the global logger
func Initialize(verbose, debug bool) {
	once.Do(func() {
		level := LevelInfo
		if verbose {
			level = LevelVerbose
		}
		if debug {
			level = LevelDebug
		}

		defaultLogger = &Logger{
			level:   level,
			output:  os.Stderr,
			colors:  isTerminal(),
			timings: make(map[string]time.Time),
		}

		// Also log to file in debug mode
		if debug {
			logDir := os.ExpandEnv("$HOME/.mitl/logs")
			_ = os.MkdirAll(logDir, 0o755)
			logFile := filepath.Join(logDir, fmt.Sprintf("mitl-%s.log", time.Now().Format("2006-01-02")))
			if file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				defaultLogger.file = file
				Debugf("Logging to %s", logFile)
			}
		}
	})
}

// Close closes any resources used by the logger
func Close() {
	if defaultLogger != nil && defaultLogger.file != nil {
		_ = defaultLogger.file.Close()
	}
}

// Info logs at info level (always shown)
func Info(msg string) {
	if defaultLogger != nil {
		defaultLogger.log(LevelInfo, msg)
	}
}
func Infof(format string, args ...interface{}) { Info(fmt.Sprintf(format, args...)) }

// Verbose logs at verbose level (shown with -v)
func Verbose(msg string) {
	if defaultLogger != nil {
		defaultLogger.log(LevelVerbose, msg)
	}
}
func Verbosef(format string, args ...interface{}) { Verbose(fmt.Sprintf(format, args...)) }

// Debug logs at debug level (shown with --debug)
func Debug(msg string) {
	if defaultLogger != nil {
		defaultLogger.log(LevelDebug, msg)
	}
}
func Debugf(format string, args ...interface{}) { Debug(fmt.Sprintf(format, args...)) }

// Warn logs warnings
func Warn(msg string) {
	if defaultLogger != nil {
		defaultLogger.log(LevelWarn, msg)
	}
}
func Warnf(format string, args ...interface{}) { Warn(fmt.Sprintf(format, args...)) }

// Error logs errors (always shown)
func Error(msg string) {
	if defaultLogger != nil {
		defaultLogger.log(LevelError, msg)
	}
}
func Errorf(format string, args ...interface{}) { Error(fmt.Sprintf(format, args...)) }

// StartTimer begins timing an operation
func StartTimer(operation string) {
	if defaultLogger != nil && defaultLogger.level >= LevelVerbose {
		defaultLogger.mu.Lock()
		defaultLogger.timings[operation] = time.Now()
		defaultLogger.mu.Unlock()
		Verbosef("⏱  Starting: %s", operation)
	}
}

// EndTimer logs the duration of an operation
func EndTimer(operation string) {
	if defaultLogger != nil && defaultLogger.level >= LevelVerbose {
		defaultLogger.mu.Lock()
		if start, ok := defaultLogger.timings[operation]; ok {
			delete(defaultLogger.timings, operation)
			defaultLogger.mu.Unlock()
			Verbosef("✓ Completed %s in %v", operation, time.Since(start))
		} else {
			defaultLogger.mu.Unlock()
		}
	}
}

// log writes a log message with level, timestamp and optional caller
func (l *Logger) log(level Level, msg string) {
	if level > l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	var prefix, color string
	switch level {
	case LevelError:
		prefix, color = "ERROR", "\033[31m" // red
	case LevelWarn:
		prefix, color = "WARN", "\033[33m" // yellow
	case LevelInfo:
		prefix, color = "INFO", "\033[32m" // green
	case LevelVerbose:
		prefix, color = "VERBOSE", "\033[36m" // cyan
	case LevelDebug:
		prefix, color = "DEBUG", "\033[35m" // magenta
	}

	caller := ""
	if level == LevelDebug {
		if _, file, line, ok := runtime.Caller(3); ok {
			caller = fmt.Sprintf(" [%s:%d]", filepath.Base(file), line)
		}
	}

	var output string
	if l.colors {
		// Colorize the level prefix only
		output = fmt.Sprintf("[%s] %s%s%s: %s\n", timestamp, color, prefix, "\033[0m", strings.TrimRight(msg, "\n"))
		if caller != "" {
			output = fmt.Sprintf("[%s] %s%s%s%s: %s\n", timestamp, color, prefix, "\033[0m", caller, strings.TrimRight(msg, "\n"))
		}
	} else {
		output = fmt.Sprintf("[%s] %s%s: %s\n", timestamp, prefix, caller, strings.TrimRight(msg, "\n"))
	}

	fmt.Fprint(l.output, output)
	if l.file != nil {
		fmt.Fprint(l.file, output)
	}
}

func isTerminal() bool {
	fi, _ := os.Stderr.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}
