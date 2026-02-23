package logging

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger handles connection and session logging
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	enabled  bool
	filePath string
}

// NewLogger creates a new logger
func NewLogger() *Logger {
	return &Logger{}
}

// Start begins logging to a file
func (l *Logger) Start(filePath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = f
	l.filePath = filePath
	l.enabled = true

	// Write header
	header := fmt.Sprintf("\n=== Genpilot Session Log ===\n=== Started: %s ===\n\n",
		time.Now().Format("2006-01-02 15:04:05"))
	l.file.WriteString(header)

	return nil
}

// Stop stops logging
func (l *Logger) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.WriteString(fmt.Sprintf("\n=== Session ended: %s ===\n",
			time.Now().Format("2006-01-02 15:04:05")))
		l.file.Close()
		l.file = nil
	}
	l.enabled = false
}

// Write writes a log entry
func (l *Logger) Write(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled || l.file == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	l.file.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, text))
}

// WriteRaw writes raw text without timestamp
func (l *Logger) WriteRaw(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled || l.file == nil {
		return
	}

	l.file.WriteString(text)
}

// IsEnabled returns whether logging is active
func (l *Logger) IsEnabled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enabled
}

// GetFilePath returns the current log file path
func (l *Logger) GetFilePath() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.filePath
}
