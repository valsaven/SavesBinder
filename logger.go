package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger provides thread-safe logging to both file and optional callback.
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	writer   io.Writer
	callback func(level, msg string)
}

var logger *Logger

// InitLogger initializes the global logger with the specified log file path.
// If callback is provided, log messages are also sent to it (e.g., for UI display).
func InitLogger(logPath string, callback func(level, msg string)) error {
	if logPath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logger = &Logger{
		file:     f,
		writer:   f,
		callback: callback,
	}

	return nil
}

// Close closes the log file.
func CloseLogger() {
	if logger != nil && logger.file != nil {
		logger.file.Close()
	}
}

// Logf writes a formatted log message with timestamp and level.
func Logf(level, format string, args ...interface{}) {
	if logger == nil {
		return
	}

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullMsg := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	logger.mu.Lock()
	defer logger.mu.Unlock()

	if logger.writer != nil {
		logger.writer.Write([]byte(fullMsg))
	}

	if logger.callback != nil {
		logger.callback(level, msg)
	}
}

// Error logs an error-level message.
func Error(format string, args ...interface{}) {
	Logf("ERROR", format, args...)
}

// Warn logs a warning-level message.
func Warn(format string, args ...interface{}) {
	Logf("WARN", format, args...)
}

// Info logs an info-level message.
func Info(format string, args ...interface{}) {
	Logf("INFO", format, args...)
}

// Debug logs a debug-level message.
func Debug(format string, args ...interface{}) {
	Logf("DEBUG", format, args...)
}
