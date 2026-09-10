package dynamic_mock_server

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "logger_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	logger, err := NewLogger(tmpFileName)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	t.Run("LogEntryStructure", func(t *testing.T) {
		details := map[string]interface{}{"key": "value", "id": 123}
		logger.Log("TestEvent", 500*time.Millisecond, details)

		// Read file
		file, err := os.Open(tmpFileName)
		if err != nil {
			t.Fatalf("Failed to open log file: %v", err)
		}
		defer file.Close()

		var entry LogEntry
		if err := json.NewDecoder(file).Decode(&entry); err != nil {
			t.Fatalf("Failed to decode log entry: %v", err)
		}

		if entry.Type != "TestEvent" {
			t.Errorf("Expected Type 'TestEvent', got '%s'", entry.Type)
		}
		if entry.Duration != "500ms" {
			t.Errorf("Expected Duration '500ms', got '%s'", entry.Duration)
		}

		d, ok := entry.Details.(map[string]interface{})
		if !ok {
			t.Errorf("Details expected to be map, got %T", entry.Details)
		}
		if d["key"] != "value" || d["id"] != float64(123) { // JSON numbers are float64
			t.Errorf("Details content mismatch: %v", d)
		}
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		// We append to the same file
		count := 50
		done := make(chan bool)

		for i := 0; i < count; i++ {
			go func(idx int) {
				logger.Log("ConcurrentEvent", 0, idx)
				done <- true
			}(i)
		}

		for i := 0; i < count; i++ {
			<-done
		}

		// Verify line count
		file, err := os.Open(tmpFileName)
		if err != nil {
			t.Fatalf("Failed to open log file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineCount := 0
		for scanner.Scan() {
			lineCount++
		}

		// 1 from previous test + 50 from this test = 51
		if lineCount != count+1 {
			t.Errorf("Expected %d lines, got %d", count+1, lineCount)
		}
	})
}
