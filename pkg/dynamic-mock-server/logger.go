package dynamic_mock_server

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      string      `json:"type"`
	Duration  string      `json:"duration,omitempty"`
	Details   interface{} `json:"details"`
}

type Logger struct {
	mu      sync.Mutex
	encoder *json.Encoder
	file    *os.File
}

func NewLogger(filename string) (*Logger, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

func NewConsoleLogger() *Logger {
	return &Logger{
		encoder: json.NewEncoder(os.Stdout),
	}
}

func (l *Logger) Log(logType string, duration time.Duration, details interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      logType,
		Details:   details,
	}
	if duration > 0 {
		entry.Duration = duration.String()
	}

	if err := l.encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
	}
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
