package v1

import (
	"bytes"
	"net/http"
	"testing"
)

func TestNewRequestWrapper(t *testing.T) {
	req, err := http.NewRequest("POST", "http://example.com", bytes.NewBufferString("test body"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	wrapper := NewRequestWrapper(req)

	if wrapper.Method != "POST" {
		t.Errorf("Expected method POST, got %s", wrapper.Method)
	}
	if wrapper.URL != "http://example.com" {
		t.Errorf("Expected URL http://example.com, got %s", wrapper.URL)
	}
	if wrapper.Body != "test body" {
		t.Errorf("Expected body 'test body', got '%s'", wrapper.Body)
	}
	if wrapper.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Expected header Content-Type text/plain, got %s", wrapper.Header.Get("Content-Type"))
	}
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse(200, "hello")

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != "hello" {
		t.Errorf("Expected body 'hello', got '%s'", resp.Body)
	}
	if resp.Header == nil {
		t.Error("Expected initialized header map")
	}
}
