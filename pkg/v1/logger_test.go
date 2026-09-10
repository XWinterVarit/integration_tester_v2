package v1

import (
	"sync"
	"testing"
)

func TestLogger(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	var captured LogEntry
	handler := func(e LogEntry) {
		captured = e
		wg.Done()
	}

	logHandlers = nil                    // Clear previous handlers
	defer func() { logHandlers = nil }() // Clear after test
	RegisterLogHandler(handler)

	Log(LogTypeInfo, "Test Summary", "Test Detail")

	wg.Wait()

	if captured.Type != LogTypeInfo {
		t.Errorf("Expected LogTypeInfo, got %s", captured.Type)
	}
	if captured.Summary != "Test Summary" {
		t.Errorf("Expected 'Test Summary', got '%s'", captured.Summary)
	}
	if captured.Detail != "Test Detail" {
		t.Errorf("Expected 'Test Detail', got '%s'", captured.Detail)
	}
}

func TestLogf(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	var captured LogEntry
	handler := func(e LogEntry) {
		captured = e
		wg.Done()
	}

	// Registering another handler adds to the list, doesn't replace.
	// We need to be careful if global state persists.
	// Since tests run in parallel or sequentially in same process, RegisterLogHandler appends.
	// To isolate, we might just assume it works or clear handlers if possible (not exposed).
	// But RegisterLogHandler appends. So the previous handler might also run.
	// Let's just register a new one and wait for it.
	logHandlers = nil                    // Clear previous handlers
	defer func() { logHandlers = nil }() // Clear after test
	RegisterLogHandler(handler)

	Logf(LogTypeInfo, "Hello %s", "World")

	wg.Wait()

	if captured.Summary != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", captured.Summary)
	}
}
