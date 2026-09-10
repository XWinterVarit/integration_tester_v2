package v1

import (
	"fmt"
	"log"
	"sync"
)

// LogType defines the category of the log.
type LogType string

const (
	LogTypeStage   LogType = "Stage"
	LogTypeDB      LogType = "DB"
	LogTypeRedis   LogType = "Redis"
	LogTypeRequest LogType = "Request"
	LogTypeMock    LogType = "Mock"
	LogTypeApp     LogType = "App"
	LogTypeExpect  LogType = "Expect"
	LogTypeError   LogType = "Error"
	LogTypeInfo    LogType = "Info"
)

// LogEntry represents a single log event.
type LogEntry struct {
	Type    LogType
	Summary string
	Detail  string
}

// LogHandler is a function that handles log entries (e.g., UI updater).
type LogHandler func(entry LogEntry)

var (
	logHandlers []LogHandler
	logMu       sync.Mutex
)

// RegisterLogHandler adds a handler for log events.
func RegisterLogHandler(h LogHandler) {
	logMu.Lock()
	defer logMu.Unlock()
	logHandlers = append(logHandlers, h)
}

// Log records a log entry and notifies handlers.
func Log(t LogType, summary string, detail string) {
	// 1. Print to standard console for debugging/history
	if detail != "" {
		log.Printf("[%s] %s - %s", t, summary, detail)
	} else {
		log.Printf("[%s] %s", t, summary)
	}

	// 2. Notify handlers (UI)
	entry := LogEntry{
		Type:    t,
		Summary: summary,
		Detail:  detail,
	}

	logMu.Lock()
	defer logMu.Unlock()
	for _, h := range logHandlers {
		// Call handler directly. Handlers should handle concurrency (e.g. fyne.Do)
		h(entry)
	}
}

// Logf is a helper to log formatted simple info.
func Logf(t LogType, format string, v ...interface{}) {
	Log(t, fmt.Sprintf(format, v...), "")
}
